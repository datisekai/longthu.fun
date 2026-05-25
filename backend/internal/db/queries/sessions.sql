-- name: InsertSession :execresult
-- Creates a draft session. status defaults to 'draft' via the table default,
-- but we set it explicitly for grep-ability.
INSERT INTO sessions (group_id, `date`, title, location, status, created_by_user_id)
VALUES (?, ?, ?, ?, 'draft', ?);

-- name: GetSessionByIDForHost :one
-- Tenant-isolated session read. Joins through groups so a host can't
-- read another host's session, no enumeration leak.
SELECT s.id, s.group_id, s.`date`, s.title, s.location,
       s.start_time, s.end_time, s.total_cost, s.share_code,
       s.status, s.created_by_user_id, s.finalized_at, s.created_at, s.updated_at
FROM sessions s
JOIN `groups` g ON g.id = s.group_id
WHERE s.id = ? AND g.host_user_id = ?;

-- name: UpdateSessionDraftMeta :exec
-- Updates date/title/location on a draft session (host edits before finalize).
UPDATE sessions
SET `date` = ?, title = ?, location = ?
WHERE id = ?;

-- name: InsertSessionCostItem :execresult
-- Adds a cost item to a session draft.
INSERT INTO session_cost_items (session_id, type, label, amount, is_included_in_split)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteSessionCostItem :exec
-- Removes a single cost item by its ID. Caller already verified session ownership.
DELETE FROM session_cost_items WHERE id = ? AND session_id = ?;

-- name: ListSessionCostItems :many
-- Returns all cost items for a session in insertion order.
SELECT id, session_id, type, label, amount, notes, is_included_in_split, created_at
FROM session_cost_items
WHERE session_id = ?
ORDER BY id;

-- name: DeleteAllSessionParticipants :exec
-- First half of the participant-replace tx (PUT semantics for the participant set).
DELETE FROM session_participants WHERE session_id = ?;

-- name: InsertSessionParticipant :exec
-- Second half of the participant-replace tx. Weight defaults to 1.00.
INSERT INTO session_participants (session_id, player_id, weight)
VALUES (?, ?, 1.00);

-- name: ListSessionParticipants :many
-- Returns the player IDs participating in a session.
SELECT id, session_id, player_id, weight
FROM session_participants
WHERE session_id = ?
ORDER BY id;

-- name: CountPlayersInGroupByIDs :one
-- Used to validate that all submitted playerIds belong to the session's Group.
-- Returns the count of matching active players; caller compares to len(submitted).
SELECT COUNT(*)
FROM players
WHERE group_id = ? AND id IN (sqlc.slice('player_ids')) AND is_active = 1;

-- name: InsertSessionCharge :execresult
-- One row per participant at finalize time. Story 1.11.
INSERT INTO session_charges (session_id, player_id, amount, status)
VALUES (?, ?, ?, 'unpaid');

-- name: ListSessionCharges :many
-- Returns all charges for a session, ordered by id (insertion order).
SELECT id, session_id, player_id, amount, status, paid_at, paid_via, description, created_at, updated_at
FROM session_charges
WHERE session_id = ?
ORDER BY id;

-- name: UpdateSessionFinalize :exec
-- Sets the session to finalized + records share_code + total_cost + finalized_at.
UPDATE sessions
SET status = 'finalized', share_code = ?, total_cost = ?, finalized_at = ?
WHERE id = ?;

-- name: SessionShareCodeExists :one
-- shortcode.GenerateUnique callback against sessions.share_code.
SELECT EXISTS(SELECT 1 FROM sessions WHERE share_code = ?) AS found;

-- name: GetChargeByIDForHost :one
-- Tenant-isolated charge read via session → group → host.
SELECT sc.id, sc.session_id, sc.player_id, sc.amount, sc.status, sc.paid_at, sc.paid_via, sc.description
FROM session_charges sc
JOIN sessions s ON s.id = sc.session_id
JOIN `groups` g ON g.id = s.group_id
WHERE sc.id = ? AND g.host_user_id = ?;

-- name: UpdateChargeStatusManual :exec
-- Mark charge as paid (manual confirm) or revert to unpaid (undo).
UPDATE session_charges
SET status = ?, paid_at = ?, paid_via = ?
WHERE id = ?;

-- Story 4.1: Edit finalized session
-- name: UpdateSessionFinalizedMeta :exec
UPDATE sessions
SET title = ?, location = ?, `date` = ?
WHERE id = ? AND status = 'finalized';

-- Story 4.4: Waive charge
-- name: UpdateChargeWaived :exec
UPDATE session_charges SET status = 'waived' WHERE id = ?;

-- Story 4.7: Waive charge with note
-- name: UpdateChargeStatusWaived :exec
UPDATE session_charges SET status = ?, description = ? WHERE id = ?;
