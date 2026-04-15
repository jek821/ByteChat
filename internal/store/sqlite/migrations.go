package sqlite

import (
	"database/sql"
)

type migration struct {
	version int
	stmts   []string
}

var migrations = []migration{
	{
		version: 1,
		stmts: []string{
			"CREATE TABLE IF NOT EXISTS users (user_id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash BLOB NOT NULL, created_at INTEGER NOT NULL)",

			"CREATE TABLE IF NOT EXISTS sessions (session_id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, token_hash BLOB NOT NULL UNIQUE, created_at INTEGER NOT NULL, revoked_at INTEGER, FOREIGN KEY(user_id) REFERENCES users(user_id))",

			"CREATE TABLE IF NOT EXISTS messages (message_id INTEGER PRIMARY KEY, from_user_id INTEGER NOT NULL, to_user_id INTEGER NOT NULL, body TEXT NOT NULL, created_at INTEGER NOT NULL, delivered_at INTEGER, FOREIGN KEY(from_user_id) REFERENCES users(user_id), FOREIGN KEY(to_user_id) REFERENCES users(user_id))",

			"CREATE INDEX IF NOT EXISTS idx_messages_to_user_delivered ON messages(to_user_id, delivered_at)",
		},
	},
	{
		version: 2,
		stmts: []string{
			"ALTER TABLE users ADD COLUMN e2e_public_key BLOB",
		},
	},
	{
		version: 3,
		stmts: []string{
			"ALTER TABLE users ADD COLUMN e2e_encrypted_private_key BLOB",
			"ALTER TABLE users ADD COLUMN e2e_key_salt BLOB",
		},
	},
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)"); err != nil {
		return err
	}

	var latestVersion int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&latestVersion); err != nil {
		return err
	}

	for _, m := range migrations {
		if m.version <= latestVersion {
			continue
		}
		if err := applyMigration(db, m); err != nil {
			return err
		}

	}
	return nil
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range m.stmts {
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return err
		}
	}
	if _, err := tx.Exec("INSERT INTO schema_migrations(version) VALUES (?)", m.version); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
