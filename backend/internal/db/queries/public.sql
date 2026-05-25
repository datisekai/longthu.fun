-- Story 2.1: Public Group Bill — get session + charges + players by share_code
-- No auth required. Returns session summary + player rows with charge status.

-- name: GetPublicGroupBill :one
SELECT
  s.id           AS session_id,
  s.date         AS session_date,
  s.title        AS session_title,
  s.total_cost   AS session_total_cost,
  s.status       AS session_status,
  g.name         AS group_name,
  g.privacy_mode AS group_privacy_mode
FROM sessions s
JOIN `groups` g ON g.id = s.group_id
WHERE s.share_code = ?
  AND s.status = 'finalized'
  AND g.archived_at IS NULL
LIMIT 1;

-- name: ListPublicSessionCharges :many
-- All charges for the session, ordered by id.
SELECT
  sc.id         AS charge_id,
  sc.player_id  AS player_id,
  sc.amount     AS charge_amount,
  sc.status     AS charge_status,
  p.display_name
FROM session_charges sc
JOIN players p ON p.id = sc.player_id
WHERE sc.session_id = ?
ORDER BY sc.id;

-- name: GetPlayerCrossSessionDebt :one
-- Sum of unpaid charges across ALL sessions for a player (for privacy_mode=public rows).
SELECT COALESCE(SUM(sc.amount), 0) AS total_unpaid
FROM session_charges sc
JOIN sessions s ON s.id = sc.session_id
JOIN `groups` g ON g.id = s.group_id
WHERE sc.player_id = ?
  AND sc.status IN ('unpaid', 'pending_confirmation', 'suspected')
  AND s.status = 'finalized'
  AND g.archived_at IS NULL;

-- name: GetPlayerByPublicCode :one
SELECT id, group_id, display_name, public_code, is_active FROM players WHERE public_code = ?;

-- name: ListPlayerChargesForLedger :many
-- Current session charge + cross-session unpaid for player ledger.
SELECT
  sc.id         AS charge_id,
  sc.session_id AS session_id,
  sc.amount     AS amount,
  sc.status     AS status,
  sc.paid_at    AS paid_at,
  s.date        AS session_date,
  s.title       AS session_title,
  g.name        AS group_name
FROM session_charges sc
JOIN sessions s ON s.id = sc.session_id
JOIN `groups` g ON g.id = s.group_id
WHERE sc.player_id = ?
  AND s.status = 'finalized'
ORDER BY s.date DESC, sc.id DESC;
