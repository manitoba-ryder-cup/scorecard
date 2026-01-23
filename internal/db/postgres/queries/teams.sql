-- name: CreateTeam :one
INSERT INTO teams (
    tenant_id,
    name
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams
WHERE id = $1 AND tenant_id = $2;

-- name: GetTeamByName :one
SELECT * FROM teams
WHERE name = $1 AND tenant_id = $2;

-- name: ListTeams :many
SELECT * FROM teams
WHERE tenant_id = $1
ORDER BY name;

-- name: UpdateTeam :one
UPDATE teams
SET name = $3
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1 AND tenant_id = $2;
