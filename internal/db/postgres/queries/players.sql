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

-- PlayerRecords returns players with their all-time record and cups (both 0 with no
-- finished matches / no winning tournaments) joined in — one query for the whole list
-- (id null) or a single player (id set). Cups is pre-aggregated once (a per-player CTE)
-- rather than a subquery per row; the record aggregates the per-match outcomes.
-- name: PlayerRecords :many
WITH cups AS (
    SELECT tm.player_id, COUNT(*) AS cups_won
    FROM team_members tm
    JOIN tournament_winners w
        ON w.tenant_id = tm.tenant_id AND w.tournament_id = tm.tournament_id AND w.team_id = tm.team_id
    WHERE tm.tenant_id = @tenant_id
    GROUP BY tm.player_id
)
SELECT
    p.*,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties,
    COALESCE(MAX(c.cups_won), 0)::bigint AS cups_won
FROM players p
LEFT JOIN player_match_outcomes o ON o.player_id = p.id AND o.tenant_id = p.tenant_id
LEFT JOIN cups c ON c.player_id = p.id
WHERE p.tenant_id = @tenant_id
  AND (sqlc.narg('id')::uuid IS NULL OR p.id = sqlc.narg('id'))
GROUP BY p.id
ORDER BY p.last_name, p.first_name;
