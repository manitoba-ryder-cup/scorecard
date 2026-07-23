-- name: CreatePlayer :one
INSERT INTO players (
    tenant_id,
    user_id,
    email,
    first_name,
    last_name,
    photo_path
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- GetPlayer returns a player with their all-time W-L-T joined in (see ListPlayers).
-- name: GetPlayer :one
SELECT
    p.*,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id = mp.team_id) AS wins,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NOT NULL AND mr.leader_team_id <> mp.team_id) AS losses,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NULL) AS ties
FROM players p
LEFT JOIN match_participants mp ON mp.player_id = p.id AND mp.tenant_id = p.tenant_id
LEFT JOIN match_results mr ON mr.match_id = mp.match_id AND mr.tenant_id = mp.tenant_id
WHERE p.id = $1 AND p.tenant_id = $2
GROUP BY p.id;

-- ListPlayers returns every player with their all-time W-L-T (0/0/0 for a player with no
-- finished matches) joined in, so the players listing is a single query. The record
-- counts every finished match the player has played (won when their team led, etc.).
-- name: ListPlayers :many
SELECT
    p.*,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id = mp.team_id) AS wins,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NOT NULL AND mr.leader_team_id <> mp.team_id) AS losses,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NULL) AS ties
FROM players p
LEFT JOIN match_participants mp ON mp.player_id = p.id AND mp.tenant_id = p.tenant_id
LEFT JOIN match_results mr ON mr.match_id = mp.match_id AND mr.tenant_id = mp.tenant_id
WHERE p.tenant_id = $1
GROUP BY p.id
ORDER BY p.last_name, p.first_name;
