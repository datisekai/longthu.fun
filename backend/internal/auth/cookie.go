package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CookieName is the session cookie's name. Short prefix `lt_` = longthu.
const CookieName = "lt_session"

// SetSession writes the JWT as an HTTP-only Secure SameSite=Lax cookie.
// `Secure` is enabled when baseURL begins with `https://` (production); for
// `http://localhost...` we keep it off so the dev server works without HTTPS.
func SetSession(c *gin.Context, token string, baseURL string) {
	maxAge := int(SessionTTL.Seconds())
	secure := strings.HasPrefix(baseURL, "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, token, maxAge, "/", "", secure, true /* httpOnly */)
}

// ClearSession wipes the session cookie (Max-Age=0 tells the browser to drop it).
func ClearSession(c *gin.Context, baseURL string) {
	secure := strings.HasPrefix(baseURL, "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, "", -1, "/", "", secure, true)
}
