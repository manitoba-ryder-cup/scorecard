-- name: CreateHole :one
INSERT INTO holes (
    course_id,
    tee_color_id,
    number,
    tenant_id,
    par,
    hdcp,
    yards
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetHole :one
SELECT * FROM holes
WHERE course_id = $1 AND tee_color_id = $2 AND number = $3 AND tenant_id = $4;

-- name: ListHolesByTeeSet :many
SELECT * FROM holes
WHERE course_id = $1 AND tee_color_id = $2 AND tenant_id = $3
ORDER BY number;

-- name: UpdateHole :one
UPDATE holes
SET
    par = $5,
    hdcp = $6,
    yards = $7
WHERE course_id = $1 AND tee_color_id = $2 AND number = $3 AND tenant_id = $4
RETURNING *;

-- name: DeleteHole :exec
DELETE FROM holes
WHERE course_id = $1 AND tee_color_id = $2 AND number = $3 AND tenant_id = $4;
