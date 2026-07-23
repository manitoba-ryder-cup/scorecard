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
-- (id null) or a single player (id set). Cups counts finished tournaments the player's
-- team won outright.
-- name: PlayerRecords :many
SELECT
    p.*,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id = mp.team_id) AS wins,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NOT NULL AND mr.leader_team_id <> mp.team_id) AS losses,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NULL) AS ties,
    (
        SELECT COUNT(*)
        FROM team_members tm
        JOIN tournament_winners w
            ON w.tenant_id = tm.tenant_id AND w.tournament_id = tm.tournament_id AND w.team_id = tm.team_id
        WHERE tm.player_id = p.id AND tm.tenant_id = p.tenant_id
    ) AS cups_won
FROM players p
LEFT JOIN match_participants mp ON mp.player_id = p.id AND mp.tenant_id = p.tenant_id
LEFT JOIN match_results mr ON mr.match_id = mp.match_id AND mr.tenant_id = mp.tenant_id
WHERE p.tenant_id = @tenant_id
  AND (sqlc.narg('id')::uuid IS NULL OR p.id = sqlc.narg('id'))
GROUP BY p.id
ORDER BY p.last_name, p.first_name;
