-- name: CreateMatchFormat :one
INSERT INTO match_formats (
    tenant_id,
    name
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetMatchFormat :one
SELECT * FROM match_formats
WHERE id = $1 AND tenant_id = $2;

-- name: GetMatchFormatByName :one
SELECT * FROM match_formats
WHERE name = $1 AND tenant_id = $2;

-- name: ListMatchFormats :many
SELECT * FROM match_formats
WHERE tenant_id = $1
ORDER BY name;

-- name: UpdateMatchFormat :one
UPDATE match_formats
SET name = $3
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteMatchFormat :exec
DELETE FROM match_formats
WHERE id = $1 AND tenant_id = $2;
