package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

const defaultHistoryLimit = 200

type HistoryEntry struct {
	FromUsername string
	Body         string
	CreatedAt    int64
	Self         bool
}

func (s *MessageService) GetConversationHistory(ctx context.Context, userID int64, selfUsername, peerUsername string) ([]HistoryEntry, error) {
	peerUsername = strings.TrimSpace(peerUsername)
	if peerUsername == "" {
		return nil, ErrUserNotFound
	}

	peerID, _, err := s.store.GetUserByUsername(ctx, peerUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	friends, err := s.store.AreFriends(ctx, userID, peerID)
	if err != nil {
		return nil, err
	}
	if !friends {
		return nil, ErrNotFriends
	}

	stored, err := s.store.ListConversationMessages(ctx, userID, peerID, defaultHistoryLimit)
	if err != nil {
		return nil, err
	}

	out := make([]HistoryEntry, len(stored))
	for i, msg := range stored {
		out[i] = HistoryEntry{
			FromUsername: msg.FromUsername,
			Body:         msg.Body,
			CreatedAt:    msg.CreatedAt,
			Self:         msg.FromUsername == selfUsername,
		}
	}
	return out, nil
}
