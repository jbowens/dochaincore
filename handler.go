package dochaincore

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

const indexPageHTML = `<!DOCTYPE html>
<html>
	<head>
		<title>One-Click Chain Core DigitalOcean</title>
		<link rel="stylesheet" href="/static/style.css">
	</head>
	<body>
		<div id="content">
			<div id="header">
				<a href="https://chain.com"><img src="https://chain.com/docs/images/chain-brand.png" alt="Chain" class="mainsite" /></a>
			</div>

			<p>Install <a href="https://chain.com">Chain Core</a> on a DigitalOcean droplet. This installer creates a new 1gb droplet and a 100gb block storage volume on your DigitalOcean account. It installs Chain Core on the droplet using the attached volume for storage. The approximate cost on DigitalOcean is $20/month.</p>

			<a href="{{.InstallLink}}" class="btn-success" id="install-btn">Install Chain Core</a>
  		</div>
	</body>
</html>`
const progressPageHTML = `
<!DOCTYPE html>
<html>
	<head>
		<title>One-Click Chain Core DigitalOcean</title>
		<link rel="stylesheet" href="/static/style.css">
		<script src="https://chain.com/docs/js/jquery.min.js"></script>
		<script type="text/javascript">
			window.installID = "{{.InstallID}}";
		</script>
		<script src="/static/progress.js"></script>
	</head>
	<body>
		<div id="content">
			<div id="header">
				<a href="https://chain.com"><img src="https://chain.com/docs/images/chain-brand.png" alt="Chain" class="mainsite" /></a>
			</div>

			<div id="progress-bar">
				<div id="current-progress"></div>
			</div>
			<p id="status-line">Initializing droplet&hellip;</p>

			<div id="core-info">
				<code id="client-token">
				</code>
				<a href="http://:1999/dashboard" class="btn-success" id="open-dashboard">Open dashboard</a>
			</div>
		</div>
	</body>
</html>
`

func Handler(oauthClientID, oauthClientSecret, host string) http.Handler {
	h := &handler{
		oauthClientID:     oauthClientID,
		oauthClientSecret: oauthClientSecret,
		host:              host,
		progressTmpl:      template.Must(template.New("progresspage").Parse(progressPageHTML)),
		indexTmpl:         template.Must(template.New("index").Parse(indexPageHTML)),
		installs:          make(map[string]*install),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status/", h.status)
	mux.HandleFunc("/progress", h.progressPage)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", h.index)
	return mux
}

type handler struct {
	oauthClientID     string
	oauthClientSecret string
	host              string
	progressTmpl      *template.Template
	indexTmpl         *template.Template

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

	tmplData := struct {
		InstallLink string
	}{
		InstallLink: u.String(),
	}
	h.indexTmpl.Execute(rw, tmplData)
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
		return
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
		return
	}
	if decodedResponse.AccessToken == "" {
		http.Error(rw, "missing access token", http.StatusBadRequest)
		return
	}
	if decodedResponse.Scope != "read write" {
		http.Error(rw, "need read write OAuth scope", http.StatusBadRequest)
		return
	}

	go curr.init(decodedResponse.AccessToken, state)

	tmplData := struct {
		InstallID string
	}{
		InstallID: state,
	}
	err = h.progressTmpl.Execute(rw, tmplData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "executing template: %s", err.Error())
	}
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
	IPAddress   string `json:"ip_address"`
	accessToken string
	c           *Core
}

func (i *install) setStatus(status string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Status = status
}

func (i *install) init(accessToken, state string) {
	defer revoke(accessToken)

	var core *Core
	var err error

	defer func() {
		if err != nil {
			i.setStatus(err.Error())
		}
	}()

	// Start deploying and create the droplet.
	core, err = Deploy(accessToken, DropletName("chain-core-"+state[:6]))
	if err != nil {
		return
	}

	i.mu.Lock()
	i.IPAddress = core.IPv4Address
	i.accessToken = accessToken
	i.c = core
	i.Status = "waiting for ssh"
	i.mu.Unlock()

	err = WaitForSSH(core)
	if err != nil {
		return
	}

	i.setStatus("waiting for http")
	err = WaitForHTTP(core)
	if err != nil {
		return
	}

	i.setStatus("creating client token")
	token, err := CreateClientToken(core)
	if err != nil {
		return
	}

	i.mu.Lock()
	i.Status = "done"
	i.ClientToken = token
	i.c = nil // garbage collect the SSH keys
	i.mu.Unlock()
}

func revoke(accessToken string) error {
	body := strings.NewReader(url.Values{"token": {accessToken}}.Encode())
	req, err := http.NewRequest("POST", "https://cloud.digitalocean.com/v1/oauth/revoke", body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke endpoint returned %d status code", resp.StatusCode)
	}
	return nil
}
