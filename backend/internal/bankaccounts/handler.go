package bankaccounts

import (
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
	r.POST("/bank-accounts", h.handleCreate)
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
