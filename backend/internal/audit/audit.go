// Package audit writes append-only rows into the audit_log table.
//
// EVERY payment-state-mutating operation MUST call Record() inside the same
// DB transaction as its state change. The nil-tx check enforces this — if
// you find yourself wanting to pass nil, you're calling audit at the wrong
// layer.
//
// See architecture.md §Implementation Patterns → Audit log events for the
// canonical event names and §Cross-cutting decisions → Audit logging for
// the transaction co-residency contract.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ActorType matches the audit_log.actor_type CHECK constraint values
// (migration 0001_init).
type ActorType string

const (
	ActorHost          ActorType = "host"
	ActorPlayerLink    ActorType = "player_link"
	ActorSystem        ActorType = "system"
	ActorSystemWebhook ActorType = "system_webhook"
	ActorSystemCron    ActorType = "system_cron"
	ActorAdmin         ActorType = "admin"
)

// EventType is the canonical event-name string. The DB column is open-set
// VARCHAR(48) — new events get added per story without a schema change.
type EventType string

// Canonical event types (from architecture §Implementation Patterns →
// Audit log events). Story 1.4 ships the foundation set; future stories
// add entries here as they introduce new state transitions.
const (
	// Session lifecycle
	EventSessionFinalized EventType = "session_finalized"
	EventSessionEdited    EventType = "session_edited"

	// Charge transitions
	EventChargeCreated             EventType = "charge_created"
	EventChargePaidAuto            EventType = "charge_paid_auto"
	EventChargePaidManual          EventType = "charge_paid_manual"
	EventChargePendingConfirmation EventType = "charge_pending_confirmation"
	EventChargeWaived              EventType = "charge_waived"
	EventChargeUnwaived            EventType = "charge_unwaived"

	// Payment Intent transitions
	EventPaymentIntentCreated   EventType = "payment_intent_created"
	EventPaymentIntentCancelled EventType = "payment_intent_cancelled"
	EventPaymentIntentExpired   EventType = "payment_intent_expired"

	// Payment record transitions (from webhook + manual confirm + linking)
	EventPaymentMatched      EventType = "payment_matched"
	EventPaymentSuspected    EventType = "payment_suspected"
	EventPaymentUnmatched    EventType = "payment_unmatched"
	EventPaymentLinkedManual EventType = "payment_linked_manual"

	// Auto-Detect setup
	EventAutoDetectEnabled  EventType = "auto_detect_enabled"
	EventAutoDetectDisabled EventType = "auto_detect_disabled"

	// Admin actions
	EventTierChanged     EventType = "tier_changed"
	EventPlayerCodeReset EventType = "player_code_reset"

	// Group lifecycle — added in Story 1.8.
	EventGroupCreated EventType = "group_created"
)

// Event captures what to write. Callers populate the fields they have;
// unset Before/After/Context JSON fields become DB NULLs.
type Event struct {
	Type        EventType
	ActorType   ActorType
	HostUserID  *int64
	EntityType  string // 'session', 'charge', 'payment_intent', 'payment', 'group', 'player'
	EntityID    int64
	BeforeState json.RawMessage
	AfterState  json.RawMessage
	ContextJSON json.RawMessage
	OccurredAt  time.Time // defaults to time.Now().UTC() if zero
}

// ErrNilTx is returned when Record is called with a nil transaction.
// Audit rows MUST live in the same DB transaction as the state change
// they record — passing nil breaks atomicity.
var ErrNilTx = errors.New("audit.Record: tx is nil — audit rows must live in the same transaction as the state change")

const insertSQL = `
INSERT INTO audit_log
    (host_user_id, actor_type, event_type, entity_type, entity_id,
     before_state, after_state, context_json, occurred_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// Record writes one audit_log row inside the supplied transaction.
// Returns ErrNilTx when tx is nil.
//
// Optional fields (HostUserID, BeforeState, AfterState, ContextJSON,
// OccurredAt) are nullable / defaulted. The caller controls Commit /
// Rollback — Record only INSERTs.
func Record(ctx context.Context, tx *sql.Tx, e Event) error {
	if tx == nil {
		return ErrNilTx
	}

	if e.Type == "" {
		return fmt.Errorf("audit.Record: Event.Type is required")
	}
	if e.ActorType == "" {
		return fmt.Errorf("audit.Record: Event.ActorType is required")
	}
	if e.EntityType == "" {
		return fmt.Errorf("audit.Record: Event.EntityType is required")
	}

	occurredAt := e.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	hostUserID := nullInt64Ptr(e.HostUserID)
	before := nullJSON(e.BeforeState)
	after := nullJSON(e.AfterState)
	contextJSON := nullJSON(e.ContextJSON)

	_, err := tx.ExecContext(
		ctx, insertSQL,
		hostUserID,
		string(e.ActorType),
		string(e.Type),
		e.EntityType,
		e.EntityID,
		before,
		after,
		contextJSON,
		occurredAt,
	)
	if err != nil {
		return fmt.Errorf("audit.Record: INSERT: %w", err)
	}
	return nil
}

func nullInt64Ptr(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullJSON(raw json.RawMessage) sql.NullString {
	if len(raw) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(raw), Valid: true}
}
