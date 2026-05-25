// Package sessions owns the Session lifecycle: draft (create / add cost
// items / set participants — Story 1.10) and finalize (mint share code,
// generate Charges, audit — Story 1.11).
package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/datisekai/longthu.fun/backend/internal/audit"
	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/shortcode"
)

// shareCodeLength per epics §1.11: GroupShare codes are 6 Crockford-base32
// chars (~1B keyspace; comfortable with the 10-retry GenerateUnique loop).
const shareCodeLength = 6

// CostItemType matches the session_cost_items.type CHECK constraint.
type CostItemType string

const (
	CostCourt    CostItemType = "court"
	CostShuttle  CostItemType = "shuttle"
	CostWater    CostItemType = "water"
	CostOther    CostItemType = "other"
	CostDiscount CostItemType = "discount"
)

func validCostItemType(t string) bool {
	switch CostItemType(t) {
	case CostCourt, CostShuttle, CostWater, CostOther, CostDiscount:
		return true
	}
	return false
}

// Public response shapes — kept distinct from sqlc row structs so the JSON
// contract is stable even if the schema migrates.
type PublicSession struct {
	ID          uint64  `json:"id"`
	GroupID     uint64  `json:"groupId"`
	Date        string  `json:"date"` // YYYY-MM-DD
	Title       *string `json:"title,omitempty"`
	Location    *string `json:"location,omitempty"`
	Status      string  `json:"status"`
	TotalCost   int64   `json:"totalCost"`
	ShareCode   *string `json:"shareCode,omitempty"`
	FinalizedAt *string `json:"finalizedAt,omitempty"`
}

type PublicCostItem struct {
	ID                uint64 `json:"id"`
	SessionID         uint64 `json:"sessionId"`
	Type              string `json:"type"`
	Label             string `json:"label"`
	Amount            int64  `json:"amount"`
	IsIncludedInSplit bool   `json:"isIncludedInSplit"`
}

type PublicParticipant struct {
	PlayerID uint64 `json:"playerId"`
}

// Typed errors mapped to HTTP statuses by the handler.
var (
	ErrSessionNotFound           = errors.New("sessions: not found")
	ErrGroupNotFound             = errors.New("sessions: group not found for this host")
	ErrInvalidDate               = errors.New("sessions: invalid date")
	ErrInvalidCostItemType       = errors.New("sessions: invalid cost item type")
	ErrInvalidCostItemAmount     = errors.New("sessions: invalid cost item amount")
	ErrInvalidCostItemLabel      = errors.New("sessions: invalid cost item label")
	ErrParticipantsEmpty         = errors.New("sessions: at least one participant required")
	ErrParticipantOutsideRoster  = errors.New("sessions: participant not in group roster")
	ErrNoBankAccount            = errors.New("sessions: host has no bank account; cannot finalize")
	ErrNoCostItems              = errors.New("sessions: cannot finalize without at least 1 cost item")
	ErrCostItemNotFound         = errors.New("sessions: cost item not found")
	ErrChargeNotFound           = errors.New("sessions: charge not found")
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateDraftParams holds the validated input for CreateDraft.
type CreateDraftParams struct {
	GroupID  uint64
	Date     string // YYYY-MM-DD (Asia/Ho_Chi_Minh local; backend stores as DATE)
	Title    string
	Location string
}

// CreateDraft creates a new draft Session under the given Group (which must
// belong to hostID). Returns the public shape.
func (s *Service) CreateDraft(ctx context.Context, hostID uint64, params CreateDraftParams) (PublicSession, error) {
	date, err := parseDate(params.Date)
	if err != nil {
		return PublicSession{}, ErrInvalidDate
	}

	// Tenant + existence check: host owns the group.
	q := dbgen.New(s.db)
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: params.GroupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicSession{}, ErrGroupNotFound
		}
		return PublicSession{}, fmt.Errorf("sessions.CreateDraft: group check: %w", err)
	}

	title := nullString(strings.TrimSpace(params.Title))
	location := nullString(strings.TrimSpace(params.Location))

	res, err := q.InsertSession(ctx, dbgen.InsertSessionParams{
		GroupID:         params.GroupID,
		Date:            date,
		Title:           title,
		Location:        location,
		CreatedByUserID: hostID,
	})
	if err != nil {
		return PublicSession{}, fmt.Errorf("sessions.CreateDraft: insert: %w", err)
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return PublicSession{}, fmt.Errorf("sessions.CreateDraft: lastInsertId: %w", err)
	}

	row, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: uint64(newID), HostUserID: hostID,
	})
	if err != nil {
		return PublicSession{}, fmt.Errorf("sessions.CreateDraft: re-read: %w", err)
	}
	return toPublicSession(row), nil
}

