package client

import (
	"errors"
	"fmt"
)

type Credentials struct {
	Username string
	Token    string
}

type AuthClient interface {
	Login(username, password string) (Credentials, error)
	Register(username, password string) (Credentials, error)
}

type MockAuth struct{}

func (MockAuth) Login(username, password string) (Credentials, error) {
	if username == "" || password == "" {
		return Credentials{}, errors.New("username and password are required")
	}
	return Credentials{Username: username, Token: "mock-token"}, nil
}

func (MockAuth) Register(username, password string) (Credentials, error) {
	if username == "" || password == "" {
		return Credentials{}, errors.New("username and password are required")
	}
	if len(password) < 8 {
		return Credentials{}, fmt.Errorf("password must be at least 8 characters")
	}
	return Credentials{Username: username, Token: "mock-token"}, nil
}
