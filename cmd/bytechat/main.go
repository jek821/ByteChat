package main

import (
	"flag"
	"fmt"
	"os"

	"ByteChat/internal/client"
	"ByteChat/internal/tui"
)

func main() {
	serverURL := flag.String("server", "https://localhost:8443", "HTTPS auth server URL")
	tcpAddr := flag.String("tcp", "localhost:8444", "TCP+TLS chat server address")
	flag.Parse()

	cfg := tui.Config{
		Auth:    client.NewHTTPAuth(*serverURL),
		Admin:   client.NewAdminClient(*serverURL),
		TCPAddr: *tcpAddr,
	}
	if err := tui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
