package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ByteChat/internal/service"
	"ByteChat/internal/store/sqlite"
)

func TestRegisterAndLoginHTTP(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	auth := service.NewAuthService(store)
	srv := httptest.NewServer(New(auth))
	t.Cleanup(srv.Close)

	registerBody := map[string]string{
		"username": "alice",
		"password": "password123",
	}
	registerResp := postJSON(t, srv.URL+"/api/register", registerBody)
	if registerResp["token"] == "" {
		t.Fatal("expected register token")
	}

	loginBody := map[string]string{
		"username": "alice",
		"password": "password123",
	}
	loginResp := postJSON(t, srv.URL+"/api/login", loginBody)
	if loginResp["token"] == "" {
		t.Fatal("expected login token")
	}
	if loginResp["username"] != "alice" {
		t.Fatalf("username: got %v want alice", loginResp["username"])
	}

	_, err = auth.Login(ctx, "alice", "wrong")
	if err == nil {
		t.Fatal("expected login failure for sanity check")
	}
}

func postJSON(t *testing.T, url string, body any) map[string]string {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	res, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", res.StatusCode)
	}

	var out map[string]string
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}
