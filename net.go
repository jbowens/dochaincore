package dochaincore

import (
	"fmt"
	"net"
	"time"
)

func waitForPort(host string, port int) error {
	for attempt := uint(0); attempt < 15; attempt++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		} else if err != nil {
			return err
		}
		conn.Close()
		return nil
	}
	return fmt.Errorf("timed out waiting for port %d to open", port)
}