// AddCostItemParams holds validated input for AddCostItem.
type AddCostItemParams struct {
	SessionID         uint64
	Type              string
	Label             string
	Amount            int64
	IsIncludedInSplit bool
}

func (s *Service) AddCostItem(ctx context.Context, hostID uint64, params AddCostItemParams) (PublicCostItem, error) {
	if !validCostItemType(params.Type) {
		return PublicCostItem{}, ErrInvalidCostItemType
	}
	if params.Amount == 0 {
		return PublicCostItem{}, ErrInvalidCostItemAmount
	}
	label := strings.TrimSpace(params.Label)
	if label == "" || len(label) > 80 {
		return PublicCostItem{}, ErrInvalidCostItemLabel
	}

	q := dbgen.New(s.db)
	// Tenant check: session must belong to host.
	if _, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: params.SessionID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicCostItem{}, ErrSessionNotFound
		}
		return PublicCostItem{}, fmt.Errorf("sessions.AddCostItem: session check: %w", err)
	}

	res, err := q.InsertSessionCostItem(ctx, dbgen.InsertSessionCostItemParams{
		SessionID:         params.SessionID,
		Type:              params.Type,
		Label:             label,
		Amount:            params.Amount,
		IsIncludedInSplit: params.IsIncludedInSplit,
	})
	if err != nil {
		return PublicCostItem{}, fmt.Errorf("sessions.AddCostItem: insert: %w", err)
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return PublicCostItem{}, fmt.Errorf("sessions.AddCostItem: lastInsertId: %w", err)
	}
	return PublicCostItem{
		ID:                uint64(newID),
		SessionID:         params.SessionID,
		Type:              params.Type,
		Label:             label,
		Amount:            params.Amount,
		IsIncludedInSplit: params.IsIncludedInSplit,
	}, nil
}

// RemoveCostItem deletes a single cost item from a draft session.
func (s *Service) RemoveCostItem(ctx context.Context, hostID uint64, sessionID uint64, itemID uint64) error {
	q := dbgen.New(s.db)
	if _, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: sessionID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("sessions.RemoveCostItem: session check: %w", err)
	}
	if err := q.DeleteSessionCostItem(ctx, dbgen.DeleteSessionCostItemParams{
		ID: itemID, SessionID: sessionID,
	}); err != nil {
		return fmt.Errorf("sessions.RemoveCostItem: delete: %w", err)
	}
	return nil
}

