// Package payos wraps the official payOS Go SDK with typed interfaces
// for webhook verification and payment link creation.
package payos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/payOSHQ/payos-lib-golang/v2"
)

// ErrMissingSignature is returned when the webhook payload is missing the
// signature field entirely.
var ErrMissingSignature = errors.New("payos: webhook payload missing signature field")

// ErrSignatureInvalid is returned when the webhook signature verification fails.
var ErrSignatureInvalid = errors.New("payos: webhook signature verification failed")

// Config holds the three credentials required to instantiate a payOS client.
// These should come from environment variables, never hardcoded.
type Config struct {
	ClientID    string
	APIKey      string
	ChecksumKey string
}

// NewClient creates a new payOS client from the given config.
// Returns an error if any credential is empty.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("payos: PAYOS_CLIENT_ID is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("payos: PAYOS_API_KEY is required")
	}
	if cfg.ChecksumKey == "" {
		return nil, errors.New("payos: PAYOS_CHECKSUM_KEY is required")
	}

	sdk, err := payos.NewPayOS(&payos.PayOSOptions{
		ClientId:    cfg.ClientID,
		ApiKey:      cfg.APIKey,
		ChecksumKey: cfg.ChecksumKey,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		sdk:         sdk,
		checksumKey: cfg.ChecksumKey,
	}, nil
}

// Client is a thin wrapper around the official payOS Go SDK.
// It provides typed interfaces for webhook verification and payment link creation.
type Client struct {
	sdk         *payos.PayOS
	checksumKey string
}

// WebhookData represents the verified data extracted from a payOS webhook payload.
// This is the inner "data" object from the webhook POST body.
type WebhookData struct {
	OrderCode            int64   `json:"orderCode"`
	Amount               int     `json:"amount"`
	Description          string  `json:"description"`
	AccountNumber        string  `json:"accountNumber"`
	Reference            string  `json:"reference"`
	TransactionDateTime  string  `json:"transactionDateTime"`
	Currency             string  `json:"currency"`
	PaymentLinkId        string  `json:"paymentLinkId"`
	Code                 string  `json:"code"`
	Desc                 string  `json:"desc"`
	CounterAccountBankId   *string `json:"counterAccountBankId"`
	CounterAccountBankName *string `json:"counterAccountBankName"`
	CounterAccountName     *string `json:"counterAccountName"`
	CounterAccountNumber   *string `json:"counterAccountNumber"`
	VirtualAccountName     *string `json:"virtualAccountName"`
	VirtualAccountNumber   *string `json:"virtualAccountNumber"`
}

// RawWebhook represents the full webhook POST body from payOS.
type RawWebhook struct {
	Code      string       `json:"code"`
	Desc      string       `json:"desc"`
	Success   bool         `json:"success"`
	Data      *WebhookData `json:"data"`
	Signature string       `json:"signature"`
}

// VerifyWebhook parses and verifies the HMAC-SHA256 signature of a raw webhook payload.
// Returns the verified WebhookData if the signature is valid.
// Returns ErrMissingSignature if the payload has no signature field.
// Returns ErrSignatureInvalid if the signature does not match.
// Returns a JSON parse error if the payload is not valid JSON.
func (c *Client) VerifyWebhook(ctx context.Context, rawJSON []byte) (*WebhookData, error) {
	// Parse the raw JSON to check for signature field and extract data.
	var webhook RawWebhook
	if err := json.Unmarshal(rawJSON, &webhook); err != nil {
		return nil, fmt.Errorf("payos: failed to parse webhook JSON: %w", err)
	}

	// Check for missing signature.
	if webhook.Signature == "" {
		return nil, ErrMissingSignature
	}

	// Parse the raw JSON as map[string]interface{} for SDK verification.
	var webhookBody map[string]interface{}
	if err := json.Unmarshal(rawJSON, &webhookBody); err != nil {
		return nil, fmt.Errorf("payos: failed to parse webhook body: %w", err)
	}

	// Delegate to the SDK's built-in verifier.
	// The SDK handles the sorted-key=value&... HMAC-SHA256 serialization internally.
	result, err := c.sdk.Webhooks.VerifyData(ctx, webhookBody)
	if err != nil {
		return nil, ErrSignatureInvalid
	}

	// Type assert the result to our struct.
	// The SDK returns the data object from the webhook.
	dataMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payos: unexpected webhook data type")
	}

	dataBytes, err := json.Marshal(dataMap)
	if err != nil {
		return nil, fmt.Errorf("payos: failed to marshal webhook data: %w", err)
	}

	var data WebhookData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, fmt.Errorf("payos: failed to unmarshal webhook data: %w", err)
	}

	return &data, nil
}

// CreatePaymentLinkParams holds the parameters for creating a payOS payment link.
// Amount is in VND (smallest unit, no decimals).
type CreatePaymentLinkParams struct {
	OrderCode   int64  // Unique order code (use timestamp + random)
	Description string // Transfer content (max 9 chars for non-linked accounts)
	Amount      int    // Amount in VND
	ReturnURL   string
	CancelURL   string
}

// PaymentLink represents a created payOS payment link.
type PaymentLink struct {
	Bin           string `json:"bin"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
	Amount        int    `json:"amount"`
	Description   string `json:"description"`
	OrderCode     int64  `json:"orderCode"`
	Currency      string `json:"currency"`
	PaymentLinkId string `json:"paymentLinkId"`
	Status        string `json:"status"`
	CheckoutUrl   string `json:"checkoutUrl"`
	QrCode        string `json:"qrCode"`
}

// CreatePaymentLink creates a new payOS payment link (Payment Intent) and
// returns the payment link details.
func (c *Client) CreatePaymentLink(ctx context.Context, params CreatePaymentLinkParams) (*PaymentLink, error) {
	resp, err := c.sdk.PaymentRequests.Create(ctx, payos.CreatePaymentLinkRequest{
		OrderCode:   params.OrderCode,
		Amount:      params.Amount,
		Description: params.Description,
		ReturnUrl:   params.ReturnURL,
		CancelUrl:   params.CancelURL,
	})
	if err != nil {
		return nil, fmt.Errorf("payos: failed to create payment link: %w", err)
	}

	return &PaymentLink{
		Bin:           resp.Bin,
		AccountNumber: resp.AccountNumber,
		AccountName:   resp.AccountName,
		Amount:        resp.Amount,
		Description:   resp.Description,
		OrderCode:     resp.OrderCode,
		Currency:      resp.Currency,
		PaymentLinkId: resp.PaymentLinkId,
		Status:        string(resp.Status),
		CheckoutUrl:   resp.CheckoutUrl,
		QrCode:        resp.QrCode,
	}, nil
}

// CancelPaymentLink cancels an existing payment link by its order code.
func (c *Client) CancelPaymentLink(ctx context.Context, orderCode int64) error {
	_, err := c.sdk.PaymentRequests.Cancel(ctx, orderCode, nil)
	return err
}

// ConfirmWebhookURL confirms a webhook URL with payOS.
func (c *Client) ConfirmWebhookURL(ctx context.Context, webhookURL string) (string, error) {
	return c.sdk.Webhooks.Confirm(ctx, webhookURL)
}

// EnvConfig reads payOS credentials from environment variables.
// Use this for standard application startup.
func EnvConfig() *Config {
	return &Config{
		ClientID:    os.Getenv("PAYOS_CLIENT_ID"),
		APIKey:      os.Getenv("PAYOS_API_KEY"),
		ChecksumKey: os.Getenv("PAYOS_CHECKSUM_KEY"),
	}
}
