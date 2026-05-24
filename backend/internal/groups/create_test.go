package groups_test

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
	"github.com/datisekai/longthu.fun/backend/internal/bankaccounts"
	"github.com/datisekai/longthu.fun/backend/internal/groups"
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
		panic("groups handler test: sql.Open: " + err.Error())
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
		t.Skip("DATABASE_URL not set — skipping groups handler integration tests")
	}
}

func uniqueEmail(t *testing.T) string {
	t.Helper()
	return "story18-" + t.Name() + "-" + time.Now().Format("150405.000000000") + "@example.test"
}

// cleanupHostByEmail removes everything attached to the host so tests don't
// leak. FK order: audit_log (SET NULL host_user_id is automatic) → groups →
// bank_accounts → users.
func cleanupHostByEmail(t *testing.T, email string) {
	t.Helper()
	var id int64
	if err := testDB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id); err != nil {
		if !strings.Contains(err.Error(), "no rows") {
			t.Logf("cleanup lookup: %v", err)
		}
		return
	}
	if _, err := testDB.Exec("DELETE FROM `groups` WHERE host_user_id = ?", id); err != nil {
		t.Logf("cleanup groups: %v", err)
	}
	if _, err := testDB.Exec("DELETE FROM bank_accounts WHERE user_id = ?", id); err != nil {
		t.Logf("cleanup bank_accounts: %v", err)
	}
	if _, err := testDB.Exec("DELETE FROM users WHERE id = ?", id); err != nil {
		t.Logf("cleanup users: %v", err)
	}
}

func newTestRouter() *gin.Engine {
	r := gin.New()
	authSvc := auth.NewService(testDB)
	authHandler := auth.NewHandler(authSvc, testSecret, baseURL)
	v1Public := r.Group("/api/v1")
	authHandler.RegisterAuthRoutes(v1Public)

	v1Auth := r.Group("/api/v1")
	v1Auth.Use(auth.SessionMiddleware(testSecret))
	authHandler.RegisterMeRoute(v1Auth)

	groupsSvc := groups.NewService(testDB)
	groupsHandler := groups.NewHandler(groupsSvc)
	groupsHandler.RegisterRoutes(v1Auth)

	bankSvc := bankaccounts.NewService(testDB)
	bankHandler := bankaccounts.NewHandler(bankSvc)
	bankHandler.RegisterRoutes(v1Auth)

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

func sessionCookie(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	return nil
}

// register creates a user and returns the session cookie.
func register(t *testing.T, r *gin.Engine, email string) *http.Cookie {
	t.Helper()
	w := postJSON(r, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Test",
	}, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d; body=%s", email, w.Code, w.Body.String())
	}
	c := sessionCookie(w)
	if c == nil {
		t.Fatalf("register %s: no session cookie", email)
	}
	return c
}

