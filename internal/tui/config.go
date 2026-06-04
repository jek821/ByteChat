package tui

import "ByteChat/internal/client"

type Config struct {
	Auth        client.AuthClient
	Admin       *client.AdminClient
	TCPAddr     string
	ServerLabel string
	InsecureTLS bool
}
