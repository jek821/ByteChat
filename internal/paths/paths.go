package paths

import (
	"os"
	"path/filepath"
)

const (
	dot  = ".gochat"
	serv = "server"
	cli  = "client"
	db   = "gochat.db"
)

func RootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	rootDir := filepath.Join(home, dot)
	if err := os.MkdirAll(rootDir, 0700); err != nil {
		return "", err
	}
	return rootDir, nil
}

func ServerDir() (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	serverDir := filepath.Join(root, serv)
	if err := os.MkdirAll(serverDir, 0700); err != nil {
		return "", err
	}

	return serverDir, nil
}

func ClientDir() (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	clientDir := filepath.Join(root, cli)
	if err := os.MkdirAll(clientDir, 0700); err != nil {
		return "", err
	}
	return clientDir, nil
}
func DBPath() (string, error) {
	root, err := ServerDir()
	if err != nil {
		return "", err
	}

	dbPath := filepath.Join(root, db)
	return dbPath, nil
}
