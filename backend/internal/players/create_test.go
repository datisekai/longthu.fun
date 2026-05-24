package players_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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
		panic("players test: sql.Open: " + err.Error())
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
		t.Skip("DATABASE_URL not set or unreachable — skipping players integration tests")
	}
}

func newTestServer() http.Handler {
	cfg := &config.Config{
		Port:        "0",
		AppBaseURL:  "http://localhost:5173",
		DatabaseURL: "test",
		JWTSecret:   "players-test-secret-32-bytes-long!!",
	}
	return server.New(cfg, testDB, "test").Router()
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.test", prefix, time.Now().UnixNano())
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

func registerHost(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()
	w := postJSON(router, "/api/v1/auth/register", map[string]any{
		"email":       email,
		"password":    "supersecret",
		"displayName": "Players Test",
	}, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d; body=%s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "lt_session" {
			return c
		}
	}
	t.Fatal("register: no lt_session cookie")
	return nil
}

func createGroup(t *testing.T, router http.Handler, cookie *http.Cookie, name string) uint64 {
	t.Helper()
	w := postJSON(router, "/api/v1/groups", map[string]any{"name": name}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("createGroup status = %d; body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	return uint64(got["id"].(float64))
}

func cleanupHostByEmail(t *testing.T, email string) {
	t.Helper()
	var id uint64
	if err := testDB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id); err != nil {
		if err != sql.ErrNoRows {
			t.Logf("cleanup lookup: %v", err)
		}
		return
	}
	if _, err := testDB.Exec(
		"DELETE FROM players WHERE group_id IN (SELECT id FROM `groups` WHERE host_user_id = ?)",
		id,
	); err != nil {
		t.Logf("cleanup players: %v", err)
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

// AC1 — happy path: bulk insert with diacritics preserved, unique codes, all is_active.
func TestBulkCreate_HappyPath(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-happy")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Tối thứ 3")

	names := []string{"Đạt", "Lý", "Tâm Đạt", "Bảo Châu"}
	w := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": names},
		[]*http.Cookie{cookie},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body=%s", w.Code, w.Body.String())
	}

	var got struct {
		Players []struct {
			ID          uint64 `json:"id"`
			GroupID     uint64 `json:"groupId"`
			DisplayName string `json:"displayName"`
			PublicCode  string `json:"publicCode"`
			IsActive    bool   `json:"isActive"`
		} `json:"players"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, w.Body.String())
	}
	if len(got.Players) != len(names) {
		t.Fatalf("len(players) = %d; want %d", len(got.Players), len(names))
	}

	seenCodes := map[string]struct{}{}
	for i, p := range got.Players {
		if p.DisplayName != names[i] {
			t.Errorf("player[%d].displayName = %q; want %q (diacritics dropped?)", i, p.DisplayName, names[i])
		}
		if !p.IsActive {
			t.Errorf("player[%d].isActive = false; want true", i)
		}
		if p.GroupID != groupID {
			t.Errorf("player[%d].groupId = %d; want %d", i, p.GroupID, groupID)
		}
		if len(p.PublicCode) != 8 {
			t.Errorf("player[%d].publicCode = %q; want 8 chars", i, p.PublicCode)
		}
		if _, dup := seenCodes[p.PublicCode]; dup {
			t.Errorf("player[%d].publicCode = %q is duplicated within submit", i, p.PublicCode)
		}
		seenCodes[p.PublicCode] = struct{}{}
	}
}

// AC2 — tier cap: Free tier (6) + adding 7 names → 422.
func TestBulkCreate_TierCapExceeded(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-cap")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Cap test")

	// 7 names > free tier cap (6) on an empty group.
	names := []string{"A", "B", "C", "D", "E", "F", "G"}
	w := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": names},
		[]*http.Cookie{cookie},
	)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "tối đa 6") {
		t.Errorf("body missing free tier cap detail (tối đa 6): %s", body)
	}

	// All-or-nothing: zero players inserted.
	var count int
	_ = testDB.QueryRow("SELECT COUNT(*) FROM players WHERE group_id = ?", groupID).Scan(&count)
	if count != 0 {
		t.Errorf("partial insert on cap reject: %d players in group; want 0", count)
	}
}

// AC3 — per-submit duplicate (case-insensitive) → 422 naming the duplicate.
func TestBulkCreate_DuplicateInSubmit(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-dupsubmit")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Dup submit")

	w := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": []string{"Đạt", "Lý", "đạt"}},
		[]*http.Cookie{cookie},
	)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "trùng nhau") {
		t.Errorf("body missing 'trùng nhau' message: %s", w.Body.String())
	}
}

// AC4 — duplicate against existing roster (case-insensitive) → 409.
func TestBulkCreate_DuplicateAgainstRoster(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-duproster")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Dup roster")

	// Seed roster.
	w1 := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": []string{"Đạt", "Lý"}},
		[]*http.Cookie{cookie},
	)
	if w1.Code != http.StatusCreated {
		t.Fatalf("seed status = %d; body=%s", w1.Code, w1.Body.String())
	}

	// Re-add "đạt" (case-flip) → conflict.
	w2 := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": []string{"đạt"}},
		[]*http.Cookie{cookie},
	)
	if w2.Code != http.StatusConflict {
		t.Fatalf("status = %d; want 409; body=%s", w2.Code, w2.Body.String())
	}
}

// AC5 — tenant isolation: Host B POSTing against Host A's groupId → 404.
func TestBulkCreate_TenantIsolation(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	emailA := uniqueEmail("p19-tenantA")
	emailB := uniqueEmail("p19-tenantB")
	defer cleanupHostByEmail(t, emailA)
	defer cleanupHostByEmail(t, emailB)

	cookieA := registerHost(t, router, emailA)
	cookieB := registerHost(t, router, emailB)

	groupA := createGroup(t, router, cookieA, "A's group")

	// B attempts to bulk-add into A's group.
	w := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupA, 10)+"/players",
		map[string]any{"names": []string{"intruder"}},
		[]*http.Cookie{cookieB},
	)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404; body=%s", w.Code, w.Body.String())
	}
}

// AC6 — audit row `player_added` written per inserted Player in the same tx.
func TestBulkCreate_AuditPerPlayer(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-audit")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Audit test")

	w := postJSON(router,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": []string{"X", "Y", "Z"}},
		[]*http.Cookie{cookie},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}

	var auditCount int
	err := testDB.QueryRow(`
		SELECT COUNT(*) FROM audit_log
		WHERE event_type = 'player_added'
		  AND entity_type = 'player'
		  AND entity_id IN (SELECT id FROM players WHERE group_id = ?)
	`, groupID).Scan(&auditCount)
	if err != nil {
		t.Fatalf("audit count query: %v", err)
	}
	if auditCount != 3 {
		t.Errorf("audit player_added rows = %d; want 3", auditCount)
	}
}

// Validation: empty names list → 422.
func TestBulkCreate_EmptyNames(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("p19-empty")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Empty test")

	for _, payload := range []map[string]any{
		{"names": []string{}},
		{"names": []string{"", "  ", "\t"}},
	} {
		w := postJSON(router,
			"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
			payload, []*http.Cookie{cookie},
		)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("payload=%v status = %d; want 422; body=%s", payload, w.Code, w.Body.String())
		}
	}
}
