-- team_members records the draft: an entered player assigned to a team. Per-tournament
-- attributes (tier/biography/hdcp) live on tournament_players. Roster listings are
-- served from the tournament_players queries (which carry the team assignment).

-- name: CreateTeamMember :one
INSERT INTO team_members (
    team_id,
    player_id,
    tournament_id,
    tenant_id
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- Undraft a player. ON DELETE CASCADE pulls them from any match_participants too.
-- name: DeleteTeamMember :execrows
DELETE FROM team_members
WHERE team_id = $1 AND player_id = $2 AND tenant_id = $3;

-- Every draft record, linking a Cup win to the players who won it.
-- name: ListTeamMemberships :many
SELECT player_id, tournament_id, team_id FROM team_members
WHERE tenant_id = $1;

-- Whether undrafting this player would take scores with it. Joined on the match rather
-- than the participant row, so a one-ball format counts too: those scores carry no player
-- and so are invisible to the foreign key that guards the per-player case.
-- name: PlayerHasScoredMatches :one
SELECT EXISTS (
    SELECT 1
    FROM match_participants mp
    JOIN scores s ON s.match_id = mp.match_id AND s.tenant_id = mp.tenant_id
    WHERE mp.team_id = @team_id AND mp.player_id = @player_id AND mp.tenant_id = @tenant_id
)::boolean;
