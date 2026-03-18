package sqlite

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
)

func TestCreateNewUser(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	t.Logf("package sqlite

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
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
	var fakePasswordHash = []byte{0, 1, 0, 1, 0, 1, 0, 1}

	userId, err := store.CreateUser("jacob", fakePasswordHash)
	expectedName := "jacob"
	expectedPassHash := fakePasswordHash
	expectedUserId := userId

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	userRows, err := db.query(`SELECT * FROM users`)
	if err != nil {
		t.Fatalf("Failed to query rows from users table: %v", err)
	}
	defer userRows.Close()

	t.Log("Users found in users table: ")

	for userRows.Next() {
		var userName string
		var userId int
		var userPassHash []byte
		if err := userRows.Scan()
	}
}temp dir: %s", tempDir)
	t.Logf("db path: %s", dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer store.Close()
	var fakePasswordHash = []byte{0, 1, 0, 1, 0, 1, 0, 1}

	userId, err := store.CreateUser("jacob", fakePasswordHash)
	expectedName := "jacob"
	expectedPassHash := fakePasswordHash
	expectedUserId := userId

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	userRows, err := db.query(`SELECT * FROM users`)
	if err != nil {
		t.Fatalf("Failed to query rows from users table: %v", err)
	}
	defer userRows.Close()

	t.Log("Users found in users table: ")

	for userRows.Next() {
		var userName string
		var userId int
		var userPassHash []byte
		if err := userRows.Scan()
	}
}
