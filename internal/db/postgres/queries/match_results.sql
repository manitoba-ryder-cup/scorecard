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

-- Every match in the tournament with its stored outcome. A match with no result row has
-- not been scored, so it reads as neither started nor finished — the standings rules live
-- in the domain and take these rows as input.
-- name: ListMatchOutcomes :many
SELECT
    (mr.match_id IS NOT NULL)::boolean AS started,
    COALESCE(mr.finished, false)::boolean AS finished,
    mr.leader_team_id
FROM matches m
LEFT JOIN match_results mr ON mr.match_id = m.id AND mr.tenant_id = m.tenant_id
WHERE m.tournament_id = @tournament_id AND m.tenant_id = @tenant_id;

-- Every tournament's match outcomes, for the all-time aggregates the domain derives
-- (which Cup each side won).
-- name: ListAllMatchOutcomes :many
SELECT
    m.tournament_id,
    (mr.match_id IS NOT NULL)::boolean AS started,
    COALESCE(mr.finished, false)::boolean AS finished,
    mr.leader_team_id
FROM matches m
LEFT JOIN match_results mr ON mr.match_id = m.id AND mr.tenant_id = m.tenant_id
WHERE m.tenant_id = @tenant_id;

-- Deleted rather than zeroed: the row's existence is what marks a match started.
-- name: DeleteMatchResult :execrows
DELETE FROM match_results
WHERE match_id = @match_id AND tenant_id = @tenant_id;
