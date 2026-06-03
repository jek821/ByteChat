package store

import "context"

type StoredMessage struct {
	ID           int64
	FromUsername string
	Body         string
	CreatedAt    int64
}

type UserSummary struct {
	ID        int64
	Username  string
	CreatedAt int64
	IsAdmin   bool
}

type ServerStats struct {
	UserCount    int
	MessageCount int
	SessionCount int
	FriendCount  int
	RequestCount int
}

type Store interface {
	Close() error

	CreateUser(ctx context.Context, username string, passwordHash []byte) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (userID int64, passwordHash []byte, err error)
	GetUserByTokenHash(ctx context.Context, tokenHash []byte) (userID int64, username string, err error)
	CreateSession(ctx context.Context, userID int64, tokenHash []byte) error

	SaveMessage(ctx context.Context, fromUserID, toUserID int64, body string) (int64, error)
	ListUndeliveredMessages(ctx context.Context, userID int64) ([]StoredMessage, error)
	MarkMessageDelivered(ctx context.Context, messageID int64) error

	ListFriends(ctx context.Context, userID int64) ([]string, error)
	ListIncomingFriendRequests(ctx context.Context, userID int64) ([]string, error)
	ListOutgoingFriendRequests(ctx context.Context, userID int64) ([]string, error)
	ListConversationMessages(ctx context.Context, userID, peerUserID int64, limit int) ([]StoredMessage, error)
	CreateFriendRequest(ctx context.Context, fromUserID, toUserID int64) error
	AcceptFriendRequest(ctx context.Context, userID, fromUserID int64) error
	AreFriends(ctx context.Context, userID, otherUserID int64) (bool, error)

	SetE2EKeyBundle(ctx context.Context, userID int64, pubKey, encPrivKey, salt []byte) error
	GetE2EPublicKey(ctx context.Context, username string) ([]byte, error)
	GetE2EKeyBundle(ctx context.Context, userID int64) (encPrivKey, salt []byte, err error)

	IsAdmin(ctx context.Context, userID int64) (bool, error)
	SetAdmin(ctx context.Context, userID int64, admin bool) error
	ListUsers(ctx context.Context) ([]UserSummary, error)
	DeleteUser(ctx context.Context, userID int64) error
	WipeAllData(ctx context.Context) error
	GetServerStats(ctx context.Context) (ServerStats, error)
	HasAdmin(ctx context.Context) (bool, error)
}
