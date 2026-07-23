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

-- name: ListMatchesByTournament :many
SELECT m.* FROM matches m
WHERE m.tournament_id = $1 AND m.tenant_id = $2
ORDER BY m.tee_time;

-- Joined with format + course names so the results view resolves both in one query.
-- name: ListMatchesWithDetailsByTournament :many
SELECT m.*, mf.name AS format_name, c.name AS course_name
FROM matches m
JOIN match_formats mf ON mf.id = m.match_format_id
JOIN courses c ON c.id = m.course_id AND c.tenant_id = m.tenant_id
WHERE m.tournament_id = @tournament_id AND m.tenant_id = @tenant_id
ORDER BY m.tee_time NULLS LAST, m.id;
