package paymentintents

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/datisekai/longthu.fun/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.POST("/payment-intents", h.handleCreate)
	r.GET("/payment-intents/:intentCode", h.handleGet)
	r.POST("/payment-intents/:intentCode/mark-transferred", h.handleMarkTransferred)
}

type createIntentReq struct {
	PlayerCode string `json:"playerCode"`
	Mode      string `json:"mode"` // "current_session" | "all_unpaid"
}

func (h *Handler) handleCreate(c *gin.Context) {
	var req createIntentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Invalid request", "")
		return
	}

	if req.PlayerCode == "" {
		httpx.Reply(c, http.StatusBadRequest, "Missing playerCode", "")
		return
	}

	params := CreateParams{
		PlayerCode: req.PlayerCode,
		Mode:      ModeCurrentSession,
	}
	if req.Mode == "all_unpaid" {
		params.Mode = ModeAllUnpaid
	}

	result, err := h.svc.Create(c.Request.Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoUnpaidCharges):
			httpx.Reply(c, http.StatusUnprocessableEntity, "Không có khoản chưa trả nào", "")
		case errors.Is(err, ErrPlayerNotFound):
			httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		default:
			httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"intentCode":       result.Code,
		"amount":          result.Amount,
		"status":          result.Status,
		"transferContent": result.TransferContent,
		"expiresAt":       result.ExpiresAt,
		"bankInfo":        result.BankInfo,
	})
}

func (h *Handler) handleGet(c *gin.Context) {
	code := c.Param("intentCode")
	if code == "" {
		httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		return
	}

	result, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, ErrIntentNotFound):
			httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		case errors.Is(err, ErrIntentExpired):
			httpx.Reply(c, http.StatusGone, "QR đã hết hạn (sau 24h). Tạo QR mới?", "")
		default:
			httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":             result.Code,
		"amount":           result.Amount,
		"status":           result.Status,
		"transferContent":  result.TransferContent,
		"expiresAt":        result.ExpiresAt,
		"bankInfo":          result.BankInfo,
	})
}

func (h *Handler) handleMarkTransferred(c *gin.Context) {
	code := c.Param("intentCode")
	if code == "" {
		httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		return
	}

	err := h.svc.MarkTransferred(c.Request.Context(), code)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "pending_confirmation",
	})
}
