package client

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"

	"ByteChat/internal/paths"
)

const defaultHTTPSPort = "8443"
const defaultTCPPort = "8444"

// Settings holds everything needed to connect to a byteChat server.
type Settings struct {
	ServerURL   string
	TCPAddr     string
	InsecureTLS bool
}

// FileConfig is persisted to ~/.gochat/client/config.json.
type FileConfig struct {
	ServerURL   string `json:"server_url"`
	TCPAddr     string `json:"tcp_addr,omitempty"`
	InsecureTLS bool   `json:"insecure_tls,omitempty"`
}

func LocalSettings() Settings {
	return Settings{
		ServerURL:   "https://localhost:8443",
		TCPAddr:     "localhost:8444",
		InsecureTLS: true,
	}
}

func LoadConfigIfExists() (FileConfig, bool, error) {
	path, err := paths.ClientConfigPath()
	if err != nil {
		return FileConfig{}, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileConfig{}, false, nil
		}
		return FileConfig{}, false, err
	}
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, false, err
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		return FileConfig{}, false, errors.New("config: server_url is required")
	}
	return cfg, true, nil
}

func SaveConfig(cfg FileConfig) error {
	cfg = cfg.Resolved()
	path, err := paths.ClientConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (c FileConfig) Resolved() FileConfig {
	if strings.TrimSpace(c.TCPAddr) != "" {
		return c
	}
	if host := HostFromServerURL(c.ServerURL); host != "" {
		c.TCPAddr = net.JoinHostPort(host, defaultTCPPort)
	}
	return c
}

func (c FileConfig) Settings(insecureTLS bool) Settings {
	c = c.Resolved()
	return Settings{
		ServerURL:   strings.TrimRight(c.ServerURL, "/"),
		TCPAddr:     c.TCPAddr,
		InsecureTLS: insecureTLS,
	}
}

func (s Settings) ServerLabel() string {
	host := HostFromServerURL(s.ServerURL)
	if host == "" {
		return s.ServerURL
	}
	return host
}

func HostFromServerURL(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func IsLocalServer(serverURL string) bool {
	host := HostFromServerURL(serverURL)
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (s Settings) HTTPAuth() *HTTPAuth {
	return NewHTTPAuth(s.ServerURL, s.InsecureTLS)
}

func (s Settings) AdminClient() *AdminClient {
	return NewAdminClient(s.ServerURL, s.InsecureTLS)
}

func (s Settings) ChatClient() *ChatClient {
	return NewChatClient(s.TCPAddr, s.InsecureTLS)
}
