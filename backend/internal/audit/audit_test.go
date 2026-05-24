package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// testDB is opened in TestMain if DATABASE_URL is set; otherwise tests skip.
var testDB *sql.DB

func TestMain(m *testing.M) {
	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		// No DB available — TestMain runs nothing; individual tests gate via testDB == nil.
		os.Exit(m.Run())
	}

	// Strip the `mysql://` prefix that golang-migrate's Makefile uses;
	// the Go MySQL driver expects `user:pass@tcp(host:port)/db?...` directly.
	dsn := strings.TrimPrefix(rawDSN, "mysql://")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// Open shouldn't actually contact the DB — but if DSN parsing fails, fail the suite.
		panic("audit_test: sql.Open: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		// DB unreachable — skip all tests.
		os.Exit(m.Run())
	}
	testDB = db
	defer testDB.Close()
	os.Exit(m.Run())
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("DATABASE_URL not set or unreachable — skipping audit integration tests")
	}
}

// uniqueEntityID returns a time-based id well above any seed/test fixture,
// so tests can DELETE only their own rows without affecting other data.
func uniqueEntityID() int64 {
	return time.Now().UnixNano()
}

func cleanupAuditByEntityID(t *testing.T, entityID int64) {
	t.Helper()
	_, err := testDB.Exec("DELETE FROM audit_log WHERE entity_id = ?", entityID)
	if err != nil {
		t.Logf("cleanup DELETE failed (non-fatal): %v", err)
	}
}

func TestRecord_NilTxFails(t *testing.T) {
	// This test does NOT need a DB — it asserts the nil-tx guard before any SQL.
	err := Record(context.Background(), nil, Event{
		Type:       EventSessionFinalized,
		ActorType:  ActorHost,
		EntityType: "session",
		EntityID:   1,
	})
	if !errors.Is(err, ErrNilTx) {
		t.Fatalf("Record(nil tx) = %v; want ErrNilTx", err)
	}
}

func TestRecord_RequiresType(t *testing.T) {
	skipIfNoDB(t)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback()

	err = Record(context.Background(), tx, Event{
		// Type intentionally empty
		ActorType:  ActorHost,
		EntityType: "session",
		EntityID:   1,
	})
	if err == nil {
		t.Fatal("Record with empty Type should return error")
	}
	if !strings.Contains(err.Error(), "Type is required") {
		t.Errorf("error %q should mention missing Type", err)
	}
}

func TestRecord_InsertsRowAndCommits(t *testing.T) {
	skipIfNoDB(t)
	entityID := uniqueEntityID()
	defer cleanupAuditByEntityID(t, entityID)

	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	afterState := json.RawMessage(`{"status":"finalized","share_code":"TEST01"}`)
	err = Record(ctx, tx, Event{
		Type:       EventSessionFinalized,
		ActorType:  ActorHost,
		EntityType: "session",
		EntityID:   entityID,
		AfterState: afterState,
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Record: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify row visible after commit.
	var (
		gotType  string
		gotActor string
		gotAfter sql.NullString
	)
	err = testDB.QueryRow(
		"SELECT event_type, actor_type, after_state FROM audit_log WHERE entity_id = ?",
		entityID,
	).Scan(&gotType, &gotActor, &gotAfter)
	if err != nil {
		t.Fatalf("SELECT after commit: %v", err)
	}
	if gotType != string(EventSessionFinalized) {
		t.Errorf("event_type = %q; want %q", gotType, EventSessionFinalized)
	}
	if gotActor != string(ActorHost) {
		t.Errorf("actor_type = %q; want %q", gotActor, ActorHost)
	}
	if !gotAfter.Valid || !strings.Contains(gotAfter.String, "TEST01") {
		t.Errorf("after_state not preserved: %+v", gotAfter)
	}
}

func TestRecord_RollbackLeavesNoRow(t *testing.T) {
	skipIfNoDB(t)
	entityID := uniqueEntityID()
	defer cleanupAuditByEntityID(t, entityID) // defensive — should be no-op since we rollback

	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	err = Record(ctx, tx, Event{
		Type:       EventChargePaidManual,
		ActorType:  ActorHost,
		EntityType: "charge",
		EntityID:   entityID,
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Record: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	var count int
	err = testDB.QueryRow(
		"SELECT COUNT(*) FROM audit_log WHERE entity_id = ?",
		entityID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("SELECT after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("after Rollback: %d rows visible; want 0", count)
	}
}

func TestRecord_DefaultsOccurredAt(t *testing.T) {
	skipIfNoDB(t)
	entityID := uniqueEntityID()
	defer cleanupAuditByEntityID(t, entityID)

	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	before := time.Now().UTC()
	err = Record(ctx, tx, Event{
		Type:       EventPaymentMatched,
		ActorType:  ActorSystemWebhook,
		EntityType: "payment",
		EntityID:   entityID,
		// OccurredAt left zero — Record should default to time.Now().UTC()
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Record: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	after := time.Now().UTC()

	var occurredAt time.Time
	err = testDB.QueryRow(
		"SELECT occurred_at FROM audit_log WHERE entity_id = ?",
		entityID,
	).Scan(&occurredAt)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}

	// occurred_at should fall within [before, after] (allow 5s slack for DB clock drift).
	slack := 5 * time.Second
	if occurredAt.Before(before.Add(-slack)) || occurredAt.After(after.Add(slack)) {
		t.Errorf("occurred_at = %v; want in [%v, %v] +/- %v", occurredAt, before, after, slack)
	}
}

func TestRecord_NullableHostUserID(t *testing.T) {
	skipIfNoDB(t)
	entityID := uniqueEntityID()
	defer cleanupAuditByEntityID(t, entityID)

	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	err = Record(ctx, tx, Event{
		Type:       EventPaymentUnmatched,
		ActorType:  ActorSystemWebhook,
		HostUserID: nil, // system events have no host
		EntityType: "payment",
		EntityID:   entityID,
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Record: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var hostUserID sql.NullInt64
	err = testDB.QueryRow(
		"SELECT host_user_id FROM audit_log WHERE entity_id = ?",
		entityID,
	).Scan(&hostUserID)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if hostUserID.Valid {
		t.Errorf("host_user_id should be NULL; got %d", hostUserID.Int64)
	}
}
