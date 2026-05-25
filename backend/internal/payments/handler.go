package payments

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterDashboardRoutes(r *gin.RouterGroup) {
	r.GET("/dashboard/suspected", h.handleListSuspected)
	r.GET("/dashboard/unmatched", h.handleListUnmatched)
	r.POST("/payments/:paymentId/confirm", h.handleConfirmPayment)
	r.POST("/payments/:paymentId/reject", h.handleRejectPayment)
	r.POST("/payments/:paymentId/link", h.handleLinkPayment)
}

func (h *Handler) handleListSuspected(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	q := NewService(h.db)
	payments, err := q.ListSuspectedPayments(c.Request.Context(), hostID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

func (h *Handler) handleListUnmatched(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	q := NewService(h.db)
	payments, err := q.ListUnmatchedPayments(c.Request.Context(), hostID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

func (h *Handler) handleConfirmPayment(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	svc := NewService(h.db)
	err = svc.ConfirmSuspectedPayment(c.Request.Context(), hostID, paymentID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã xác nhận payment"})
}

func (h *Handler) handleRejectPayment(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	svc := NewService(h.db)
	err = svc.RejectSuspectedPayment(c.Request.Context(), hostID, paymentID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã chuyển sang chưa khớp"})
}

type linkReq struct {
	PlayerID  uint64   `json:"playerId"`
	ChargeIDs []uint64 `json:"chargeIds"`
}

func (h *Handler) handleLinkPayment(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	paymentID, err := strconv.ParseUint(c.Param("paymentId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	var req linkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	svc := NewService(h.db)
	err = svc.LinkPaymentToPlayer(c.Request.Context(), hostID, paymentID, req.PlayerID, req.ChargeIDs)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã khớp payment"})
}
