package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"ByteChat/internal/store/sqlite"
)

func TestAuthRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	auth := NewAuthService(store)

	result, err := auth.Register(ctx, RegisterInput{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}

	login, err := auth.Login(ctx, "alice", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if login.Token == "" {
		t.Fatal("expected non-empty login token")
	}
	if login.Username != "alice" {
		t.Fatalf("username: got %q want alice", login.Username)
	}
}

func TestAuthRegisterDuplicateUser(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	auth := NewAuthService(store)
	_, err = auth.Register(ctx, RegisterInput{Username: "bob", Password: "password123"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = auth.Register(ctx, RegisterInput{Username: "bob", Password: "password456"})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	auth := NewAuthService(store)
	_, err = auth.Register(ctx, RegisterInput{Username: "carol", Password: "password123"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = auth.Login(ctx, "carol", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	_, err = auth.Login(ctx, "missing", "password123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for missing user, got %v", err)
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	stored, err := hashPassword("secret-password")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword("secret-password", stored) {
		t.Fatal("expected password to verify")
	}
	if verifyPassword("wrong-password", stored) {
		t.Fatal("expected wrong password to fail verification")
	}
}
