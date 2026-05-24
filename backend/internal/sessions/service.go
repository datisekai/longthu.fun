// Package sessions owns the Session draft lifecycle: create draft, add cost
// items, set participants. Finalize lives in Story 1.11 (separate
// transactional path that mints charges + share code).
package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

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
	ErrSessionNotFound          = errors.New("sessions: not found")
	ErrGroupNotFound            = errors.New("sessions: group not found for this host")
	ErrInvalidDate              = errors.New("sessions: invalid date")
	ErrInvalidCostItemType      = errors.New("sessions: invalid cost item type")
	ErrInvalidCostItemAmount    = errors.New("sessions: invalid cost item amount")
	ErrInvalidCostItemLabel     = errors.New("sessions: invalid cost item label")
	ErrParticipantsEmpty        = errors.New("sessions: at least one participant required")
	ErrParticipantOutsideRoster = errors.New("sessions: participant not in group roster")
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
