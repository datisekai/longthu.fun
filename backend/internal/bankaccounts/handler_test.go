package bankaccounts_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/datisekai/longthu.fun/backend/internal/config"
	"github.com/datisekai/longthu.fun/backend/internal/server"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		os.Exit(m.Run())
	}
	dsn := strings.TrimPrefix(rawDSN, "mysql://")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("bankaccounts test: sql.Open: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		os.Exit(m.Run())
	}
	testDB = db
	defer testDB.Close()
	os.Exit(m.Run())
}

func TestCreateFirstBankAccountIsDefault(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail()
	defer cleanupUser(t, email)

	cookie := registerHost(t, router, email)
	w := postJSON(router, "/api/v1/bank-accounts", map[string]any{
		"bankCode":          "MBBANK",
		"accountNumber":     "123456789",
		"accountHolderName": "NGUYEN VAN A",
	}, []*http.Cookie{cookie})

	if w.Code != http.StatusCreated {
		t.Fatalf("create bank account status = %d; want 201; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		BankName  string `json:"bankName"`
		BankCode  string `json:"bankCode"`
		IsDefault bool   `json:"isDefault"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if body.BankName != "MBBank" || body.BankCode != "MBBANK" || !body.IsDefault {
		t.Fatalf("unexpected response: %+v", body)
	}

	var isDefault bool
	err := testDB.QueryRow(
		`SELECT is_default FROM bank_accounts ba JOIN users u ON u.id = ba.user_id WHERE u.email = ?`,
		email,
	).Scan(&isDefault)
	if err != nil {
		t.Fatalf("query inserted bank account: %v", err)
	}
	if !isDefault {
		t.Fatal("first bank account should be stored as default")
	}
}

func TestCreateBankAccountValidation(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail()
	defer cleanupUser(t, email)
	cookie := registerHost(t, router, email)

	w := postJSON(router, "/api/v1/bank-accounts", map[string]any{
		"bankCode":          "MBBANK",
		"accountNumber":     "abc",
		"accountHolderName": "NGUYEN VAN A",
	}, []*http.Cookie{cookie})

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Số tài khoản") {
		t.Fatalf("validation body missing account number message: %s", w.Body.String())
	}
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("DATABASE_URL not set or unreachable — skipping bank account integration tests")
	}
}

func newTestServer() http.Handler {
	cfg := &config.Config{
		Port:        "0",
		AppBaseURL:  "http://localhost:5173",
		DatabaseURL: "test",
		JWTSecret:   "bank-test-secret-32-bytes-long!!",
	}
	return server.New(cfg, testDB, "test").Router()
}

func uniqueEmail() string {
	return fmt.Sprintf("bank-%d@example.test", time.Now().UnixNano())
}

func registerHost(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()
	w := postJSON(router, "/api/v1/auth/register", map[string]any{
		"email":       email,
		"password":    "supersecret",
		"displayName": "Bank Test",
	}, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d; want 201; body=%s", w.Code, w.Body.String())
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "lt_session" {
			return cookie
		}
	}
	t.Fatal("register did not set lt_session cookie")
	return nil
}

func postJSON(router http.Handler, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func cleanupUser(t *testing.T, email string) {
	t.Helper()
	var userID uint64
	err := testDB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		if err != sql.ErrNoRows {
			t.Logf("cleanup lookup user %s failed: %v", email, err)
		}
		return
	}
	if _, err := testDB.Exec("DELETE FROM bank_accounts WHERE user_id = ?", userID); err != nil {
		t.Logf("cleanup bank accounts for %s failed: %v", email, err)
	}
	if _, err := testDB.Exec("DELETE FROM users WHERE id = ?", userID); err != nil {
		t.Logf("cleanup user %s failed: %v", email, err)
	}
}
