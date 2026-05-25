package players

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/auth"
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
	r.POST("/groups/:groupId/players", h.handleBulkCreate)
	r.GET("/groups/:groupId/players", h.handleListActive)
	r.PATCH("/groups/:groupId/players/:playerId", h.handleUpdate)
	r.POST("/groups/:groupId/players/:playerId/deactivate", h.handleDeactivate)
	r.POST("/groups/:groupId/players/:playerId/reactivate", h.handleReactivate)
	r.POST("/groups/:groupId/players/:playerId/reset-code", h.handleResetCode)
}

type bulkCreateReq struct {
	Names []string `json:"names"`
}

func (h *Handler) handleBulkCreate(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	tier, _ := auth.TierFrom(c)

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	var req bulkCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	players, err := h.svc.BulkCreate(c.Request.Context(), hostID, groupID, req.Names, tier)
	if err == nil {
		c.JSON(http.StatusCreated, gin.H{"players": players})
		return
	}

	switch {
	case errors.Is(err, ErrGroupNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrNamesEmpty):
		httpx.ReplyValidation(c, "Thông tin không hợp lệ", []httpx.FieldError{
			{Field: "names", Message: "Cần ít nhất 1 tên người chơi"},
		})
	case errors.Is(err, ErrTierCapExceeded):
		var typed *TypedError
		errors.As(err, &typed)
		msg := fmt.Sprintf("Tier %s tối đa %d người/group — hiện có %d, thêm %d sẽ vượt.",
			displayTier(typed.Cap.Tier), typed.Cap.Cap, typed.Cap.CurrentCount, typed.Cap.ToAdd)
		httpx.ReplyValidation(c, "Vượt giới hạn tier", []httpx.FieldError{
			{Field: "names", Message: msg},
		})
	case errors.Is(err, ErrDuplicateInSubmit):
		var typed *TypedError
		errors.As(err, &typed)
		httpx.ReplyValidation(c, "Tên trùng nhau", []httpx.FieldError{
			{Field: "names", Message: fmt.Sprintf("Tên trùng nhau trong danh sách: %s", typed.Dup.Names[0])},
		})
	case errors.Is(err, ErrDuplicateAgainstRoster):
		var typed *TypedError
		errors.As(err, &typed)
		httpx.Reply(c, http.StatusConflict, "Tên trùng nhau",
			fmt.Sprintf("%q đã có trong nhóm — mỗi nhóm chỉ được 1 tên", typed.Dup.Names[0]))
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không thêm được người chơi", "")
	}
}

func (h *Handler) handleListActive(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	players, err := h.svc.ListActive(c.Request.Context(), hostID, groupID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"players": players})
		return
	}
	if errors.Is(err, ErrGroupNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Không đọc được danh sách người chơi", "")
}

type updatePlayerReq struct {
	DisplayName string `json:"displayName"`
}

func (h *Handler) handleUpdate(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	playerID, err := strconv.ParseUint(c.Param("playerId"), 10, 64)
	if err != nil || playerID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	var req updatePlayerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	player, err := h.svc.Update(c.Request.Context(), hostID, groupID, playerID, req.DisplayName)
	if err == nil {
		c.JSON(http.StatusOK, player)
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrDuplicateAgainstRoster):
		httpx.Reply(c, http.StatusConflict, "Tên trùng nhau",
			"Tên này đã có trong nhóm rồi")
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không cập nhật được", "")
	}
}

func (h *Handler) handleDeactivate(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	playerID, err := strconv.ParseUint(c.Param("playerId"), 10, 64)
	if err != nil || playerID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	err = h.svc.Deactivate(c.Request.Context(), hostID, groupID, playerID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"inactive": true})
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Không lưu được", "")
}

func (h *Handler) handleReactivate(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	playerID, err := strconv.ParseUint(c.Param("playerId"), 10, 64)
	if err != nil || playerID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	err = h.svc.Reactivate(c.Request.Context(), hostID, groupID, playerID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"active": true})
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Không khôi phục được", "")
}

func (h *Handler) handleResetCode(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	playerID, err := strconv.ParseUint(c.Param("playerId"), 10, 64)
	if err != nil || playerID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	player, err := h.svc.ResetCode(c.Request.Context(), hostID, groupID, playerID)
	if err == nil {
		c.JSON(http.StatusOK, player)
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Không reset được code", "")
}

func displayTier(t string) string {
	switch t {
	case "pro":
		return "PRO"
	case "pro_plus":
		return "PRO Plus"
	default:
		return "Free"
	}
}
