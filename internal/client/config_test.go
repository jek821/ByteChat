package client

import "testing"

func TestFileConfigResolvedDerivesTCP(t *testing.T) {
	cfg := FileConfig{ServerURL: "https://chat.example.com:8443"}.Resolved()
	if cfg.TCPAddr != "chat.example.com:8444" {
		t.Fatalf("tcp addr: got %q want chat.example.com:8444", cfg.TCPAddr)
	}
}

func TestResolveSettingsRemoteDefaultsSecureTLS(t *testing.T) {
	s, err := ResolveSettings("https://chat.example.com:8443", "", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if s.InsecureTLS {
		t.Fatal("expected TLS verification for remote server")
	}
	if s.TCPAddr != "chat.example.com:8444" {
		t.Fatalf("tcp: got %q", s.TCPAddr)
	}
}

func TestResolveSettingsLocalDefaultsInsecure(t *testing.T) {
	s, err := ResolveSettings("", "", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !s.InsecureTLS {
		t.Fatal("expected insecure TLS for localhost default")
	}
}

func TestIsLocalServer(t *testing.T) {
	if !IsLocalServer("https://localhost:8443") {
		t.Fatal("localhost should be local")
	}
	if IsLocalServer("https://chat.example.com:8443") {
		t.Fatal("remote host should not be local")
	}
}
