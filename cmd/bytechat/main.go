package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"ByteChat/internal/client"
	"ByteChat/internal/tui"
)

func main() {
	serverURL := flag.String("server", "", "HTTPS server URL (overrides config)")
	tcpAddr := flag.String("tcp", "", "TCP+TLS chat address (overrides config; default derived from server host)")
	insecure := flag.Bool("insecure", false, "skip TLS certificate verification")
	configure := flag.Bool("configure", false, "save connection settings to ~/.gochat/client/config.json and exit")
	showConfig := flag.Bool("show-config", false, "print saved client config and exit")
	flag.Parse()

	insecureSet := flagPassed("insecure")

	if *showConfig {
		if err := client.ShowConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	settings, err := client.ResolveSettings(*serverURL, *tcpAddr, insecure, insecureSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *configure {
		if strings.TrimSpace(*serverURL) == "" {
			fmt.Fprintln(os.Stderr, "error: -configure requires -server (e.g. -server https://chat.example.com:8443)")
			os.Exit(1)
		}
		if err := client.SaveSettings(settings); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		client.PrintSettings(settings)
		return
	}

	cfg := tui.Config{
		Auth:        settings.HTTPAuth(),
		Admin:       settings.AdminClient(),
		TCPAddr:     settings.TCPAddr,
		ServerLabel: settings.ServerLabel(),
		InsecureTLS: settings.InsecureTLS,
	}
	if err := tui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func flagPassed(name string) bool {
	found := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
