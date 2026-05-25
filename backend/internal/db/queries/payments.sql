-- Story 6.3 & 6.4: Payment matching queries

-- name: InsertPayment :execresult
INSERT INTO payments (player_id, amount, bank_description, matched_intent_id, status, provider, provider_transaction_id, received_at, raw_payload_json, host_user_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPaymentIntentByCodeAndAmount :one
-- Exact match: code + amount + pending status
SELECT pi.id, pi.player_id, pi.group_id, pi.session_id, pi.amount, pi.code, pi.status, pi.covers_charge_ids_json, pi.expires_at,
       pl.group_id
FROM payment_intents pi
JOIN players pl ON pl.id = pi.player_id
WHERE pi.code = ? AND pi.amount = ? AND pi.status = 'pending';

-- name: UpdatePaymentIntentToMatched :exec
UPDATE payment_intents SET status = 'matched', provider_payment_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?;

-- name: UpdatePaymentIntentToSuspected :exec
UPDATE payment_intents SET status = 'suspected', updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?;

-- name: UpdateChargesToPaidAuto :exec
UPDATE session_charges SET status = 'paid', paid_at = NOW(), paid_via = 'auto', updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND status IN ('unpaid', 'pending_confirmation');

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = ?, matched_intent_id = ?, player_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?;

-- name: GetPaymentByProviderTx :one
-- Check for duplicate webhook (idempotency)
SELECT id FROM payments WHERE provider = ? AND provider_transaction_id = ?;

-- Story 6.5: Dashboard counters

-- name: CountSuspectedPaymentsForHost :one
SELECT COUNT(*) FROM payments WHERE host_user_id = ? AND status = 'suspected';

-- name: CountUnmatchedPaymentsForHost :one
SELECT COUNT(*) FROM payments WHERE host_user_id = ? AND status = 'unmatched';

-- name: ListSuspectedPaymentsForHost :many
-- List suspected payments for dashboard resolution
SELECT p.id, p.amount, p.bank_description, p.received_at, p.raw_payload_json,
       pi.id as intent_id, pi.code as intent_code, pi.amount as intent_amount,
       pl.id as player_id, pl.display_name as player_name
FROM payments p
LEFT JOIN payment_intents pi ON pi.id = p.matched_intent_id
LEFT JOIN players pl ON pl.id = p.player_id
WHERE p.host_user_id = ? AND p.status = 'suspected'
ORDER BY p.received_at DESC;

-- name: ListUnmatchedPaymentsForHost :many
-- List unmatched payments for dashboard resolution
SELECT p.id, p.amount, p.bank_description, p.received_at, p.raw_payload_json
FROM payments p
WHERE p.host_user_id = ? AND p.status = 'unmatched'
ORDER BY p.received_at DESC;

-- name: ConfirmSuspectedPayment :exec
UPDATE payments SET status = 'matched', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND host_user_id = ?;

-- name: RejectSuspectedPayment :exec
UPDATE payments SET status = 'unmatched', updated_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND host_user_id = ?;

-- name: LinkPaymentToPlayer :exec
UPDATE payments SET status = 'matched', player_id = ?, updated_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND host_user_id = ?;
