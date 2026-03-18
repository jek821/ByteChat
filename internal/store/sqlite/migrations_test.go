package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewRunsInitialMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	t.Logf("temp dir: %s", tempDir)
	t.Logf("db path: %s", dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer store.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	expectedTables := map[string]bool{
		"schema_migrations": false,
		"users":             false,
		"sessions":          false,
		"messages":          false,
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("querying sqlite_master failed: %v", err)
	}
	defer rows.Close()

	t.Log("tables found in sqlite_master:")

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			t.Fatalf("scanning table name failed: %v", err)
		}

		t.Logf(" - %s", tableName)

		if _, ok := expectedTables[tableName]; ok {
			expectedTables[tableName] = true
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration failed: %v", err)
	}

	for tableName, found := range expectedTables {
		if !found {
			t.Fatalf("expected table %q to exist, but it was not found", tableName)
		}
	}

	var version int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatalf("querying schema_migrations failed: %v", err)
	}

	t.Logf("latest migration version: %d", version)

	if version != 1 {
		t.Fatalf("expected migration version 1, got %d", version)
	}
}
