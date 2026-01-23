-- name: CreateTeeSet :one
INSERT INTO tee_sets (
    course_id,
    tee_color_id,
    tenant_id,
    slope,
    rating
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetTeeSet :one
SELECT * FROM tee_sets
WHERE course_id = $1 AND tee_color_id = $2 AND tenant_id = $3;

-- name: ListTeeSetsByCourse :many
SELECT ts.*, tc.color as tee_color_name
FROM tee_sets ts
JOIN tee_colors tc ON ts.tee_color_id = tc.id
WHERE ts.course_id = $1 AND ts.tenant_id = $2
ORDER BY tc.color;

-- name: UpdateTeeSet :one
UPDATE tee_sets
SET
    slope = $4,
    rating = $5
WHERE course_id = $1 AND tee_color_id = $2 AND tenant_id = $3
RETURNING *;

-- name: DeleteTeeSet :exec
DELETE FROM tee_sets
WHERE course_id = $1 AND tee_color_id = $2 AND tenant_id = $3;
