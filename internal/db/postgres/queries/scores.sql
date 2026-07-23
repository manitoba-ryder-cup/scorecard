-- Per-player score (singles / fourball): one row per player per hole.
-- name: UpsertPlayerScore :one
INSERT INTO scores (
    match_id, team_id, player_id, course_id, tee_color_id, hole_number, tenant_id, strokes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (match_id, hole_number, player_id) WHERE player_id IS NOT NULL
DO UPDATE SET strokes = EXCLUDED.strokes, updated_at = now()
RETURNING *;

-- Team score (alternate shot / scramble / mod-scotch): one row per team per hole.
-- name: UpsertTeamScore :one
INSERT INTO scores (
    match_id, team_id, player_id, course_id, tee_color_id, hole_number, tenant_id, strokes
) VALUES (
    $1, $2, NULL, $3, $4, $5, $6, $7
)
ON CONFLICT (match_id, hole_number, team_id) WHERE player_id IS NULL
DO UPDATE SET strokes = EXCLUDED.strokes, updated_at = now()
RETURNING *;

-- name: ListScoresByMatch :many
SELECT s.* FROM scores s
WHERE s.match_id = $1 AND s.tenant_id = $2
ORDER BY s.hole_number;

-- Every score across the tournament, ordered by match, so the results view computes
-- each match's progression without a per-match query.
-- name: ListScoresByTournament :many
SELECT s.*
FROM scores s
JOIN matches m ON m.id = s.match_id AND m.tenant_id = s.tenant_id
WHERE m.tournament_id = @tournament_id AND s.tenant_id = @tenant_id
ORDER BY s.match_id, s.hole_number;
