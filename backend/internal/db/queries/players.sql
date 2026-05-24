-- name: InsertPlayer :execresult
-- Single Player insert. Caller mints public_code via shortcode.GenerateUnique
-- (uniqueness checked against PlayerExistsByPublicCode below).
INSERT INTO players (group_id, display_name, public_code, is_active)
VALUES (?, ?, ?, 1);

-- name: CountActivePlayersInGroup :one
-- Used to enforce tier caps in Service.BulkCreate.
SELECT COUNT(*) FROM players WHERE group_id = ? AND is_active = 1;

-- name: ListPlayerDisplayNamesInGroup :many
-- For duplicate-against-existing-roster detection. Returns even inactive
-- players so the host can't accidentally re-add a name they marked inactive
-- (the unique index uk_players_group_display_name would block at INSERT time
-- anyway, but surfacing the conflict earlier gives a friendlier error).
SELECT display_name FROM players WHERE group_id = ?;

-- name: PlayerExistsByPublicCode :one
-- Used by shortcode.GenerateUnique's `exists` callback.
SELECT EXISTS(SELECT 1 FROM players WHERE public_code = ?) AS found;

-- name: ListActivePlayersInGroup :many
-- Returns the active roster for the participant picker (Story 1.10).
-- Ordered by display_name for stable UX.
SELECT id, group_id, display_name, public_code, is_active
FROM players
WHERE group_id = ? AND is_active = 1
ORDER BY display_name;
