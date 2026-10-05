package common

import "testing"

func TestUpstreamCredentialEncryptionRoundTrip(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "01234567890123456789012345678901")
	ciphertext, err := EncryptUpstreamCredential("jwt-secret")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "jwt-secret" || ciphertext == "" {
		t.Fatalf("credential was not encrypted")
	}
	plaintext, err := DecryptUpstreamCredential(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "jwt-secret" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}

func TestUpstreamCredentialEncryptionRequiresStableSecret(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "")
	t.Setenv("SESSION_SECRET", "")
	if _, err := EncryptUpstreamCredential("jwt-secret"); err == nil {
		t.Fatal("expected stable secret validation error")
	}
}
