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

-- PlayerRecords returns players with their all-time W-L-T — one query for the whole list
-- (id null) or a single player (id set). Cups won is derived in the domain, since which
-- side won a Cup is a scoring rule rather than a retrieval concern.
-- name: PlayerRecords :many
SELECT
    p.*,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM players p
LEFT JOIN player_match_outcomes o ON o.player_id = p.id AND o.tenant_id = p.tenant_id
WHERE p.tenant_id = @tenant_id
  AND (sqlc.narg('id')::uuid IS NULL OR p.id = sqlc.narg('id'))
GROUP BY p.id
ORDER BY p.last_name, p.first_name;
