package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"ByteChat/internal/store"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserExists         = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

const (
	passwordSaltLen = 16
	passwordHashLen = 32
)

type AuthService struct {
	store store.Store
}

func NewAuthService(s store.Store) *AuthService {
	return &AuthService{store: s}
}

type AuthResult struct {
	Token    string
	Username string
	E2EBundle
}

type E2EBundle struct {
	EncPrivKey []byte
	Salt       []byte
}

type RegisterInput struct {
	Username string
	Password string
	E2E      *E2EBundle
	PubKey   []byte
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	if in.Username == "" || in.Password == "" {
		return AuthResult{}, ErrInvalidInput
	}
	if len(in.Password) < 8 {
		return AuthResult{}, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidInput)
	}

	passwordHash, err := hashPassword(in.Password)
	if err != nil {
		return AuthResult{}, err
	}

	userID, err := s.store.CreateUser(ctx, in.Username, passwordHash)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthResult{}, ErrUserExists
		}
		return AuthResult{}, err
	}

	if in.PubKey != nil && in.E2E != nil {
		if err := s.store.SetE2EKeyBundle(ctx, userID, in.PubKey, in.E2E.EncPrivKey, in.E2E.Salt); err != nil {
			return AuthResult{}, err
		}
	}

	token, err := s.issueSession(ctx, userID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Token: token, Username: in.Username}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (AuthResult, error) {
	if username == "" || password == "" {
		return AuthResult{}, ErrInvalidInput
	}

	userID, storedHash, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}

	if !verifyPassword(password, storedHash) {
		return AuthResult{}, ErrInvalidCredentials
	}

	token, err := s.issueSession(ctx, userID)
	if err != nil {
		return AuthResult{}, err
	}

	result := AuthResult{Token: token, Username: username}
	encPrivKey, salt, err := s.store.GetE2EKeyBundle(ctx, userID)
	if err == nil && len(encPrivKey) > 0 && len(salt) > 0 {
		result.EncPrivKey = encPrivKey
		result.Salt = salt
	}

	return result, nil
}

func (s *AuthService) issueSession(ctx context.Context, userID int64) (string, error) {
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateSession(ctx, userID, tokenHash); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) SessionUser(ctx context.Context, token string) (userID int64, username string, err error) {
	tokenHash, err := HashSessionToken(token)
	if err != nil {
		return 0, "", ErrInvalidToken
	}
	userID, username, err = s.store.GetUserByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", ErrInvalidToken
		}
		return 0, "", err
	}
	return userID, username, nil
}

func hashPassword(password string) ([]byte, error) {
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, passwordHashLen)
	return append(salt, hash...), nil
}

func verifyPassword(password string, stored []byte) bool {
	if len(stored) < passwordSaltLen+passwordHashLen {
		return false
	}
	salt := stored[:passwordSaltLen]
	hash := stored[passwordSaltLen:]
	computed := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, passwordHashLen)
	return subtle.ConstantTimeCompare(hash, computed) == 1
}

func newSessionToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), sum[:], nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint")
}
