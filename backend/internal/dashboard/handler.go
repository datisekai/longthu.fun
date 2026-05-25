package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/dashboard", h.handleGetDashboard)
}

func (h *Handler) handleGetDashboard(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	dash, err := h.svc.GetDashboard(c.Request.Context(), hostID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	c.JSON(http.StatusOK, dash)
}
