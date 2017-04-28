package dochaincore

import (
	"context"
	"fmt"
	"net"
	"time"
)

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
