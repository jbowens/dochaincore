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
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Printf("Deployed to droplet %d at http://%s:1999/dashboard\n", core.DropletID, core.IPv4Address)
}
