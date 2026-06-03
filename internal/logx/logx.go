package logx

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"ByteChat/internal/paths"
)

type Category string

const (
	CatServer    Category = "server"
	CatHTTP      Category = "http"
	CatTCP       Category = "tcp"
	CatMessaging Category = "messaging"
	CatFriends   Category = "friends"
	CatAdmin     Category = "admin"
	CatStore     Category = "store"
)

var AllCategories = []Category{
	CatServer, CatHTTP, CatTCP, CatMessaging, CatFriends, CatAdmin, CatStore,
}

type Config struct {
	Enabled map[Category]bool `json:"enabled"`
}

type Logger struct {
	mu     sync.RWMutex
	cfg    Config
	logger *log.Logger
}

var defaultLogger = &Logger{
	cfg:    defaultConfig(),
	logger: log.New(os.Stderr, "", log.LstdFlags),
}

func defaultConfig() Config {
	enabled := make(map[Category]bool, len(AllCategories))
	for _, c := range AllCategories {
		enabled[c] = true
	}
	return Config{Enabled: enabled}
}

func Init() error {
	path, err := paths.LogConfigPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SaveConfig(defaultConfig())
		}
		return err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if cfg.Enabled == nil {
		cfg.Enabled = defaultConfig().Enabled
	}
	defaultLogger.mu.Lock()
	defaultLogger.cfg = cfg
	defaultLogger.mu.Unlock()
	return nil
}

func SaveConfig(cfg Config) error {
	path, err := paths.LogConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	defaultLogger.mu.Lock()
	defaultLogger.cfg = cfg
	defaultLogger.mu.Unlock()
	return nil
}

func GetConfig() Config {
	defaultLogger.mu.RLock()
	defer defaultLogger.mu.RUnlock()
	out := Config{Enabled: make(map[Category]bool, len(defaultLogger.cfg.Enabled))}
	for k, v := range defaultLogger.cfg.Enabled {
		out.Enabled[k] = v
	}
	return out
}

func SetCategory(cat Category, on bool) error {
	cfg := GetConfig()
	if cfg.Enabled == nil {
		cfg.Enabled = make(map[Category]bool)
	}
	cfg.Enabled[cat] = on
	return SaveConfig(cfg)
}

func Info(cat Category, format string, args ...any) {
	logf("INFO", cat, format, args...)
}

func Warn(cat Category, format string, args ...any) {
	logf("WARN", cat, format, args...)
}

func logf(level string, cat Category, format string, args ...any) {
	defaultLogger.mu.RLock()
	enabled := defaultLogger.cfg.Enabled[cat]
	logger := defaultLogger.logger
	defaultLogger.mu.RUnlock()

	if !enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	logger.Printf("[%s] [%s] %s", level, cat, msg)
}

func HTTP(method, path string, status int, duration time.Duration) {
	Info(CatHTTP, "%s %s -> %d (%s)", method, path, status, duration.Round(time.Millisecond))
}

func TCPConnected(username string) {
	Info(CatTCP, "client connected username=%s", username)
}

func TCPDisconnected(username string) {
	Info(CatTCP, "client disconnected username=%s", username)
}

func MessageSent(from, to string, messageID int64) {
	Info(CatMessaging, "message sent id=%d from=%s to=%s", messageID, from, to)
}

func MessageDelivered(messageID int64, to string) {
	Info(CatMessaging, "message delivered id=%d to=%s", messageID, to)
}

func FriendRequest(from, to string) {
	Info(CatFriends, "friend request from=%s to=%s", from, to)
}

func FriendAccepted(accepter, requester string) {
	Info(CatFriends, "friend accepted by=%s from=%s", accepter, requester)
}

func AdminAction(action, detail string) {
	Info(CatAdmin, "%s %s", action, detail)
}

func MigrationApplied(version int) {
	Info(CatStore, "migration applied version=%d", version)
}

func UserCreated(userID int64, username string) {
	Info(CatStore, "user created id=%d username=%s", userID, username)
}

func UserDeleted(userID int64) {
	Info(CatStore, "user deleted id=%d", userID)
}

func DatabaseWiped() {
	Info(CatStore, "database wiped all tables cleared")
}

func MessageStored(messageID, fromUserID, toUserID int64) {
	Info(CatStore, "message stored id=%d from_user_id=%d to_user_id=%d", messageID, fromUserID, toUserID)
}

func AdminFlagSet(userID int64, admin bool) {
	state := "false"
	if admin {
		state = "true"
	}
	Info(CatStore, "admin flag set user_id=%d admin=%s", userID, state)
}
