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
