package autodetect

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"

	"github.com/datisekai/longthu.fun/backend/internal/crypto"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/webhooks/payos"
)

// PayOSCredentials stored encrypted in the database.
type PayOSCredentials struct {
	ClientID    string `json:"clientId"`
	APIKey      string `json:"apiKey"`
	ChecksumKey string `json:"checksumKey"`
}

// PayOSSupportedBanks contains bank codes that support payOS auto-detect.
var PayOSSupportedBanks = map[string]bool{
	"MBBB":   true, // MBBank
	"OCB":    true, // OCB
	"KLBANK": true, // KienlongBank
	"ACB":    true, // ACB
	"BIDV":   true, // BIDV
}

type Service struct {
	db        *sql.DB
	masterKey string
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db:        db,
		masterKey: os.Getenv("SECRETS_MASTER_KEY"),
	}
}

// GetGroupAutoDetect retrieves auto-detect settings for a group.
func (s *Service) GetGroupAutoDetect(ctx context.Context, hostID, groupID uint64) (*AutoDetectSettings, error) {
	q := dbgen.New(s.db)
	row, err := q.GetGroupWithAutoDetect(ctx, dbgen.GetGroupWithAutoDetectParams{
		ID:        groupID,
		HostUserID: hostID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	settings := &AutoDetectSettings{
		GroupID:    row.ID,
		HostUserID: row.HostUserID,
		GroupName:  row.Name,
		Enabled:    row.AutoDetectEnabled,
	}

	if row.AutoDetectBankAccountID.Valid {
		settings.BankAccountID = row.AutoDetectBankAccountID.Int64
	}
	if row.BankName.Valid {
		settings.BankName = row.BankName.String
	}
	if row.BankCode.Valid {
		settings.BankCode = row.BankCode.String
	}
	if row.AccountNumber.Valid {
		settings.AccountNumber = maskAccountNumber(row.AccountNumber.String)
	}
	if row.AccountHolderName.Valid {
		settings.AccountHolderName = row.AccountHolderName.String
	}

	if row.AutoDetectCredentialsJson.Valid && row.AutoDetectCredentialsJson.String != "" {
		settings.HasCredentials = true
		settings.CredentialsMasked = "***masked***"
	}

	return settings, nil
}

// EnableAutoDetect enables auto-detect for a group.
func (s *Service) EnableAutoDetect(ctx context.Context, hostID, groupID, bankAccountID uint64, credentials *PayOSCredentials) error {
	credsJSON, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(credsJSON, s.masterKey)
	if err != nil {
		return err
	}

	q := dbgen.New(s.db)
	return q.EnableAutoDetect(ctx, dbgen.EnableAutoDetectParams{
		AutoDetectBankAccountID:   sql.NullInt64{Int64: int64(bankAccountID), Valid: true},
		AutoDetectCredentialsJson: sql.NullString{String: string(encrypted), Valid: true},
		ID:                       groupID,
		HostUserID:               hostID,
	})
}

// DisableAutoDetect disables auto-detect for a group.
func (s *Service) DisableAutoDetect(ctx context.Context, hostID, groupID uint64) error {
	q := dbgen.New(s.db)
	return q.DisableAutoDetect(ctx, dbgen.DisableAutoDetectParams{
		ID:        groupID,
		HostUserID: hostID,
	})
}

// GetSupportedBankAccounts returns bank accounts that support payOS auto-detect.
func (s *Service) GetSupportedBankAccounts(ctx context.Context, hostID uint64) ([]BankAccount, error) {
	q := dbgen.New(s.db)
	rows, err := q.GetPayosSupportedBankAccounts(ctx, hostID)
	if err != nil {
		return nil, err
	}

	accounts := make([]BankAccount, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, BankAccount{
			ID:                row.ID,
			BankName:          row.BankName,
			BankCode:          row.BankCode,
			AccountNumber:     maskAccountNumber(row.AccountNumber),
			AccountHolderName: row.AccountHolderName,
			IsDefault:         row.IsDefault,
			IsPayOSSupported:  true,
		})
	}
	return accounts, nil
}

// GetAllBankAccounts returns all bank accounts for a host.
func (s *Service) GetAllBankAccounts(ctx context.Context, hostID uint64) ([]BankAccount, error) {
	q := dbgen.New(s.db)
	rows, err := q.GetGroupBankAccounts(ctx, hostID)
	if err != nil {
		return nil, err
	}

	accounts := make([]BankAccount, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, BankAccount{
			ID:                row.ID,
			BankName:          row.BankName,
			BankCode:          row.BankCode,
			AccountNumber:     maskAccountNumber(row.AccountNumber),
			AccountHolderName: row.AccountHolderName,
			IsDefault:         row.IsDefault,
			IsPayOSSupported:  PayOSSupportedBanks[row.BankCode],
		})
	}
	return accounts, nil
}

