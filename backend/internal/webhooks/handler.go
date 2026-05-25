package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/autodetect"
	dbpkg "github.com/datisekai/longthu.fun/backend/internal/db"
	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
	"github.com/datisekai/longthu.fun/backend/internal/payments"
)

type Handler struct {
	db        *sql.DB
	matcher   *payments.Matcher
	whitelist string // Expected Host header value (webhook.longthu.fun)
}

func NewHandler(db *sql.DB, whitelist string) *Handler {
	return &Handler{
		db:      db,
		matcher: payments.NewMatcher(db),
		whitelist: whitelist,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/webhooks/payos", h.handlePayOSWebhook)
}

// handlePayOSWebhook handles incoming payOS payment webhooks.
func (h *Handler) handlePayOSWebhook(c *gin.Context) {
	ctx := c.Request.Context()
	logger := slog.Default()

	// 1. Validate Host header (defense-in-depth).
	if c.GetHeader("Host") != h.whitelist {
		logger.Warn("webhook: rejected - invalid Host header",
			"got", c.GetHeader("Host"),
			"expected", h.whitelist)
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// 2. Read raw body.
	rawJSON, err := c.GetRawData()
	if err != nil {
		logger.Warn("webhook: failed to read body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// 3. Parse JSON to check structure.
	var webhook struct {
		Code      string `json:"code"`
		Success   bool   `json:"success"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(rawJSON, &webhook); err != nil {
		logger.Warn("webhook: malformed JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// 4. Check for missing signature.
	if webhook.Signature == "" {
		logger.Warn("webhook: missing signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature"})
		return
	}

	// 5. Parse webhook body as map for SDK verification.
	var webhookBody map[string]interface{}
	if err := json.Unmarshal(rawJSON, &webhookBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// 6. Extract description to find the group and get payOS credentials.
	description := ""
	if data, ok := webhookBody["data"].(map[string]interface{}); ok {
		if desc, ok := data["description"].(string); ok {
			description = desc
		}
	}

	// 7. Find group from payment intent code.
	// Extract LT{6-char} code from description.
	code := extractPaymentCode(description)
	if code == "" {
		// No recognizable code - process as unmatched.
		h.processUnmatchedPayment(ctx, rawJSON, webhookBody, logger)
		c.JSON(http.StatusOK, gin.H{"success": true, "matched": false, "status": "unmatched"})
		return
	}

	// 8. Look up payment intent to find group.
	q := dbgen.New(h.db)
	intent, err := q.GetPaymentIntentByCode(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Info("webhook: no intent found for code", "code", code)
			h.processUnmatchedPayment(ctx, rawJSON, webhookBody, logger)
			c.JSON(http.StatusOK, gin.H{"success": true, "matched": false, "status": "unmatched"})
			return
		}
		logger.Error("webhook: error looking up intent", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 9. Get payOS credentials for the group to verify signature.
	autoDetectSvc := autodetect.NewService(h.db)
	client, _, err := autoDetectSvc.GetPayOSClientForGroup(ctx, intent.GroupID)
	if err != nil {
		// Auto-detect not enabled for this group - skip verification.
		logger.Info("webhook: auto-detect not enabled for group", "groupID", intent.GroupID)
		c.JSON(http.StatusOK, gin.H{"success": true, "matched": false, "status": "skipped"})
		return
	}

	// 10. Verify webhook signature.
	verifiedData, err := client.VerifyWebhook(ctx, rawJSON)
	if err != nil {
		logger.Warn("webhook: signature verification failed",
			"error", err,
			"description", description)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// 11. Idempotency check.
	var existingPaymentID uint64
	var _ error // ignore the error, we just check if exists
	existingPaymentID, _ = q.GetPaymentByProviderTx(ctx, dbgen.GetPaymentByProviderTxParams{
		Provider:              "payos",
		ProviderTransactionID: verifiedData.Reference,
	})
	if existingPaymentID != 0 {
		logger.Info("webhook: duplicate detected", "txID", verifiedData.Reference)
		c.JSON(http.StatusOK, gin.H{"success": true, "deduplicated": true})
		return
	}

	// 12. Get host_user_id for the group.
	hostUserID, err := q.GetGroupHostUserID(ctx, intent.GroupID)
	if err != nil {
		logger.Error("webhook: failed to get host user ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 13. Run matcher in transaction.
	err = dbpkg.WithTx(ctx, h.db, func(tx *sql.Tx) error {
		return h.matcher.Match(ctx, tx, payments.MatchParams{
			Provider:                "payos",
			ProviderTransactionID:   verifiedData.Reference,
			Amount:                  int64(verifiedData.Amount),
			Description:             verifiedData.Description,
			PaymentIntentID:        intent.ID,
			PlayerID:               intent.PlayerID,
			GroupID:                intent.GroupID,
			HostUserID:             hostUserID,
			RawPayload:             rawJSON,
		})
	})

	if err != nil {
		logger.Error("webhook: matcher error", "error", err)
		c.JSON(http.StatusOK, gin.H{"success": true, "matched": false, "status": "error"})
		return
	}

	logger.Info("webhook: processed successfully",
		"txID", verifiedData.Reference,
		"amount", verifiedData.Amount)
	c.JSON(http.StatusOK, gin.H{"success": true, "matched": true, "status": "processed"})
}

// processUnmatchedPayment records a payment with no matching intent.
func (h *Handler) processUnmatchedPayment(ctx context.Context, rawJSON []byte, webhookBody map[string]interface{}, logger *slog.Logger) {
	// Extract amount and reference from webhook.
	var amount int64
	var reference string
	if data, ok := webhookBody["data"].(map[string]interface{}); ok {
		if a, ok := data["amount"].(float64); ok {
			amount = int64(a)
		}
		if r, ok := data["reference"].(string); ok {
			reference = r
		}
	}

	rawJSONStr := string(rawJSON)

	err := dbpkg.WithTx(ctx, h.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO payments (player_id, amount, bank_description, matched_intent_id, status, provider, provider_transaction_id, received_at, raw_payload_json, host_user_id)
			VALUES (NULL, ?, ?, NULL, 'unmatched', 'payos', ?, ?, NOW(3), ?, NULL)
		`, amount, "", reference, rawJSONStr)
		return err
	})

	if err != nil {
		logger.Error("webhook: failed to insert unmatched payment", "error", err)
	}
}

// extractPaymentCode extracts the LT{6-char} code from a bank description.
// Matches patterns like "LTA8F3K2" or "LTA8F3".
func extractPaymentCode(description string) string {
	if len(description) < 8 {
		return ""
	}
	// Look for LT prefix followed by alphanumeric chars.
	for i := 0; i <= len(description)-8; i++ {
		if description[i] == 'L' && i+1 < len(description) && description[i+1] == 'T' {
			code := description[i : i+8]
			return code[2:] // Strip "LT" prefix, return 6-char code
		}
	}
	return ""
}
