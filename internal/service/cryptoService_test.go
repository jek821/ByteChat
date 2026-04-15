package service

import (
	"bytes"
	"crypto/ecdh"
	"encoding/pem"
	"os"
	"testing"
)

func TestGenerateE2EKeypair(t *testing.T) {
	pub1, priv1, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}

	// X25519 keys are always 32 bytes.
	if len(pub1) != 32 {
		t.Fatalf("expected 32-byte public key, got %d", len(pub1))
	}
	if len(priv1) != 32 {
		t.Fatalf("expected 32-byte private key, got %d", len(priv1))
	}

	// Two calls should not produce the same keypair.
	pub2, priv2, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair second call: %v", err)
	}
	if bytes.Equal(pub1, pub2) {
		t.Fatal("two keypairs share the same public key")
	}
	if bytes.Equal(priv1, priv2) {
		t.Fatal("two keypairs share the same private key")
	}
}

func TestEncryptDecryptPrivateKey(t *testing.T) {
	_, privKey, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}

	password := "correct-horse-battery-staple"

	encKey, salt, err := EncryptPrivateKey(privKey, password)
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(salt))
	}

	decrypted, err := DecryptPrivateKey(encKey, salt, password)
	if err != nil {
		t.Fatalf("DecryptPrivateKey: %v", err)
	}
	if !bytes.Equal(decrypted, privKey) {
		t.Fatal("decrypted key does not match original")
	}
}

func TestDecryptPrivateKeyWrongPassword(t *testing.T) {
	_, privKey, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}

	encKey, salt, err := EncryptPrivateKey(privKey, "correct-password")
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}

	_, err = DecryptPrivateKey(encKey, salt, "wrong-password")
	if err == nil {
		t.Fatal("expected decryption to fail with wrong password, but it succeeded")
	}
}

func TestDecryptPrivateKeyTruncated(t *testing.T) {
	_, privKey, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}

	encKey, salt, err := EncryptPrivateKey(privKey, "password")
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}

	_, err = DecryptPrivateKey(encKey[:4], salt, "password")
	if err == nil {
		t.Fatal("expected decryption to fail with truncated ciphertext, but it succeeded")
	}
}

func TestInitClientE2EKeysNewDevice(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pubKey, encPrivKey, salt, uploadNeeded, err := InitClientE2EKeys("my-password")
	if err != nil {
		t.Fatalf("InitClientE2EKeys: %v", err)
	}

	if !uploadNeeded {
		t.Fatal("expected uploadNeeded=true on a fresh device")
	}
	if len(pubKey) != 32 {
		t.Fatalf("expected 32-byte public key, got %d", len(pubKey))
	}
	if len(encPrivKey) == 0 {
		t.Fatal("expected non-empty encrypted private key")
	}
	if len(salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(salt))
	}

	// Verify the encrypted key round-trips back to the public key.
	decrypted, err := DecryptPrivateKey(encPrivKey, salt, "my-password")
	if err != nil {
		t.Fatalf("DecryptPrivateKey after init: %v", err)
	}
	priv, err := ecdh.X25519().NewPrivateKey(decrypted)
	if err != nil {
		t.Fatalf("NewPrivateKey from decrypted: %v", err)
	}
	if !bytes.Equal(priv.PublicKey().Bytes(), pubKey) {
		t.Fatal("public key derived from decrypted private key does not match returned public key")
	}
}

func TestInitClientE2EKeysExistingDevice(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// First call — generates keys.
	pubKey1, _, _, uploadNeeded1, err := InitClientE2EKeys("my-password")
	if err != nil {
		t.Fatalf("first InitClientE2EKeys: %v", err)
	}
	if !uploadNeeded1 {
		t.Fatal("expected uploadNeeded=true on first call")
	}

	// Second call — key already on disk.
	pubKey2, encPrivKey2, salt2, uploadNeeded2, err := InitClientE2EKeys("my-password")
	if err != nil {
		t.Fatalf("second InitClientE2EKeys: %v", err)
	}
	if uploadNeeded2 {
		t.Fatal("expected uploadNeeded=false when key already exists")
	}
	if encPrivKey2 != nil || salt2 != nil {
		t.Fatal("expected nil encPrivKey and salt when key already exists")
	}
	if !bytes.Equal(pubKey1, pubKey2) {
		t.Fatal("public key changed between calls on the same device")
	}
}

func TestRestoreE2EKeysFromServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Simulate what the server has stored: a real keypair, encrypted.
	pubKey, privKey, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}
	encPrivKey, salt, err := EncryptPrivateKey(privKey, "my-password")
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}

	if err := RestoreE2EKeysFromServer(encPrivKey, salt, "my-password"); err != nil {
		t.Fatalf("RestoreE2EKeysFromServer: %v", err)
	}

	// Verify private key file was written and decodes to the correct key.
	home := os.Getenv("HOME")
	privPath := home + "/.gochat/client/e2e_private.pem"
	pubPath := home + "/.gochat/client/e2e_public.pem"

	privRaw, err := os.ReadFile(privPath)
	if err != nil {
		t.Fatalf("reading restored private key: %v", err)
	}
	block, _ := pem.Decode(privRaw)
	if block == nil {
		t.Fatal("failed to PEM-decode restored private key")
	}
	if !bytes.Equal(block.Bytes, privKey) {
		t.Fatal("restored private key does not match original")
	}

	pubRaw, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("reading restored public key: %v", err)
	}
	block, _ = pem.Decode(pubRaw)
	if block == nil {
		t.Fatal("failed to PEM-decode restored public key")
	}
	if !bytes.Equal(block.Bytes, pubKey) {
		t.Fatal("restored public key does not match original")
	}
}

func TestRestoreE2EKeysFromServerWrongPassword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, privKey, err := GenerateE2EKeypair()
	if err != nil {
		t.Fatalf("GenerateE2EKeypair: %v", err)
	}
	encPrivKey, salt, err := EncryptPrivateKey(privKey, "correct-password")
	if err != nil {
		t.Fatalf("EncryptPrivateKey: %v", err)
	}

	err = RestoreE2EKeysFromServer(encPrivKey, salt, "wrong-password")
	if err == nil {
		t.Fatal("expected RestoreE2EKeysFromServer to fail with wrong password")
	}
}
