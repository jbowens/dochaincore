package dochaincore

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"sync"
)

func Handler(oauthClientID, oauthClientSecret, host string) http.Handler {
	h := &handler{
		oauthClientID:     oauthClientID,
		oauthClientSecret: oauthClientSecret,
		host:              host,
		installs:          make(map[string]*install),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", h.status)
	mux.HandleFunc("/progress", h.progressPage)
	mux.HandleFunc("/", h.index)
	return mux
}

type handler struct {
	oauthClientID     string
	oauthClientSecret string
	host              string

	installMu sync.Mutex
	installs  map[string]*install
}

func (h *handler) index(rw http.ResponseWriter, req *http.Request) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	installID := hex.EncodeToString(b)

	h.installMu.Lock()
	h.installs[installID] = &install{Status: "pending auth"}
	h.installMu.Unlock()

	vals := make(url.Values)
	vals.Set("response_type", "code")
	vals.Set("client_id", h.oauthClientID)
	vals.Set("state", installID)
	vals.Set("scope", "read write")
	vals.Set("redirect_uri", h.host+"/progress")
	u := url.URL{
		Scheme:   "https",
		Host:     "cloud.digitalocean.com",
		Path:     "/v1/oauth/authorize",
		RawQuery: vals.Encode(),
	}
	http.Redirect(rw, req, u.String(), http.StatusSeeOther)
}

func (h *handler) progressPage(rw http.ResponseWriter, req *http.Request) {
	code, state := req.FormValue("code"), req.FormValue("state")
	if code == "" || state == "" {
		http.Error(rw, "invalid oauth2 grant", http.StatusBadRequest)
		return
	}

	h.installMu.Lock()
	curr := h.installs[state]
	h.installMu.Unlock()
	if curr == nil {
		http.Error(rw, "invalid oauth2 state", http.StatusBadRequest)
		return
	}

	// Claim the code grant
	vals := make(url.Values)
	vals.Set("grant_type", "authorization_code")
	vals.Set("code", code)
	vals.Set("client_id", h.oauthClientID)
	vals.Set("client_secret", h.oauthClientSecret)
	vals.Set("redirect_uri", h.host+"/progress")
	u := url.URL{
		Scheme:   "https",
		Host:     "cloud.digitalocean.com",
		Path:     "/v1/oauth/token",
		RawQuery: vals.Encode(),
	}
	resp, err := http.Post(u.String(), "application/x-www-form-urlencoded", nil)
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	var decodedResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"bearer"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Info         struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			UUID  string `json:"uuid"`
		} `json:"info"`
	}
	err = json.NewDecoder(resp.Body).Decode(&decodedResponse)
	if err != nil {
		http.Error(rw, "err decoding access token grant", http.StatusBadRequest)
	}
	if decodedResponse.AccessToken == "" {
		http.Error(rw, "missing access token", http.StatusBadRequest)
	}
	if decodedResponse.Scope != "read write" {
		http.Error(rw, "need read write OAuth scope", http.StatusBadRequest)
	}

	// Start deploying and create the droplet.
	core, err := Deploy(decodedResponse.AccessToken)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	go curr.init(decodedResponse.AccessToken, core)

	// TODO(jackson): Print a webpage with a React app here.
	rw.Write([]byte(state))
}

func (h *handler) status(rw http.ResponseWriter, req *http.Request) {
	id := path.Base(req.URL.Path)
	h.installMu.Lock()
	curr := h.installs[id]
	h.installMu.Unlock()

	if curr == nil {
		http.NotFound(rw, req)
		return
	}

	// Marshal to a buffer first so that a really slow request can't
	// keep curr.mu locked indefinitely.
	var buf bytes.Buffer
	curr.mu.Lock()
	_ = json.NewEncoder(&buf).Encode(curr)
	curr.mu.Unlock()

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(buf.Bytes())
}

type install struct {
	mu          sync.Mutex
	Status      string `json:"status"`
	ClientToken string `json:"client_token"`
	accessToken string
	c           *Core
}

func (i *install) setStatus(status string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Status = status
}

func (i *install) init(accessToken string, c *Core) {
	i.mu.Lock()
	i.accessToken = accessToken
	i.c = c
	i.Status = "waiting for ssh"
	i.mu.Unlock()

	defer func() {
		i.mu.Lock()
		if i.Status != "done" {
			i.Status = "failed"
		}
		i.mu.Unlock()
	}()

	err := WaitForSSH(c)
	if err != nil {
		return
	}

	i.setStatus("waiting for http")
	err = WaitForHTTP(c)
	if err != nil {
		return
	}

	i.setStatus("creating client token")
	token, err := CreateClientToken(c)
	if err != nil {
		return
	}

	i.mu.Lock()
	i.Status = "done"
	i.ClientToken = token
	i.c = nil // garbage collect the SSH keys
	i.mu.Unlock()
}
