// Command dochaincore deploys Chain Core Developer Edition to
// a Digital Ocean droplet.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/jbowens/dochaincore"
)

var (
	flagServer = flag.Bool("server", false, "set to run OAuth2 server")
	flagPort   = flag.Int("port", 8080, "listen port for OAuth2 server")
)

func main() {
	flag.Parse()

	if !*flagServer {
		createDroplet()
		return
	}

	handler := dochaincore.Handler(
		os.Getenv("DIGITALOCEAN_CLIENT_ID"),
		os.Getenv("DIGITALOCEAN_CLIENT_SECRET"),
		os.Getenv("SERVER_HOST"),
	)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *flagPort), handler)
	if err != nil {
		fatal(err)
	}
}

func createDroplet() {
	core, err := dochaincore.Deploy(os.Getenv("DIGITALOCEAN_ACCESS_TOKEN"))
	if err != nil {
		fatal(err)
	}

	ctx := context.Background()
	fmt.Printf("Created DigitalOcean droplet %d.\n", core.DropletID)
	fmt.Printf("Waiting for SSH server to start...\n")
	err = dochaincore.WaitForSSH(ctx, core)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Waiting for Chain Core to start...\n")
	err = dochaincore.WaitForHTTP(ctx, core)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Creating a client token...\n")
	token, err := dochaincore.CreateClientToken(core)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Chain Core listening at: http://%s:1999\n", core.IPv4Address)
	fmt.Printf("Chain Core client token: %s\n", token)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, err.Error())
	os.Exit(1)
}
