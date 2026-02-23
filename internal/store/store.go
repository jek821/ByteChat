package store

import "context"

type Store interface {
	Close() error

	// users
	CreateUser(ctx context.Context, usernameCanonical, usernameDisplay string, passwordHash []byte) (int64, error)
	GetUserByUsernameCanonical(ctx context.Context, usernameCanonical string) (*UserRow, error)

	// sessions
	UpsertSingleSession(ctx context.Context, userID int64, tokenHash []byte) error
	GetSessionByTokenHash(ctx context.Context, tokenHash []byte) (*SessionRow, error)
	RevokeUserSessions(ctx context.Context, userID int64) error

	// messages
	InsertDirectMessage(ctx context.Context, fromUserID, toUserID int64, body string) (int64, error)
	ListUndeliveredDirectMessages(ctx context.Context, toUserID int64, limit int) ([]MessageRow, error)
	MarkMessagesDelivered(ctx context.Context, messageIDs []int64) error
}

// Keep these rows in store for now; later you can switch to pkg/types if you want.
type UserRow struct {
	ID                int64
	UsernameCanonical string
	UsernameDisplay   string
	PasswordHash      []byte
}

type SessionRow struct {
	ID        int64
	UserID    int64
	TokenHash []byte
	RevokedAt *int64 // unix seconds or null; your call later
}

type MessageRow struct {
	ID        int64
	FromUser  int64
	ToUser    int64
	Body      string
	CreatedAt int64
}
