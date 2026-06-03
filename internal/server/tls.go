package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	"ByteChat/internal/paths"
)

// EnsureCert generates a self-signed TLS certificate if one is not already on disk.
func EnsureCert() error {
	certPath, err := paths.CertPath()
	if err != nil {
		return err
	}
	keyPath, err := paths.KeyPath()
	if err != nil {
		return err
	}

	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"byteChat"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	return pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
}

// LoadTLSConfig loads or creates the server certificate and returns a TLS config.
func LoadTLSConfig() (*tls.Config, error) {
	if err := EnsureCert(); err != nil {
		return nil, err
	}

	certPath, err := paths.CertPath()
	if err != nil {
		return nil, err
	}
	keyPath, err := paths.KeyPath()
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// CertFiles returns the paths to the TLS certificate and private key.
func CertFiles() (certPath, keyPath string, err error) {
	certPath, err = paths.CertPath()
	if err != nil {
		return "", "", err
	}
	keyPath, err = paths.KeyPath()
	if err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
