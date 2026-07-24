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

-- Clears the captain when that player leaves the team (e.g. undrafted), so a team never
-- keeps a captain who is no longer on it. A no-op when the player wasn't the captain.
-- name: ClearTeamCaptainForPlayer :exec
UPDATE teams
SET captain_id = NULL
WHERE id = $1 AND captain_id = $2 AND tenant_id = $3;
