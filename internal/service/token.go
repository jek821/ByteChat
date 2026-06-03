package service

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func HashSessionToken(token string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode session token: %w", err)
	}
	sum := sha256.Sum256(raw)
	return sum[:], nil
}
