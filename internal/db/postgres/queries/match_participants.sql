-- name: CreateMatchParticipant :one
INSERT INTO match_participants (
    tournament_id,
    match_id,
    player_id,
    team_id,
    tenant_id
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetMatchParticipant :one
SELECT * FROM match_participants
WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3;

-- name: ListMatchParticipants :many
SELECT
    mp.*,
    p.first_name,
    p.last_name,
    p.email,
    tm.hdcp,
    tm.tier,
    t.color AS team_color
FROM match_participants mp
JOIN players p ON mp.player_id = p.id
JOIN team_members tm ON mp.team_id = tm.team_id AND mp.player_id = tm.player_id
JOIN teams t ON mp.team_id = t.id
WHERE mp.match_id = $1 AND mp.tenant_id = $2
ORDER BY t.color, p.last_name, p.first_name;

-- name: DeleteMatchParticipant :exec
DELETE FROM match_participants
WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3;
