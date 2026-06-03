package server

import (
	"context"
	"crypto/tls"
	"path/filepath"
	"testing"
	"time"

	"ByteChat/internal/client"
	"ByteChat/internal/service"
	"ByteChat/internal/store/sqlite"
)

func TestHubDeliverMessage(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	auth := service.NewAuthService(store)
	messages := service.NewMessageService(store)

	_, err = auth.Register(ctx, service.RegisterInput{Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatalf("register alice: %v", err)
	}
	_, err = auth.Register(ctx, service.RegisterInput{Username: "bob", Password: "password123"})
	if err != nil {
		t.Fatalf("register bob: %v", err)
	}

	alice, err := auth.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("login alice: %v", err)
	}
	bob, err := auth.Login(ctx, "bob", "password123")
	if err != nil {
		t.Fatalf("login bob: %v", err)
	}

	tlsConfig, err := LoadTLSConfig()
	if err != nil {
		t.Fatalf("LoadTLSConfig: %v", err)
	}

	hub := NewHub(messages)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go hub.handleConn(conn)
		}
	}()

	aliceClient := client.NewChatClient(ln.Addr().String())
	if err := aliceClient.Connect(alice.Token); err != nil {
		t.Fatalf("connect alice: %v", err)
	}
	t.Cleanup(func() { aliceClient.Close() })

	bobClient := client.NewChatClient(ln.Addr().String())
	if err := bobClient.Connect(bob.Token); err != nil {
		t.Fatalf("connect bob: %v", err)
	}
	t.Cleanup(func() { bobClient.Close() })

	drainContacts(t, aliceClient)
	drainContacts(t, bobClient)

	if err := aliceClient.Send("bob", "hello bob"); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case event := <-bobClient.Events():
		if event.Kind != client.EventMessage {
			t.Fatalf("expected message event, got %v", event.Kind)
		}
		if event.Message.From != "alice" || event.Message.Body != "hello bob" {
			t.Fatalf("unexpected message: %+v", event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func drainContacts(t *testing.T, c *client.ChatClient) {
	t.Helper()
	select {
	case event := <-c.Events():
		if event.Kind != client.EventContacts {
			t.Fatalf("expected contacts event, got %v", event.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for contacts")
	}
}
