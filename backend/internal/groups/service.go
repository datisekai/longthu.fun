package groups

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

var ErrNotFound = errors.New("groups: not found")

type PublicGroup struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	PrivacyMode string `json:"privacyMode"`
}

type Service struct {
	q *dbgen.Queries
}

func NewService(db *sql.DB) *Service {
	return &Service{q: dbgen.New(db)}
}

func (s *Service) GetByID(ctx context.Context, hostID uint64, groupID uint64) (PublicGroup, error) {
	group, err := s.q.GetGroupByIDForHost(ctx, dbgen.GetGroupByIDForHostParams{
		ID:         groupID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicGroup{}, ErrNotFound
		}
		return PublicGroup{}, fmt.Errorf("groups.GetByID: %w", err)
	}
	return PublicGroup{
		ID:          group.ID,
		Name:        group.Name,
		PrivacyMode: group.PrivacyMode,
	}, nil
}
