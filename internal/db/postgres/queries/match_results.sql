-- name: UpsertMatchResult :one
INSERT INTO match_results (
    match_id, tournament_id, tenant_id, finished, leader_team_id, lead, holes_remaining
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (match_id) DO UPDATE SET
    finished = EXCLUDED.finished,
    leader_team_id = EXCLUDED.leader_team_id,
    lead = EXCLUDED.lead,
    holes_remaining = EXCLUDED.holes_remaining,
    updated_at = now()
RETURNING *;

-- name: GetMatchResult :one
SELECT * FROM match_results
WHERE match_id = $1 AND tenant_id = $2;

-- name: ListTeamPoints :many
-- Ryder-cup points per team for a tournament (see the team_points view).
SELECT team_id, points FROM team_points
WHERE tournament_id = $1 AND tenant_id = $2;

-- name: IsTournamentFinished :one
SELECT EXISTS (
    SELECT 1 FROM finished_tournaments WHERE tournament_id = $1 AND tenant_id = $2
) AS finished;

-- name: GetTournamentWinner :one
-- The winning team's id, or no row when the tournament is unfinished or tied.
SELECT team_id FROM tournament_winners
WHERE tournament_id = $1 AND tenant_id = $2;

-- All-time W-L-T for every player entered in a tournament, so the roster enriches
-- without a per-player round trip. Records span every match the player has played.
-- name: ListTournamentPlayerRecords :many
SELECT
    tp.player_id,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM tournament_players tp
LEFT JOIN player_match_outcomes o ON o.player_id = tp.player_id AND o.tenant_id = tp.tenant_id
WHERE tp.tournament_id = @tournament_id AND tp.tenant_id = @tenant_id
GROUP BY tp.player_id;

-- Cups won (finished tournaments where the player's team was the sole points leader)
-- for every player entered in a tournament (finished tournaments their team won
-- outright). tournament_winners is the winning team per tournament.
-- name: ListTournamentPlayerCups :many
SELECT tp.player_id, COUNT(w.tournament_id) AS cups_won
FROM tournament_players tp
LEFT JOIN team_members tm ON tm.player_id = tp.player_id AND tm.tenant_id = tp.tenant_id
LEFT JOIN tournament_winners w ON w.tenant_id = tm.tenant_id AND w.tournament_id = tm.tournament_id AND w.team_id = tm.team_id
WHERE tp.tournament_id = @tournament_id AND tp.tenant_id = @tenant_id
GROUP BY tp.player_id;
