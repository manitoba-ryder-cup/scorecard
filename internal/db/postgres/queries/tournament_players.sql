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
SELECT ins.*, p.first_name, p.last_name, p.email, p.photo_path, tm.team_id
FROM ins
JOIN players p ON ins.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = ins.tournament_id AND tm.player_id = ins.player_id;

-- UpdateTournamentPlayer updates attributes and returns the enriched entry.
-- name: UpdateTournamentPlayer :one
WITH upd AS (
    UPDATE tournament_players
    SET tier = $4, biography = $5, hdcp = $6, updated_at = now()
    WHERE tournament_players.tournament_id = $1
      AND tournament_players.player_id = $2
      AND tournament_players.tenant_id = $3
    RETURNING *
)
SELECT upd.*, p.first_name, p.last_name, p.email, p.photo_path, tm.team_id
FROM upd
JOIN players p ON upd.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = upd.tournament_id AND tm.player_id = upd.player_id;

-- ListTournamentPlayers returns every entered player, enriched.
-- name: ListTournamentPlayers :many
SELECT tp.*, p.first_name, p.last_name, p.email, p.photo_path, tm.team_id
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
LEFT JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id
WHERE tp.tournament_id = $1 AND tp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- ListTournamentPlayersByTeam returns the same rows filtered to one team's drafted
-- players.
-- name: ListTournamentPlayersByTeam :many
SELECT tp.*, p.first_name, p.last_name, p.email, p.photo_path, tm.team_id
FROM tournament_players tp
JOIN players p ON tp.player_id = p.id
JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id
WHERE tm.team_id = $1 AND tp.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- A player's tournament history with their team (via its captain) and per-tournament
-- W-L-T. LEFT JOINs throughout, since a player can be entered but undrafted. The
-- won/lost/tied verdict needs the standings, so the caller derives it separately.
-- name: ListPlayerTournaments :many
SELECT
    t.id AS tournament_id,
    t.name,
    t.location,
    t.start_date,
    t.end_date,
    tm.team_id,
    cap.first_name AS captain_first_name,
    cap.last_name  AS captain_last_name,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id = mp.team_id) AS wins,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NOT NULL AND mr.leader_team_id <> mp.team_id) AS losses,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NULL) AS ties
FROM tournament_players tp
JOIN tournaments t ON t.id = tp.tournament_id AND t.tenant_id = tp.tenant_id
LEFT JOIN team_members tm ON tm.tournament_id = tp.tournament_id AND tm.player_id = tp.player_id AND tm.tenant_id = tp.tenant_id
LEFT JOIN teams te ON te.id = tm.team_id AND te.tenant_id = tm.tenant_id
LEFT JOIN players cap ON cap.id = te.captain_id AND cap.tenant_id = te.tenant_id
LEFT JOIN match_participants mp ON mp.player_id = tp.player_id AND mp.tournament_id = tp.tournament_id AND mp.tenant_id = tp.tenant_id
LEFT JOIN match_results mr ON mr.match_id = mp.match_id AND mr.tenant_id = mp.tenant_id
WHERE tp.player_id = @player_id AND tp.tenant_id = @tenant_id
GROUP BY t.id, t.name, t.location, t.start_date, t.end_date, tm.team_id, cap.first_name, cap.last_name
ORDER BY t.start_date DESC;
