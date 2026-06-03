package service

import (
	"context"
	"database/sql"
	"errors"

	"ByteChat/internal/store"
)

var (
	ErrInvalidToken   = errors.New("invalid session token")
	ErrUserNotFound   = errors.New("user not found")
	ErrEmptyMessage   = errors.New("message body is required")
	ErrSelfMessage    = errors.New("cannot message yourself")
)

type MessageService struct {
	store store.Store
}

func NewMessageService(s store.Store) *MessageService {
	return &MessageService{store: s}
}

type OutboundMessage struct {
	ID           int64
	FromUsername string
	Body         string
}

func (s *MessageService) AuthenticateToken(ctx context.Context, token string) (userID int64, username string, err error) {
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

func (s *MessageService) ListContacts(ctx context.Context, userID int64) ([]string, error) {
	return s.store.ListUsernames(ctx, userID)
}

func (s *MessageService) PendingMessages(ctx context.Context, userID int64) ([]OutboundMessage, error) {
	stored, err := s.store.ListUndeliveredMessages(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]OutboundMessage, len(stored))
	for i, msg := range stored {
		out[i] = OutboundMessage{
			ID:           msg.ID,
			FromUsername: msg.FromUsername,
			Body:         msg.Body,
		}
	}
	return out, nil
}

func (s *MessageService) MarkDelivered(ctx context.Context, messageID int64) error {
	return s.store.MarkMessageDelivered(ctx, messageID)
}

func (s *MessageService) Send(ctx context.Context, fromUserID int64, toUsername, body string) (msgID int64, toUserID int64, err error) {
	if body == "" {
		return 0, 0, ErrEmptyMessage
	}

	toUserID, _, err = s.store.GetUserByUsername(ctx, toUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, ErrUserNotFound
		}
		return 0, 0, err
	}
	if toUserID == fromUserID {
		return 0, 0, ErrSelfMessage
	}

	msgID, err = s.store.SaveMessage(ctx, fromUserID, toUserID, body)
	if err != nil {
		return 0, 0, err
	}
	return msgID, toUserID, nil
}
