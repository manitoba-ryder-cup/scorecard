-- name: CreateTeam :one
INSERT INTO teams (
    tenant_id,
    tournament_id,
    color,
    captain_id
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams
WHERE id = $1 AND tenant_id = $2;

-- name: GetTeamByColor :one
SELECT * FROM teams
WHERE tournament_id = $1 AND color = $2 AND tenant_id = $3;

-- name: ListTeamsByTournament :many
-- LEFT JOIN so the captain (if any) comes back in one query instead of a per-team lookup.
SELECT
    t.*,
    p.first_name AS captain_first_name,
    p.last_name  AS captain_last_name,
    p.email      AS captain_email
FROM teams t
LEFT JOIN players p ON p.id = t.captain_id
WHERE t.tournament_id = $1 AND t.tenant_id = $2
ORDER BY t.color;

-- name: SetTeamCaptain :one
UPDATE teams
SET captain_id = $3
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1 AND tenant_id = $2;
