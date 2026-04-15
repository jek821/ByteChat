package paths

import (
	"os"
	"path/filepath"
)

const (
	dot            = ".gochat"
	serv           = "server"
	cli            = "client"
	db             = "gochat.db"
	cert           = "cert.pem"
	key            = "key.pem"
	e2eKeys        = "e2e_keys"
	clientPrivKey  = "e2e_private.pem"
	clientPubKey   = "e2e_public.pem"
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

func CertPath() (string, error) {
	dir, err := ServerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cert), nil
}

func KeyPath() (string, error) {
	dir, err := ServerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key), nil
}

// E2EKeysDir is the directory that holds friends' public keys for E2E encryption.
// Each friend's key is stored as <username>.pem inside this directory.
func E2EKeysDir() (string, error) {
	client, err := ClientDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(client, e2eKeys)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// FriendE2EPubKeyPath returns the path to a specific friend's stored public key.
func FriendE2EPubKeyPath(username string) (string, error) {
	dir, err := E2EKeysDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, username+".pem"), nil
}

// ClientE2EPrivKeyPath returns the path to the client's own E2E private key.
func ClientE2EPrivKeyPath() (string, error) {
	dir, err := ClientDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, clientPrivKey), nil
}

// ClientE2EPubKeyPath returns the path to the client's own E2E public key (shared with friends/server).
func ClientE2EPubKeyPath() (string, error) {
	dir, err := ClientDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, clientPubKey), nil
}
