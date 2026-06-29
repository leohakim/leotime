package auth

import "testing"

func TestHashPasswordVerifies(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}

	if VerifyPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	first, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("hash first password: %v", err)
	}
	second, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("hash second password: %v", err)
	}

	if first == second {
		t.Fatal("expected unique hashes for the same password")
	}
}
