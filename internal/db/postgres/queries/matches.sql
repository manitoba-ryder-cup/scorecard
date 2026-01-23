-- name: CreateMatch :one
INSERT INTO matches (
    tournament_id,
    course_id,
    tee_color_id,
    match_format_id,
    tenant_id,
    tee_time,
    handicapped
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetMatch :one
SELECT * FROM matches
WHERE id = $1 AND tenant_id = $2;

-- name: GetMatchWithDetails :one
SELECT
    m.*,
    c.name as course_name,
    tc.color as tee_color_name,
    mf.name as match_format_name,
    t.name as tournament_name
FROM matches m
JOIN courses c ON m.course_id = c.id
JOIN tee_colors tc ON m.tee_color_id = tc.id
JOIN match_formats mf ON m.match_format_id = mf.id
JOIN tournaments t ON m.tournament_id = t.id
WHERE m.id = $1 AND m.tenant_id = $2;

-- name: ListMatchesByTournament :many
SELECT
    m.*,
    c.name as course_name,
    tc.color as tee_color_name
FROM matches m
JOIN courses c ON m.course_id = c.id
JOIN tee_colors tc ON m.tee_color_id = tc.id
WHERE m.tournament_id = $1 AND m.tenant_id = $2
ORDER BY m.tee_time;

-- name: UpdateMatch :one
UPDATE matches
SET
    course_id = $3,
    tee_color_id = $4,
    match_format_id = $5,
    tee_time = $6,
    handicapped = $7
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteMatch :exec
DELETE FROM matches
WHERE id = $1 AND tenant_id = $2;
