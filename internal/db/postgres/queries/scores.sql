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
SELECT
    s.*,
    h.par,
    h.hdcp AS hole_hdcp,
    h.yards
FROM scores s
JOIN holes h ON s.course_id = h.course_id
    AND s.tee_color_id = h.tee_color_id
    AND s.hole_number = h.number
WHERE s.match_id = $1 AND s.tenant_id = $2
ORDER BY s.hole_number;
