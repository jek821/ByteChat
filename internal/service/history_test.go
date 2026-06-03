package service

import (
	"context"
	"path/filepath"
	"testing"

	"ByteChat/internal/store/sqlite"
)

func TestConversationHistory(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	auth := NewAuthService(db)
	messages := NewMessageService(db)

	_, _ = auth.Register(ctx, RegisterInput{Username: "alice", Password: "password123"})
	_, _ = auth.Register(ctx, RegisterInput{Username: "bob", Password: "password123"})

	aliceID, _, _ := db.GetUserByUsername(ctx, "alice")
	bobID, _, _ := db.GetUserByUsername(ctx, "bob")

	_ = db.CreateFriendRequest(ctx, aliceID, bobID)
	_ = db.AcceptFriendRequest(ctx, bobID, aliceID)

	_, _, err = messages.Send(ctx, aliceID, "bob", "hello")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	_, _, err = messages.Send(ctx, bobID, "alice", "hey back")
	if err != nil {
		t.Fatalf("Send bob: %v", err)
	}

	history, err := messages.GetConversationHistory(ctx, aliceID, "alice", "bob")
	if err != nil {
		t.Fatalf("GetConversationHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if !history[0].Self || history[0].Body != "hello" {
		t.Fatalf("unexpected first message: %+v", history[0])
	}
	if history[1].Self || history[1].Body != "hey back" {
		t.Fatalf("unexpected second message: %+v", history[1])
	}
}
