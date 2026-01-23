-- name: CreateTeeColor :one
INSERT INTO tee_colors (
    tenant_id,
    color
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetTeeColor :one
SELECT * FROM tee_colors
WHERE id = $1 AND tenant_id = $2;

-- name: GetTeeColorByName :one
SELECT * FROM tee_colors
WHERE color = $1 AND tenant_id = $2;

-- name: ListTeeColors :many
SELECT * FROM tee_colors
WHERE tenant_id = $1
ORDER BY color;

-- name: UpdateTeeColor :one
UPDATE tee_colors
SET color = $3
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteTeeColor :exec
DELETE FROM tee_colors
WHERE id = $1 AND tenant_id = $2;
