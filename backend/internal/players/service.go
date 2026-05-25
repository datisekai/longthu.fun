// Package players owns the host-scoped Player roster — bulk-add, rename,
// reset-code, mark-inactive. Story 1.9 ships BulkCreate; future stories
// extend.
package players

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/datisekai/longthu.fun/backend/internal/audit"
	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/shortcode"
)

// Tier caps come from PRD §4.2 FR-6. 0 = unlimited.
const (
	freeCap    = 8
	proCap     = 20
	proPlusCap = 0 // unlimited
)

// playerCodeLength is the per-Player short code length per FR-30.
const playerCodeLength = 8

// PublicPlayer is the response shape.
type PublicPlayer struct {
	ID          uint64 `json:"id"`
	GroupID     uint64 `json:"groupId"`
	DisplayName string `json:"displayName"`
	PublicCode  string `json:"publicCode"`
	IsActive    bool   `json:"isActive"`
}

// Typed errors returned by BulkCreate, rename, reset-code, mark-inactive. Mapped to HTTP status by the handler.
var (
	ErrGroupNotFound          = errors.New("players: group not found for this host")
	ErrNamesEmpty             = errors.New("players: at least one name required")
	ErrTierCapExceeded        = errors.New("players: tier cap exceeded")
	ErrDuplicateInSubmit      = errors.New("players: duplicate name in submission")
	ErrDuplicateAgainstRoster = errors.New("players: name already in group roster")
	ErrNotFound              = errors.New("players: not found")
)

// CapExceededDetail carries the cap + attempted total back to the handler.
type CapExceededDetail struct {
	Tier         string
	Cap          int
	CurrentCount int64
	ToAdd        int
}

// DuplicateDetail carries the offending names back to the handler.
type DuplicateDetail struct {
	Names []string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// BulkCreate atomically inserts `names` as Players of `groupID` (must be
// owned by `hostID`). Returns ErrGroupNotFound if host doesn't own the group,
// ErrTierCapExceeded if the result would exceed the tier cap (with a typed
// detail on the wrapped error), ErrDuplicateInSubmit / ErrDuplicateAgainstRoster
// for duplicate names. All-or-nothing.
func (s *Service) BulkCreate(ctx context.Context, hostID uint64, groupID uint64, names []string, tier string) ([]PublicPlayer, error) {
	cleaned := cleanNames(names)
	if len(cleaned) == 0 {
		return nil, ErrNamesEmpty
	}

	// Per-submit duplicate detection (case-insensitive, accent-insensitive
	// MySQL collation will catch the rest at INSERT time; we surface a friendlier
	// error here with the actual duplicate name).
	if dup := firstDuplicateInSubmit(cleaned); dup != "" {
		return nil, &TypedError{Inner: ErrDuplicateInSubmit, Dup: DuplicateDetail{Names: []string{dup}}}
	}

	cap := capForTier(tier)
	var inserted []PublicPlayer

	err := dbpkg.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		q := dbgen.New(tx)

		// Tenant + existence check (Story 1.6 enforcement pattern).
		if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
			ID:         groupID,
			HostUserID: hostID,
		}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrGroupNotFound
			}
			return fmt.Errorf("players.BulkCreate: group check: %w", err)
		}

		// Tier cap check (live count + new count ≤ cap; skip when cap is 0 = unlimited).
		currentCount, err := q.CountActivePlayersInGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("players.BulkCreate: count: %w", err)
		}
		if cap > 0 && int(currentCount)+len(cleaned) > cap {
			return &TypedError{
				Inner: ErrTierCapExceeded,
				Cap: CapExceededDetail{
					Tier:         tier,
					Cap:          cap,
					CurrentCount: currentCount,
					ToAdd:        len(cleaned),
				},
			}
		}

		// Duplicate-against-roster check (case-insensitive via Go side; final
		// safety net is the uk_players_group_display_name UNIQUE index).
		existing, err := q.ListPlayerDisplayNamesInGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("players.BulkCreate: list existing: %w", err)
		}
		if dup := firstDuplicateAgainstRoster(cleaned, existing); dup != "" {
			return &TypedError{Inner: ErrDuplicateAgainstRoster, Dup: DuplicateDetail{Names: []string{dup}}}
		}

		// Insert each Player with a unique public_code.
		hostIDInt := int64(hostID)
		for _, name := range cleaned {
			code, err := shortcode.GenerateUnique(ctx, playerCodeLength, shortcode.PlayerCode, func(ctx context.Context, candidate string) (bool, error) {
				return q.PlayerExistsByPublicCode(ctx, candidate)
			})
			if err != nil {
				return fmt.Errorf("players.BulkCreate: mint code: %w", err)
			}

			res, err := q.InsertPlayer(ctx, dbgen.InsertPlayerParams{
				GroupID:     groupID,
				DisplayName: name,
				PublicCode:  code,
			})
			if err != nil {
				return fmt.Errorf("players.BulkCreate: insert %q: %w", name, err)
			}
			newID, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("players.BulkCreate: lastInsertId: %w", err)
			}

			if err := audit.Record(ctx, tx, audit.Event{
				Type:       audit.EventPlayerAdded,
				ActorType:  audit.ActorHost,
				HostUserID: &hostIDInt,
				EntityType: "player",
				EntityID:   newID,
			}); err != nil {
				return fmt.Errorf("players.BulkCreate: audit: %w", err)
			}

			inserted = append(inserted, PublicPlayer{
				ID:          uint64(newID),
				GroupID:     groupID,
				DisplayName: name,
				PublicCode:  code,
				IsActive:    true,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return inserted, nil
}

// ListActive returns the active roster for a Group (after verifying host ownership).
// Story 1.10 uses this to populate the participant picker.
func (s *Service) ListActive(ctx context.Context, hostID uint64, groupID uint64) ([]PublicPlayer, error) {
	q := dbgen.New(s.db)
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: groupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("players.ListActive: group check: %w", err)
	}
	rows, err := q.ListActivePlayersInGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("players.ListActive: list: %w", err)
	}
	out := make([]PublicPlayer, 0, len(rows))
	for _, r := range rows {
		out = append(out, PublicPlayer{
			ID:          r.ID,
			GroupID:     r.GroupID,
			DisplayName: r.DisplayName,
			PublicCode:  r.PublicCode,
			IsActive:    r.IsActive,
		})
	}
	return out, nil
}

