package dochaincore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type Core struct {
	DropletID   int
	IPv4Address string
	IPv6Address string

	ssh *sshKeyPair
}

type Option func(*options)

func DropletName(name string) Option {
	return func(opt *options) { opt.dropletName = name }
}

func DropletRegion(region string) Option {
	return func(opt *options) { opt.dropletRegion = region }
}

func DropletSize(size string) Option {
	return func(opt *options) { opt.dropletSize = size }
}

func VolumeSizeGB(gb int64) Option {
	return func(opt *options) { opt.volumeSize = gb }
}

type options struct {
	dropletName   string
	dropletRegion string
	dropletSize   string
	volumeSize    int64
}

// Deploy builds and deploys an instance of Chain Core on a DigitalOcean
// droplet. It requires a DigitalOcean access token and optionally takes
// a variadic number of configuration options.
func Deploy(ctx context.Context, accessToken string, opts ...Option) (*Core, error) {
	opt := options{
		dropletName:   "chain-core",
		dropletRegion: "sfo2",
		dropletSize:   "1gb",
		volumeSize:    100,
	}
	for _, o := range opts {
		o(&opt)
	}

	keypair, err := createSSHKeyPair()
	if err != nil {
		return nil, err
	}

	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	))
	client := godo.NewClient(oauthClient)

	// Blockchains require storage. Make a volume that we can attach
	// to the droplet. Chain Core will store blockchain data on the volume.
	volume, _, err := client.Storage.CreateVolume(ctx, &godo.VolumeCreateRequest{
		Region:        opt.dropletRegion,
		Name:          fmt.Sprintf("%s-storage", opt.dropletName),
		Description:   "Chain Core storage volume",
		SizeGigaBytes: opt.volumeSize,
	})
	if err != nil {
		return nil, err
	}

	// Query all the SSH keys on the account so we can include them
	// in the droplet.
	sshKeys, _, err := client.Keys.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Build user data to initialize the droplet as a Chain Core
	// instance.
	userData, err := buildUserData(keypair)
	if err != nil {
		return nil, err
	}

	// Launch the DigitalOcean droplet.
	createRequest := &godo.DropletCreateRequest{
		Name:       opt.dropletName,
		Region:     opt.dropletRegion,
		Size:       opt.dropletSize,
		IPv6:       true,
		Monitoring: true,
		UserData:   userData,
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-17-04-x64",
		},
		Volumes: []godo.DropletCreateVolume{
			{ID: volume.ID},
		},
	}
	for _, key := range sshKeys {
		keyToAdd := godo.DropletCreateSSHKey{ID: key.ID}
		createRequest.SSHKeys = append(createRequest.SSHKeys, keyToAdd)
	}

	droplet, _, err := client.Droplets.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	core := &Core{
		DropletID: droplet.ID,
		ssh:       keypair,
	}

	// A just-created droplet won't have any of the network IP addresses
	// quite yet. We have to poll until the droplet is provisioned and
	// they're populated.
	for attempt := 1; core.IPv4Address == "" || core.IPv6Address == ""; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second): /// linear backoff
		}

		droplet, _, err := client.Droplets.Get(ctx, core.DropletID)
		if err != nil {
			return nil, err
		}

		for _, nv4 := range droplet.Networks.V4 {
			if nv4.IPAddress != "" {
				core.IPv4Address = nv4.IPAddress
			}
		}
		for _, nv6 := range droplet.Networks.V6 {
			if nv6.IPAddress != "" {
				core.IPv6Address = nv6.IPAddress
			}
		}
		if attempt >= 10 {
			return nil, fmt.Errorf("timeout waiting for provisioning of droplet %d", core.DropletID)
		}
	}
	return core, nil
}

// WaitForSSH waits until port 22 on the provided Chain Core's host is opened.
func WaitForSSH(ctx context.Context, c *Core) error {
	return waitForPort(ctx, c.IPv4Address, 22)
}

// WaitForHTTP waits until Chain Core begins listening on port 1999.
func WaitForHTTP(ctx context.Context, c *Core) error {
	return waitForPort(ctx, c.IPv4Address, 1999)
}

func waitForPort(ctx context.Context, host string, port int) (err error) {
	var conn net.Conn
	for conn == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		}
	}
	conn.Close()
	return err
}

// CreateClientToken sets up a Chain Core client token for the
// provided Core.
func CreateClientToken(ctx context.Context, c *Core) (string, error) {
	const createClientToken = `
	docker exec dochaincore /usr/bin/chain/corectl create-token do client-readwrite
	`
	// TODO(jackson): remove the ssh key from authorized_keys before
	// closing the SSH session.

	session, err := connect(ctx, c.IPv4Address, c.ssh)
	if err != nil {
		return "", err
	}
	defer session.Close()

	rOut, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	rErr, err := session.StderrPipe()
	if err != nil {
		return "", err
	}

	err = session.Start(createClientToken)
	if err != nil {
		return "", err
	}
	outputBytes, err := ioutil.ReadAll(io.MultiReader(rOut, rErr))
	if err != nil {
		return "", err
	}
	output := strings.TrimSpace(string(outputBytes))
	if !strings.HasPrefix(output, "do:") {
		return "", errors.New(output)
	}
	return output, nil
}
