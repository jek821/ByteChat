package client

import (
	"crypto/tls"
	"net/http"
	"time"
)

func TLSConfig(insecure bool) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	}
}

func NewHTTPClient(insecure bool, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: TLSConfig(insecure),
		},
	}
}
