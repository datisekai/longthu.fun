package paymentintents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/shortcode"
)

var (
	ErrNoUnpaidCharges  = errors.New("paymentintents: no unpaid charges found")
	ErrPlayerNotFound   = errors.New("paymentintents: player not found")
	ErrIntentNotFound   = errors.New("paymentintents: intent not found")
	ErrIntentExpired    = errors.New("paymentintents: intent expired")
	ErrIntentCancelled  = errors.New("paymentintents: intent cancelled")
)

const (
	intentCodeLength   = 6
	intentExpiryHours = 24
	intentStatusPending   = "pending"
	intentStatusMatched   = "matched"
	intentStatusExpired   = "expired"
	intentStatusCancelled = "cancelled"
)

// Mode determines what charges an intent covers.
type IntentMode string

const (
	ModeCurrentSession IntentMode = "current_session"
	ModeAllUnpaid     IntentMode = "all_unpaid"
)

// CreateParams for building a new PaymentIntent.
type CreateParams struct {
	PlayerCode string     // public_code from URL
	Mode      IntentMode
	SessionID *uint64 // optional for mode=current_session
}

// PaymentIntent is the domain model.
type PaymentIntent struct {
	ID            uint64
	PlayerID     uint64
	GroupID      uint64
	SessionID    *uint64
	Amount       int64
	Code         string
	Status       string
	Provider     string
	CoversChargeIDs []uint64
	ExpiresAt    time.Time
}

// PublicIntent is the JSON-safe response shape.
type PublicIntent struct {
	Code          string `json:"code"`
	Amount       int64  `json:"amount"`
	Status       string `json:"status"`
	TransferContent string `json:"transferContent"`
	ExpiresAt    string `json:"expiresAt"`
	BankInfo     BankInfo `json:"bankInfo"`
}

