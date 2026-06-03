package service

import (
	"context"
	"path/filepath"
	"testing"

	"ByteChat/internal/store/sqlite"
)

func TestMessageServiceSendAndPending(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	auth := NewAuthService(db)
	messages := NewMessageService(db)

	_, err = auth.Register(ctx, RegisterInput{Username: "alice", Password: "password123"})
	if err != nil {
		t.Fatalf("register alice: %v", err)
	}
	_, err = auth.Register(ctx, RegisterInput{Username: "bob", Password: "password123"})
	if err != nil {
		t.Fatalf("register bob: %v", err)
	}

	aliceLogin, err := auth.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("login alice: %v", err)
	}
	bobLogin, err := auth.Login(ctx, "bob", "password123")
	if err != nil {
		t.Fatalf("login bob: %v", err)
	}

	aliceID, _, err := db.GetUserByTokenHash(ctx, mustHashToken(t, aliceLogin.Token))
	if err != nil {
		t.Fatalf("alice token: %v", err)
	}
	bobID, _, err := db.GetUserByTokenHash(ctx, mustHashToken(t, bobLogin.Token))
	if err != nil {
		t.Fatalf("bob token: %v", err)
	}

	if err := db.CreateFriendRequest(ctx, aliceID, bobID); err != nil {
		t.Fatalf("CreateFriendRequest: %v", err)
	}
	if err := db.AcceptFriendRequest(ctx, bobID, aliceID); err != nil {
		t.Fatalf("AcceptFriendRequest: %v", err)
	}

	msgID, toID, err := messages.Send(ctx, aliceID, "bob", "hello bob")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if msgID == 0 || toID != bobID {
		t.Fatalf("unexpected send result: id=%d to=%d", msgID, toID)
	}

	pending, err := messages.PendingMessages(ctx, bobID)
	if err != nil {
		t.Fatalf("PendingMessages: %v", err)
	}
	if len(pending) != 1 || pending[0].Body != "hello bob" {
		t.Fatalf("unexpected pending: %+v", pending)
	}
}

func mustHashToken(t *testing.T, token string) []byte {
	t.Helper()
	hash, err := HashSessionToken(token)
	if err != nil {
		t.Fatalf("HashSessionToken: %v", err)
	}
	return hash
}
