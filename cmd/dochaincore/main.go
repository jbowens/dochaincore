// Command dochaincore deploys Chain Core Developer Edition to
// a Digital Ocean droplet.
package main

import (
	"fmt"
	"os"

	"github.com/jbowens/dochaincore"
)

func main() {
	core, err := dochaincore.Deploy(os.Getenv("DIGITAL_OCEAN_PAT"))
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Created DigitalOcean droplet %d.\n", core.DropletID)
	fmt.Printf("Waiting for SSH server to start...\n")
	err = dochaincore.WaitForSSH(core)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Waiting for Chain Core to start...\n")
	err = dochaincore.WaitForHTTP(core)
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
