package sqlite

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"time"

	"ByteChat/internal/store"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	newDb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := newDb.Ping(); err != nil {
		newDb.Close()
		return nil, err
	}
	if _, err = newDb.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		newDb.Close()
		return nil, err
	}

	if _, err = newDb.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		newDb.Close()
		return nil, err
	}

	if _, err = newDb.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
		newDb.Close()
		return nil, err
	}

	if err := migrate(newDb); err != nil {
		newDb.Close()
		return nil, err
	}

	return &Store{db: newDb}, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (int64, []byte, error) {
	var userID int64
	var passwordHash []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, password_hash FROM users WHERE username = ?`, username,
	).Scan(&userID, &passwordHash)
	if err != nil {
		return 0, nil, err
	}
	return userID, passwordHash, nil
}

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions(user_id, token_hash, created_at) VALUES (?, ?, ?)`,
		userID, tokenHash, time.Now().Unix(),
	)
	return err
}

func (s *Store) CreateUser(ctx context.Context, username string, passwordHash []byte) (int64, error) {
	createdAt := time.Now().Unix()
	userResult, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users(username, password_hash, created_at) VALUES (?, ?, ?)`,
		username,
		passwordHash,
		createdAt,
	)
	if err != nil {
		return 0, err
	}
	userId, err := userResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	return userId, nil
}

// SetE2EKeyBundle stores the user's public key, encrypted private key, and Argon2 salt.
// Called after the client generates a keypair (registration or key rotation).
func (s *Store) SetE2EKeyBundle(ctx context.Context, userID int64, pubKey, encPrivKey, salt []byte) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET e2e_public_key = ?, e2e_encrypted_private_key = ?, e2e_key_salt = ? WHERE user_id = ?`,
		pubKey, encPrivKey, salt, userID,
	)
	return err
}

// GetE2EPublicKey returns a user's public key by username.
// Called by other clients who want to start an encrypted session.
func (s *Store) GetE2EPublicKey(ctx context.Context, username string) ([]byte, error) {
	var pubKey []byte
	err := s.db.QueryRowContext(ctx, `SELECT e2e_public_key FROM users WHERE username = ?`, username).Scan(&pubKey)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// GetE2EKeyBundle returns the encrypted private key and salt for a user.
// Called on new device login so the client can decrypt the private key locally using their password.
func (s *Store) GetE2EKeyBundle(ctx context.Context, userID int64) (encPrivKey, salt []byte, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT e2e_encrypted_private_key, e2e_key_salt FROM users WHERE user_id = ?`, userID,
	).Scan(&encPrivKey, &salt)
	return
}

func (s *Store) GetUserByTokenHash(ctx context.Context, tokenHash []byte) (int64, string, error) {
	var userID int64
	var username string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.user_id, u.username
		FROM sessions s
		JOIN users u ON s.user_id = u.user_id
		WHERE s.token_hash = ? AND s.revoked_at IS NULL`,
		tokenHash,
	).Scan(&userID, &username)
	if err != nil {
		return 0, "", err
	}
	return userID, username, nil
}

func (s *Store) SaveMessage(ctx context.Context, fromUserID, toUserID int64, body string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO messages(from_user_id, to_user_id, body, created_at) VALUES (?, ?, ?, ?)`,
		fromUserID, toUserID, body, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) ListUndeliveredMessages(ctx context.Context, userID int64) ([]store.StoredMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.message_id, u.username, m.body
		FROM messages m
		JOIN users u ON m.from_user_id = u.user_id
		WHERE m.to_user_id = ? AND m.delivered_at IS NULL
		ORDER BY m.created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []store.StoredMessage
	for rows.Next() {
		var msg store.StoredMessage
		if err := rows.Scan(&msg.ID, &msg.FromUsername, &msg.Body); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *Store) MarkMessageDelivered(ctx context.Context, messageID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET delivered_at = ? WHERE message_id = ?`,
		time.Now().Unix(), messageID,
	)
	return err
}

func (s *Store) ListFriends(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username
		FROM friends f
		JOIN users u ON f.friend_user_id = u.user_id
		WHERE f.user_id = ?
		ORDER BY u.username ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *Store) ListIncomingFriendRequests(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username
		FROM friend_requests fr
		JOIN users u ON fr.from_user_id = u.user_id
		WHERE fr.to_user_id = ?
		ORDER BY fr.created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *Store) CreateFriendRequest(ctx context.Context, fromUserID, toUserID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO friend_requests(from_user_id, to_user_id, created_at) VALUES (?, ?, ?)`,
		fromUserID, toUserID, time.Now().Unix(),
	)
	return err
}

func (s *Store) ListOutgoingFriendRequests(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.username
		FROM friend_requests fr
		JOIN users u ON fr.to_user_id = u.user_id
		WHERE fr.from_user_id = ?
		ORDER BY fr.created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *Store) ListConversationMessages(ctx context.Context, userID, peerUserID int64, limit int) ([]store.StoredMessage, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.message_id, u.username, m.body, m.created_at
		FROM messages m
		JOIN users u ON m.from_user_id = u.user_id
		WHERE (m.from_user_id = ? AND m.to_user_id = ?)
		   OR (m.from_user_id = ? AND m.to_user_id = ?)
		ORDER BY m.created_at ASC
		LIMIT ?`,
		userID, peerUserID, peerUserID, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []store.StoredMessage
	for rows.Next() {
		var msg store.StoredMessage
		if err := rows.Scan(&msg.ID, &msg.FromUsername, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *Store) AcceptFriendRequest(ctx context.Context, userID, fromUserID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`DELETE FROM friend_requests WHERE from_user_id = ? AND to_user_id = ?`,
		fromUserID, userID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO friends(user_id, friend_user_id, created_at) VALUES (?, ?, ?)`,
		userID, fromUserID, now,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO friends(user_id, friend_user_id, created_at) VALUES (?, ?, ?)`,
		fromUserID, userID, now,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) AreFriends(ctx context.Context, userID, otherUserID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM friends WHERE user_id = ? AND friend_user_id = ?`,
		userID, otherUserID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
