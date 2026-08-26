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

-- A null argument leaves the column alone, so a caller changing one field never has to
-- read the match first just to echo the rest back. match_format_id is absent rather than
-- optional: a match is created in a format and deleted to leave it.
-- name: UpdateMatch :one
UPDATE matches
SET course_id       = COALESCE(sqlc.narg('course_id'), course_id),
    tee_color_id    = COALESCE(sqlc.narg('tee_color_id'), tee_color_id),
    tee_time        = COALESCE(sqlc.narg('tee_time'), tee_time),
    handicapped     = COALESCE(sqlc.narg('handicapped'), handicapped)
WHERE id = sqlc.arg('id') AND tenant_id = sqlc.arg('tenant_id')
RETURNING *;

-- Serializes writes to a match. Each of them reads to decide and then writes on that answer,
-- so without this lock two can decide against the same snapshot and the later one acts on an
-- answer that stopped being true before it landed.
-- Returns the row, not just its id: a caller that locks a match to decide something about it
-- needs the match, and reading it again inside the lock cannot say anything different.
-- name: LockMatch :one
SELECT * FROM matches
WHERE id = $1 AND tenant_id = $2
FOR UPDATE;

-- name: ListMatchesByTournament :many
SELECT m.* FROM matches m
WHERE m.tournament_id = $1 AND m.tenant_id = $2
ORDER BY m.tee_time;

-- Joined with the format and course so the results view resolves all of it in one query. The
-- format's rules ride along: a client reading a hole or drawing an unfilled one needs them, and
-- deriving them from the name would be this table's data copied into a list somewhere else.
-- name: ListMatchesWithDetailsByTournament :many
SELECT m.*, mf.name AS format_name, mf.players_per_side, mf.scores_per_player, c.name AS course_name
FROM matches m
JOIN match_formats mf ON mf.id = m.match_format_id
JOIN courses c ON c.id = m.course_id AND c.tenant_id = m.tenant_id
WHERE m.tournament_id = @tournament_id AND m.tenant_id = @tenant_id
ORDER BY m.tee_time NULLS LAST, m.id;

-- name: DeleteMatch :execrows
DELETE FROM matches WHERE id = @id AND tenant_id = @tenant_id;
