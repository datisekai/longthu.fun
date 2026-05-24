-- name: CountBankAccountsForHost :one
SELECT COUNT(*)
FROM bank_accounts
WHERE user_id = ?;

-- name: InsertBankAccount :execresult
INSERT INTO bank_accounts (user_id, bank_name, bank_code, account_number, account_holder_name, is_default)
VALUES (?, ?, ?, ?, ?, ?);
