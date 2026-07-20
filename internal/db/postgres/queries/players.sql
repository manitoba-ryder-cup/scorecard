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

-- name: GetPlayer :one
SELECT * FROM players
WHERE id = $1 AND tenant_id = $2;

-- name: ListPlayers :many
SELECT * FROM players
WHERE tenant_id = $1
ORDER BY last_name, first_name;
