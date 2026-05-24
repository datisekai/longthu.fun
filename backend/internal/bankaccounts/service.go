package bankaccounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

type PublicBankAccount struct {
	ID                uint64 `json:"id"`
	BankName          string `json:"bankName"`
	BankCode          string `json:"bankCode"`
	AccountNumber     string `json:"accountNumber"`
	AccountHolderName string `json:"accountHolderName"`
	IsDefault         bool   `json:"isDefault"`
}

type CreateParams struct {
	BankCode          string
	AccountNumber     string
	AccountHolderName string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, hostID uint64, params CreateParams) (PublicBankAccount, error) {
	bankName := bankNameForCode(params.BankCode)
	if bankName == "" {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: unsupported bank code")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: begin tx: %w", err)
	}
	defer tx.Rollback()

	q := dbgen.New(tx)
	count, err := q.CountBankAccountsForHost(ctx, hostID)
	if err != nil {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: count: %w", err)
	}
	isDefault := count == 0

	accountNumber := strings.TrimSpace(params.AccountNumber)
	holderName := strings.TrimSpace(params.AccountHolderName)
	res, err := q.InsertBankAccount(ctx, dbgen.InsertBankAccountParams{
		UserID:            hostID,
		BankName:          bankName,
		BankCode:          params.BankCode,
		AccountNumber:     accountNumber,
		AccountHolderName: holderName,
		IsDefault:         isDefault,
	})
	if err != nil {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: lastInsertId: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return PublicBankAccount{}, fmt.Errorf("bankaccounts.Create: commit: %w", err)
	}

	return PublicBankAccount{
		ID:                uint64(id),
		BankName:          bankName,
		BankCode:          params.BankCode,
		AccountNumber:     accountNumber,
		AccountHolderName: holderName,
		IsDefault:         isDefault,
	}, nil
}

func bankNameForCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "MBBANK":
		return "MBBank"
	case "VCB":
		return "Vietcombank"
	case "TPB":
		return "TPBank"
	default:
		return ""
	}
}
