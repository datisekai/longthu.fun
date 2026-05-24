package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/httpx"
)

const minPasswordLen = 8

type registerReq struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler wires HTTP routes to the auth Service.
type Handler struct {
	svc        *Service
	secret     []byte
	appBaseURL string
}

// NewHandler builds a Handler. secret is the JWT signing key.
func NewHandler(svc *Service, secret []byte, appBaseURL string) *Handler {
	return &Handler{svc: svc, secret: secret, appBaseURL: appBaseURL}
}

// RegisterAuthRoutes mounts the unauthenticated auth endpoints
// (/api/v1/auth/register, /login, /logout) under the supplied group.
// Logout is unauthenticated because clearing a cookie should always work,
// even when the cookie is missing or expired.
func (h *Handler) RegisterAuthRoutes(r *gin.RouterGroup) {
	g := r.Group("/auth")
	{
		g.POST("/register", h.handleRegister)
		g.POST("/login", h.handleLogin)
		g.POST("/logout", h.handleLogout)
	}
}

// RegisterMeRoute mounts /api/v1/auth/me on the authenticated group.
// Must be called on a group that has SessionMiddleware applied.
func (h *Handler) RegisterMeRoute(r *gin.RouterGroup) {
	r.GET("/auth/me", h.handleMe)
}

func (h *Handler) handleRegister(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	var fieldErrs []httpx.FieldError
	if strings.TrimSpace(req.Email) == "" || !strings.Contains(req.Email, "@") {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "email", Message: "Email không hợp lệ"})
	}
	if len(req.Password) < minPasswordLen {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "password", Message: "Mật khẩu cần ít nhất 8 ký tự"})
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		fieldErrs = append(fieldErrs, httpx.FieldError{Field: "displayName", Message: "Cần điền tên hiển thị"})
	}
	if len(fieldErrs) > 0 {
		httpx.ReplyValidation(c, "Thông tin không hợp lệ", fieldErrs)
		return
	}

	user, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, strings.TrimSpace(req.DisplayName))
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			httpx.Reply(c, http.StatusConflict, "Email đã có tài khoản", "Đăng nhập thay vì đăng ký?")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Đăng ký thất bại", "")
		return
	}

	token, err := IssueToken(h.secret, user.ID, user.Tier)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không tạo được phiên đăng nhập", "")
		return
	}
	SetSession(c, token, h.appBaseURL)
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) handleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Reply(c, http.StatusBadRequest, "Yêu cầu sai định dạng", err.Error())
		return
	}

	user, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Reply(c, http.StatusUnauthorized, "Email hoặc mật khẩu sai", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Đăng nhập thất bại", "")
		return
	}

	token, err := IssueToken(h.secret, user.ID, user.Tier)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Không tạo được phiên đăng nhập", "")
		return
	}
	SetSession(c, token, h.appBaseURL)
	c.JSON(http.StatusOK, user)
}

func (h *Handler) handleLogout(c *gin.Context) {
	ClearSession(c, h.appBaseURL)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleMe returns the currently-authenticated user, or 401. Used by the
// frontend's AuthGuard / session bootstrap on page load.
func (h *Handler) handleMe(c *gin.Context) {
	id, ok := HostUserIDFrom(c)
	if !ok {
		httpx.Reply(c, http.StatusUnauthorized, "Chưa đăng nhập", "")
		return
	}
	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			ClearSession(c, h.appBaseURL)
			httpx.Reply(c, http.StatusUnauthorized, "Tài khoản không tồn tại", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Không lấy được thông tin tài khoản", "")
		return
	}
	c.JSON(http.StatusOK, user)
}
