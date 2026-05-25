package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/datisekai/longthu.fun/backend/internal/httpx"
)

// Tier management for admin: flip host tiers manually.

type Handler struct {
	db            *sql.DB
	adminEmail   string
	adminPassword string
}

func NewHandler(db *sql.DB, adminEmail, adminPassword string) *Handler {
	return &Handler{db: db, adminEmail: adminEmail, adminPassword: adminPassword}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.handleLogin)
	r.GET("/pending-upgrades", h.handleListPending)
	r.POST("/flip-tier", h.handleFlipTier)
}

func (h *Handler) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Invalid request", "")
		return
	}
	if req.Email != h.adminEmail || req.Password != h.adminPassword {
		httpx.Reply(c, http.StatusUnauthorized, "Sai thông tin đăng nhập", "")
		return
	}
	// Simple token-based auth for MVP
	c.JSON(http.StatusOK, gin.H{"ok": true, "token": "admin-token"})
}

func (h *Handler) handleListPending(c *gin.Context) {
	// List hosts with pending tier change requests
	// In MVP this is simplified - just list all hosts
	q := NewAdminQueries(h.db)
	hosts, err := q.ListHosts(c.Request.Context())
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}
	c.JSON(http.StatusOK, gin.H{"hosts": hosts})
}

func (h *Handler) handleFlipTier(c *gin.Context) {
	var req struct {
		HostID uint64 `json:"hostId"`
		Tier   string `json:"tier"` // "free" | "pro" | "pro_plus"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Invalid request", "")
		return
	}
	if req.Tier != "free" && req.Tier != "pro" && req.Tier != "pro_plus" {
		httpx.Reply(c, http.StatusBadRequest, "Tier không hợp lệ", "")
		return
	}

	q := NewAdminQueries(h.db)
	err := q.UpdateHostTier(c.Request.Context(), req.HostID, req.Tier)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Admin queries (simplified - reads from users table)
type adminQueries struct {
	db *sql.DB
}

func NewAdminQueries(db *sql.DB) *adminQueries {
	return &adminQueries{db: db}
}

type HostInfo struct {
	ID          uint64 `json:"id"`
	Email      string `json:"email"`
	DisplayName string `json:"displayName"`
	Tier       string `json:"tier"`
}

func (q *adminQueries) ListHosts(ctx context.Context) ([]HostInfo, error) {
	rows, err := q.db.QueryContext(ctx, "SELECT id, email, display_name, tier FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []HostInfo
	for rows.Next() {
		var h HostInfo
		if err := rows.Scan(&h.ID, &h.Email, &h.DisplayName, &h.Tier); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

func (q *adminQueries) UpdateHostTier(ctx context.Context, hostID uint64, tier string) error {
	_, err := q.db.ExecContext(ctx, "UPDATE users SET tier = ?, tier_changed_at = NOW() WHERE id = ?", tier, hostID)
	return err
}

// Parsing helper
func parseUint64(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err
}

var _ = fmt.Sprintf // avoid unused import
