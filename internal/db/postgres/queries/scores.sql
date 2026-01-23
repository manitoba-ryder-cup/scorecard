-- name: CreateScore :one
INSERT INTO scores (
    match_id,
    player_id,
    course_id,
    tee_color_id,
    hole_number,
    tenant_id,
    strokes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetScore :one
SELECT * FROM scores
WHERE match_id = $1 AND player_id = $2 AND hole_number = $3 AND tenant_id = $4;

-- name: ListScoresByMatch :many
SELECT
    s.*,
    h.par,
    h.hdcp as hole_hdcp,
    p.first_name,
    p.last_name,
    p.hdcp as player_hdcp
FROM scores s
JOIN holes h ON s.course_id = h.course_id
    AND s.tee_color_id = h.tee_color_id
    AND s.hole_number = h.number
JOIN players p ON s.player_id = p.id
WHERE s.match_id = $1 AND s.tenant_id = $2
ORDER BY s.hole_number, p.last_name;

-- name: ListScoresByMatchAndPlayer :many
SELECT
    s.*,
    h.par,
    h.hdcp as hole_hdcp,
    h.yards
FROM scores s
JOIN holes h ON s.course_id = h.course_id
    AND s.tee_color_id = h.tee_color_id
    AND s.hole_number = h.number
WHERE s.match_id = $1 AND s.player_id = $2 AND s.tenant_id = $3
ORDER BY s.hole_number;

-- name: UpdateScore :one
UPDATE scores
SET strokes = $4
WHERE match_id = $1 AND player_id = $2 AND hole_number = $3 AND tenant_id = $5
RETURNING *;

-- name: DeleteScore :exec
DELETE FROM scores
WHERE match_id = $1 AND player_id = $2 AND hole_number = $3 AND tenant_id = $4;

-- name: DeleteScoresByMatch :exec
DELETE FROM scores
WHERE match_id = $1 AND tenant_id = $2;
