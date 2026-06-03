package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ByteChat/internal/logx"
	"ByteChat/internal/service"
	"ByteChat/internal/store"
)

type AdminHandler interface {
	Login(ctx context.Context, username, password string) (service.AdminLoginResult, error)
	ValidateToken(ctx context.Context, token string) (string, error)
	Dashboard(ctx context.Context) (service.AdminDashboard, error)
	ListUsers(ctx context.Context) ([]store.UserSummary, error)
	DeleteUser(ctx context.Context, adminUsername, targetUsername string) error
	WipeDatabase(ctx context.Context, adminUsername, confirm string) error
	GetLogConfig() logx.Config
	SetLogCategory(cat logx.Category, enabled bool) error
}

type UserSummary = store.UserSummary

func registerAdminRoutes(mux *http.ServeMux, admin AdminHandler) {
	mux.HandleFunc("POST /api/admin/login", adminLoginHandler(admin))
	mux.HandleFunc("GET /api/admin/dashboard", adminAuth(admin, adminDashboardHandler(admin)))
	mux.HandleFunc("GET /api/admin/users", adminAuth(admin, adminListUsersHandler(admin)))
	mux.HandleFunc("DELETE /api/admin/users/{username}", adminAuth(admin, adminDeleteUserHandler(admin)))
	mux.HandleFunc("POST /api/admin/wipe", adminAuth(admin, adminWipeHandler(admin)))
	mux.HandleFunc("GET /api/admin/logs", adminAuth(admin, adminGetLogsHandler(admin)))
	mux.HandleFunc("PUT /api/admin/logs/{category}", adminAuth(admin, adminSetLogHandler(admin)))
}

func adminLoginHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		result, err := admin.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrNotAdmin) || errors.Is(err, service.ErrInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "invalid admin credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"token":    result.Token,
			"username": result.Username,
		})
	}
}

func adminAuth(admin AdminHandler, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing admin token")
			return
		}
		username, err := admin.ValidateToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}
		ctx := context.WithValue(r.Context(), adminUserKey{}, username)
		next(w, r.WithContext(ctx))
	}
}

type adminUserKey struct{}

func adminUsername(r *http.Request) string {
	v, _ := r.Context().Value(adminUserKey{}).(string)
	return v
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}

func adminDashboardHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dash, err := admin.Dashboard(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, dash)
	}
}

func adminListUsersHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := admin.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	}
}

func adminDeleteUserHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.PathValue("username")
		if err := admin.DeleteUser(r.Context(), adminUsername(r), target); err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

type wipeRequest struct {
	Confirm string `json:"confirm"`
}

func adminWipeHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req wipeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := admin.WipeDatabase(r.Context(), adminUsername(r), req.Confirm); err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "wiped"})
	}
}

func adminGetLogsHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, admin.GetLogConfig())
	}
}

func adminSetLogHandler(admin AdminHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cat := logx.Category(r.PathValue("category"))
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := admin.SetLogCategory(cat, req.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, admin.GetLogConfig())
	}
}

func writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrCannotDeleteAdmin):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrConfirmRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logx.HTTP(r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
