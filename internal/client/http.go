package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ByteChat/internal/paths"
	"ByteChat/internal/service"
)

type HTTPAuth struct {
	baseURL string
	client  *http.Client
}

func NewHTTPAuth(baseURL string) *HTTPAuth {
	baseURL = strings.TrimRight(baseURL, "/")
	return &HTTPAuth{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, // self-signed localhost cert
				},
			},
		},
	}
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

func (c *HTTPAuth) Register(username, password string) (Credentials, error) {
	req := authRequest{Username: username, Password: password}

	pubKey, encPrivKey, salt, uploadNeeded, err := service.InitClientE2EKeys(password)
	if err != nil {
		return Credentials{}, err
	}
	if uploadNeeded {
		req.E2EPublicKey = base64.StdEncoding.EncodeToString(pubKey)
		req.E2EEncryptedPrivateKey = base64.StdEncoding.EncodeToString(encPrivKey)
		req.E2EKeySalt = base64.StdEncoding.EncodeToString(salt)
	}

	resp, err := c.post("/api/register", req)
	if err != nil {
		return Credentials{}, err
	}

	if err := c.restoreE2EIfNeeded(password, resp); err != nil {
		return Credentials{}, err
	}

	return Credentials{Username: resp.Username, Token: resp.Token}, nil
}

func (c *HTTPAuth) Login(username, password string) (Credentials, error) {
	resp, err := c.post("/api/login", authRequest{Username: username, Password: password})
	if err != nil {
		return Credentials{}, err
	}

	if err := c.restoreE2EIfNeeded(password, resp); err != nil {
		return Credentials{}, err
	}

	return Credentials{Username: resp.Username, Token: resp.Token}, nil
}

func (c *HTTPAuth) restoreE2EIfNeeded(password string, resp authResponse) error {
	hasLocalKey, err := clientHasE2EKey()
	if err != nil {
		return err
	}
	if hasLocalKey || resp.E2EEncryptedPrivateKey == "" {
		return nil
	}

	encPrivKey, err := base64.StdEncoding.DecodeString(resp.E2EEncryptedPrivateKey)
	if err != nil {
		return fmt.Errorf("decode e2e_encrypted_private_key: %w", err)
	}
	salt, err := base64.StdEncoding.DecodeString(resp.E2EKeySalt)
	if err != nil {
		return fmt.Errorf("decode e2e_key_salt: %w", err)
	}

	return service.RestoreE2EKeysFromServer(encPrivKey, salt, password)
}

func clientHasE2EKey() (bool, error) {
	path, err := paths.ClientE2EPrivKeyPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (c *HTTPAuth) post(path string, body any) (authResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return authResponse{}, err
	}

	res, err := c.client.Post(c.baseURL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return authResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return authResponse{}, err
	}

	if res.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(raw, &errResp) == nil && errResp.Error != "" {
			return authResponse{}, fmt.Errorf("%s", errResp.Error)
		}
		return authResponse{}, fmt.Errorf("request failed with status %d", res.StatusCode)
	}

	var resp authResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return authResponse{}, err
	}
	return resp, nil
}
