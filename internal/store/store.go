package store

import (
	"context"

	"GoChat/pkg/types"
)

type Store interface {
	Close() error

	CreateUser(ctx context.Context, user *types.User) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*types.User, error)
}
