package types

type User struct {
	UserId       int64
	UserName     string
	PasswordHash []byte
	CreatedAt    int64
}
