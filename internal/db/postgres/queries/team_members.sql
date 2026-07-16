-- team_members records the draft: an entered player assigned to a team. Per-tournament
-- attributes (tier/biography/hdcp) live on tournament_players, not here.

-- name: CreateTeamMember :one
INSERT INTO team_members (
    team_id,
    player_id,
    tournament_id,
    tenant_id
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetTeamMember :one
SELECT * FROM team_members
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;

-- name: ListTeamMembersByTournament :many
SELECT tm.*, p.first_name, p.last_name, p.email, t.color AS team_color
FROM team_members tm
JOIN players p ON tm.player_id = p.id
JOIN teams t ON tm.team_id = t.id
WHERE tm.tournament_id = $1 AND tm.tenant_id = $2
ORDER BY t.color, p.last_name, p.first_name;

-- name: ListTeamMembersByTeam :many
SELECT tm.*, p.first_name, p.last_name, p.email
FROM team_members tm
JOIN players p ON tm.player_id = p.id
WHERE tm.team_id = $1 AND tm.tenant_id = $2
ORDER BY p.last_name, p.first_name;

-- name: GetTeamCaptain :one
SELECT p.*
FROM teams t
JOIN players p ON t.captain_id = p.id
WHERE t.id = $1 AND t.tenant_id = $2;

-- name: DeleteTeamMember :exec
DELETE FROM team_members
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;
