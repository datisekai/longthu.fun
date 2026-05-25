package sessions_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// helper: add a bank account so finalize's bank gate is satisfied.
func addBank(t *testing.T, router http.Handler, cookie *http.Cookie) {
	t.Helper()
	w := doJSON(router, http.MethodPost, "/api/v1/bank-accounts", map[string]any{
		"bankCode": "MBBANK", "accountNumber": "12345678", "accountHolderName": "TEST HOST",
	}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("addBank: %d / %s", w.Code, w.Body.String())
	}
}

// Build a draft with N players, K cost items totalling `total`. Returns sessionID.
func seedDraft(t *testing.T, router http.Handler, cookie *http.Cookie, groupID uint64, names []string, costs []map[string]any) (sessionID uint64, playerIDs []uint64) {
	t.Helper()
	playerIDs = addPlayers(t, router, cookie, groupID, names)
	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookie})
	if w.Code != http.StatusCreated {
		t.Fatalf("create draft: %d / %s", w.Code, w.Body.String())
	}
	var draft map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &draft)
	sessionID = uint64(draft["id"].(float64))

	for _, c := range costs {
		w := doJSON(router, http.MethodPost,
			"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/cost-items",
			c, []*http.Cookie{cookie})
		if w.Code != http.StatusCreated {
			t.Fatalf("add cost item %v: %d / %s", c, w.Code, w.Body.String())
		}
	}
	w2 := doJSON(router, http.MethodPut,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/participants",
		map[string]any{"playerIds": playerIDs}, []*http.Cookie{cookie})
	if w2.Code != http.StatusOK {
		t.Fatalf("set participants: %d / %s", w2.Code, w2.Body.String())
	}
	return sessionID, playerIDs
}

