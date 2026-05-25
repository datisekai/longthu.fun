package payments

import (
	"context"
	"database/sql"
	"encoding/json"

	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

// Matcher handles payment matching logic.
type Matcher struct {
	db *sql.DB
}

// NewMatcher creates a new Matcher.
func NewMatcher(db *sql.DB) *Matcher {
	return &Matcher{db: db}
}

// Service handles payment-related operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new payment service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// SuspectedPayment represents a suspected payment with context.
type SuspectedPayment struct {
	ID               uint64  `json:"id"`
	Amount          int64   `json:"amount"`
	BankDescription string  `json:"bankDescription"`
	ReceivedAt      string  `json:"receivedAt"`
	IntentCode      *string `json:"intentCode,omitempty"`
	IntentAmount    *int64  `json:"intentAmount,omitempty"`
	PlayerName      *string `json:"playerName,omitempty"`
}

// UnmatchedPayment represents an unmatched payment.
type UnmatchedPayment struct {
	ID               uint64  `json:"id"`
	Amount          int64   `json:"amount"`
	BankDescription string  `json:"bankDescription"`
	ReceivedAt      string  `json:"receivedAt"`
	CounterAccount  *string `json:"counterAccount,omitempty"`
}

// ListSuspectedPayments returns suspected payments for a host.
func (s *Service) ListSuspectedPayments(ctx context.Context, hostID uint64) ([]SuspectedPayment, error) {
	q := dbgen.New(s.db)
	rows, err := q.ListSuspectedPaymentsForHost(ctx, sql.NullInt64{Int64: int64(hostID), Valid: true})
	if err != nil {
		return nil, err
	}

	payments := make([]SuspectedPayment, 0, len(rows))
	for _, r := range rows {
		p := SuspectedPayment{
			ID:               r.ID,
			Amount:          r.Amount,
			BankDescription: r.BankDescription,
			ReceivedAt:      r.ReceivedAt.Format("02/01/2006 15:04"),
		}
		if r.IntentCode.Valid {
			p.IntentCode = &r.IntentCode.String
		}
		if r.IntentAmount.Valid {
			p.IntentAmount = &r.IntentAmount.Int64
		}
		if r.PlayerName.Valid {
			p.PlayerName = &r.PlayerName.String
		}
		payments = append(payments, p)
	}
	return payments, nil
}

// ListUnmatchedPayments returns unmatched payments for a host.
func (s *Service) ListUnmatchedPayments(ctx context.Context, hostID uint64) ([]UnmatchedPayment, error) {
	q := dbgen.New(s.db)
	rows, err := q.ListUnmatchedPaymentsForHost(ctx, sql.NullInt64{Int64: int64(hostID), Valid: true})
	if err != nil {
		return nil, err
	}

	payments := make([]UnmatchedPayment, 0, len(rows))
	for _, r := range rows {
		payments = append(payments, UnmatchedPayment{
			ID:               r.ID,
			Amount:          r.Amount,
			BankDescription: r.BankDescription,
			ReceivedAt:      r.ReceivedAt.Format("02/01/2006 15:04"),
		})
	}
	return payments, nil
}

// ConfirmSuspectedPayment confirms a suspected payment.
func (s *Service) ConfirmSuspectedPayment(ctx context.Context, hostID, paymentID uint64) error {
	matcher := NewMatcher(s.db)
	return matcher.ConfirmMatch(ctx, hostID, paymentID)
}

// RejectSuspectedPayment rejects a suspected payment (marks as unmatched).
func (s *Service) RejectSuspectedPayment(ctx context.Context, hostID, paymentID uint64) error {
	return dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE payments 
			SET status = 'unmatched', updated_at = CURRENT_TIMESTAMP(3)
			WHERE id = ? AND status = 'suspected' AND host_user_id = ?
		`, paymentID, hostID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id, context_json)
			VALUES (?, 'host', 'payment_suspected_rejected', 'payment', ?, '{"action": "rejected"}')
		`, hostID, paymentID)
		return err
	})
}

// LinkPaymentToPlayer links an unmatched payment to a player and charges.
func (s *Service) LinkPaymentToPlayer(ctx context.Context, hostID, paymentID, playerID uint64, chargeIDs []uint64) error {
	matcher := NewMatcher(s.db)
	return matcher.LinkManually(ctx, hostID, paymentID, playerID, chargeIDs)
}

