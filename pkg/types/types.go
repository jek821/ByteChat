package types

type User struct {
	ID           int64
	Username     string
	PasswordHash []byte
	CreatedAt    int64
}

type Session struct {
	ID        int64
	UserID    int64
	TokenHash []byte
	CreatedAt int64
}