// SetParticipants replaces the participant set for a draft session in one tx.
// Validates that every submitted playerID belongs to the session's Group.
func (s *Service) SetParticipants(ctx context.Context, hostID uint64, sessionID uint64, playerIDs []uint64) ([]PublicParticipant, error) {
	if len(playerIDs) == 0 {
		return nil, ErrParticipantsEmpty
	}

	q := dbgen.New(s.db)
	session, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: sessionID, HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessions.SetParticipants: session check: %w", err)
	}

	dedup := dedupUint64(playerIDs)
	count, err := q.CountPlayersInGroupByIDs(ctx, dbgen.CountPlayersInGroupByIDsParams{
		GroupID: session.GroupID, PlayerIds: dedup,
	})
	if err != nil {
		return nil, fmt.Errorf("sessions.SetParticipants: validate roster: %w", err)
	}
	if int(count) != len(dedup) {
		return nil, ErrParticipantOutsideRoster
	}

	var result []PublicParticipant
	err = dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		txq := dbgen.New(tx)
		if err := txq.DeleteAllSessionParticipants(ctx, sessionID); err != nil {
			return fmt.Errorf("sessions.SetParticipants: clear: %w", err)
		}
		for _, pid := range dedup {
			if err := txq.InsertSessionParticipant(ctx, dbgen.InsertSessionParticipantParams{
				SessionID: sessionID, PlayerID: pid,
			}); err != nil {
				return fmt.Errorf("sessions.SetParticipants: insert %d: %w", pid, err)
			}
			result = append(result, PublicParticipant{PlayerID: pid})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDraftWithDetails returns the session + its cost items + participants
// (used by the handler's GET path for refresh / hydration).
type DraftSnapshot struct {
	Session      PublicSession       `json:"session"`
	CostItems    []PublicCostItem    `json:"costItems"`
	Participants []PublicParticipant `json:"participants"`
}

func (s *Service) GetDraftWithDetails(ctx context.Context, hostID uint64, sessionID uint64) (DraftSnapshot, error) {
	q := dbgen.New(s.db)
	row, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: sessionID, HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DraftSnapshot{}, ErrSessionNotFound
		}
		return DraftSnapshot{}, fmt.Errorf("sessions.GetDraft: read: %w", err)
	}
	items, err := q.ListSessionCostItems(ctx, sessionID)
	if err != nil {
		return DraftSnapshot{}, fmt.Errorf("sessions.GetDraft: list items: %w", err)
	}
	participants, err := q.ListSessionParticipants(ctx, sessionID)
	if err != nil {
		return DraftSnapshot{}, fmt.Errorf("sessions.GetDraft: list participants: %w", err)
	}

	out := DraftSnapshot{
		Session:      toPublicSession(row),
		CostItems:    make([]PublicCostItem, 0, len(items)),
		Participants: make([]PublicParticipant, 0, len(participants)),
	}
	for _, it := range items {
		out.CostItems = append(out.CostItems, PublicCostItem{
			ID: it.ID, SessionID: it.SessionID, Type: it.Type, Label: it.Label,
			Amount: it.Amount, IsIncludedInSplit: it.IsIncludedInSplit,
		})
	}
	for _, p := range participants {
		out.Participants = append(out.Participants, PublicParticipant{PlayerID: p.PlayerID})
	}
	return out, nil
}

// PublicCharge is the response shape for a finalized session_charges row.
type PublicCharge struct {
	ID        uint64  `json:"id"`
	SessionID uint64  `json:"sessionId"`
	PlayerID  uint64  `json:"playerId"`
	Amount    int64   `json:"amount"`
	Status    string  `json:"status"`
	PaidVia   *string `json:"paidVia,omitempty"`
}

// FinalizeResult is what the handler returns on a successful Finalize call
// (or on idempotent re-finalize).
type FinalizeResult struct {
	Session   PublicSession  `json:"session"`
	Charges   []PublicCharge `json:"charges"`
	ShareCode string         `json:"shareCode"`
}

