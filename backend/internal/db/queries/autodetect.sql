-- Story 6.2: Auto-Detect setup queries

-- name: GetGroupWithAutoDetect :one
-- Get a group with auto-detect settings for the host
SELECT g.id, g.host_user_id, g.name, g.auto_detect_enabled, g.auto_detect_bank_account_id,
       g.auto_detect_credentials_json,
       b.bank_name, b.bank_code, b.account_number, b.account_holder_name
FROM `groups` g
LEFT JOIN bank_accounts b ON b.id = g.auto_detect_bank_account_id
WHERE g.id = ? AND g.host_user_id = ?;

-- name: EnableAutoDetect :exec
-- Enable auto-detect for a group: update bank account + encrypted credentials
UPDATE `groups`
SET auto_detect_enabled = 1,
    auto_detect_bank_account_id = ?,
    auto_detect_credentials_json = ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND host_user_id = ?;

-- name: DisableAutoDetect :exec
-- Disable auto-detect for a group
UPDATE `groups`
SET auto_detect_enabled = 0,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND host_user_id = ?;

-- name: GetPayosSupportedBankAccounts :many
-- Get bank accounts that support payOS auto-detect
-- Supported: MBBank, OCB, KienlongBank, ACB, BIDV
SELECT id, bank_name, bank_code, account_number, account_holder_name, is_default
FROM bank_accounts
WHERE user_id = ? AND is_default = 1
  AND bank_code IN ('MBBB', 'OCB', 'KLBANK', 'ACB', 'BIDV');

-- name: GetGroupBankAccounts :many
-- Get all bank accounts for a group (for Auto-Detect wizard)
SELECT id, bank_name, bank_code, account_number, account_holder_name, is_default
FROM bank_accounts
WHERE user_id = ?;

-- name: GetGroupHostUserID :one
-- Get the host_user_id for a group (for payment matching)
SELECT g.host_user_id
FROM `groups` g
WHERE g.id = ?;

-- name: UpdatePaymentsHostUserID :exec
-- Update host_user_id denormalized column for payments
UPDATE payments p
JOIN players pl ON pl.id = p.player_id
JOIN `groups` g ON g.id = pl.group_id
SET p.host_user_id = g.host_user_id
WHERE p.host_user_id IS NULL;
