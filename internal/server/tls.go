package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"

	"ByteChat/internal/paths"
)

// TLSOptions configures how the server loads or generates TLS certificates.
type TLSOptions struct {
	CertPath string // optional; defaults to ~/.gochat/server/cert.pem
	KeyPath  string // optional; defaults to ~/.gochat/server/key.pem
	Hostname string // DNS name or IP for auto-generated self-signed certs
}

// EnsureCert generates a self-signed TLS certificate if one is not already on disk.
func EnsureCert(hostname string) error {
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

	dnsNames, ipAddrs := certSANs(hostname)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"byteChat"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
		IPAddresses:  ipAddrs,
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

func certSANs(hostname string) (dns []string, ips []net.IP) {
	seen := map[string]bool{}
	addDNS := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		dns = append(dns, name)
	}

	addDNS("localhost")
	if hostname != "" {
		if ip := net.ParseIP(hostname); ip != nil {
			ips = append(ips, ip)
		} else {
			addDNS(hostname)
		}
	}
	return dns, ips
}

// LoadTLSConfig loads or creates the server certificate and returns a TLS config.
func LoadTLSConfig(opts TLSOptions) (*tls.Config, error) {
	certPath := opts.CertPath
	keyPath := opts.KeyPath

	if certPath == "" || keyPath == "" {
		if err := EnsureCert(opts.Hostname); err != nil {
			return nil, err
		}
		var err error
		certPath, err = paths.CertPath()
		if err != nil {
			return nil, err
		}
		keyPath, err = paths.KeyPath()
		if err != nil {
			return nil, err
		}
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