func addBank(t *testing.T, r *gin.Engine, cookie *http.Cookie) {
	t.Helper()
	w := postJSON(r, "/api/v1/bank-accounts", map[string]any{
		"bankCode":          "MBBANK",
		"accountNumber":     "12345678",
		"accountHolderName": "TEST HOST",
	}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("addBank status = %d; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateGroup_SuccessWithDefaultBank(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail(t)
	defer cleanupHostByEmail(t, email)
	r := newTestRouter()

	cookie := register(t, r, email)
	addBank(t, r, cookie)

	w := postJSON(r, "/api/v1/groups", map[string]any{"name": "Tối thứ 3"}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body=%s", w.Code, w.Body.String())
	}

	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["name"] != "Tối thứ 3" {
		t.Errorf("name = %v; want 'Tối thứ 3'", got["name"])
	}
	if got["privacyMode"] != "public" {
		t.Errorf("privacyMode = %v; want public", got["privacyMode"])
	}
	if got["autoDetectEnabled"] != false {
		t.Errorf("autoDetectEnabled = %v; want false", got["autoDetectEnabled"])
	}
	if got["defaultBankAccountId"] == nil {
		t.Errorf("defaultBankAccountId should be populated when host has a default bank; got nil. body=%s", w.Body.String())
	}

	// Audit row exists for this entity.
	var actor string
	groupID := uint64(got["id"].(float64))
	err := testDB.QueryRow(
		"SELECT actor_type FROM audit_log WHERE entity_type = 'group' AND entity_id = ? AND event_type = 'group_created'",
		groupID,
	).Scan(&actor)
	if err != nil {
		t.Fatalf("audit row missing for group %d: %v", groupID, err)
	}
	if actor != "host" {
		t.Errorf("audit actor_type = %q; want 'host'", actor)
	}
}

func TestCreateGroup_NoBank_NullsDefault(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail(t)
	defer cleanupHostByEmail(t, email)
	r := newTestRouter()

	cookie := register(t, r, email)
	// No bank added.

	w := postJSON(r, "/api/v1/groups", map[string]any{"name": "Group không bank"}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body=%s", w.Code, w.Body.String())
	}

	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if v, present := got["defaultBankAccountId"]; present && v != nil {
		t.Errorf("expected defaultBankAccountId to be absent/null; got %v", v)
	}
}

func TestCreateGroup_EmptyName_422(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail(t)
	defer cleanupHostByEmail(t, email)
	r := newTestRouter()
	cookie := register(t, r, email)

	for _, badName := range []string{"", "   ", "\t\n"} {
		w := postJSON(r, "/api/v1/groups", map[string]any{"name": badName}, []*http.Cookie{cookie})
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("name=%q status = %d; want 422; body=%s", badName, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Tên nhóm không được") {
			t.Errorf("name=%q body missing field-error: %s", badName, w.Body.String())
		}
	}
}

func TestCreateGroup_NameTooLong_422(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail(t)
	defer cleanupHostByEmail(t, email)
	r := newTestRouter()
	cookie := register(t, r, email)

	tooLong := strings.Repeat("a", 121)
	w := postJSON(r, "/api/v1/groups", map[string]any{"name": tooLong}, []*http.Cookie{cookie})
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateGroup_SecondGroupSucceeds_NoCap(t *testing.T) {
	skipIfNoDB(t)
	email := uniqueEmail(t)
	defer cleanupHostByEmail(t, email)
	r := newTestRouter()
	cookie := register(t, r, email)

	for i := 1; i <= 3; i++ {
		w := postJSON(r, "/api/v1/groups", map[string]any{"name": "Group " + strings.Repeat("X", i)}, []*http.Cookie{cookie})
		if w.Code != http.StatusCreated {
			t.Fatalf("attempt %d status = %d; want 201; body=%s", i, w.Code, w.Body.String())
		}
	}
}

func TestCreateGroup_TenantIsolation(t *testing.T) {
	skipIfNoDB(t)
	emailA := uniqueEmail(t) + "-A"
	emailB := uniqueEmail(t) + "-B"
	defer cleanupHostByEmail(t, emailA)
	defer cleanupHostByEmail(t, emailB)
	r := newTestRouter()

	cookieA := register(t, r, emailA)
	cookieB := register(t, r, emailB)

	// Host A creates a group.
	w := postJSON(r, "/api/v1/groups", map[string]any{"name": "A's secret"}, []*http.Cookie{cookieA})
	if w.Code != http.StatusCreated {
		t.Fatalf("A create status = %d; body=%s", w.Code, w.Body.String())
	}
	var aGroup map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &aGroup)
	aGroupID := uint64(aGroup["id"].(float64))

	// Host B should NOT see it.
	get := getReq(r, "/api/v1/groups/"+formatUint(aGroupID), []*http.Cookie{cookieB})
	if get.Code != http.StatusNotFound {
		t.Errorf("B reading A's group: status = %d; want 404; body=%s", get.Code, get.Body.String())
	}
}

// formatUint without importing strconv into the test scope (already imported elsewhere).
func formatUint(n uint64) string {
	return strings.TrimSpace(strings.Trim(string(jsonNumber(n)), "\""))
}

func jsonNumber(n uint64) []byte {
	b, _ := json.Marshal(n)
	return b
}