// TypedError carries handler-relevant detail attached to a sentinel error.
// Handler does `errors.As(err, &typed)` to unwrap.
type TypedError struct {
	Inner error
	Cap   CapExceededDetail
	Dup   DuplicateDetail
}

func (e *TypedError) Error() string { return e.Inner.Error() }
func (e *TypedError) Unwrap() error { return e.Inner }

// cleanNames trims each name + drops empty strings. Preserves original Vietnamese
// diacritics (no normalization).
func cleanNames(in []string) []string {
	out := make([]string, 0, len(in))
	for _, n := range in {
		t := strings.TrimSpace(n)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// firstDuplicateInSubmit returns the first name seen twice (case + accent
// insensitive via lowercase compare); empty if none.
func firstDuplicateInSubmit(names []string) string {
	seen := map[string]string{} // lower → original
	for _, n := range names {
		key := strings.ToLower(n)
		if orig, found := seen[key]; found {
			return orig
		}
		seen[key] = n
	}
	return ""
}

// firstDuplicateAgainstRoster checks `incoming` against `existing`; case-insensitive.
func firstDuplicateAgainstRoster(incoming, existing []string) string {
	existingSet := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		existingSet[strings.ToLower(e)] = struct{}{}
	}
	for _, n := range incoming {
		if _, hit := existingSet[strings.ToLower(n)]; hit {
			return n
		}
	}
	return ""
}

func capForTier(tier string) int {
	switch tier {
	case "pro":
		return proCap
	case "pro_plus":
		return proPlusCap // 0 = unlimited
	default:
		return freeCap
	}
}

// Update renames a player.
func (s *Service) Update(ctx context.Context, hostID uint64, groupID uint64, playerID uint64, newName string) (PublicPlayer, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return PublicPlayer{}, ErrNamesEmpty
	}

	q := dbgen.New(s.db)

	// Verify host owns group and player belongs to group
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: groupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicPlayer{}, ErrNotFound
		}
		return PublicPlayer{}, fmt.Errorf("players.Update: group: %w", err)
	}

	player, err := q.GetPlayerByIDForGroup(ctx, dbgen.GetPlayerByIDForGroupParams{
		ID: playerID, GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicPlayer{}, ErrNotFound
		}
		return PublicPlayer{}, fmt.Errorf("players.Update: player: %w", err)
	}

	// Check for duplicate name in group
	existing, err := q.ListPlayerDisplayNamesInGroup(ctx, groupID)
	if err != nil {
		return PublicPlayer{}, fmt.Errorf("players.Update: list names: %w", err)
	}
	// Exclude current player from duplicate check
	for _, name := range existing {
		if strings.EqualFold(name, newName) && name != player.DisplayName {
			return PublicPlayer{}, ErrDuplicateAgainstRoster
		}
	}

	err = q.UpdatePlayerName(ctx, dbgen.UpdatePlayerNameParams{
		DisplayName: newName,
		ID:          playerID,
	})
	if err != nil {
		return PublicPlayer{}, fmt.Errorf("players.Update: update: %w", err)
	}

	return PublicPlayer{
		ID:          player.ID,
		GroupID:     player.GroupID,
		DisplayName: newName,
		PublicCode:  player.PublicCode,
		IsActive:    player.IsActive,
	}, nil
}

