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
