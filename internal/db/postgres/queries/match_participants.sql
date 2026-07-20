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

-- name: ListMatchParticipants :many
SELECT * FROM match_participants
WHERE match_id = $1 AND tenant_id = $2
ORDER BY team_id, player_id;
