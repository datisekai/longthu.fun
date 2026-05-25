-- name: CountBankAccountsForHost :one
SELECT COUNT(*)
FROM bank_accounts
WHERE user_id = ?;

-- name: InsertBankAccount :execresult
INSERT INTO bank_accounts (user_id, bank_name, bank_code, account_number, account_holder_name, is_default)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetDefaultBankAccountForHost :one
-- Added in Story 1.8 — used by Group create to auto-pick the host's default
-- bank as `default_bank_account_id`. Returns sql.ErrNoRows if the host has
-- no bank account yet; callers should treat that as "default_bank_account_id = NULL".
SELECT id, user_id, bank_name, bank_code, account_number, account_holder_name, is_default,
       default_flag, created_at, updated_at
FROM bank_accounts
WHERE user_id = ? AND is_default = 1
LIMIT 1;

-- Story 4.2: List all bank accounts for host
-- name: ListBankAccountsForHost :many
SELECT id, user_id, bank_name, bank_code, account_number, account_holder_name, is_default,
       default_flag, created_at, updated_at
FROM bank_accounts
WHERE user_id = ?
ORDER BY is_default DESC, id ASC;

-- Story 4.2: Set bank account as default
-- name: SetDefaultBankAccount :exec
-- First clear existing default, then set new one
UPDATE bank_accounts SET is_default = 0 WHERE user_id = ?;
UPDATE bank_accounts SET is_default = 1 WHERE id = ?;
