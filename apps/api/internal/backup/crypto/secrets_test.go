package crypto

import (
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decrypt(enc, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "secret" {
		t.Fatalf("got %q", plain)
	}
}

func TestParseKeyBase64(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	encoded := "dGVzdC1rZXktdGVzdC1rZXktdGVzdC1rZXktdGVzdC1rZXk="
	parsed, err := ParseKey(encoded)
	if err == nil && len(parsed) != 32 {
		t.Fatalf("unexpected parsed key length %d", len(parsed))
	}

	parsed, err = ParseKey("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}
