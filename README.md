# dochaincore
[![GoDoc](https://godoc.org/github.com/jbowens/dochaincore?status.svg)](https://godoc.org/github.com/jbowens/dochaincore)

[Chain Core Developer Edition](https://chain.com) one-click deploy to a [DigitalOcean](https://digitalocean.com) droplet.

## Command line

dochaincore exposes a simple command-line utility for deploying Chain Core.

```bash
go install github.com/jbowens/dochaincore/cmd/...
DIGITALOCEAN_ACCESS_TOKEN=... dochaincore
```

Prints output like:
```
Created DigitalOcean droplet 30977065.
Waiting for SSH server to start...
Waiting for Chain Core to start...
Creating a client token...
Chain Core listening at: http://138.68.52.205:1999
Chain Core client token: dochaincore:6de76c428a8ce9805777a60fffed21889240f434e72eef902c49e9822b8a87eb
```
