-- name: GetGroupByIDForHost :one
-- Host-scoped lookup. Returning sql.ErrNoRows for both "missing" and
-- "belongs to another host" preserves tenant isolation.
SELECT id, host_user_id, name, slug, default_bank_account_id, telegram_chat_id,
       privacy_mode, auto_detect_enabled, auto_detect_bank_account_id,
       archived_at, created_at, updated_at
FROM `groups`
WHERE id = ? AND host_user_id = ?
LIMIT 1;

-- name: InsertGroup :execresult
-- Added in Story 1.8. privacy_mode defaults to 'public' and
-- auto_detect_enabled defaults to 0 per the schema; slug stays NULL
-- (vestigial per Story 1.2 §Completion Notes #2).
INSERT INTO `groups` (host_user_id, name, default_bank_account_id)
VALUES (?, ?, ?);
