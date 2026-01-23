-- name: CreatePlayer :one
INSERT INTO players (
    tenant_id,
    email,
    first_name,
    last_name,
    hdcp,
    photo_path,
    biography,
    tier
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetPlayer :one
SELECT * FROM players
WHERE id = $1 AND tenant_id = $2;

-- name: GetPlayerByEmail :one
SELECT * FROM players
WHERE email = $1 AND tenant_id = $2;

-- name: ListPlayers :many
SELECT * FROM players
WHERE tenant_id = $1
ORDER BY last_name, first_name;

-- name: UpdatePlayer :one
UPDATE players
SET
    email = $3,
    first_name = $4,
    last_name = $5,
    hdcp = $6,
    photo_path = $7,
    biography = $8,
    tier = $9
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdatePlayerStats :one
UPDATE players
SET
    wins = $3,
    losses = $4,
    ties = $5,
    cups = $6
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE id = $1 AND tenant_id = $2;
