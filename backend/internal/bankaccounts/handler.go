package bankaccounts

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
	"github.com/datisekai/longthu.fun/backend/internal/tenant"
)

var numericAccount = regexp.MustCompile(`^[0-9]{8,16}$`)

type Handler struct {
	svc *Service
}

type createReq struct {
	BankCode          string `json:"bankCode"`
	AccountNumber     string `json:"accountNumber"`
	AccountHolderName string `json:"accountHolderName"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/bank-accounts", h.handleList)
	r.POST("/bank-accounts", h.handleCreate)
	r.PATCH("/bank-accounts/:bankId/default", h.handleSetDefault)
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

	var fieldErrs []httpx.FieldError
	if bankNameForCode(req.BankCode) == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "bankCode", Message: "Chọn ngân hàng hợp lệ"})
	}
	if !numericAccount.MatchString(strings.TrimSpace(req.AccountNumber)) {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "accountNumber", Message: "Số tài khoản chỉ gồm 8-16 chữ số"})
	}
	if strings.TrimSpace(req.AccountHolderName) == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "accountHolderName", Message: "Nhập tên chủ tài khoản"})
	}
	if len(fieldErrs) > 0 {
		httpx.ReplyValidation(c, "Thông tin tài khoản ngân hàng chưa hợp lệ", fieldErrs)
		return
	}

	account, err := h.svc.Create(c.Request.Context(), hostID, CreateParams{
		BankCode:          strings.ToUpper(strings.TrimSpace(req.BankCode)),
		AccountNumber:     req.AccountNumber,
		AccountHolderName: req.AccountHolderName,
	})
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không lưu được tài khoản ngân hàng", "")
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *Handler) handleList(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	accounts, err := h.svc.List(c.Request.Context(), hostID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không lấy được danh sách", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"bankAccounts": accounts})
}

func (h *Handler) handleSetDefault(c *gin.Context) {
	hostID, ok := tenant.HostID(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}

	bankIDStr := c.Param("bankId")
	var bankID uint64
	if _, err := fmt.Sscanf(bankIDStr, "%d", &bankID); err != nil || bankID == 0 {
		httpx.Reply(c, http.StatusNotFound, "Not found", "")
		return
	}

	err := h.svc.SetDefault(c.Request.Context(), hostID, bankID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không cập nhật được", "")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
