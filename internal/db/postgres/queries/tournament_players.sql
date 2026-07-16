-- tournament_players holds a player's per-tournament attributes (tier, biography,
-- handicap), set independently of the team draft.

-- name: CreateTournamentPlayer :one
INSERT INTO tournament_players (
    tournament_id,
    player_id,
    tenant_id,
    tier,
    biography,
    hdcp
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetTournamentPlayer :one
SELECT * FROM tournament_players
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;

-- name: ListTournamentPlayers :many
SELECT tp.*, p.first_name, p.last_name, p.email, p.photo_path
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
WHERE tp.tournament_id = $1 AND tp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- name: UpdateTournamentPlayer :one
UPDATE tournament_players
SET tier = $4, biography = $5, hdcp = $6, updated_at = now()
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3
RETURNING *;

-- name: DeleteTournamentPlayer :exec
DELETE FROM tournament_players
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;
