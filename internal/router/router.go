package router

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"ByteChat/internal/service"
)

type AuthHandler interface {
	Register(ctx context.Context, in service.RegisterInput) (service.AuthResult, error)
	Login(ctx context.Context, username, password string) (service.AuthResult, error)
}

func New(auth AuthHandler, admin AdminHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", registerHandler(auth))
	mux.HandleFunc("POST /api/login", loginHandler(auth))
	if admin != nil {
		registerAdminRoutes(mux, admin)
	}
	return loggingMiddleware(mux)
}

type authRequest struct {
	Username               string `json:"username"`
	Password               string `json:"password"`
	E2EPublicKey           string `json:"e2e_public_key,omitempty"`
	E2EEncryptedPrivateKey string `json:"e2e_encrypted_private_key,omitempty"`
	E2EKeySalt             string `json:"e2e_key_salt,omitempty"`
}

type authResponse struct {
	Token                  string `json:"token"`
	Username               string `json:"username"`
	E2EEncryptedPrivateKey string `json:"e2e_encrypted_private_key,omitempty"`
	E2EKeySalt             string `json:"e2e_key_salt,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func registerHandler(auth AuthHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		in := service.RegisterInput{
			Username: req.Username,
			Password: req.Password,
		}

		if req.E2EPublicKey != "" {
			pubKey, err := base64.StdEncoding.DecodeString(req.E2EPublicKey)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid e2e_public_key")
				return
			}
			encPrivKey, err := base64.StdEncoding.DecodeString(req.E2EEncryptedPrivateKey)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid e2e_encrypted_private_key")
				return
			}
			salt, err := base64.StdEncoding.DecodeString(req.E2EKeySalt)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid e2e_key_salt")
				return
			}
			in.PubKey = pubKey
			in.E2E = &service.E2EBundle{EncPrivKey: encPrivKey, Salt: salt}
		}

		result, err := auth.Register(r.Context(), in)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeAuthSuccess(w, result)
	}
}

func loginHandler(auth AuthHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := auth.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeAuthSuccess(w, result)
	}
}

func writeAuthSuccess(w http.ResponseWriter, result service.AuthResult) {
	resp := authResponse{
		Token:    result.Token,
		Username: result.Username,
	}
	if len(result.EncPrivKey) > 0 {
		resp.E2EEncryptedPrivateKey = base64.StdEncoding.EncodeToString(result.EncPrivKey)
	}
	if len(result.Salt) > 0 {
		resp.E2EKeySalt = base64.StdEncoding.EncodeToString(result.Salt)
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrUserExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