// TestCredentials validates payOS credentials.
func (s *Service) TestCredentials(ctx context.Context, credentials *PayOSCredentials) error {
	client, err := payos.NewClient(ctx, &payos.Config{
		ClientID:    credentials.ClientID,
		APIKey:      credentials.APIKey,
		ChecksumKey: credentials.ChecksumKey,
	})
	if err != nil {
		return err
	}

	_, err = client.CreatePaymentLink(ctx, payos.CreatePaymentLinkParams{
		OrderCode:   1000,
		Description: "LTPING01",
		Amount:      1000,
		ReturnURL:   "https://longthu.fun/test-return",
		CancelURL:   "https://longthu.fun/test-cancel",
	})
	return err
}

// GetPayOSClientForGroup returns a payOS client for a group.
func (s *Service) GetPayOSClientForGroup(ctx context.Context, groupID uint64) (*payos.Client, *PayOSCredentials, error) {
	q := dbgen.New(s.db)

	hostUserID, err := q.GetGroupHostUserID(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}

	row, err := q.GetGroupWithAutoDetect(ctx, dbgen.GetGroupWithAutoDetectParams{
		ID:        groupID,
		HostUserID: hostUserID,
	})
	if err != nil {
		return nil, nil, err
	}

	if !row.AutoDetectEnabled || !row.AutoDetectCredentialsJson.Valid || row.AutoDetectCredentialsJson.String == "" {
		return nil, nil, ErrNotEnabled
	}

	decrypted, err := crypto.Decrypt([]byte(row.AutoDetectCredentialsJson.String), s.masterKey)
	if err != nil {
		return nil, nil, err
	}

	var credentials PayOSCredentials
	if err := json.Unmarshal(decrypted, &credentials); err != nil {
		return nil, nil, err
	}

	client, err := payos.NewClient(ctx, &payos.Config{
		ClientID:    credentials.ClientID,
		APIKey:      credentials.APIKey,
		ChecksumKey: credentials.ChecksumKey,
	})
	if err != nil {
		return nil, nil, err
	}

	return client, &credentials, nil
}

func maskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return accountNumber
	}
	return "****" + accountNumber[len(accountNumber)-4:]
}

var (
	ErrNotFound   = errors.New("autodetect: not found")
	ErrNotEnabled = errors.New("autodetect: not enabled")
)

type AutoDetectSettings struct {
	GroupID           uint64 `json:"groupId"`
	HostUserID        uint64 `json:"hostUserId"`
	GroupName         string `json:"groupName"`
	Enabled           bool   `json:"enabled"`
	BankAccountID     int64  `json:"bankAccountId,omitempty"`
	BankName          string `json:"bankName,omitempty"`
	BankCode          string `json:"bankCode,omitempty"`
	AccountNumber     string `json:"accountNumber,omitempty"`
	AccountHolderName string `json:"accountHolderName,omitempty"`
	HasCredentials    bool   `json:"hasCredentials"`
	CredentialsMasked string `json:"credentialsMasked,omitempty"`
}

type BankAccount struct {
	ID                uint64 `json:"id"`
	BankName          string `json:"bankName"`
	BankCode          string `json:"bankCode"`
	AccountNumber     string `json:"accountNumber"`
	AccountHolderName string `json:"accountHolderName"`
	IsDefault         bool   `json:"isDefault"`
	IsPayOSSupported  bool   `json:"isPayOSSupported"`
}
