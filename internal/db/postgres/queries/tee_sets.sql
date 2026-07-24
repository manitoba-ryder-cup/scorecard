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

-- A course's configured tee sets with their colour name joined in, so a match-setup
-- picker can offer valid (course, tee) options labelled without a second lookup.
-- name: ListTeeSetsByCourse :many
SELECT ts.course_id, ts.tee_color_id, tc.color, ts.slope, ts.rating
FROM tee_sets ts
JOIN tee_colors tc ON tc.id = ts.tee_color_id AND tc.tenant_id = ts.tenant_id
WHERE ts.course_id = $1 AND ts.tenant_id = $2
ORDER BY tc.color;
