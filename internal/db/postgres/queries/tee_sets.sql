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
