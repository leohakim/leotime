package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrInvalidKey   = errors.New("invalid secrets key")
	ErrInvalidValue = errors.New("invalid encrypted value")
)

func ParseKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidKey)
	}

	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	return nil, fmt.Errorf("%w: expected 32 bytes as base64 or hex", ErrInvalidKey)
}

func Encrypt(plaintext, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("%w: expected 32 bytes", ErrInvalidKey)
	}
	if len(plaintext) == 0 {
		return "", fmt.Errorf("%w: empty plaintext", ErrInvalidValue)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: expected 32 bytes", ErrInvalidKey)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("%w: decode base64: %v", ErrInvalidValue, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidValue)
	}

	nonce, payload := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypt failed", ErrInvalidValue)
	}

	return plaintext, nil
}
