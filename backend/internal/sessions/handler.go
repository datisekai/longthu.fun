package sessions

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

type Handler struct {
	svc        *Service
	appBaseURL string
}

func NewHandler(svc *Service, appBaseURL string) *Handler {
	return &Handler{svc: svc, appBaseURL: strings.TrimRight(appBaseURL, "/")}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/groups/:groupId/sessions", h.handleCreateDraft)
	r.GET("/sessions/:sessionId", h.handleGetDraft)
	r.POST("/sessions/:sessionId/cost-items", h.handleAddCostItem)
	r.DELETE("/sessions/:sessionId/cost-items/:itemId", h.handleRemoveCostItem)
	r.PUT("/sessions/:sessionId/participants", h.handleSetParticipants)
	r.POST("/sessions/:sessionId/finalize", h.handleFinalize)
}

type createDraftReq struct {
	Date     string `json:"date"`
	Title    string `json:"title"`
	Location string `json:"location"`
}

func (h *Handler) handleCreateDraft(c *gin.Context) {
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
	var req createDraftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	session, err := h.svc.CreateDraft(c.Request.Context(), hostID, CreateDraftParams{
		GroupID: groupID, Date: req.Date, Title: req.Title, Location: req.Location,
	})
	if err == nil {
		c.JSON(http.StatusCreated, session)
		return
	}
	switch {
	case errors.Is(err, ErrGroupNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrInvalidDate):
		httpx.ReplyValidation(c, "Thông tin không hợp lệ", []httpx.FieldError{
			{Field: "date", Message: "Ngày không hợp lệ (YYYY-MM-DD)"},
		})
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không tạo được buổi đánh", "")
	}
}

func (h *Handler) handleGetDraft(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 64)
	if err != nil || sessionID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	snap, err := h.svc.GetDraftWithDetails(c.Request.Context(), hostID, sessionID)
	if err == nil {
		c.JSON(http.StatusOK, snap)
		return
	}
	if errors.Is(err, ErrSessionNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Không đọc được buổi đánh", "")
}

type addCostItemReq struct {
	Type              string `json:"type"`
	Label             string `json:"label"`
	Amount            int64  `json:"amount"`
	IsIncludedInSplit *bool  `json:"isIncludedInSplit"`
}

func (h *Handler) handleAddCostItem(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 64)
	if err != nil || sessionID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	var req addCostItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}
	included := true
	if req.IsIncludedInSplit != nil {
		included = *req.IsIncludedInSplit
	}

	item, err := h.svc.AddCostItem(c.Request.Context(), hostID, AddCostItemParams{
		SessionID:         sessionID,
		Type:              req.Type,
		Label:             req.Label,
		Amount:            req.Amount,
		IsIncludedInSplit: included,
	})
	if err == nil {
		c.JSON(http.StatusCreated, item)
		return
	}
	switch {
	case errors.Is(err, ErrSessionNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrInvalidCostItemType):
		httpx.ReplyValidation(c, "Loại không hợp lệ", []httpx.FieldError{
			{Field: "type", Message: "Loại phải là court / shuttle / water / other / discount"},
		})
	case errors.Is(err, ErrInvalidCostItemAmount):
		httpx.ReplyValidation(c, "Số tiền không hợp lệ", []httpx.FieldError{
			{Field: "amount", Message: "Số tiền không hợp lệ"},
		})
	case errors.Is(err, ErrInvalidCostItemLabel):
		httpx.ReplyValidation(c, "Mô tả không hợp lệ", []httpx.FieldError{
			{Field: "label", Message: "Mô tả cần 1-80 ký tự"},
		})
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không thêm được khoản chi", "")
	}
}

func (h *Handler) handleRemoveCostItem(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 64)
	if err != nil || sessionID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 64)
	if err != nil || itemID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	if err := h.svc.RemoveCostItem(c.Request.Context(), hostID, sessionID, itemID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			httpx.Reply(c, http.StatusNotFound, "Not found", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Không xóa được khoản chi", "")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleFinalize(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 64)
	if err != nil || sessionID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	result, err := h.svc.Finalize(c.Request.Context(), hostID, sessionID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"session":   result.Session,
			"charges":   result.Charges,
			"shareCode": result.ShareCode,
			"shareUrl":  h.appBaseURL + "/g/" + result.ShareCode,
		})
		return
	}
	switch {
	case errors.Is(err, ErrSessionNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrNoBankAccount):
		httpx.ReplyValidation(c, "Chưa có tài khoản nhận tiền", []httpx.FieldError{
			{Field: "bankAccount", Message: "Bạn cần thêm tài khoản nhận tiền trước khi chốt buổi."},
		})
	case errors.Is(err, ErrNoCostItems):
		httpx.ReplyValidation(c, "Chưa đủ thông tin", []httpx.FieldError{
			{Field: "costItems", Message: "Cần ít nhất 1 khoản chi để chốt buổi"},
		})
	case errors.Is(err, ErrParticipantsEmpty):
		httpx.ReplyValidation(c, "Chưa có người chơi", []httpx.FieldError{
			{Field: "participants", Message: "Cần ít nhất 1 người chơi để chốt buổi"},
		})
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không chốt được buổi", "")
	}
}

type setParticipantsReq struct {
	PlayerIDs []uint64 `json:"playerIds"`
}

func (h *Handler) handleSetParticipants(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	sessionID, err := strconv.ParseUint(c.Param("sessionId"), 10, 64)
	if err != nil || sessionID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	var req setParticipantsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}
	parts, err := h.svc.SetParticipants(c.Request.Context(), hostID, sessionID, req.PlayerIDs)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"participants": parts})
		return
	}
	switch {
	case errors.Is(err, ErrSessionNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrParticipantsEmpty):
		httpx.ReplyValidation(c, "Cần ít nhất 1 người chơi", []httpx.FieldError{
			{Field: "playerIds", Message: "Cần ít nhất 1 người chơi"},
		})
	case errors.Is(err, ErrParticipantOutsideRoster):
		httpx.ReplyValidation(c, "Có người chơi không thuộc group", []httpx.FieldError{
			{Field: "playerIds", Message: "Có ID không thuộc group hiện tại"},
		})
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Không cập nhật được người chơi", "")
	}
}
