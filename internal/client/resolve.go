package client

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"ByteChat/internal/paths"
)

// ResolveSettings loads config from disk, applies CLI overrides, and picks TLS verification mode.
func ResolveSettings(serverFlag, tcpFlag string, insecureFlag *bool, insecureSet bool) (Settings, error) {
	s := LocalSettings()
	var fileCfg FileConfig
	fromFile := false

	if cfg, loaded, err := LoadConfigIfExists(); err != nil {
		return Settings{}, err
	} else if loaded {
		fromFile = true
		fileCfg = cfg.Resolved()
		s.ServerURL = strings.TrimRight(fileCfg.ServerURL, "/")
		s.TCPAddr = fileCfg.TCPAddr
		s.InsecureTLS = fileCfg.InsecureTLS
	}

	if strings.TrimSpace(serverFlag) != "" {
		s.ServerURL = strings.TrimRight(serverFlag, "/")
	}
	if strings.TrimSpace(tcpFlag) != "" {
		s.TCPAddr = tcpFlag
	} else if host := HostFromServerURL(s.ServerURL); host != "" {
		s.TCPAddr = net.JoinHostPort(host, defaultTCPPort)
	}

	switch {
	case insecureSet:
		s.InsecureTLS = *insecureFlag
	case fromFile:
		s.InsecureTLS = fileCfg.InsecureTLS
	default:
		s.InsecureTLS = IsLocalServer(s.ServerURL)
	}

	return s, nil
}

// SaveSettings writes the current connection settings to the client config file.
func SaveSettings(s Settings) error {
	return SaveConfig(FileConfig{
		ServerURL:   s.ServerURL,
		TCPAddr:     s.TCPAddr,
		InsecureTLS: s.InsecureTLS,
	})
}

// PrintSettings shows where the client will connect (used by -configure).
func PrintSettings(s Settings) {
	path, err := paths.ClientConfigPath()
	if err != nil {
		path = "~/.gochat/client/config.json"
	}
	fmt.Printf("Server:  %s\n", s.ServerURL)
	fmt.Printf("Chat:    %s\n", s.TCPAddr)
	fmt.Printf("TLS:     %s\n", tlsModeLabel(s.InsecureTLS))
	fmt.Printf("Saved:   %s\n", path)
}

func tlsModeLabel(insecure bool) string {
	if insecure {
		return "skip verification (dev/self-signed)"
	}
	return "verify certificate"
}

// ExampleConfigJSON returns an example config for documentation.
func ExampleConfigJSON(host string) string {
	cfg := FileConfig{
		ServerURL:   fmt.Sprintf("https://%s:8443", host),
		TCPAddr:     fmt.Sprintf("%s:8444", host),
		InsecureTLS: false,
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return string(data)
}

// ShowConfig prints the saved client config, if any.
func ShowConfig() error {
	cfg, loaded, err := LoadConfigIfExists()
	if err != nil {
		return err
	}
	if !loaded {
		path, _ := paths.ClientConfigPath()
		fmt.Printf("No client config found (%s).\n", path)
		fmt.Println("Run with -configure -server https://your-host:8443 to create one.")
		return nil
	}
	cfg = cfg.Resolved()
	PrintSettings(cfg.Settings(cfg.InsecureTLS))
	return nil
}
