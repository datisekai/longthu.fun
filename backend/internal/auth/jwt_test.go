package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret-32-bytes-long-for-hs256!!")

func TestJWT_IssueThenVerify(t *testing.T) {
	token, err := IssueToken(testSecret, 42, "pro")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if !strings.Contains(token, ".") {
		t.Fatal("token should be a dotted JWT string")
	}

	claims, err := VerifyToken(testSecret, token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.HostUserID != 42 {
		t.Errorf("HostUserID = %d; want 42", claims.HostUserID)
	}
	if claims.Tier != "pro" {
		t.Errorf("Tier = %q; want pro", claims.Tier)
	}
	if claims.Issuer != "longthu.fun" {
		t.Errorf("Issuer = %q; want longthu.fun", claims.Issuer)
	}
}

func TestJWT_WrongSecretRejected(t *testing.T) {
	token, _ := IssueToken(testSecret, 1, "free")
	_, err := VerifyToken([]byte("different-secret-xxxxxxxxxxxxxxxxxxxxx"), token)
	if err == nil {
		t.Fatal("VerifyToken with wrong secret should fail")
	}
}

func TestJWT_ExpiredRejected(t *testing.T) {
	// Hand-roll an already-expired token.
	claims := Claims{
		Tier: "free",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Issuer:    "longthu.fun",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := tok.SignedString(testSecret)

	_, err := VerifyToken(testSecret, signed)
	if err == nil {
		t.Fatal("expired token should be rejected")
	}
}

func TestJWT_MalformedRejected(t *testing.T) {
	for _, bad := range []string{"", "not-a-jwt", "a.b.c", "header.payload.sig"} {
		_, err := VerifyToken(testSecret, bad)
		if err == nil {
			t.Errorf("malformed token %q should be rejected", bad)
		}
	}
}
