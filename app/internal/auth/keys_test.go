package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// writeTestKeyPair generates a throwaway RSA key pair and writes it as PEM
// files into a temp dir, returning the two paths. Tests must not depend on
// the real keys/ directory: it is gitignored, so it does not exist in CI.
func writeTestKeyPair(t *testing.T) (privatePath, publicPath string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	dir := t.TempDir()
	privatePath = filepath.Join(dir, "private.pem")
	publicPath = filepath.Join(dir, "public.pem")

	write := func(path, blockType string, der []byte) {
		data := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(privatePath, "PRIVATE KEY", privateDER)
	write(publicPath, "PUBLIC KEY", publicDER)

	return privatePath, publicPath
}

func TestLoadKeys(t *testing.T) {
	privatePath, publicPath := writeTestKeyPair(t)

	privateKey, err := LoadPrivateKey(privatePath)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	publicKey, err := LoadPublicKey(publicPath)
	if err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}

	if privateKey == nil {
		t.Fatal("private key is nil")
	}

	if publicKey == nil {
		t.Fatal("public key is nil")
	}

	// The loaded public key must match the loaded private key's own public half.
	if !privateKey.PublicKey.Equal(publicKey) {
		t.Fatal("public key does not match private key")
	}
}

func TestLoadPrivateKeyErrors(t *testing.T) {
	if _, err := LoadPrivateKey(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("expected an error for a missing file")
	}

	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	if err := os.WriteFile(garbage, []byte("not a pem file"), 0o600); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	if _, err := LoadPrivateKey(garbage); err == nil {
		t.Fatal("expected an error for a non-PEM file")
	}
}
