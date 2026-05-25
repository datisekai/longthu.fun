-- Story 3.2: Host Dashboard queries

-- name: GetDashboard :one
SELECT
  COALESCE(SUM(sc.amount), 0) AS total_unpaid
FROM session_charges sc
JOIN sessions s ON s.id = sc.session_id
JOIN `groups` g ON g.id = s.group_id
WHERE g.host_user_id = ?
  AND sc.status IN ('unpaid', 'pending_confirmation', 'suspected')
  AND s.status = 'finalized';

-- name: ListRecentSessionsForHost :many
-- Latest 5 finalized sessions across all groups.
SELECT
  s.id AS session_id,
  s.date AS session_date,
  s.title AS session_title,
  g.id AS group_id,
  g.name AS group_name,
  s.share_code AS share_code,
  s.total_cost AS total_cost,
  s.finalized_at AS finalized_at
FROM sessions s
JOIN `groups` g ON g.id = s.group_id
WHERE g.host_user_id = ? AND s.status = 'finalized'
ORDER BY s.finalized_at DESC
LIMIT 5;

-- name: ListPlayersWithUnpaidForHost :many
-- Top 5 players with highest unpaid amounts.
SELECT
  p.id AS player_id,
  p.display_name AS player_name,
  p.public_code AS player_code,
  g.id AS group_id,
  g.name AS group_name,
  COALESCE(SUM(sc.amount), 0) AS total_unpaid
FROM players p
JOIN `groups` g ON g.id = p.group_id
JOIN session_charges sc ON sc.player_id = p.id
JOIN sessions s ON s.id = sc.session_id
WHERE g.host_user_id = ?
  AND sc.status IN ('unpaid', 'pending_confirmation', 'suspected')
  AND s.status = 'finalized'
  AND p.is_active = 1
GROUP BY p.id, p.display_name, p.public_code, g.id, g.name
ORDER BY total_unpaid DESC
LIMIT 5;

-- name: CountGroupsForHost :one
SELECT COUNT(*) FROM `groups` WHERE host_user_id = ?;

-- name: CountFinalizedSessionsForHost :one
SELECT COUNT(*) FROM sessions s
JOIN `groups` g ON g.id = s.group_id
WHERE g.host_user_id = ? AND s.status = 'finalized';
