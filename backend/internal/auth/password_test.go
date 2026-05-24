package auth

import "testing"

func TestPassword_RoundTrip(t *testing.T) {
	plain := "correct-horse-battery-staple"
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == plain {
		t.Fatal("hash equals plaintext — bcrypt broken?")
	}
	if !VerifyPassword(hash, plain) {
		t.Error("VerifyPassword(hash, plain) = false; want true")
	}
}

func TestPassword_WrongPasswordRejected(t *testing.T) {
	hash, err := HashPassword("right-one")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if VerifyPassword(hash, "wrong-one") {
		t.Error("VerifyPassword(hash, wrongPassword) = true; want false")
	}
}

func TestPassword_DifferentHashesEachTime(t *testing.T) {
	a, _ := HashPassword("same-plain")
	b, _ := HashPassword("same-plain")
	if a == b {
		t.Error("bcrypt produced identical hashes for same input — salt broken?")
	}
}
