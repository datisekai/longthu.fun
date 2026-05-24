package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionTTL is the lifetime of an issued JWT (and the matching cookie Max-Age).
// 30 days per PRD §4.1 FR-1.
const SessionTTL = 30 * 24 * time.Hour

// Claims is the JWT payload for an authenticated host session.
// `sub` carries the host_user_id (RFC 7519 standard claim, stringified for safety).
// `tier` is snapshot-at-issue; the session middleware re-fetches the current
// tier from DB on each request to avoid stale-tier authorization.
type Claims struct {
	HostUserID uint64 `json:"-"` // populated by Verify after parsing `sub`
	Tier       string `json:"tier"`
	jwt.RegisteredClaims
}

// IssueToken signs a session token for the given host.
func IssueToken(secret []byte, hostUserID uint64, tier string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Tier: tier,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", hostUserID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionTTL)),
			Issuer:    "longthu.fun",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// VerifyToken parses + validates the token, returning typed Claims.
// Returns an error on expired, malformed, or wrong-signature tokens.
func VerifyToken(secret []byte, tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("auth.VerifyToken: unexpected signing method %v", t.Header["alg"])
			}
			return secret, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("auth.VerifyToken: %w", err)
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("auth.VerifyToken: invalid token")
	}
	// Hydrate HostUserID from `sub`.
	var id uint64
	if _, err := fmt.Sscanf(claims.Subject, "%d", &id); err != nil {
		return nil, fmt.Errorf("auth.VerifyToken: bad subject %q: %w", claims.Subject, err)
	}
	claims.HostUserID = id
	return claims, nil
}
