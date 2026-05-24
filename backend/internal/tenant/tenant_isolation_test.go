package tenant_test

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
		panic("tenant isolation test: sql.Open: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		os.Exit(m.Run())
	}
	testDB = db
	defer testDB.Close()
	os.Exit(m.Run())
}

func TestGroupTenantIsolation(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()

	hostAEmail := uniqueEmail("host-a")
	hostBEmail := uniqueEmail("host-b")
	defer cleanupUsers(t, hostAEmail, hostBEmail)

	hostACookie := registerHost(t, router, hostAEmail)
	hostBCookie := registerHost(t, router, hostBEmail)
	hostAID := userIDByEmail(t, hostAEmail)

	groupID := insertGroup(t, hostAID, "Tối thứ 3")

	own := getGroup(t, router, groupID, hostACookie)
	if own.Code != http.StatusOK {
		t.Fatalf("Host A reading own group status = %d; want 200; body=%s", own.Code, own.Body.String())
	}

	crossTenant := getGroup(t, router, groupID, hostBCookie)
	if crossTenant.Code != http.StatusNotFound {
		t.Fatalf("Host B reading Host A group status = %d; want 404; body=%s", crossTenant.Code, crossTenant.Body.String())
	}
	assertProblemTitle(t, crossTenant, "Not found")

	missing := getGroup(t, router, groupID+999_999, hostBCookie)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing group status = %d; want 404; body=%s", missing.Code, missing.Body.String())
	}
	assertProblemTitle(t, missing, "Not found")
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("DATABASE_URL not set or unreachable — skipping tenant isolation integration tests")
	}
}

func newTestServer() http.Handler {
	cfg := &config.Config{
		Port:        "0",
		AppBaseURL:  "http://localhost:5173",
		DatabaseURL: "test",
		JWTSecret:   "tenant-test-secret-32-bytes-long!!",
	}
	return server.New(cfg, testDB, "test").Router()
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.test", prefix, time.Now().UnixNano())
}

func registerHost(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()
	body := map[string]any{
		"email":       email,
		"password":    "supersecret",
		"displayName": "Tenant Test",
	}
	w := postJSON(router, "/api/v1/auth/register", body, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d; want 201; body=%s", email, w.Code, w.Body.String())
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "lt_session" {
			return cookie
		}
	}
	t.Fatalf("register %s did not set lt_session cookie", email)
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

func getGroup(t *testing.T, router http.Handler, groupID uint64, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/groups/%d", groupID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func userIDByEmail(t *testing.T, email string) uint64 {
	t.Helper()
	var id uint64
	err := testDB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id)
	if err != nil {
		t.Fatalf("lookup user %s: %v", email, err)
	}
	return id
}

func insertGroup(t *testing.T, hostID uint64, name string) uint64 {
	t.Helper()
	res, err := testDB.Exec("INSERT INTO `groups` (host_user_id, name) VALUES (?, ?)", hostID, name)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("group LastInsertId: %v", err)
	}
	return uint64(id)
}

func assertProblemTitle(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	var problem struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem JSON: %v; body=%s", err, w.Body.String())
	}
	if problem.Title != want {
		t.Fatalf("problem title = %q; want %q; body=%s", problem.Title, want, w.Body.String())
	}
}

func cleanupUsers(t *testing.T, emails ...string) {
	t.Helper()
	for _, email := range emails {
		var userID uint64
		err := testDB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
		if err != nil {
			if err != sql.ErrNoRows {
				t.Logf("cleanup lookup user %s failed: %v", email, err)
			}
			continue
		}
		if _, err := testDB.Exec("DELETE FROM `groups` WHERE host_user_id = ?", userID); err != nil {
			t.Logf("cleanup groups for %s failed: %v", email, err)
		}
		if _, err := testDB.Exec("DELETE FROM users WHERE id = ?", userID); err != nil {
			t.Logf("cleanup user %s failed: %v", email, err)
		}
	}
}
