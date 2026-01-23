-- name: CreateTeamMember :one
INSERT INTO team_members (
    tournament_id,
    player_id,
    team_id,
    tenant_id,
    is_captain
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetTeamMember :one
SELECT * FROM team_members
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;

-- name: ListTeamMembersByTournament :many
SELECT tm.*, p.first_name, p.last_name, p.email, t.name as team_name
FROM team_members tm
JOIN players p ON tm.player_id = p.id
JOIN teams t ON tm.team_id = t.id
WHERE tm.tournament_id = $1 AND tm.tenant_id = $2
ORDER BY t.name, p.last_name, p.first_name;

-- name: ListTeamMembersByTeam :many
SELECT tm.*, p.first_name, p.last_name, p.email
FROM team_members tm
JOIN players p ON tm.player_id = p.id
WHERE tm.tournament_id = $1 AND tm.team_id = $2 AND tm.tenant_id = $3
ORDER BY p.last_name, p.first_name;

-- name: GetTeamCaptain :one
SELECT tm.*, p.first_name, p.last_name, p.email
FROM team_members tm
JOIN players p ON tm.player_id = p.id
WHERE tm.tournament_id = $1 AND tm.team_id = $2 AND tm.is_captain = true AND tm.tenant_id = $3
LIMIT 1;

-- name: UpdateTeamMemberCaptain :one
UPDATE team_members
SET is_captain = $4
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3
RETURNING *;

-- name: DeleteTeamMember :exec
DELETE FROM team_members
WHERE tournament_id = $1 AND player_id = $2 AND tenant_id = $3;
