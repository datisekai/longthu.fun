package sessions

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

type Handler struct {
	svc         *Service
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
	r.PATCH("/session-charges/:chargeId", h.handlePatchCharge)
	r.POST("/session-charges/batch-patch", h.handleBatchPatchCharge)
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
	var req struct {
		Date     string  `json:"date"`
		Title   *string `json:"title,omitempty"`
		Location *string `json:"location,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	var titleVal, locationVal string
	if req.Title != nil { titleVal = *req.Title }
	if req.Location != nil { locationVal = *req.Location }
	session, err := h.svc.CreateDraft(c.Request.Context(), hostID, CreateDraftParams{
		GroupID: groupID, Date: req.Date, Title: titleVal, Location: locationVal,
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
	httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
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
	var req struct {
		Type             string  `json:"type"`
		Label            string  `json:"label"`
		Amount           int64   `json:"amount"`
		IsIncludedInSplit *bool  `json:"isIncludedInSplit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}
	if req.IsIncludedInSplit == nil {
		req.IsIncludedInSplit = new(bool)
		*req.IsIncludedInSplit = true
	}
	item, err := h.svc.AddCostItem(c.Request.Context(), hostID, AddCostItemParams{
		SessionID: sessionID,
		Type: req.Type,
		Label: req.Label,
		Amount: req.Amount,
		IsIncludedInSplit: *req.IsIncludedInSplit,
	})
	if err == nil {
		c.JSON(http.StatusCreated, item)
		return
	}
	switch {
	case errors.Is(err, ErrSessionNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	case errors.Is(err, ErrInvalidCostItemType):
		httpx.ReplyValidation(c, "Loại chi phí không hợp lệ", []httpx.FieldError{
			{Field: "type", Message: "Loại không hợp lệ"},
		})
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
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
	err = h.svc.RemoveCostItem(c.Request.Context(), hostID, sessionID, itemID)
	if err == nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}
	if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrCostItemNotFound) {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
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

type patchChargeReq struct {
	Action string `json:"action"` // "confirm_paid" | "undo_paid" | "waive"
	Note  string `json:"note"`   // optional note for waive action
}

// PatchChargeResult is the response for charge patch operations.
type PatchChargeResult struct {
	ID       uint64 `json:"id"`
	Status   string `json:"status"`
	PaidAt   string `json:"paidAt,omitempty"`
	PaidVia  string `json:"paidVia,omitempty"`
}

func (h *Handler) handlePatchCharge(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	chargeID, err := strconv.ParseUint(c.Param("chargeId"), 10, 64)
	if err != nil || chargeID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}
	var req patchChargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", "")
		return
	}

	charge, err := h.svc.PatchCharge(c.Request.Context(), hostID, chargeID, req.Action, req.Note)
	if err == nil {
		result := PatchChargeResult{
			ID:     charge.ID,
			Status: charge.Status,
		}
		if charge.PaidAt.Valid {
			result.PaidAt = charge.PaidAt.Time.Format(time.RFC3339)
		}
		if charge.PaidVia.Valid {
			result.PaidVia = charge.PaidVia.String
		}
		c.JSON(http.StatusOK, result)
		return
	}
	switch {
	case errors.Is(err, ErrChargeNotFound):
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
	default:
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
	}
}

type batchPatchChargeReq struct {
	ChargeIDs []uint64 `json:"chargeIds"`
	Action   string    `json:"action"` // "confirm_paid" | "undo_paid" | "waive"
	Note     string    `json:"note"`   // optional note for waive action
}

func (h *Handler) handleBatchPatchCharge(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	var req batchPatchChargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", "")
		return
	}
	if len(req.ChargeIDs) == 0 {
		httpx.Reply(c, http.StatusBadRequest, "Không có charge nào", "")
		return
	}

	results := make([]PatchChargeResult, 0, len(req.ChargeIDs))
	for _, chargeID := range req.ChargeIDs {
		charge, err := h.svc.PatchCharge(c.Request.Context(), hostID, chargeID, req.Action, req.Note)
		if err == nil {
			result := PatchChargeResult{
				ID:     charge.ID,
				Status: charge.Status,
			}
			if charge.PaidAt.Valid {
				result.PaidAt = charge.PaidAt.Time.Format(time.RFC3339)
			}
			if charge.PaidVia.Valid {
				result.PaidVia = charge.PaidVia.String
			}
			results = append(results, result)
		}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
