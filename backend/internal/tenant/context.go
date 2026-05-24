// Package tenant centralizes host-scoped request helpers.
package tenant

import (
	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/auth"
)

// HostID extracts the authenticated host ID injected by auth.SessionMiddleware.
// Host-scoped handlers should call this and pass the returned hostID as the
// first non-context parameter to their service/repository methods.
func HostID(c *gin.Context) (uint64, bool) {
	return auth.HostUserIDFrom(c)
}
