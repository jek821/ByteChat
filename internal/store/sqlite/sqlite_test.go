package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateNewUser(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	t.Logf("temp dir: %s", tempDir)
	t.Logf("db path: %s", dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer store.Close()

	fakePasswordHash := []byte{0, 1, 0, 1, 0, 1, 0, 1}

	expectedUserId, err := store.CreateUser(context.Background(), "jacob", fakePasswordHash)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	expectedName := "jacob"
	expectedPassHash := fakePasswordHash

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	userRows, err := db.Query(`SELECT user_id, username, password_hash FROM users`)
	if err != nil {
		t.Fatalf("Failed to query rows from users table: %v", err)
	}
	defer userRows.Close()

	t.Log("Users found in users table:")

	found := false

	for userRows.Next() {
		var actualUserName string
		var actualUserId int64
		var actualUserPassHash []byte

		if err := userRows.Scan(&actualUserId, &actualUserName, &actualUserPassHash); err != nil {
			t.Fatalf("Failed to scan rows in users table: %v", err)
		}

		if actualUserName != expectedName {
			t.Fatalf("Unexpected username: got %s want %s", actualUserName, expectedName)
		}
		if actualUserId != expectedUserId {
			t.Fatalf("Unexpected userId: got %d want %d", actualUserId, expectedUserId)
		}
		if !bytes.Equal(actualUserPassHash, expectedPassHash) {
			t.Fatalf("Unexpected userPassHash: got %v want %v", actualUserPassHash, expectedPassHash)
		}

		t.Logf(
			"Test user data accurate: username=%s userId=%d userPassHash=%v",
			actualUserName,
			actualUserId,
			actualUserPassHash,
		)

		found = true
	}

	if err := userRows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}

	if !found {
		t.Fatal("Expected to find one user row, found none")
	}
}

func TestSetAndGetE2EKeyBundle(t *testing.T) {
	tempDir := t.TempDir()
	store, err := New(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	userID, err := store.CreateUser(ctx, "alice", []byte("fakehash"))
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	pubKey := []byte("fake-public-key")
	encPrivKey := []byte("fake-encrypted-private-key")
	salt := []byte("fake-salt")

	if err := store.SetE2EKeyBundle(ctx, userID, pubKey, encPrivKey, salt); err != nil {
		t.Fatalf("SetE2EKeyBundle: %v", err)
	}

	gotPub, err := store.GetE2EPublicKey(ctx, "alice")
	if err != nil {
		t.Fatalf("GetE2EPublicKey: %v", err)
	}
	if !bytes.Equal(gotPub, pubKey) {
		t.Fatalf("GetE2EPublicKey: got %v want %v", gotPub, pubKey)
	}

	gotEncPriv, gotSalt, err := store.GetE2EKeyBundle(ctx, userID)
	if err != nil {
		t.Fatalf("GetE2EKeyBundle: %v", err)
	}
	if !bytes.Equal(gotEncPriv, encPrivKey) {
		t.Fatalf("GetE2EKeyBundle encPrivKey: got %v want %v", gotEncPriv, encPrivKey)
	}
	if !bytes.Equal(gotSalt, salt) {
		t.Fatalf("GetE2EKeyBundle salt: got %v want %v", gotSalt, salt)
	}
}
