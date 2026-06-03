package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"ByteChat/internal/store/sqlite"
)

func TestFriendRequestFlow(t *testing.T) {
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

	aliceID, _, _ := db.GetUserByUsername(ctx, "alice")
	bobID, _, _ := db.GetUserByUsername(ctx, "bob")

	if _, err := messages.SendFriendRequest(ctx, aliceID, "bob"); err != nil {
		t.Fatalf("SendFriendRequest: %v", err)
	}

	pending, err := messages.ListContacts(ctx, bobID)
	if err != nil {
		t.Fatalf("ListContacts bob: %v", err)
	}
	if len(pending.PendingRequests) != 1 || pending.PendingRequests[0] != "alice" {
		t.Fatalf("unexpected pending: %+v", pending)
	}

	if _, err := messages.AcceptFriendRequest(ctx, bobID, "alice"); err != nil {
		t.Fatalf("AcceptFriendRequest: %v", err)
	}

	aliceContacts, err := messages.ListContacts(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListContacts alice: %v", err)
	}
	if len(aliceContacts.Friends) != 1 || aliceContacts.Friends[0] != "bob" {
		t.Fatalf("unexpected alice friends: %+v", aliceContacts)
	}

	_, _, err = messages.Send(ctx, aliceID, "bob", "hi")
	if err != nil {
		t.Fatalf("Send after friend: %v", err)
	}

	_, _, err = messages.Send(ctx, aliceID, "missing", "hi")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
