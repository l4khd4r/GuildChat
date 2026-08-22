package auth

import "testing"

func TestLoadKeys(t *testing.T) {
	privateKey, err := LoadPrivateKey("../../../keys/private.pem")
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	publicKey, err := LoadPublicKey("../../../keys/public.pem")
	if err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}

	if privateKey == nil {
		t.Fatal("private key is nil")
	}

	if publicKey == nil {
		t.Fatal("public key is nil")
	}
}
