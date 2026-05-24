package groups

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

const maxGroupNameLen = 120

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/groups/:id", h.handleGet)
	r.POST("/groups", h.handleCreate)
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

type createReq struct {
	Name string `json:"name"`
}

func (h *Handler) handleCreate(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.ReplyValidation(c, "Thông tin không hợp lệ", []httpx.FieldError{
			{Field: "name", Message: "Tên nhóm không được để trống"},
		})
		return
	}
	if len(name) > maxGroupNameLen {
		httpx.ReplyValidation(c, "Thông tin không hợp lệ", []httpx.FieldError{
			{Field: "name", Message: "Tên nhóm tối đa 120 ký tự"},
		})
		return
	}

	group, err := h.svc.Create(c.Request.Context(), hostID, CreateParams{Name: name})
	if err != nil {
		if errors.Is(err, ErrNameMissing) {
			httpx.ReplyValidation(c, "Thông tin không hợp lệ", []httpx.FieldError{
				{Field: "name", Message: "Tên nhóm không được để trống"},
			})
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Không tạo được nhóm", "")
		return
	}

	c.JSON(http.StatusCreated, group)
}
