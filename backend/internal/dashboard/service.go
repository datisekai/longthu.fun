package dashboard

import (
	"context"
	"database/sql"

	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type Dashboard struct {
	TotalUnpaid       int64              `json:"totalUnpaid"`
	RecentSessions    []DashboardSession `json:"recentSessions"`
	PlayersWithUnpaid []PlayerUnpaid    `json:"playersWithUnpaid"`
	GroupCount       int                `json:"groupCount"`
	SessionCount     int                `json:"sessionCount"`
	SuspectedCount   int                `json:"suspectedCount"`
	UnmatchedCount   int                `json:"unmatchedCount"`
}

type DashboardSession struct {
	SessionID  uint64  `json:"sessionId"`
	Date      string  `json:"date"`
	Title     *string `json:"title,omitempty"`
	GroupID   uint64  `json:"groupId"`
	GroupName string  `json:"groupName"`
	ShareCode *string `json:"shareCode,omitempty"`
	TotalCost int64   `json:"totalCost"`
	FinalizedAt string `json:"finalizedAt"`
}

type PlayerUnpaid struct {
	PlayerID    uint64 `json:"playerId"`
	PlayerName  string `json:"playerName"`
	GroupID     uint64 `json:"groupId"`
	GroupName   string `json:"groupName"`
	TotalUnpaid int64  `json:"totalUnpaid"`
}

func (s *Service) GetDashboard(ctx context.Context, hostID uint64) (*Dashboard, error) {
	q := dbgen.New(s.db)

	// Total unpaid - COALESCE returns interface{}
	var totalUnpaid int64
	result, err := q.GetDashboard(ctx, hostID)
	if err == nil {
		if v, ok := result.(int64); ok {
			totalUnpaid = v
		} else if v, ok := result.([]uint8); ok {
			// MySQL returns numbers as []byte sometimes
			totalUnpaid = 0
			for _, b := range v {
				totalUnpaid = totalUnpaid*10 + int64(b-'0')
			}
		}
	}

	// Recent sessions
	sessions, err := q.ListRecentSessionsForHost(ctx, hostID)
	if err != nil {
		sessions = nil
	}
	dashSessions := make([]DashboardSession, 0, len(sessions))
	for _, sess := range sessions {
		d := DashboardSession{
			SessionID:  sess.SessionID,
			Date:       sess.SessionDate.Format("02/01/2006"),
			GroupID:    sess.GroupID,
			GroupName:  sess.GroupName,
			TotalCost:  sess.TotalCost,
		}
		if sess.SessionTitle.Valid {
			d.Title = &sess.SessionTitle.String
		}
		if sess.ShareCode.Valid {
			d.ShareCode = &sess.ShareCode.String
		}
		if sess.FinalizedAt.Valid {
			d.FinalizedAt = sess.FinalizedAt.Time.Format("02/01/2006 15:04")
		}
		dashSessions = append(dashSessions, d)
	}

	// Players with unpaid
	players, err := q.ListPlayersWithUnpaidForHost(ctx, hostID)
	if err != nil {
		players = nil
	}
	playerUnpaids := make([]PlayerUnpaid, 0, len(players))
	for _, p := range players {
		var unpaid int64
		if v, ok := p.TotalUnpaid.(int64); ok {
			unpaid = v
		}
		playerUnpaids = append(playerUnpaids, PlayerUnpaid{
			PlayerID:    p.PlayerID,
			PlayerName:  p.PlayerName,
			GroupID:     p.GroupID,
			GroupName:   p.GroupName,
			TotalUnpaid: unpaid,
		})
	}

	// Counts
	groupCount, _ := q.CountGroupsForHost(ctx, hostID)
	sessionCount, _ := q.CountFinalizedSessionsForHost(ctx, hostID)

	// Suspected and unmatched counts (Story 6.5)
	suspectedCount, _ := q.CountSuspectedPaymentsForHost(ctx, sql.NullInt64{Int64: int64(hostID), Valid: true})
	unmatchedCount, _ := q.CountUnmatchedPaymentsForHost(ctx, sql.NullInt64{Int64: int64(hostID), Valid: true})

	return &Dashboard{
		TotalUnpaid:       totalUnpaid,
		RecentSessions:    dashSessions,
		PlayersWithUnpaid: playerUnpaids,
		GroupCount:       int(groupCount),
		SessionCount:     int(sessionCount),
		SuspectedCount:   int(suspectedCount),
		UnmatchedCount:   int(unmatchedCount),
	}, nil
}
