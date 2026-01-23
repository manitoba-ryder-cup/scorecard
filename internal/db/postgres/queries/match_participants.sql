-- name: CreateMatchParticipant :one
INSERT INTO match_participants (
    tournament_id,
    match_id,
    player_id,
    tenant_id
) VALUES (
    $1, $2, $3, $4
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
    p.hdcp,
    p.tier
FROM match_participants mp
JOIN players p ON mp.player_id = p.id
WHERE mp.match_id = $1 AND mp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- name: ListMatchParticipantsWithTeam :many
SELECT
    mp.*,
    p.first_name,
    p.last_name,
    p.email,
    p.hdcp,
    p.tier,
    t.name as team_name
FROM match_participants mp
JOIN players p ON mp.player_id = p.id
JOIN team_members tm ON mp.tournament_id = tm.tournament_id AND mp.player_id = tm.player_id
JOIN teams t ON tm.team_id = t.id
WHERE mp.match_id = $1 AND mp.tenant_id = $2
ORDER BY t.name, p.last_name, p.first_name;

-- name: DeleteMatchParticipant :exec
DELETE FROM match_participants
WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3;
