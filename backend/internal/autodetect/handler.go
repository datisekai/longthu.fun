package autodetect

import (
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
	r.GET("/groups/:groupId/auto-detect", h.handleGetSettings)
	r.POST("/groups/:groupId/auto-detect/enable", h.handleEnable)
	r.POST("/groups/:groupId/auto-detect/disable", h.handleDisable)
	r.POST("/groups/:groupId/auto-detect/test", h.handleTestCredentials)
}

// GET /api/v1/groups/:groupId/auto-detect
func (h *Handler) handleGetSettings(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	settings, err := h.svc.GetGroupAutoDetect(c.Request.Context(), hostID, groupID)
	if err != nil {
		if err == ErrNotFound {
			httpx.Reply(c, http.StatusNotFound, "Not found", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Không tải được cài đặt", "")
		return
	}

	c.JSON(http.StatusOK, settings)
}

// POST /api/v1/groups/:groupId/auto-detect/enable
type enableReq struct {
	BankAccountID uint64 `json:"bankAccountId"`
	ClientID      string `json:"clientId"`
	APIKey        string `json:"apiKey"`
	ChecksumKey   string `json:"checksumKey"`
}

func (h *Handler) handleEnable(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	var req enableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	var fieldErrs []httpx.FieldError
	if req.BankAccountID == 0 {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "bankAccountId", Message: "Chọn tài khoản ngân hàng"})
	}
	if req.ClientID == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "clientId", Message: "Nhập Client ID"})
	}
	if req.APIKey == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "apiKey", Message: "Nhập API Key"})
	}
	if req.ChecksumKey == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "checksumKey", Message: "Nhập Checksum Key"})
	}
	if len(fieldErrs) > 0 {
		httpx.ReplyValidation(c, "Thông tin chưa đầy đủ", fieldErrs)
		return
	}

	credentials := &PayOSCredentials{
		ClientID:    req.ClientID,
		APIKey:      req.APIKey,
		ChecksumKey: req.ChecksumKey,
	}

	err = h.svc.EnableAutoDetect(c.Request.Context(), hostID, groupID, req.BankAccountID, credentials)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không bật được auto-detect", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã bật auto-detect"})
}

// POST /api/v1/groups/:groupId/auto-detect/disable
func (h *Handler) handleDisable(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	err = h.svc.DisableAutoDetect(c.Request.Context(), hostID, groupID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không tắt được auto-detect", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Đã tắt auto-detect"})
}

// POST /api/v1/groups/:groupId/auto-detect/test
type testReq struct {
	ClientID    string `json:"clientId"`
	APIKey      string `json:"apiKey"`
	ChecksumKey string `json:"checksumKey"`
}

func (h *Handler) handleTestCredentials(c *gin.Context) {
	var req testReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	credentials := &PayOSCredentials{
		ClientID:    req.ClientID,
		APIKey:      req.APIKey,
		ChecksumKey: req.ChecksumKey,
	}

	err := h.svc.TestCredentials(c.Request.Context(), credentials)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Credentials sai — kiểm tra lại",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test ping ok — sẵn sàng bật auto-detect",
	})
}
