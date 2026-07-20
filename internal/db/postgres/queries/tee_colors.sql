-- name: CreateTeeColor :one
INSERT INTO tee_colors (
    tenant_id,
    color
) VALUES (
    $1, $2
) RETURNING *;

-- name: ListTeeColors :many
SELECT * FROM tee_colors
WHERE tenant_id = $1
ORDER BY color;
