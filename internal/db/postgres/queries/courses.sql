-- name: CreateCourse :one
INSERT INTO courses (
    tenant_id,
    name
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetCourse :one
SELECT * FROM courses
WHERE id = $1 AND tenant_id = $2;

-- name: ListCourses :many
SELECT * FROM courses
WHERE tenant_id = $1
ORDER BY name;
