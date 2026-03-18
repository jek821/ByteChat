package sqlite

import (
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

func (s *Store) CreateUser(username string, passwordHash []byte) (int64, error) {
	createdAt := time.Now().Unix()
	userResult, err := s.db.Exec(
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

func (s *Store) Close() error {
	return s.db.Close()
}