// Finalize mints share_code + generates Charges + writes audit, all in one tx.
// Idempotent: if the session is already finalized, returns the existing
// snapshot without re-executing the tx.
//
// Bank gate: requires the host to have at least one bank account.
func (s *Service) Finalize(ctx context.Context, hostID uint64, sessionID uint64) (FinalizeResult, error) {
	q := dbgen.New(s.db)

	// Tenant + existence check.
	sess, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: sessionID, HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FinalizeResult{}, ErrSessionNotFound
		}
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: session check: %w", err)
	}

	// Idempotent short-circuit.
	if sess.Status == "finalized" {
		charges, err := q.ListSessionCharges(ctx, sessionID)
		if err != nil {
			return FinalizeResult{}, fmt.Errorf("sessions.Finalize: re-read charges: %w", err)
		}
		out := FinalizeResult{Session: toPublicSession(sess)}
		if sess.ShareCode.Valid {
			out.ShareCode = sess.ShareCode.String
		}
		for _, c := range charges {
			out.Charges = append(out.Charges, toPublicCharge(c))
		}
		return out, nil
	}

	// Bank gate.
	bankCount, err := q.CountBankAccountsForHost(ctx, hostID)
	if err != nil {
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: bank check: %w", err)
	}
	if bankCount == 0 {
		return FinalizeResult{}, ErrNoBankAccount
	}

	// Pre-read cost items + participants for the split math.
	items, err := q.ListSessionCostItems(ctx, sessionID)
	if err != nil {
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: list items: %w", err)
	}
	if len(items) == 0 {
		return FinalizeResult{}, ErrNoCostItems
	}
	participants, err := q.ListSessionParticipants(ctx, sessionID)
	if err != nil {
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: list participants: %w", err)
	}
	if len(participants) == 0 {
		return FinalizeResult{}, ErrParticipantsEmpty
	}

	totalCost, splittable := sumCostItems(items)
	playerIDs := make([]uint64, 0, len(participants))
	for _, p := range participants {
		playerIDs = append(playerIDs, p.PlayerID)
	}
	amounts := distributeSplit(splittable, len(playerIDs))

	// Mint share code BEFORE the tx (so collision retries don't hold a lock).
	shareCode, err := shortcode.GenerateUnique(ctx, shareCodeLength, shortcode.GroupShare, func(ctx context.Context, candidate string) (bool, error) {
		return q.SessionShareCodeExists(ctx, sql.NullString{String: candidate, Valid: true})
	})
	if err != nil {
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: mint share code: %w", err)
	}

	hostIDInt := int64(hostID)
	finalizedAt := time.Now().UTC()
	var charges []PublicCharge

	err = dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		txq := dbgen.New(tx)

		// Update session row.
		if err := txq.UpdateSessionFinalize(ctx, dbgen.UpdateSessionFinalizeParams{
			ShareCode:   sql.NullString{String: shareCode, Valid: true},
			TotalCost:   totalCost,
			FinalizedAt: sql.NullTime{Time: finalizedAt, Valid: true},
			ID:          sessionID,
		}); err != nil {
			return fmt.Errorf("update session: %w", err)
		}

		// Audit: session_finalized (host actor).
		if err := audit.Record(ctx, tx, audit.Event{
			Type: audit.EventSessionFinalized, ActorType: audit.ActorHost,
			HostUserID: &hostIDInt, EntityType: "session", EntityID: int64(sessionID),
		}); err != nil {
			return fmt.Errorf("audit session_finalized: %w", err)
		}

		// Insert charges + per-charge audit.
		for i, pid := range playerIDs {
			res, err := txq.InsertSessionCharge(ctx, dbgen.InsertSessionChargeParams{
				SessionID: sessionID, PlayerID: pid, Amount: amounts[i],
			})
			if err != nil {
				return fmt.Errorf("insert charge for player %d: %w", pid, err)
			}
			chargeID, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("lastInsertId charge: %w", err)
			}
			if err := audit.Record(ctx, tx, audit.Event{
				Type: audit.EventChargeCreated, ActorType: audit.ActorSystem,
				HostUserID: &hostIDInt, EntityType: "charge", EntityID: chargeID,
			}); err != nil {
				return fmt.Errorf("audit charge_created: %w", err)
			}
			charges = append(charges, PublicCharge{
				ID: uint64(chargeID), SessionID: sessionID, PlayerID: pid,
				Amount: amounts[i], Status: "unpaid",
			})
		}
		return nil
	})
	if err != nil {
		return FinalizeResult{}, err
	}

	// Re-read for accurate FinalizedAt/Status fields.
	updated, err := q.GetSessionByIDForHost(ctx, dbgen.GetSessionByIDForHostParams{
		ID: sessionID, HostUserID: hostID,
	})
	if err != nil {
		return FinalizeResult{}, fmt.Errorf("sessions.Finalize: re-read: %w", err)
	}
	return FinalizeResult{
		Session: toPublicSession(updated), Charges: charges, ShareCode: shareCode,
	}, nil
}

// sumCostItems returns (total, splittable). Splittable excludes items with
// is_included_in_split=false.
func sumCostItems(items []dbgen.ListSessionCostItemsRow) (int64, int64) {
	var total, splittable int64
	for _, it := range items {
		total += it.Amount
		if it.IsIncludedInSplit {
			splittable += it.Amount
		}
	}
	return total, splittable
}

