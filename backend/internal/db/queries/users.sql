-- name: InsertUser :execresult
-- Insert a new host user. Returns the result (use LastInsertId() for the new id).
INSERT INTO users (email, password_hash, display_name, tier)
VALUES (?, ?, ?, 'free');

-- name: GetUserByEmail :one
-- Used at login. Returns sql.ErrNoRows when email is not registered.
SELECT id, email, password_hash, display_name, tier, tier_changed_at, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByID :one
-- Used by the session middleware to enrich the request context with current tier.
-- Returns sql.ErrNoRows if the user has been hard-deleted (rare).
SELECT id, email, password_hash, display_name, tier, tier_changed_at, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;