// AC1: happy path — finalize generates charges, mints share_code, writes audit.
func TestFinalize_HappyPath(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-fin-happy")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	addBank(t, router, cookie)
	groupID := createGroup(t, router, cookie, "Finalize happy")
	// Free tier cap is 6 players; 600001 / 6 = 100000.166... so the
	// distributeSplit residual path gets exercised (first player gets +1).
	sessionID, playerIDs := seedDraft(t, router, cookie, groupID,
		[]string{"A", "B", "C", "D", "E", "F"},
		[]map[string]any{
			{"type": "court", "label": "Sân", "amount": 360000},
			{"type": "shuttle", "label": "Cầu", "amount": 200000},
			{"type": "water", "label": "Nước", "amount": 40001},
		})

	w := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookie})
	if w.Code != http.StatusOK {
		t.Fatalf("finalize: %d / %s", w.Code, w.Body.String())
	}
	var got struct {
		Session   map[string]any   `json:"session"`
		Charges   []map[string]any `json:"charges"`
		ShareCode string           `json:"shareCode"`
		ShareURL  string           `json:"shareUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Charges) != len(playerIDs) {
		t.Errorf("charges = %d; want %d", len(got.Charges), len(playerIDs))
	}
	if got.ShareCode == "" || len(got.ShareCode) != 6 {
		t.Errorf("shareCode = %q; want 6 chars", got.ShareCode)
	}
	if !strings.HasSuffix(got.ShareURL, "/g/"+got.ShareCode) {
		t.Errorf("shareUrl = %q; want suffix /g/%s", got.ShareURL, got.ShareCode)
	}
	if got.Session["status"] != "finalized" {
		t.Errorf("session.status = %v; want finalized", got.Session["status"])
	}

	// Sum of charge amounts == sum of splittable cost items (600001).
	var sum int64
	for _, c := range got.Charges {
		sum += int64(c["amount"].(float64))
	}
	if sum != 600001 {
		t.Errorf("sum(charges) = %d; want 600001", sum)
	}
	// Residual: first player gets 100001, rest get 100000.
	if int64(got.Charges[0]["amount"].(float64)) != 100001 {
		t.Errorf("charge[0].amount = %v; want 100001 (residual recipient)", got.Charges[0]["amount"])
	}

	// Audit rows: 1 session_finalized + N charge_created.
	var sFin int
	_ = testDB.QueryRow(
		"SELECT COUNT(*) FROM audit_log WHERE event_type='session_finalized' AND entity_id = ?",
		sessionID,
	).Scan(&sFin)
	if sFin != 1 {
		t.Errorf("session_finalized rows = %d; want 1", sFin)
	}
	var chCreated int
	_ = testDB.QueryRow(
		"SELECT COUNT(*) FROM audit_log WHERE event_type='charge_created' AND entity_id IN (SELECT id FROM session_charges WHERE session_id = ?)",
		sessionID,
	).Scan(&chCreated)
	if chCreated != len(playerIDs) {
		t.Errorf("charge_created rows = %d; want %d", chCreated, len(playerIDs))
	}
}

// AC2: idempotent — calling finalize a second time returns the same result, no new rows.
func TestFinalize_Idempotent(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-fin-idem")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	addBank(t, router, cookie)
	groupID := createGroup(t, router, cookie, "Idempotent")
	sessionID, _ := seedDraft(t, router, cookie, groupID,
		[]string{"A", "B"},
		[]map[string]any{{"type": "court", "label": "Sân", "amount": 200000}},
	)

	w1 := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookie})
	if w1.Code != http.StatusOK {
		t.Fatalf("first finalize: %d / %s", w1.Code, w1.Body.String())
	}
	var first struct {
		Charges   []map[string]any `json:"charges"`
		ShareCode string           `json:"shareCode"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &first)

	w2 := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookie})
	if w2.Code != http.StatusOK {
		t.Fatalf("second finalize: %d / %s", w2.Code, w2.Body.String())
	}
	var second struct {
		Charges   []map[string]any `json:"charges"`
		ShareCode string           `json:"shareCode"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &second)

	if first.ShareCode != second.ShareCode {
		t.Errorf("shareCode changed across calls: %q vs %q", first.ShareCode, second.ShareCode)
	}
	if len(second.Charges) != len(first.Charges) {
		t.Errorf("second charge count = %d; want same as first = %d", len(second.Charges), len(first.Charges))
	}

	// Audit row count unchanged after second call.
	var sFinAfter int
	_ = testDB.QueryRow(
		"SELECT COUNT(*) FROM audit_log WHERE event_type='session_finalized' AND entity_id = ?",
		sessionID,
	).Scan(&sFinAfter)
	if sFinAfter != 1 {
		t.Errorf("session_finalized rows after re-finalize = %d; want 1", sFinAfter)
	}
}

// AC3: bank gate — no bank → 422.
func TestFinalize_BankGate(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-fin-nobank")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	// NO bank.
	groupID := createGroup(t, router, cookie, "No bank")
	sessionID, _ := seedDraft(t, router, cookie, groupID, []string{"A"},
		[]map[string]any{{"type": "court", "label": "Sân", "amount": 100000}})

	w := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookie})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "tài khoản nhận tiền") {
		t.Errorf("body missing bank message: %s", w.Body.String())
	}
}

// AC4: finalize blocked without cost items.
func TestFinalize_NoCostItems(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	email := uniqueEmail("sess-fin-noitems")
	defer cleanupHostByEmail(t, email)
	cookie := registerHost(t, router, email)
	addBank(t, router, cookie)
	groupID := createGroup(t, router, cookie, "No items")
	playerIDs := addPlayers(t, router, cookie, groupID, []string{"A"})
	w := doJSON(router, http.MethodPost,
		"/api/v1/groups/"+strconv.FormatUint(groupID, 10)+"/sessions",
		map[string]any{"date": "2026-05-24"}, []*http.Cookie{cookie})
	var d map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &d)
	sessionID := uint64(d["id"].(float64))
	_ = doJSON(router, http.MethodPut,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/participants",
		map[string]any{"playerIds": playerIDs}, []*http.Cookie{cookie})

	w2 := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookie})
	if w2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; want 422; body=%s", w2.Code, w2.Body.String())
	}
}

// AC5: tenant isolation — Host B finalizing Host A's session → 404.
func TestFinalize_TenantIsolation(t *testing.T) {
	skipIfNoDB(t)
	router := newTestServer()
	emailA := uniqueEmail("sess-fin-tenantA")
	emailB := uniqueEmail("sess-fin-tenantB")
	defer cleanupHostByEmail(t, emailA)
	defer cleanupHostByEmail(t, emailB)
	cookieA := registerHost(t, router, emailA)
	cookieB := registerHost(t, router, emailB)
	addBank(t, router, cookieA)
	groupA := createGroup(t, router, cookieA, "A's")
	sessionID, _ := seedDraft(t, router, cookieA, groupA, []string{"X"},
		[]map[string]any{{"type": "court", "label": "Sân", "amount": 100000}})

	w := doJSON(router, http.MethodPost,
		"/api/v1/sessions/"+strconv.FormatUint(sessionID, 10)+"/finalize", nil,
		[]*http.Cookie{cookieB})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404; body=%s", w.Code, w.Body.String())
	}
}
