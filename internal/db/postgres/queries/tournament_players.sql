-- tournament_players holds a player's per-tournament attributes (tier, biography,
-- handicap), set independently of the team draft. All reads/writes return the same
-- enriched shape: the entry plus the player's identity and team assignment (team_id
-- NULL when entered but undrafted).

-- CreateTournamentPlayer inserts the entry and returns it enriched with identity/team.
-- name: CreateTournamentPlayer :one
WITH ins AS (
    INSERT INTO tournament_players (tournament_id, player_id, tenant_id, tier, biography, hdcp)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING *
)
SELECT ins.*, p.first_name, p.last_name, p.photo_path, tm.team_id
FROM ins
JOIN players p ON ins.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = ins.tournament_id AND tm.player_id = ins.player_id;

-- UpdateTournamentPlayer writes only the attributes that were supplied: a null argument
-- leaves the column alone, so setting a biography never has to restate the tier and
-- handicap. Most entries carry a handicap, and echoing one back is how it gets lost.
--
-- Separate from the enriched read below rather than one CTE, because sqlc cannot analyse
-- an UPDATE inside a CTE once its SET clause references the columns it is updating. The
-- repository runs both in a single transaction, so this costs a statement, not a trip.
-- name: UpdateTournamentPlayer :one
UPDATE tournament_players
SET tier = COALESCE(sqlc.narg('tier'), tier),
    biography = COALESCE(sqlc.narg('biography'), biography),
    hdcp = COALESCE(sqlc.narg('hdcp'), hdcp),
    updated_at = now()
WHERE tournament_id = sqlc.arg('tournament_id')
  AND player_id = sqlc.arg('player_id')
  AND tenant_id = sqlc.arg('tenant_id')
RETURNING player_id;

-- GetTournamentPlayer returns one entered player, enriched the same way the list is.
-- name: GetTournamentPlayer :one
SELECT tp.*, p.first_name, p.last_name, p.photo_path, tm.team_id
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id
WHERE tp.tournament_id = $1 AND tp.player_id = $2 AND tp.tenant_id = $3;

-- ListTournamentPlayers returns every entered player, enriched.
-- name: ListTournamentPlayers :many
SELECT tp.*, p.first_name, p.last_name, p.photo_path, tm.team_id
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id
WHERE tp.tournament_id = $1 AND tp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- ListTournamentPlayersByTeam returns the same rows filtered to one team's drafted
-- players.
-- name: ListTournamentPlayersByTeam :many
SELECT tp.*, p.first_name, p.last_name, p.photo_path, tm.team_id
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id
WHERE tm.team_id = $1 AND tp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- A player's tournament history: their side that year (via its captain) and their
-- per-tournament W-L-T. The verdict for that side is derived in the domain from the
-- returned team_id. LEFT JOINs throughout, since a player can be entered but undrafted.
-- name: ListPlayerTournaments :many
SELECT
    t.id AS tournament_id,
    t.name,
    t.location,
    t.start_date,
    t.end_date,
    cap.first_name AS captain_first_name,
    cap.last_name  AS captain_last_name,
    tm.team_id,
    tp.tier,
    tp.biography,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM tournament_players tp
JOIN tournaments t ON t.id = tp.tournament_id AND t.tenant_id = tp.tenant_id
LEFT JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id AND tm.tenant_id = tp.tenant_id
LEFT JOIN teams te ON te.id = tm.team_id AND te.tenant_id = tm.tenant_id
LEFT JOIN players cap ON cap.id = te.captain_id AND cap.tenant_id = te.tenant_id
LEFT JOIN player_match_outcomes o ON o.player_id = tp.player_id AND o.tournament_id = tp.tournament_id AND o.tenant_id = tp.tenant_id
WHERE tp.player_id = @player_id AND tp.tenant_id = @tenant_id
GROUP BY t.id, t.name, t.location, t.start_date, t.end_date, tm.team_id, cap.first_name, cap.last_name, tp.tier, tp.biography
ORDER BY t.start_date DESC;
