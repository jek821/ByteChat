package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ByteChat/internal/logx"
	"ByteChat/internal/store"
)

var (
	ErrNotAdmin          = errors.New("admin access required")
	ErrCannotDeleteAdmin = errors.New("cannot delete an admin account")
	ErrConfirmRequired   = errors.New("confirmation phrase required")
)

type OnlineTracker interface {
	OnlineUsernames() []string
}

type AdminService struct {
	store  store.Store
	auth   *AuthService
	online OnlineTracker
}

func NewAdminService(s store.Store, auth *AuthService, online OnlineTracker) *AdminService {
	return &AdminService{store: s, auth: auth, online: online}
}

type AdminLoginResult struct {
	Token    string
	Username string
}

type AdminDashboard struct {
	Stats         store.ServerStats
	OnlineUsers   []string
	OnlineCount   int
	LoggingConfig logx.Config
}

func (s *AdminService) CreateAdmin(ctx context.Context, username, password string) error {
	userID, _, err := s.store.GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := s.auth.Register(ctx, RegisterInput{Username: username, Password: password}); err != nil {
			return err
		}
		userID, _, err = s.store.GetUserByUsername(ctx, username)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := s.store.SetAdmin(ctx, userID, true); err != nil {
		return err
	}
	logx.AdminAction("create_admin", fmt.Sprintf("username=%s", username))
	return nil
}

func (s *AdminService) Login(ctx context.Context, username, password string) (AdminLoginResult, error) {
	result, err := s.auth.Login(ctx, username, password)
	if err != nil {
		return AdminLoginResult{}, err
	}
	userID, _, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return AdminLoginResult{}, err
	}
	isAdmin, err := s.store.IsAdmin(ctx, userID)
	if err != nil {
		return AdminLoginResult{}, err
	}
	if !isAdmin {
		return AdminLoginResult{}, ErrNotAdmin
	}
	logx.AdminAction("login", fmt.Sprintf("username=%s", username))
	return AdminLoginResult{Token: result.Token, Username: username}, nil
}

func (s *AdminService) ValidateToken(ctx context.Context, token string) (string, error) {
	userID, username, err := s.auth.SessionUser(ctx, token)
	if err != nil {
		return "", err
	}
	isAdmin, err := s.store.IsAdmin(ctx, userID)
	if err != nil {
		return "", err
	}
	if !isAdmin {
		return "", ErrNotAdmin
	}
	return username, nil
}

func (s *AdminService) Dashboard(ctx context.Context) (AdminDashboard, error) {
	stats, err := s.store.GetServerStats(ctx)
	if err != nil {
		return AdminDashboard{}, err
	}
	dash := AdminDashboard{
		Stats:         stats,
		LoggingConfig: logx.GetConfig(),
	}
	if s.online != nil {
		dash.OnlineUsers = s.online.OnlineUsernames()
		dash.OnlineCount = len(dash.OnlineUsers)
	}
	return dash, nil
}

func (s *AdminService) ListUsers(ctx context.Context) ([]store.UserSummary, error) {
	return s.store.ListUsers(ctx)
}

func (s *AdminService) DeleteUser(ctx context.Context, adminUsername, targetUsername string) error {
	if targetUsername == adminUsername {
		return fmt.Errorf("cannot delete your own account")
	}
	userID, _, err := s.store.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	isAdmin, err := s.store.IsAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if isAdmin {
		return ErrCannotDeleteAdmin
	}
	if err := s.store.DeleteUser(ctx, userID); err != nil {
		return err
	}
	logx.AdminAction("delete_user", fmt.Sprintf("by=%s target=%s", adminUsername, targetUsername))
	return nil
}

func (s *AdminService) WipeDatabase(ctx context.Context, adminUsername, confirm string) error {
	if confirm != "WIPE DATABASE" {
		return ErrConfirmRequired
	}
	if err := s.store.WipeAllData(ctx); err != nil {
		return err
	}
	logx.AdminAction("wipe_database", fmt.Sprintf("by=%s", adminUsername))
	return nil
}

func (s *AdminService) GetLogConfig() logx.Config {
	return logx.GetConfig()
}

func (s *AdminService) SetLogCategory(cat logx.Category, enabled bool) error {
	if err := logx.SetCategory(cat, enabled); err != nil {
		return err
	}
	state := "off"
	if enabled {
		state = "on"
	}
	logx.AdminAction("log_toggle", fmt.Sprintf("category=%s state=%s", cat, state))
	return nil
}

func (s *AdminService) HasAdmin(ctx context.Context) (bool, error) {
	return s.store.HasAdmin(ctx)
}
