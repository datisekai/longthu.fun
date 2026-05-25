package public

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/httpx"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/group-bill/:shareCode", h.handleGroupBill)
	r.GET("/player-ledger/:playerCode", h.handlePlayerLedger)
}

// PublicGroupBill is the response shape for a public group bill page.
// Excludes host PII (email, bank holder name beyond what's needed for QR).
type PublicGroupBill struct {
	Session      PublicSessionInfo `json:"session"`
	Players     []PlayerRow      `json:"players"`
	Summary     BillSummary      `json:"summary"`
	PrivacyMode string           `json:"privacyMode"`
}

type PublicSessionInfo struct {
	ID        uint64  `json:"id"`
	Date      string  `json:"date"`
	Title     *string `json:"title,omitempty"`
	TotalCost int64   `json:"totalCost"`
}

type PlayerRow struct {
	PlayerID     uint64  `json:"playerId"`
	DisplayName string  `json:"displayName"`
	PublicCode  string  `json:"publicCode"`
	ChargeAmount int64   `json:"chargeAmount"`
	ChargeStatus string `json:"chargeStatus"`
	CrossDebt   *int64  `json:"crossDebt,omitempty"`
}

type BillSummary struct {
	TotalCost   int64 `json:"totalCost"`
	TotalPaid  int64 `json:"totalPaid"`
	TotalUnpaid int64 `json:"totalUnpaid"`
}

func (h *Handler) handleGroupBill(c *gin.Context) {
	code := strings.TrimSpace(c.Param("shareCode"))
	if code == "" {
		httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		return
	}

	q := dbgen.New(h.db)
	bill, err := q.GetPublicGroupBill(c.Request.Context(), sql.NullString{String: code, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	charges, err := q.ListPublicSessionCharges(c.Request.Context(), bill.SessionID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	playerRows := make([]PlayerRow, 0, len(charges))
	var totalPaid, totalUnpaid int64

	for _, ch := range charges {
		row := PlayerRow{
			PlayerID:      ch.PlayerID,
			DisplayName:  ch.DisplayName,
			ChargeAmount: ch.ChargeAmount,
			ChargeStatus: ch.ChargeStatus,
		}
		player, err := q.GetPlayerByID(c.Request.Context(), ch.PlayerID)
		if err == nil {
			row.PublicCode = player.PublicCode
		}

		if bill.GroupPrivacyMode == "public" {
			debt, err := q.GetPlayerCrossSessionDebt(c.Request.Context(), ch.PlayerID)
			if err == nil {
				if v, ok := debt.(int64); ok && v > 0 {
					row.CrossDebt = &v
				}
			}
		}

		if ch.ChargeStatus == "paid" {
			totalPaid += ch.ChargeAmount
		} else {
			totalUnpaid += ch.ChargeAmount
		}

		playerRows = append(playerRows, row)
	}

	dateStr := bill.SessionDate.Format("02/01/2006")
	var title *string
	if bill.SessionTitle.Valid {
		title = &bill.SessionTitle.String
	}

	c.JSON(http.StatusOK, PublicGroupBill{
		Session: PublicSessionInfo{
			ID:        bill.SessionID,
			Date:      dateStr,
			Title:     title,
			TotalCost: bill.SessionTotalCost,
		},
		Players:     playerRows,
		Summary: BillSummary{
			TotalCost:   bill.SessionTotalCost,
			TotalPaid:  totalPaid,
			TotalUnpaid: totalUnpaid,
		},
		PrivacyMode: bill.GroupPrivacyMode,
	})
}

// PlayerLedger response for /p/:playerCode
type PlayerLedger struct {
	Player         PlayerInfo         `json:"player"`
	CurrentCharge *CurrentChargeInfo  `json:"currentCharge,omitempty"`
	Charges       []LedgerCharge     `json:"charges"`
	Summary       LedgerSummary      `json:"summary"`
}

type PlayerInfo struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"displayName"`
	PublicCode string `json:"publicCode"`
}

type CurrentChargeInfo struct {
	SessionID   uint64 `json:"sessionId"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	SessionDate string `json:"sessionDate"`
	SessionTitle *string `json:"sessionTitle,omitempty"`
}

type LedgerCharge struct {
	ID          uint64  `json:"id"`
	SessionID  uint64  `json:"sessionId"`
	Amount     int64   `json:"amount"`
	Status     string  `json:"status"`
	PaidAt     *string `json:"paidAt,omitempty"`
	SessionDate string `json:"sessionDate"`
	SessionTitle *string `json:"sessionTitle,omitempty"`
	GroupName  string  `json:"groupName"`
}

type LedgerSummary struct {
	TotalUnpaid int64 `json:"totalUnpaid"`
	TotalPaid  int64 `json:"totalPaid"`
	HasUnpaid  bool  `json:"hasUnpaid"`
}

func (h *Handler) handlePlayerLedger(c *gin.Context) {
	code := strings.TrimSpace(c.Param("playerCode"))
	if code == "" {
		httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
		return
	}

	q := dbgen.New(h.db)
	player, err := q.GetPlayerByPublicCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Reply(c, http.StatusNotFound, "Link không hợp lệ 🏸", "")
			return
		}
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	charges, err := q.ListPlayerChargesForLedger(c.Request.Context(), player.ID)
	if err != nil {
		httpx.Reply(c, http.StatusInternalServerError, "Lỗi server", "")
		return
	}

	var totalPaid, totalUnpaid int64
	ledgerCharges := make([]LedgerCharge, 0, len(charges))
	var currentCharge *CurrentChargeInfo

	for i, ch := range charges {
		dateStr := ch.SessionDate.Format("02/01/2006")
		var title *string
		if ch.SessionTitle.Valid {
			title = &ch.SessionTitle.String
		}
		
		charge := LedgerCharge{
			ID:          ch.ChargeID,
			SessionID:  ch.SessionID,
			Amount:     ch.Amount,
			Status:     ch.Status,
			SessionDate: dateStr,
			SessionTitle: title,
			GroupName:  ch.GroupName,
		}
		
		if ch.PaidAt.Valid {
			paidStr := ch.PaidAt.Time.Format("02/01/2006")
			charge.PaidAt = &paidStr
			totalPaid += ch.Amount
		} else {
			totalUnpaid += ch.Amount
		}

		ledgerCharges = append(ledgerCharges, charge)

		// Current charge = first unpaid charge (most recent due to ORDER BY date DESC)
		if currentCharge == nil && ch.Status != "paid" && ch.Status != "waived" {
			currentCharge = &CurrentChargeInfo{
				SessionID:   ch.SessionID,
				Amount:     ch.Amount,
				Status:     ch.Status,
				SessionDate: dateStr,
				SessionTitle: title,
			}
		}

		_ = i // avoid unused variable
	}

	c.JSON(http.StatusOK, PlayerLedger{
		Player: PlayerInfo{
			ID:          player.ID,
			DisplayName: player.DisplayName,
			PublicCode: player.PublicCode,
		},
		CurrentCharge: currentCharge,
		Charges:      ledgerCharges,
		Summary: LedgerSummary{
			TotalUnpaid: totalUnpaid,
			TotalPaid:  totalPaid,
			HasUnpaid:  totalUnpaid > 0,
		},
	})
}
