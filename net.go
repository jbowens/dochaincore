package dochaincore

import (
	"fmt"
	"net"
	"strings"
	"time"
)

func waitForPort(host string, port int) error {
	for attempt := uint(0); attempt < 10; attempt++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "refused") {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		} else if err != nil {
			return err
		}
		conn.Close()
		return nil
	}
	return fmt.Errorf("timed out waiting for port %d to open", port)
}