// Deactivate marks a player as inactive.
func (s *Service) Deactivate(ctx context.Context, hostID uint64, groupID uint64, playerID uint64) error {
	q := dbgen.New(s.db)

	// Verify ownership
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: groupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("players.Deactivate: group: %w", err)
	}

	_, err := q.GetPlayerByIDForGroup(ctx, dbgen.GetPlayerByIDForGroupParams{
		ID: playerID, GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("players.Deactivate: player: %w", err)
	}

	return q.UpdatePlayerActive(ctx, dbgen.UpdatePlayerActiveParams{
		IsActive: false,
		ID:       playerID,
	})
}

// Reactivate marks a player as active.
func (s *Service) Reactivate(ctx context.Context, hostID uint64, groupID uint64, playerID uint64) error {
	q := dbgen.New(s.db)

	// Verify ownership
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: groupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("players.Reactivate: group: %w", err)
	}

	_, err := q.GetPlayerByIDForGroup(ctx, dbgen.GetPlayerByIDForGroupParams{
		ID: playerID, GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("players.Reactivate: player: %w", err)
	}

	return q.UpdatePlayerActive(ctx, dbgen.UpdatePlayerActiveParams{
		IsActive: true,
		ID:       playerID,
	})
}

// ResetCode generates a new public_code for a player.
func (s *Service) ResetCode(ctx context.Context, hostID uint64, groupID uint64, playerID uint64) (PublicPlayer, error) {
	q := dbgen.New(s.db)

	// Verify ownership
	if _, err := q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID: groupID, HostUserID: hostID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicPlayer{}, ErrNotFound
		}
		return PublicPlayer{}, fmt.Errorf("players.ResetCode: group: %w", err)
	}

	player, err := q.GetPlayerByIDForGroup(ctx, dbgen.GetPlayerByIDForGroupParams{
		ID: playerID, GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicPlayer{}, ErrNotFound
		}
		return PublicPlayer{}, fmt.Errorf("players.ResetCode: player: %w", err)
	}

	// Generate new unique code
	newCode, err := shortcode.GenerateUnique(ctx, playerCodeLength, shortcode.PlayerCode, func(ctx context.Context, candidate string) (bool, error) {
		return q.PlayerExistsByPublicCode(ctx, candidate)
	})
	if err != nil {
		return PublicPlayer{}, fmt.Errorf("players.ResetCode: generate code: %w", err)
	}

	// Audit the reset
	hostIDInt := int64(hostID)
	_ = audit.Record(ctx, nil, audit.Event{
		Type:       "player_code_reset",
		ActorType:  audit.ActorHost,
		HostUserID: &hostIDInt,
		EntityType: "player",
		EntityID:   int64(playerID),
	})

	err = q.UpdatePlayerPublicCode(ctx, dbgen.UpdatePlayerPublicCodeParams{
		PublicCode: newCode,
		ID:         playerID,
	})
	if err != nil {
		return PublicPlayer{}, fmt.Errorf("players.ResetCode: update: %w", err)
	}

	return PublicPlayer{
		ID:          player.ID,
		GroupID:     player.GroupID,
		DisplayName: player.DisplayName,
		PublicCode:  newCode,
		IsActive:    player.IsActive,
	}, nil
}
