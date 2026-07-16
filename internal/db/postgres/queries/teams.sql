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
SELECT * FROM teams
WHERE tournament_id = $1 AND tenant_id = $2
ORDER BY color;

-- name: SetTeamCaptain :one
UPDATE teams
SET captain_id = $3
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1 AND tenant_id = $2;
