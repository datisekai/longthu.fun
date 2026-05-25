package payos_test

import (
	"testing"

	"github.com/datisekai/longthu.fun/backend/internal/webhooks/payos"
)

// TestConfigValidation tests that the payOS client validates required credentials.
func TestConfigValidation(t *testing.T) {
	t.Run("empty ClientID", func(t *testing.T) {
		_, err := payos.NewClient(nil, &payos.Config{
			ClientID:    "",
			APIKey:      "test-api-key",
			ChecksumKey: "test-checksum",
		})
		if err == nil {
			t.Error("expected error for empty ClientID")
		}
	})

	t.Run("empty APIKey", func(t *testing.T) {
		_, err := payos.NewClient(nil, &payos.Config{
			ClientID:    "test-client-id",
			APIKey:      "",
			ChecksumKey: "test-checksum",
		})
		if err == nil {
			t.Error("expected error for empty APIKey")
		}
	})

	t.Run("empty ChecksumKey", func(t *testing.T) {
		_, err := payos.NewClient(nil, &payos.Config{
			ClientID:    "test-client-id",
			APIKey:      "test-api-key",
			ChecksumKey: "",
		})
		if err == nil {
			t.Error("expected error for empty ChecksumKey")
		}
	})
}

// TestEnvConfig tests that EnvConfig reads from environment variables.
func TestEnvConfig(t *testing.T) {
	cfg := payos.EnvConfig()
	// EnvConfig should return a Config regardless of env vars presence.
	// Actual credential validation happens in NewClient.
	if cfg == nil {
		t.Error("EnvConfig should never return nil")
	}
}

// TestWebhookDataStruct tests that WebhookData struct has expected fields.
func TestWebhookDataStruct(t *testing.T) {
	data := &payos.WebhookData{
		OrderCode:           123456,
		Amount:              120000,
		Description:         "LTA8F3K2",
		AccountNumber:       "12345678",
		Reference:           "TF230204212323",
		TransactionDateTime: "2023-02-04 18:25:00",
		Currency:            "VND",
		PaymentLinkId:       "abc123",
	}

	if data.OrderCode != 123456 {
		t.Errorf("expected OrderCode 123456, got %d", data.OrderCode)
	}
	if data.Amount != 120000 {
		t.Errorf("expected Amount 120000, got %d", data.Amount)
	}
	if data.Description != "LTA8F3K2" {
		t.Errorf("expected Description LTA8F3K2, got %s", data.Description)
	}
}

// TestPaymentLinkStruct tests that PaymentLink struct has expected fields.
func TestPaymentLinkStruct(t *testing.T) {
	link := &payos.PaymentLink{
		Bin:           "970415",
		AccountNumber: "12345678",
		AccountName:   "NGUYEN VAN A",
		Amount:        120000,
		Description:   "LTA8F3K2",
		OrderCode:     123456,
		Currency:      "VND",
		PaymentLinkId: "abc123",
		Status:        "PENDING",
		CheckoutUrl:   "https://pay.payos.vn/checkout/abc123",
		QrCode:        "https://pay.payos.vn/qr/abc123",
	}

	if link.CheckoutUrl == "" {
		t.Error("expected non-empty CheckoutUrl")
	}
	if link.QrCode == "" {
		t.Error("expected non-empty QrCode")
	}
	if link.Status != "PENDING" {
		t.Errorf("expected Status PENDING, got %s", link.Status)
	}
}
