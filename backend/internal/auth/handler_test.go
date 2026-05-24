package auth_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/datisekai/longthu.fun/backend/internal/auth"
)

var (
	testDB     *sql.DB
	testSecret = []byte("integration-test-secret-32-bytes-long!!")
	baseURL    = "http://localhost:5173"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		os.Exit(m.Run())
	}
	dsn := strings.TrimPrefix(rawDSN, "mysql://")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("auth handler test: sql.Open: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		os.Exit(m.Run())
	}
	testDB = db
	defer testDB.Close()
	os.Exit(m.Run())
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("DATABASE_URL not set or unreachable — skipping auth handler integration tests")
	}
}

// uniqueEmail returns a per-test email so parallel runs don't collide.
func uniqueEmail() string {
	return "test-" + time.Now().Format("20060102150405.000000000") + "@example.test"
}

// cleanupUser removes the test user (cascades to nothing in Story 1.5 scope).
func cleanupUser(t *testing.T, email string) {
	t.Helper()
	if _, err := testDB.Exec("DELETE FROM users WHERE email = ?", email); err != nil {
		t.Logf("cleanup DELETE failed (non-fatal): %v", err)
	}
}

func newTestRouter() *gin.Engine {
	r := gin.New()
	svc := auth.NewService(testDB)
	h := auth.NewHandler(svc, testSecret, baseURL)
	v1 := r.Group("/api/v1")
	h.RegisterAuthRoutes(v1)
	v1Auth := r.Group("/api/v1")
	v1Auth.Use(auth.SessionMiddleware(testSecret))
	h.RegisterMeRoute(v1Auth)
	return r
}

func postJSON(r *gin.Engine, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func getReq(r *gin.Engine, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sessionCookieFrom(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	return nil
}

func TestHandler_Register_Success(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail()
	defer cleanupUser(t, email)
	r := newTestRouter()

	w := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Test Host",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body=%s", w.Code, w.Body.String())
	}
	if sessionCookieFrom(w) == nil {
		t.Fatal("expected lt_session cookie to be set")
	}
	var user map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &user)
	if user["email"] != email {
		t.Errorf("email in response = %v; want %q", user["email"], email)
	}
	if user["tier"] != "free" {
		t.Errorf("tier = %v; want free", user["tier"])
	}
	if _, has := user["passwordHash"]; has {
		t.Error("response leaked passwordHash field")
	}
}

func TestHandler_Register_DuplicateEmailIs409(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail()
	defer cleanupUser(t, email)
	r := newTestRouter()

	first := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "A",
	}, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("first register status = %d", first.Code)
	}

	second := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret2", "displayName": "B",
	}, nil)
	if second.Code != http.StatusConflict {
		t.Fatalf("second register status = %d; want 409", second.Code)
	}
	if !strings.Contains(second.Body.String(), "Email đã có tài khoản") {
		t.Errorf("conflict body missing expected message: %s", second.Body.String())
	}
}

func TestHandler_Register_ShortPasswordIs422(t *testing.T) {
	skipIfNoDB(t)
	r := newTestRouter()
	w := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": "valid@example.test", "password": "short", "displayName": "X",
	}, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ít nhất 8 ký tự") {
		t.Errorf("422 body missing password error: %s", w.Body.String())
	}
}

func TestHandler_Login_Success(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail()
	defer cleanupUser(t, email)
	r := newTestRouter()

	_ = postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Test",
	}, nil)

	w := postJSON(r, "/api/v1/auth/login", map[string]any{
		"email": email, "password": "supersecret",
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if sessionCookieFrom(w) == nil {
		t.Fatal("login should set lt_session cookie")
	}
}

func TestHandler_Login_WrongPasswordIs401(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail()
	defer cleanupUser(t, email)
	r := newTestRouter()

	_ = postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Test",
	}, nil)

	w := postJSON(r, "/api/v1/auth/login", map[string]any{
		"email": email, "password": "WRONG-password",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Email hoặc mật khẩu sai") {
		t.Errorf("401 body missing expected single-error message: %s", w.Body.String())
	}
}

func TestHandler_Login_NonexistentEmailIs401(t *testing.T) {
	skipIfNoDB(t)
	r := newTestRouter()
	w := postJSON(r, "/api/v1/auth/login", map[string]any{
		"email": "definitely-doesnt-exist@example.test", "password": "anything",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
	// Important: SAME message as wrong-password (no enumeration).
	if !strings.Contains(w.Body.String(), "Email hoặc mật khẩu sai") {
		t.Errorf("body should match wrong-password message (no enumeration): %s", w.Body.String())
	}
}

func TestHandler_MeAndLogout(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail()
	defer cleanupUser(t, email)
	r := newTestRouter()

	// Register, capture cookie.
	reg := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Me",
	}, nil)
	cookie := sessionCookieFrom(reg)
	if cookie == nil {
		t.Fatal("no session cookie after register")
	}

	// /me with cookie → 200.
	me := getReq(r, "/api/v1/auth/me", []*http.Cookie{cookie})
	if me.Code != http.StatusOK {
		t.Fatalf("/me status = %d; want 200", me.Code)
	}
	var meBody map[string]any
	_ = json.Unmarshal(me.Body.Bytes(), &meBody)
	if meBody["email"] != email {
		t.Errorf("/me email = %v; want %q", meBody["email"], email)
	}

	// /me without cookie → 401.
	noCookie := getReq(r, "/api/v1/auth/me", nil)
	if noCookie.Code != http.StatusUnauthorized {
		t.Fatalf("/me no-cookie status = %d; want 401", noCookie.Code)
	}

	// Logout clears the cookie.
	logout := postJSON(r, "/api/v1/auth/logout", map[string]any{}, []*http.Cookie{cookie})
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d; want 200", logout.Code)
	}
	cleared := sessionCookieFrom(logout)
	if cleared == nil || cleared.MaxAge != -1 {
		t.Errorf("logout did not properly clear cookie; got %+v", cleared)
	}
}

func TestMiddleware_TamperedTokenIs401(t *testing.T) {
	skipIfNoDB(t)
	r := newTestRouter()
	bad := &http.Cookie{Name: auth.CookieName, Value: "not.a.valid.jwt", Path: "/"}
	w := getReq(r, "/api/v1/auth/me", []*http.Cookie{bad})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tampered cookie status = %d; want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Phiên") {
		t.Errorf("401 body should mention session: %s", w.Body.String())
	}
}
