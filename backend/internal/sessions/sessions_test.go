package sessions_test

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
		panic("sessions test: sql.Open: " + err.Error())
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
		t.Skip("DATABASE_URL not set or unreachable — skipping sessions integration tests")
	}
}

func newTestServer() http.Handler {
	cfg := &config.Config{
		Port:        "0",
		AppBaseURL:  "http://localhost:5173",
		DatabaseURL: "test",
		JWTSecret:   "sessions-test-secret-32-bytes-long!!",
	}
	return server.New(cfg, testDB, "test").Router()
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.test", prefix, time.Now().UnixNano())
}

func doJSON(router http.Handler, method, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func registerHost(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()
	w := doJSON(router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": email, "password": "supersecret", "displayName": "Sess Test",
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
	w := doJSON(router, http.MethodPost, "/api/v1/groups", map[string]any{"name": name}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("createGroup status = %d; body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	return uint64(got["id"].(float64))
}

func addPlayers(t *testing.T, router http.Handler, cookie *http.Cookie, groupID uint64, names []string) []uint64 {
	t.Helper()
	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players",
		map[string]any{"names": names}, []*http.Cookie{cookie},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("addPlayers status = %d; body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Players []struct {
			ID uint64 `json:"id"`
		} `json:"players"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	ids := make([]uint64, len(got.Players))
	for i, p := range got.Players {
		ids[i] = p.ID
	}
	return ids
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
	// FK order: session_charges → session_participants → session_cost_items →
	// sessions → players → groups → bank_accounts → users.
	_, _ = testDB.Exec(`
		DELETE FROM session_participants
		WHERE session_id IN (
			SELECT id FROM sessions
			WHERE group_id IN (SELECT id FROM `+"`groups`"+` WHERE host_user_id = ?)
		)`, id)
	_, _ = testDB.Exec(`
		DELETE FROM session_cost_items
		WHERE session_id IN (
			SELECT id FROM sessions
			WHERE group_id IN (SELECT id FROM `+"`groups`"+` WHERE host_user_id = ?)
		)`, id)
	_, _ = testDB.Exec(`
		DELETE FROM sessions
		WHERE group_id IN (SELECT id FROM `+"`groups`"+` WHERE host_user_id = ?)`, id)
	_, _ = testDB.Exec(
		"DELETE FROM players WHERE group_id IN (SELECT id FROM `groups` WHERE host_user_id = ?)", id,
	)
	_, _ = testDB.Exec("DELETE FROM `groups` WHERE host_user_id = ?", id)
	_, _ = testDB.Exec("DELETE FROM bank_accounts WHERE user_id = ?", id)
	_, _ = testDB.Exec("DELETE FROM users WHERE id = ?", id)
}

// --- AC1: create draft ---
func TestCreateDraft_HappyPath(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-create")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Group draft test")

	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24", "title": "Tối thứ 3", "location": "Sân K34"},
		[]*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["date"] != "2026-05-24" {
		t.Errorf("date = %v; want 2026-05-24", got["date"])
	}
	if got["status"] != "draft" {
		t.Errorf("status = %v; want draft", got["status"])
	}
	if got["groupId"] == nil {
		t.Errorf("missing groupId; body=%s", w.Body.String())
	}
}

// AC7: invalid date → 422
func TestCreateDraft_InvalidDate(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-baddate")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Bad date")

	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "not-a-date"}, []*http.Cookie{cookie})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
}

// AC6: tenant isolation on session create
func TestCreateDraft_TenantIsolation(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	emailA := uniqueEmail("sess-tenantA")
	emailB := uniqueEmail("sess-tenantB")
	defer cleanupHostByEmail(t, emailA)
	defer cleanupHostByEmail(t, emailB)
	cookieA := registerHost(t, router, emailA)
	cookieB := registerHost(t, router, emailB)
	groupA := createGroup(t, router, cookieA, "A group")

	// B trying to create a session under A's group → 404.
	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupA, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookieB})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404; body=%s", w.Code, w.Body.String())
	}
}

// AC2 + AC7: add cost items + validate amount
func TestCostItems_AddAndValidate(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-costitem")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Cost item test")

	// Create draft.
	w1 := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookie})
	if w1.Code != http.StatusCreated {
		t.Fatalf("create draft: %d / %s", w1.Code, w1.Body.String())
	}
	var draft map[string]any
	_ = json.Unmarshal(w1.Body.Bytes(), &draft)
	sessionID := uint64(draft["id"].(float64))
	base := "/api/v1/sessions/" + strconv.FormatUint(sessionID, 10) + "/cost-items"

	// Valid court item.
	w2 := doJSON(router, http.MethodPost, base,
		map[string]any{"type": "court", "label": "Sân 360k", "amount": 360000}, []*http.Cookie{cookie})
	if w2.Code != http.StatusCreated {
		t.Fatalf("add court: %d / %s", w2.Code, w2.Body.String())
	}

	// Zero amount → 422.
	w3 := doJSON(router, http.MethodPost, base,
		map[string]any{"type": "water", "label": "Nước", "amount": 0}, []*http.Cookie{cookie})
	if w3.Code != http.StatusUnprocessableEntity {
		t.Fatalf("zero amount: %d / %s", w3.Code, w3.Body.String())
	}
	if !strings.Contains(w3.Body.String(), "Số tiền không hợp lệ") {
		t.Errorf("expected 'Số tiền không hợp lệ' message; got %s", w3.Body.String())
	}

	// Invalid type → 422.
	w4 := doJSON(router, http.MethodPost, base,
		map[string]any{"type": "tip", "label": "Tip", "amount": 10000}, []*http.Cookie{cookie})
	if w4.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid type: %d / %s", w4.Code, w4.Body.String())
	}

	// Discount (negative) → accepted.
	w5 := doJSON(router, http.MethodPost, base,
		map[string]any{"type": "discount", "label": "Voucher", "amount": -50000}, []*http.Cookie{cookie})
	if w5.Code != http.StatusCreated {
		t.Fatalf("discount: %d / %s", w5.Code, w5.Body.String())
	}
}

// AC3: delete cost item
func TestCostItems_Delete(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-delci")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Delete CI test")
	w1 := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookie})
	var draft map[string]any
	_ = json.Unmarshal(w1.Body.Bytes(), &draft)
	sessionID := uint64(draft["id"].(float64))

	w2 := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/cost-items",
		map[string]any{"type": "shuttle", "label": "Cầu", "amount": 200000}, []*http.Cookie{cookie})
	var item map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &item)
	itemID := uint64(item["id"].(float64))

	w3 := doJSON(router, http.MethodDelete,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/cost-items/"+strconv.FormatUint(itemID, 10),
		nil, []*http.Cookie{cookie})
	if w3.Code != http.StatusNoContent {
		t.Fatalf("delete: %d / %s", w3.Code, w3.Body.String())
	}
}

// AC4 + AC6: set participants + roster validation
func TestParticipants_SetAndValidate(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	emailA := uniqueEmail("sess-partA")
	emailB := uniqueEmail("sess-partB")
	defer cleanupHostByEmail(t, emailA)
	defer cleanupHostByEmail(t, emailB)
	cookieA := registerHost(t, router, emailA)
	cookieB := registerHost(t, router, emailB)

	groupA := createGroup(t, router, cookieA, "Part A")
	groupB := createGroup(t, router, cookieB, "Part B")
	playersA := addPlayers(t, router, cookieA, groupA, []string{"X", "Y", "Z"})
	playersB := addPlayers(t, router, cookieB, groupB, []string{"M", "N"})

	// Create A's draft.
	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupA, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookieA})
	var draft map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &draft)
	sessionID := uint64(draft["id"].(float64))
	endpoint := "/api/v1/sessions/" + strconv.FormatUint(sessionID, 10) + "/participants"

	// Happy path: all A's players.
	w2 := doJSON(router, http.MethodPut, endpoint,
		map[string]any{"playerIds": playersA}, []*http.Cookie{cookieA})
	if w2.Code != http.StatusOK {
		t.Fatalf("set participants: %d / %s", w2.Code, w2.Body.String())
	}

	// Empty list → 422.
	w3 := doJSON(router, http.MethodPut, endpoint,
		map[string]any{"playerIds": []uint64{}}, []*http.Cookie{cookieA})
	if w3.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty: %d / %s", w3.Code, w3.Body.String())
	}

	// Cross-group player (one of B's players in A's session) → 422.
	mixed := append([]uint64{}, playersA[0], playersB[0])
	w4 := doJSON(router, http.MethodPut, endpoint,
		map[string]any{"playerIds": mixed}, []*http.Cookie{cookieA})
	if w4.Code != http.StatusUnprocessableEntity {
		t.Fatalf("cross-group: %d / %s", w4.Code, w4.Body.String())
	}
}

// AC5: list active players
func TestListActivePlayers(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-listp")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "List active")
	addPlayers(t, router, cookie, groupID, []string{"Đạt", "Lý", "Tâm"})

	w := doJSON(router, http.MethodGet,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/players", nil, []*http.Cookie{cookie})
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d / %s", w.Code, w.Body.String())
	}
	var got struct {
		Players []map[string]any `json:"players"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got.Players) != 3 {
		t.Errorf("len(players) = %d; want 3", len(got.Players))
	}
}

// GET draft snapshot returns the session + items + participants.
func TestGetDraftSnapshot(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-snap")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	groupID := createGroup(t, router, cookie, "Snapshot")
	playerIDs := addPlayers(t, router, cookie, groupID, []string{"A", "B"})

	w1 := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookie})
	var draft map[string]any
	_ = json.Unmarshal(w1.Body.Bytes(), &draft)
	sessionID := uint64(draft["id"].(float64))

	_ = doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/cost-items",
		map[string]any{"type": "court", "label": "Sân", "amount": 360000}, []*http.Cookie{cookie})
	_ = doJSON(router, http.MethodPut,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/participants",
		map[string]any{"playerIds": playerIDs}, []*http.Cookie{cookie})

	w := doJSON(router, http.MethodGet,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10), nil, []*http.Cookie{cookie})
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d / %s", w.Code, w.Body.String())
	}
	var snap struct {
		Session      map[string]any   `json:"session"`
		CostItems    []map[string]any `json:"costItems"`
		Participants []map[string]any `json:"participants"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snap.CostItems) != 1 || len(snap.Participants) != 2 {
		t.Errorf("snapshot mismatch: items=%d, parts=%d", len(snap.CostItems), len(snap.Participants))
	}
}
