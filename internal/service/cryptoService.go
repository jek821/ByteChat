package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"

	"ByteChat/internal/paths"
	"golang.org/x/crypto/argon2"
)

func generateServerAuth() error {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"My App"}},
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

	certPath, err := paths.CertPath()
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

	keyPath, err := paths.KeyPath()
	if err != nil {
		return err
	}
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	return nil
}

// GenerateE2EKeypair generates a new X25519 keypair for E2E encryption.
// Returns raw public and private key bytes.
func GenerateE2EKeypair() (pubKey, privKey []byte, err error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv.PublicKey().Bytes(), priv.Bytes(), nil
}

// EncryptPrivateKey encrypts a raw private key using AES-256-GCM with an
// Argon2-derived key from the user's password. Returns the ciphertext and the salt.
func EncryptPrivateKey(privKey []byte, password string) (encryptedKey, salt []byte, err error) {
	salt = make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, err
	}

	aesKey := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	// Prepend nonce to ciphertext so we can extract it on decryption.
	ciphertext := gcm.Seal(nonce, nonce, privKey, nil)
	return ciphertext, salt, nil
}

// DecryptPrivateKey reverses EncryptPrivateKey using the same password and salt.
func DecryptPrivateKey(encryptedKey, salt []byte, password string) ([]byte, error) {
	aesKey := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedKey) < nonceSize {
		return nil, errors.New("encrypted key too short")
	}
	nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// InitClientE2EKeys checks whether the client already has a local E2E private key.
// If not, it generates a new keypair, saves it to disk, and returns the public key
// along with the encrypted private key bundle to be uploaded to the server.
// If a key already exists locally, uploadNeeded is false and encPrivKey/salt are nil.
func InitClientE2EKeys(password string) (pubKey []byte, encPrivKey []byte, salt []byte, uploadNeeded bool, err error) {
	privKeyPath, err := paths.ClientE2EPrivKeyPath()
	if err != nil {
		return nil, nil, nil, false, err
	}

	// Key already exists on this device — nothing to do.
	if _, err := os.Stat(privKeyPath); err == nil {
		raw, err := os.ReadFile(privKeyPath)
		if err != nil {
			return nil, nil, nil, false, err
		}
		block, _ := pem.Decode(raw)
		if block == nil {
			return nil, nil, nil, false, errors.New("failed to decode local E2E private key PEM")
		}
		priv, err := ecdh.X25519().NewPrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, nil, false, err
		}
		return priv.PublicKey().Bytes(), nil, nil, false, nil
	}

	// No key on disk — generate a fresh keypair.
	pubKeyBytes, privKeyBytes, err := GenerateE2EKeypair()
	if err != nil {
		return nil, nil, nil, false, err
	}

	pubKeyPath, err := paths.ClientE2EPubKeyPath()
	if err != nil {
		return nil, nil, nil, false, err
	}

	if err := writePEM(privKeyPath, "E2E PRIVATE KEY", privKeyBytes, 0600); err != nil {
		return nil, nil, nil, false, err
	}
	if err := writePEM(pubKeyPath, "E2E PUBLIC KEY", pubKeyBytes, 0644); err != nil {
		return nil, nil, nil, false, err
	}

	encPrivKey, salt, err = EncryptPrivateKey(privKeyBytes, password)
	if err != nil {
		return nil, nil, nil, false, err
	}

	return pubKeyBytes, encPrivKey, salt, true, nil
}

// RestoreE2EKeysFromServer decrypts an encrypted private key bundle received from the
// server and saves the keypair to disk. Called on first login on a new device.
func RestoreE2EKeysFromServer(encPrivKey, salt []byte, password string) error {
	privKeyBytes, err := DecryptPrivateKey(encPrivKey, salt, password)
	if err != nil {
		return err
	}

	priv, err := ecdh.X25519().NewPrivateKey(privKeyBytes)
	if err != nil {
		return err
	}

	privKeyPath, err := paths.ClientE2EPrivKeyPath()
	if err != nil {
		return err
	}
	pubKeyPath, err := paths.ClientE2EPubKeyPath()
	if err != nil {
		return err
	}

	if err := writePEM(privKeyPath, "E2E PRIVATE KEY", privKeyBytes, 0600); err != nil {
		return err
	}
	return writePEM(pubKeyPath, "E2E PUBLIC KEY", priv.PublicKey().Bytes(), 0644)
}

func writePEM(path, pemType string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: pemType, Bytes: data})
}
