-- name: CreatePlayer :one
INSERT INTO players (
    tenant_id,
    user_id,
    email,
    first_name,
    last_name,
    photo_path
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- PlayerRecords returns players with their all-time W-L-T — one query for the whole list
-- (id null) or a single player (id set). Cups won is derived in the domain, since which
-- side won a Cup is a scoring rule rather than a retrieval concern.
-- name: PlayerRecords :many
SELECT
    p.*,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM players p
LEFT JOIN player_match_outcomes o ON o.player_id = p.id AND o.tenant_id = p.tenant_id
WHERE p.tenant_id = @tenant_id
  AND (sqlc.narg('id')::uuid IS NULL OR p.id = sqlc.narg('id'))
GROUP BY p.id
ORDER BY p.last_name, p.first_name;

-- name: PlayerRecordByFormat :many
-- A player's W-L-T split by the format the match was played in. Reads from
-- player_match_outcomes so the definition of a result lives in one place.
SELECT
    mf.name AS format_name,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM player_match_outcomes o
JOIN matches m ON m.id = o.match_id AND m.tenant_id = o.tenant_id
JOIN match_formats mf ON mf.id = m.match_format_id
WHERE o.player_id = @player_id AND o.tenant_id = @tenant_id
GROUP BY mf.name
ORDER BY COUNT(*) DESC, mf.name;

-- name: PlayerRecordByTeammate :many
-- Who this player has been paired with, and how the pair did. The match count is the
-- repeat-pairing signal on its own: a captain reuses partnerships, and this says whether
-- that has been working. Singles have no teammate and drop out of the join.
SELECT
    other.id AS player_id,
    other.first_name,
    other.last_name,
    COUNT(*) AS matches,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM player_match_outcomes o
JOIN match_participants me
    ON me.match_id = o.match_id AND me.player_id = o.player_id AND me.tenant_id = o.tenant_id
JOIN match_participants mate
    ON mate.match_id = me.match_id AND mate.tenant_id = me.tenant_id
   AND mate.team_id = me.team_id AND mate.player_id <> me.player_id
JOIN players other ON other.id = mate.player_id AND other.tenant_id = mate.tenant_id
WHERE o.player_id = @player_id AND o.tenant_id = @tenant_id
GROUP BY other.id, other.first_name, other.last_name
ORDER BY COUNT(*) DESC, other.last_name, other.first_name;

-- name: PlayerRecordByOpponent :many
-- Who this player has faced, and how they fared. Same shape as the teammate query but
-- across the match rather than alongside: for a pairs format every opponent is counted,
-- so one match contributes two rows.
SELECT
    other.id AS player_id,
    other.first_name,
    other.last_name,
    COUNT(*) AS matches,
    COUNT(*) FILTER (WHERE o.won) AS wins,
    COUNT(*) FILTER (WHERE o.lost) AS losses,
    COUNT(*) FILTER (WHERE o.tied) AS ties
FROM player_match_outcomes o
JOIN match_participants me
    ON me.match_id = o.match_id AND me.player_id = o.player_id AND me.tenant_id = o.tenant_id
JOIN match_participants opp
    ON opp.match_id = me.match_id AND opp.tenant_id = me.tenant_id AND opp.team_id <> me.team_id
JOIN players other ON other.id = opp.player_id AND other.tenant_id = opp.tenant_id
WHERE o.player_id = @player_id AND o.tenant_id = @tenant_id
GROUP BY other.id, other.first_name, other.last_name
ORDER BY COUNT(*) DESC, other.last_name, other.first_name;
