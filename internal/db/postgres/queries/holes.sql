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