// distributeSplit returns `count` amounts summing exactly to `splittable`.
// Residual VND from integer division goes to the FIRST `residual` participants.
// Example: 600000 / 7 = 85714 base + residual 2 → [85715, 85715, 85714, 85714, 85714, 85714, 85714].
func distributeSplit(splittable int64, count int) []int64 {
	if count <= 0 {
		return nil
	}
	base := splittable / int64(count)
	residual := splittable - base*int64(count)
	out := make([]int64, count)
	for i := 0; i < count; i++ {
		amt := base
		if int64(i) < residual {
			amt++
		}
		out[i] = amt
	}
	return out
}

func toPublicCharge(r dbgen.SessionCharge) PublicCharge {
	out := PublicCharge{
		ID:        r.ID,
		SessionID: r.SessionID,
		PlayerID:  r.PlayerID,
		Amount:    r.Amount,
		Status:    r.Status,
	}
	if r.PaidVia.Valid {
		v := r.PaidVia.String
		out.PaidVia = &v
	}
	return out
}

// --- helpers ---

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty date")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toPublicSession(s dbgen.Session) PublicSession {
	out := PublicSession{
		ID:        s.ID,
		GroupID:   s.GroupID,
		Date:      s.Date.Format("2006-01-02"),
		Status:    s.Status,
		TotalCost: s.TotalCost,
	}
	if s.Title.Valid {
		t := s.Title.String
		out.Title = &t
	}
	if s.Location.Valid {
		l := s.Location.String
		out.Location = &l
	}
	if s.ShareCode.Valid {
		c := s.ShareCode.String
		out.ShareCode = &c
	}
	if s.FinalizedAt.Valid {
		f := s.FinalizedAt.Time.UTC().Format(time.RFC3339)
		out.FinalizedAt = &f
	}
	return out
}

func dedupUint64(in []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(in))
	out := make([]uint64, 0, len(in))
	for _, v := range in {
		if _, hit := seen[v]; hit {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (s *Service) PatchCharge(ctx context.Context, hostID uint64, chargeID uint64, action string) (dbgen.GetChargeByIDForHostRow, error) {
	q := dbgen.New(s.db)

	// Tenant-isolated read
	charge, err := q.GetChargeByIDForHost(ctx, dbgen.GetChargeByIDForHostParams{
		ID:        chargeID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dbgen.GetChargeByIDForHostRow{}, ErrChargeNotFound
		}
		return dbgen.GetChargeByIDForHostRow{}, fmt.Errorf("get charge: %w", err)
	}

	switch action {
	case "confirm_paid":
		// Idempotent — if already paid, return as-is
		if charge.Status == "paid" {
			return charge, nil
		}
		err = q.UpdateChargeStatusManual(ctx, dbgen.UpdateChargeStatusManualParams{
			Status: "paid",
			PaidAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
			PaidVia: sql.NullString{String: "manual", Valid: true},
			ID:     chargeID,
		})
		if err != nil {
			return dbgen.GetChargeByIDForHostRow{}, fmt.Errorf("confirm paid: %w", err)
		}
		return q.GetChargeByIDForHost(ctx, dbgen.GetChargeByIDForHostParams{
			ID:        chargeID,
			HostUserID: hostID,
		})

	case "undo_paid":
		err = q.UpdateChargeStatusManual(ctx, dbgen.UpdateChargeStatusManualParams{
			Status: "unpaid",
			PaidAt: sql.NullTime{Valid: false},
			PaidVia: sql.NullString{Valid: false},
			ID:     chargeID,
		})
		if err != nil {
			return dbgen.GetChargeByIDForHostRow{}, fmt.Errorf("undo paid: %w", err)
		}
		return q.GetChargeByIDForHost(ctx, dbgen.GetChargeByIDForHostParams{
			ID:        chargeID,
			HostUserID: hostID,
		})

	default:
		return dbgen.GetChargeByIDForHostRow{}, fmt.Errorf("unknown action: %s", action)
	}
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
