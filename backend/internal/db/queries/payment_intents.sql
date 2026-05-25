-- Story 2.4: Payment Intent queries

-- name: GetLatestFinalizedSessionForGroup :one
SELECT id, group_id, date, title, total_cost, status
FROM sessions
WHERE group_id = ? AND status = 'finalized'
ORDER BY finalized_at DESC
LIMIT 1;

-- name: ListUnpaidChargesForPlayerSession :many
SELECT id, session_id, player_id, amount, status
FROM session_charges
WHERE player_id = ? AND session_id = ? AND status IN ('unpaid', 'pending_confirmation', 'suspected');

-- name: ListAllUnpaidChargesForPlayer :many
SELECT sc.id, sc.session_id, sc.player_id, sc.amount, sc.status
FROM session_charges sc
JOIN sessions s ON s.id = sc.session_id
WHERE sc.player_id = ? AND sc.status IN ('unpaid', 'pending_confirmation', 'suspected') AND s.status = 'finalized'
ORDER BY s.date DESC;

-- name: ListPendingIntentsForPlayer :many
SELECT id, player_id, code, status, expires_at
FROM payment_intents
WHERE player_id = ? AND status = 'pending';

-- name: PaymentIntentCodeExists :one
SELECT EXISTS(SELECT 1 FROM payment_intents WHERE code = ?) AS found;

-- name: CreatePaymentIntent :execresult
INSERT INTO payment_intents (player_id, group_id, session_id, amount, code, status, provider, covers_charge_ids_json, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPaymentIntentByCode :one
SELECT id, player_id, group_id, session_id, amount, code, status, provider, covers_charge_ids_json, expires_at
FROM payment_intents
WHERE code = ?;

-- name: CancelPaymentIntent :exec
UPDATE payment_intents SET status = ? WHERE id = ?;

-- name: UpdatePaymentIntentStatus :exec
UPDATE payment_intents SET status = ? WHERE id = ?;

-- name: UpdateChargeStatus :exec
UPDATE session_charges SET status = ? WHERE id = ?;

-- name: GetDefaultBankAccountByGroup :one
SELECT b.id, b.bank_name, b.bank_code, b.account_number, b.account_holder_name
FROM bank_accounts b
JOIN `groups` g ON g.default_bank_account_id = b.id
WHERE g.id = ? AND b.is_default = 1
LIMIT 1;
