package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
)

const (
	ctxHostUserID = "lt_host_user_id"
	ctxTier       = "lt_tier"
)

// SessionMiddleware validates the lt_session cookie and attaches host_user_id
// + tier to the Gin context. On any failure (missing / bad / expired token)
// it short-circuits with a 401 Problem Details response.
//
// Tier in the JWT is "snapshot at issue"; downstream tier-gated stories
// (e.g. Story 6.2 FR-32 Auto-Detect) should re-read tier from DB via
// svc.GetByID(ctx, id) if they need authoritative current tier — the cookie
// alone is acceptable for non-billing-critical reads.
func SessionMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(CookieName)
		if err != nil || raw == "" {
			httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
			c.Abort()
			return
		}
		claims, err := VerifyToken(secret, raw)
		if err != nil {
			httpx.Reply(c, http.StatusUnauthorized, "Phiên đã hết hạn", "")
			c.Abort()
			return
		}
		c.Set(ctxHostUserID, claims.HostUserID)
		c.Set(ctxTier, claims.Tier)
		c.Next()
	}
}

// HostUserIDFrom extracts the authenticated host_user_id, or returns ok=false
// when the middleware hasn't run / has failed. Always check ok before using.
func HostUserIDFrom(c *gin.Context) (uint64, bool) {
	v, exists := c.Get(ctxHostUserID)
	if !exists {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}

// TierFrom extracts the snapshot-at-issue tier. Re-read DB if you need
// real-time tier (post-admin-flip scenarios).
func TierFrom(c *gin.Context) (string, bool) {
	v, exists := c.Get(ctxTier)
	if !exists {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
