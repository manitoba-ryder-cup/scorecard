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

-- Remove a player from a match (leaves their team draft untouched).
-- name: DeleteMatchParticipant :execrows
DELETE FROM match_participants
WHERE match_id = $1 AND player_id = $2 AND tenant_id = $3;

-- name: ListMatchParticipants :many
SELECT * FROM match_participants
WHERE match_id = $1 AND tenant_id = $2
ORDER BY team_id, player_id;

-- Every participant across the tournament, with player names, so the results view
-- builds each match's sides without a per-match lookup.
-- name: ListParticipantsWithPlayersByTournament :many
SELECT mp.match_id, mp.team_id, mp.player_id, p.first_name, p.last_name
FROM match_participants mp
JOIN players p ON p.id = mp.player_id AND p.tenant_id = mp.tenant_id
WHERE mp.tournament_id = @tournament_id AND mp.tenant_id = @tenant_id
ORDER BY mp.match_id, mp.team_id, p.last_name, p.first_name;
