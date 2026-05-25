package groups

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/datisekai/longthu.fun/backend/internal/audit"
	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

var (
	ErrNotFound    = errors.New("groups: not found")
	ErrNameMissing = errors.New("groups: name required")
)

// PublicGroup is the JSON shape returned by handlers — never leaks fields
// that aren't part of the read contract (e.g. archived_at is intentionally
// not surfaced until Story 4.1 needs it).
type PublicGroup struct {
	ID                   uint64  `json:"id"`
	Name                 string  `json:"name"`
	PrivacyMode          string  `json:"privacyMode"`
	AutoDetectEnabled    bool    `json:"autoDetectEnabled"`
	DefaultBankAccountID *uint64 `json:"defaultBankAccountId,omitempty"`
}

// CreateParams holds the validated input for Service.Create.
type CreateParams struct {
	Name string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// GetByID returns the Group iff host owns it. Returns ErrNotFound for both
// "doesn't exist" and "exists but other tenant" — no enumeration.
func (s *Service) GetByID(ctx context.Context, hostID uint64, groupID uint64) (PublicGroup, error) {
	q := dbgen.New(s.db)
	group, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         groupID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicGroup{}, ErrNotFound
		}
		return PublicGroup{}, fmt.Errorf("groups.GetByID: %w", err)
	}
	return toPublic(group), nil
}

// Create inserts a new Group + a `group_created` audit row in one transaction.
// Auto-picks the host's default bank as `default_bank_account_id` if one exists;
// leaves NULL otherwise (host can still create Group without a bank, but FR-12
// finalize-Session will block until a bank is added — staged copy
// `vi.onboarding.finalizeBlocked` covers this).
func (s *Service) Create(ctx context.Context, hostID uint64, params CreateParams) (PublicGroup, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return PublicGroup{}, ErrNameMissing
	}

	var newID int64
	err := dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		q := dbgen.New(tx)

		// Try to auto-pick the host's default bank. Missing bank is fine — NULL FK.
		var defaultBankID sql.NullInt64
		bank, err := q.GetDefaultBankAccountForHost(ctx, hostID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("groups.Create: lookup default bank: %w", err)
		}
		if err == nil {
			defaultBankID = sql.NullInt64{Int64: int64(bank.ID), Valid: true}
		}

		res, err := q.InsertGroup(ctx, dbgen.InsertGroupParams{
			HostUserID:           hostID,
			Name:                 name,
			DefaultBankAccountID: defaultBankID,
		})
		if err != nil {
			return fmt.Errorf("groups.Create: insert: %w", err)
		}
		newID, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("groups.Create: lastInsertId: %w", err)
		}

		// Audit row in the SAME transaction.
		hostIDInt := int64(hostID)
		return audit.Record(ctx, tx, audit.Event{
			Type:       audit.EventGroupCreated,
			ActorType:  audit.ActorHost,
			HostUserID: &hostIDInt,
			EntityType: "group",
			EntityID:   newID,
		})
	})
	if err != nil {
		return PublicGroup{}, err
	}

	// Re-read to return the persisted row (picks up DB defaults like
	// privacy_mode='public', auto_detect_enabled=0, generated timestamps).
	group, err := dbgen.New(s.db).GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         uint64(newID),
		HostUserID: hostID,
	})
	if err != nil {
		return PublicGroup{}, fmt.Errorf("groups.Create: re-read after insert: %w", err)
	}
	return toPublic(group), nil
}

func toPublic(g dbgen.GetGroupByIDForHostRow) PublicGroup {
	out := PublicGroup{
		ID:                g.ID,
		Name:              g.Name,
		PrivacyMode:       g.PrivacyMode,
		AutoDetectEnabled: g.AutoDetectEnabled,
	}
	if g.DefaultBankAccountID.Valid {
		bid := uint64(g.DefaultBankAccountID.Int64)
		out.DefaultBankAccountID = &bid
	}
	return out
}

func (s *Service) Update(ctx context.Context, hostID uint64, groupID uint64, name string) (PublicGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PublicGroup{}, ErrNameMissing
	}
	if len(name) > maxGroupNameLen {
		return PublicGroup{}, fmt.Errorf("name too long")
	}

	q := dbgen.New(s.db)

	// Verify ownership
	_, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         groupID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicGroup{}, ErrNotFound
		}
		return PublicGroup{}, fmt.Errorf("groups.Update: %w", err)
	}

	err = dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		q := dbgen.New(tx)
		return q.UpdateGroupName(ctx, dbgen.UpdateGroupNameParams{
			Name: name,
			ID:   groupID,
		})
	})
	if err != nil {
		return PublicGroup{}, fmt.Errorf("groups.Update: %w", err)
	}

	// Re-read
	group, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         groupID,
		HostUserID: hostID,
	})
	if err != nil {
		return PublicGroup{}, fmt.Errorf("groups.Update: re-read: %w", err)
	}
	return toPublic(group), nil
}

func (s *Service) Archive(ctx context.Context, hostID uint64, groupID uint64) error {
	q := dbgen.New(s.db)

	// Verify ownership
	_, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         groupID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("groups.Archive: %w", err)
	}

	return dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		q := dbgen.New(tx)
		return q.ArchiveGroup(ctx, groupID)
	})
}
