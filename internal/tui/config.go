package tui

import "ByteChat/internal/client"

type Config struct {
	Auth    client.AuthClient
	TCPAddr string
}
