package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrNotFriends        = errors.New("you can only message friends")
	ErrAlreadyFriends    = errors.New("already friends")
	ErrRequestExists     = errors.New("friend request already sent")
	ErrCannotFriendSelf  = errors.New("cannot friend yourself")
	ErrNotFriendRequest  = errors.New("no pending friend request from that user")
)

type Contacts struct {
	Friends         []string
	PendingRequests []string
}

func (s *MessageService) ListContacts(ctx context.Context, userID int64) (Contacts, error) {
	friends, err := s.store.ListFriends(ctx, userID)
	if err != nil {
		return Contacts{}, err
	}
	pending, err := s.store.ListIncomingFriendRequests(ctx, userID)
	if err != nil {
		return Contacts{}, err
	}
	return Contacts{Friends: friends, PendingRequests: pending}, nil
}

func (s *MessageService) SendFriendRequest(ctx context.Context, fromUserID int64, toUsername string) (toUserID int64, err error) {
	toUsername = strings.TrimSpace(toUsername)
	if toUsername == "" {
		return 0, ErrUserNotFound
	}

	toUserID, _, err = s.store.GetUserByUsername(ctx, toUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	if toUserID == fromUserID {
		return 0, ErrCannotFriendSelf
	}

	friends, err := s.store.AreFriends(ctx, fromUserID, toUserID)
	if err != nil {
		return 0, err
	}
	if friends {
		return 0, ErrAlreadyFriends
	}

	if err := s.store.CreateFriendRequest(ctx, fromUserID, toUserID); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrRequestExists
		}
		return 0, err
	}
	return toUserID, nil
}

func (s *MessageService) AcceptFriendRequest(ctx context.Context, userID int64, fromUsername string) (fromUserID int64, err error) {
	fromUsername = strings.TrimSpace(fromUsername)
	if fromUsername == "" {
		return 0, ErrNotFriendRequest
	}

	fromUserID, _, err = s.store.GetUserByUsername(ctx, fromUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFriendRequest
		}
		return 0, err
	}

	if err := s.store.AcceptFriendRequest(ctx, userID, fromUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFriendRequest
		}
		return 0, err
	}
	return fromUserID, nil
}
