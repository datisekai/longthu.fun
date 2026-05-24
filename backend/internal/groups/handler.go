package groups

import (
	"errors"
	"net/http"
	"strconv"

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
	r.GET("/groups/:id", h.handleGet)
}

func (h *Handler) handleGet(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	group, err := h.svc.GetByID(c.Request.Context(), hostID, groupID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Reply(c, http.StatusNotFound, "Not found", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Không tải được nhóm", "")
		return
	}

	c.JSON(http.StatusOK, group)
}
