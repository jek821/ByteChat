package sqlite

import "database/sql"

type Migration struct {
	version int
	stmts   []string
}

var migrations = []Migration{
	{
		version: 1,
		stmts: []string{
			"CREATE TABLE IF NOT EXISTS users (user_id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash BLOB NOT NULL, created_at INTEGER NOT NULL)",

			"CREATE TABLE IF NOT EXISTS sessions (session_id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, token_hash BLOB NOT NULL UNIQUE, created_at INTEGER NOT NULL, revoked_at INTEGER, FOREIGN KEY(user_id) REFERENCES users(user_id))",

			"CREATE TABLE IF NOT EXISTS messages (message_id INTEGER PRIMARY KEY, from_user_id INTEGER NOT NULL, to_user_id INTEGER NOT NULL, body TEXT NOT NULL, created_at INTEGER NOT NULL, delivered_at INTEGER, FOREIGN KEY(from_user_id) REFERENCES users(user_id), FOREIGN KEY(to_user_id) REFERENCES users(user_id))",

			"CREATE INDEX IF NOT EXISTS idx_messages_to_user_delivered ON messages(to_user_id, delivered_at)",
		},
	},
}

func migrate(db *sql.DB) error {
	if err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)"); err != nil {
		return err
	}
}
