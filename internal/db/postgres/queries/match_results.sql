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
-- Ryder-cup points per team for a tournament: 1 for a won match, 0.5 for a tie.
-- A tie awards 0.5 to both sides because every finished match joins to both teams.
SELECT
    t.id AS team_id,
    COALESCE(SUM(
        CASE
            WHEN mr.finished AND mr.leader_team_id = t.id THEN 1.0
            WHEN mr.finished AND mr.leader_team_id IS NULL THEN 0.5
            ELSE 0
        END
    ), 0)::float8 AS points
FROM teams t
LEFT JOIN match_results mr ON mr.tournament_id = t.tournament_id
WHERE t.tournament_id = $1 AND t.tenant_id = $2
GROUP BY t.id;

-- name: IsTournamentFinished :one
-- Finished when the tournament has at least one match and all are finished.
SELECT
    EXISTS (SELECT 1 FROM matches m WHERE m.tournament_id = $1 AND m.tenant_id = $2)
    AND NOT EXISTS (
        SELECT 1 FROM matches m
        LEFT JOIN match_results mr ON mr.match_id = m.id
        WHERE m.tournament_id = $1 AND m.tenant_id = $2
          AND COALESCE(mr.finished, false) = false
    ) AS finished;

-- The batched form of GetPlayerRecord: all-time W-L-T for every player entered in a
-- tournament, so the roster enriches without a per-player round trip.
-- name: ListTournamentPlayerRecords :many
SELECT
    tp.player_id,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id = mp.team_id) AS wins,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NOT NULL AND mr.leader_team_id <> mp.team_id) AS losses,
    COUNT(*) FILTER (WHERE mr.finished AND mr.leader_team_id IS NULL) AS ties
FROM tournament_players tp
LEFT JOIN match_participants mp ON mp.player_id = tp.player_id AND mp.tenant_id = tp.tenant_id
LEFT JOIN match_results mr ON mr.match_id = mp.match_id AND mr.tenant_id = mp.tenant_id
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