// MatchParams holds parameters for matching a payment.
type MatchParams struct {
	Provider              string
	ProviderTransactionID string
	Amount               int64
	Description          string
	PaymentIntentID      uint64
	PlayerID             uint64
	GroupID              uint64
	HostUserID           uint64
	RawPayload           []byte
}

// Match attempts to match a payment with a payment intent.
// This should be called within a transaction.
func (m *Matcher) Match(ctx context.Context, tx *sql.Tx, params MatchParams) error {
	q := dbgen.New(tx)

	// Get the payment intent by ID.
	intent, err := q.GetPaymentIntentByID(ctx, params.PaymentIntentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return m.recordUnmatched(ctx, tx, params)
		}
		return err
	}

	// Check if intent is still pending.
	if intent.Status != "pending" {
		return m.recordSuspected(ctx, tx, params, intent.ID)
	}

	// Check if amount matches exactly.
	if params.Amount != intent.Amount {
		return m.recordSuspected(ctx, tx, params, intent.ID)
	}

	// Exact match! Record the payment and update the intent + charges.
	return m.recordMatched(ctx, tx, params)
}

// recordMatched records a matched payment.
func (m *Matcher) recordMatched(ctx context.Context, tx *sql.Tx, params MatchParams) error {
	hostUserID := int64(params.HostUserID)

	// Insert the payment record.
	res, err := tx.ExecContext(ctx, `
		INSERT INTO payments (player_id, amount, bank_description, matched_intent_id, status, provider, provider_transaction_id, received_at, raw_payload_json, host_user_id)
		VALUES (?, ?, ?, ?, 'matched', ?, ?, NOW(3), ?, ?)
	`, params.PlayerID, params.Amount, params.Description, params.PaymentIntentID, params.Provider, params.ProviderTransactionID, string(params.RawPayload), hostUserID)
	if err != nil {
		return err
	}

	paymentID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// Update payment intent to matched.
	_, err = tx.ExecContext(ctx, `
		UPDATE payment_intents SET status = 'matched', provider_payment_id = ?, updated_at = CURRENT_TIMESTAMP(3)
		WHERE id = ?
	`, params.ProviderTransactionID, params.PaymentIntentID)
	if err != nil {
		return err
	}

	// Get intent to parse covers_charge_ids_json.
	intent, _ := dbgen.New(tx).GetPaymentIntentByID(ctx, params.PaymentIntentID)
	var chargeIDs []uint64
	if len(intent.CoversChargeIdsJson) > 0 {
		json.Unmarshal(intent.CoversChargeIdsJson, &chargeIDs)
	}

	// Update each charge to paid.
	for _, chargeID := range chargeIDs {
		_, err = tx.ExecContext(ctx, `
			UPDATE session_charges SET status = 'paid', paid_at = NOW(), paid_via = 'auto', updated_at = CURRENT_TIMESTAMP(3)
			WHERE id = ? AND status IN ('unpaid', 'pending_confirmation')
		`, chargeID)
		if err != nil {
			return err
		}

		// Audit log for charge.
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id)
			VALUES (?, 'system_webhook', 'charge_paid_auto', 'charge', ?)
		`, hostUserID, chargeID)
	}

	// Audit log for payment matched.
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id, context_json)
		VALUES (?, 'system_webhook', 'payment_matched', 'payment', ?, ?)
	`, hostUserID, paymentID, string(params.RawPayload))

	return nil
}

// recordSuspected records a suspected payment.
func (m *Matcher) recordSuspected(ctx context.Context, tx *sql.Tx, params MatchParams, intentID uint64) error {
	hostUserID := int64(params.HostUserID)

	// Insert the payment record.
	res, err := tx.ExecContext(ctx, `
		INSERT INTO payments (player_id, amount, bank_description, matched_intent_id, status, provider, provider_transaction_id, received_at, raw_payload_json, host_user_id)
		VALUES (?, ?, ?, ?, 'suspected', ?, ?, NOW(3), ?, ?)
	`, params.PlayerID, params.Amount, params.Description, intentID, params.Provider, params.ProviderTransactionID, string(params.RawPayload), hostUserID)
	if err != nil {
		return err
	}

	paymentID, _ := res.LastInsertId()

	// Audit log for suspected payment.
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id, context_json)
		VALUES (?, 'system_webhook', 'payment_suspected', 'payment', ?, ?)
	`, hostUserID, paymentID, string(params.RawPayload))

	return nil
}

// recordUnmatched records an unmatched payment.
func (m *Matcher) recordUnmatched(ctx context.Context, tx *sql.Tx, params MatchParams) error {
	hostUserID := int64(params.HostUserID)

	// Insert the payment record.
	res, err := tx.ExecContext(ctx, `
		INSERT INTO payments (player_id, amount, bank_description, matched_intent_id, status, provider, provider_transaction_id, received_at, raw_payload_json, host_user_id)
		VALUES (?, ?, ?, NULL, 'unmatched', ?, ?, NOW(3), ?, ?)
	`, nil, params.Amount, params.Description, params.Provider, params.ProviderTransactionID, string(params.RawPayload), hostUserID)
	if err != nil {
		return err
	}

	paymentID, _ := res.LastInsertId()

	// Audit log for unmatched payment.
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id, context_json)
		VALUES (?, 'system_webhook', 'payment_unmatched', 'payment', ?, ?)
	`, hostUserID, paymentID, string(params.RawPayload))

	return nil
}

