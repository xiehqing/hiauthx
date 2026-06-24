package security

import (
	"strings"
	"testing"
)

func TestGenerateRSAKeyPair(t *testing.T) {
	pair, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair returned error: %v", err)
	}
	if !strings.Contains(pair.PublicKey, "-----BEGIN PUBLIC KEY-----") {
		t.Fatalf("public key should be PEM encoded: %q", pair.PublicKey)
	}
	if !strings.Contains(pair.PrivateKey, "-----BEGIN PRIVATE KEY-----") {
		t.Fatalf("private key should be PKCS#8 PEM encoded: %q", pair.PrivateKey)
	}
	if _, err := ParseRSAPrivateKey(pair.PrivateKey); err != nil {
		t.Fatalf("generated private key should be parseable: %v", err)
	}
}

func TestGenerateRSAKeyPairDefaultBits(t *testing.T) {
	pair, err := GenerateRSAKeyPair(0)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair returned error: %v", err)
	}
	if pair.PublicKey == "" || pair.PrivateKey == "" {
		t.Fatal("generated key pair should not be empty")
	}
}