type BankInfo struct {
	BankName       string `json:"bankName"`
	BankCode       string `json:"bankCode"`
	AccountNumber  string `json:"accountNumber"`
	AccountHolder  string `json:"accountHolder"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*PublicIntent, error) {
	q := dbgen.New(s.db)

	// Get player by public_code
	player, err := q.GetPlayerByPublicCode(ctx, p.PlayerCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}
		return nil, fmt.Errorf("get player: %w", err)
	}

	var chargeIDs []uint64
	var amount int64
	var sessionID *uint64

	switch p.Mode {
	case ModeCurrentSession:
		// Get latest finalized session for the player's group
		session, err := q.GetLatestFinalizedSessionForGroup(ctx, player.GroupID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrNoUnpaidCharges
			}
			return nil, fmt.Errorf("get session: %w", err)
		}
		sessionID = &session.ID

		// Get unpaid charges for this session + player
		charges, err := q.ListUnpaidChargesForPlayerSession(ctx, dbgen.ListUnpaidChargesForPlayerSessionParams{
			PlayerID:   player.ID,
			SessionID: session.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("get charges: %w", err)
		}
		if len(charges) == 0 {
			return nil, ErrNoUnpaidCharges
		}
		for _, ch := range charges {
			chargeIDs = append(chargeIDs, ch.ID)
			amount += ch.Amount
		}

	case ModeAllUnpaid:
		// Get all unpaid charges for player across all sessions
		charges, err := q.ListAllUnpaidChargesForPlayer(ctx, player.ID)
		if err != nil {
			return nil, fmt.Errorf("get all charges: %w", err)
		}
		if len(charges) == 0 {
			return nil, ErrNoUnpaidCharges
		}
		for _, ch := range charges {
			chargeIDs = append(chargeIDs, ch.ID)
			amount += ch.Amount
		}
	}

	if amount <= 0 {
		return nil, ErrNoUnpaidCharges
	}

	// Cancel any existing pending intents for this player
	existing, err := q.ListPendingIntentsForPlayer(ctx, player.ID)
	if err == nil {
		for _, intent := range existing {
			_ = q.CancelPaymentIntent(ctx, dbgen.CancelPaymentIntentParams{
				ID:     intent.ID,
				Status: intentStatusCancelled,
			})
		}
	}

	// Mint unique intent code
	intentCode, err := shortcode.GenerateUnique(ctx, intentCodeLength, shortcode.PaymentIntent, func(ctx context.Context, candidate string) (bool, error) {
		return q.PaymentIntentCodeExists(ctx, candidate)
	})
	if err != nil {
		return nil, fmt.Errorf("mint code: %w", err)
	}

	coversJSON, _ := json.Marshal(chargeIDs)
	expiresAt := time.Now().UTC().Add(intentExpiryHours * time.Hour)

	// Get default bank account for the group
	var bankName, bankCode, accountNumber, accountHolder string
	bank, err := q.GetDefaultBankAccountByGroup(ctx, player.GroupID)
	if err == nil {
		bankName = bank.BankName
		bankCode = bank.BankCode
		accountNumber = bank.AccountNumber
		accountHolder = bank.AccountHolderName
	}

	// Insert payment intent
	_, err = q.CreatePaymentIntent(ctx, dbgen.CreatePaymentIntentParams{
		PlayerID:            player.ID,
		GroupID:             player.GroupID,
		SessionID:          sql.NullInt64{Int64: int64(ptrToUint(sessionID)), Valid: sessionID != nil},
		Amount:             amount,
		Code:               intentCode,
		Status:             intentStatusPending,
		Provider:           "manual", // manual until payOS integrates
		CoversChargeIdsJson: coversJSON,
		ExpiresAt:          sql.NullTime{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create intent: %w", err)
	}

	// Transfer content = LT prefix + code
	transferContent := "LT" + intentCode

	return &PublicIntent{
		Code:            intentCode,
		Amount:          amount,
		Status:          intentStatusPending,
		TransferContent: transferContent,
		ExpiresAt:       expiresAt.Format(time.RFC3339),
		BankInfo: BankInfo{
			BankName:      bankName,
			BankCode:      bankCode,
			AccountNumber: accountNumber,
			AccountHolder: accountHolder,
		},
	}, nil
}

func ptrToUint(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}

func (s *Service) GetByCode(ctx context.Context, code string) (*PublicIntent, error) {
	q := dbgen.New(s.db)
	intent, err := q.GetPaymentIntentByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIntentNotFound
		}
		return nil, fmt.Errorf("get intent: %w", err)
	}

	// Check if expired
	if intent.ExpiresAt.Valid && intent.ExpiresAt.Time.Before(time.Now().UTC()) {
		_ = q.UpdatePaymentIntentStatus(ctx, dbgen.UpdatePaymentIntentStatusParams{
			ID: intent.ID,
			Status: intentStatusExpired,
		})
		return nil, ErrIntentExpired
	}

	// Get bank info
	var bankName, bankCode, accountNumber, accountHolder string
	if intent.GroupID != 0 {
		bank, err := q.GetDefaultBankAccountByGroup(ctx, intent.GroupID)
		if err == nil {
			bankName = bank.BankName
			bankCode = bank.BankCode
			accountNumber = bank.AccountNumber
			accountHolder = bank.AccountHolderName
		}
	}

	return &PublicIntent{
		Code:            intent.Code,
		Amount:          intent.Amount,
		Status:          intent.Status,
		TransferContent: "LT" + intent.Code,
		ExpiresAt:       intent.ExpiresAt.Time.Format(time.RFC3339),
		BankInfo: BankInfo{
			BankName:      bankName,
			BankCode:      bankCode,
			AccountNumber: accountNumber,
			AccountHolder: accountHolder,
		},
	}, nil
}

func (s *Service) MarkTransferred(ctx context.Context, code string) error {
	q := dbgen.New(s.db)
	intent, err := q.GetPaymentIntentByCode(ctx, code)
	if err != nil {
		return ErrIntentNotFound
	}

	if intent.Status != intentStatusPending {
		return fmt.Errorf("cannot mark transferred: status is %s", intent.Status)
	}

	// Mark charges as pending_confirmation
	var chargeIDs []uint64
	_ = json.Unmarshal(intent.CoversChargeIdsJson, &chargeIDs)

	for _, id := range chargeIDs {
		_ = q.UpdateChargeStatus(ctx, dbgen.UpdateChargeStatusParams{
			Status: "pending_confirmation",
			ID:     id,
		})
	}

	return nil
}
