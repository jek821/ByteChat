package sqlite

import (
	"bytes"
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

	expectedUserId, err := store.CreateUser("jacob", fakePasswordHash)
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