// ConfirmMatch confirms a suspected payment manually.
func (m *Matcher) ConfirmMatch(ctx context.Context, hostID uint64, paymentID uint64) error {
	return dbpkg.WithTx(ctx, m.db, func(tx *sql.Tx) error {
		hostUserID := int64(hostID)

		// Get payment.
		var payment struct {
			Amount   int64
			PlayerID sql.NullInt64
			IntentID sql.NullInt64
		}
		err := tx.QueryRowContext(ctx, `
			SELECT amount, player_id, matched_intent_id FROM payments WHERE id = ?
		`, paymentID).Scan(&payment.Amount, &payment.PlayerID, &payment.IntentID)
		if err != nil {
			return err
		}

		// Update payment to matched.
		_, err = tx.ExecContext(ctx, `
			UPDATE payments SET status = 'matched', updated_at = CURRENT_TIMESTAMP(3)
			WHERE id = ?
		`, paymentID)
		if err != nil {
			return err
		}

		// If there's a matched_intent_id, update the intent.
		if payment.IntentID.Valid {
			_, _ = tx.ExecContext(ctx, `
				UPDATE payment_intents SET status = 'matched', updated_at = CURRENT_TIMESTAMP(3)
				WHERE id = ?
			`, payment.IntentID.Int64)
		}

		// Get player's unpaid charges and update them.
		if payment.PlayerID.Valid {
			_, _ = tx.ExecContext(ctx, `
				UPDATE session_charges SET status = 'paid', paid_at = NOW(), paid_via = 'linked', updated_at = CURRENT_TIMESTAMP(3)
				WHERE player_id = ? AND status IN ('unpaid', 'pending_confirmation', 'suspected')
			`, payment.PlayerID.Int64)
		}

		// Audit log.
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id)
			VALUES (?, 'host', 'payment_linked_manual', 'payment', ?)
		`, hostUserID, paymentID)

		return nil
	})
}

// LinkManually links an unmatched payment to a player and charges.
func (m *Matcher) LinkManually(ctx context.Context, hostID uint64, paymentID, playerID uint64, chargeIDs []uint64) error {
	return dbpkg.WithTx(ctx, m.db, func(tx *sql.Tx) error {
		hostUserID := int64(hostID)

		// Update payment to matched with player.
		_, err := tx.ExecContext(ctx, `
			UPDATE payments SET status = 'matched', player_id = ?, updated_at = CURRENT_TIMESTAMP(3)
			WHERE id = ? AND status = 'unmatched'
		`, playerID, paymentID)
		if err != nil {
			return err
		}

		// Update charges to paid.
		for _, chargeID := range chargeIDs {
			_, _ = tx.ExecContext(ctx, `
				UPDATE session_charges SET status = 'paid', paid_at = NOW(), paid_via = 'linked', updated_at = CURRENT_TIMESTAMP(3)
				WHERE id = ? AND status IN ('unpaid', 'pending_confirmation', 'suspected')
			`, chargeID)

			// Audit log for charge.
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id)
				VALUES (?, 'host', 'charge_paid_manual', 'charge', ?)
			`, hostUserID, chargeID)
		}

		// Audit log for payment linked.
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO audit_log (host_user_id, actor_type, event_type, entity_type, entity_id)
			VALUES (?, 'host', 'payment_linked_manual', 'payment', ?)
		`, hostUserID, paymentID)

		return nil
	})
}
