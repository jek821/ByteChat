package sqlite

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
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

func (s *Store) Close() error {
	return s.db.Close()
}
