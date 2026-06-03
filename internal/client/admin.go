package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ByteChat/internal/logx"
	"ByteChat/internal/service"
	"ByteChat/internal/store"
)

type AdminClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewAdminClient(baseURL string) *AdminClient {
	return &AdminClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

func (c *AdminClient) Login(username, password string) (Credentials, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	res, err := c.client.Post(c.baseURL+"/api/admin/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return Credentials{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return Credentials{}, parseError(raw, res.StatusCode)
	}
	var resp struct {
		Token    string `json:"token"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Credentials{}, err
	}
	c.token = resp.Token
	return Credentials{Username: resp.Username, Token: resp.Token}, nil
}

func (c *AdminClient) Dashboard() (service.AdminDashboard, error) {
	var dash service.AdminDashboard
	err := c.getJSON("/api/admin/dashboard", &dash)
	return dash, err
}

func (c *AdminClient) ListUsers() ([]store.UserSummary, error) {
	var resp struct {
		Users []store.UserSummary `json:"users"`
	}
	if err := c.getJSON("/api/admin/users", &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}

func (c *AdminClient) DeleteUser(username string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/admin/users/"+username, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *AdminClient) WipeDatabase(confirm string) error {
	body, _ := json.Marshal(map[string]string{"confirm": confirm})
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/admin/wipe", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}

func (c *AdminClient) SetLogCategory(cat logx.Category, enabled bool) error {
	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/api/admin/logs/"+string(cat), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}

func (c *AdminClient) getJSON(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *AdminClient) do(req *http.Request, out any) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseError(raw, res.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func parseError(raw []byte, status int) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}
	return fmt.Errorf("request failed with status %d", status)
}
