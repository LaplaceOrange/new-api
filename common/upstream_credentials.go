package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
)

// Upstream credentials require an explicitly configured, stable deployment secret.
func upstreamCredentialCipher() (cipher.AEAD, error) {
	secret := os.Getenv("CRYPTO_SECRET")
	if secret == "" {
		secret = os.Getenv("SESSION_SECRET")
	}
	if len(secret) < 32 || secret == "random_string" {
		return nil, errors.New("Configure CRYPTO_SECRET or SESSION_SECRET with at least 32 characters before storing upstream credentials.")
	}
	key := sha256.Sum256([]byte("new-api:upstream-credentials:v1:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptUpstreamCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	aead, err := upstreamCredentialCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := aead.Seal(nonce, nonce, []byte(value), []byte("upstream:v1"))
	return base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptUpstreamCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	aead, err := upstreamCredentialCipher()
	if err != nil {
		return "", err
	}
	payload, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(payload) < aead.NonceSize() {
		return "", errors.New("Upstream credentials cannot be decrypted; configure them again.")
	}
	plain, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], []byte("upstream:v1"))
	if err != nil {
		return "", errors.New("Upstream credentials cannot be decrypted; configure them again.")
	}
	return string(plain), nil
}
