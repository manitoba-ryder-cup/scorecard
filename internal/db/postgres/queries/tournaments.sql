-- name: CreateTournament :one
INSERT INTO tournaments (
    tenant_id,
    name,
    start_date,
    end_date,
    location
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetTournament :one
SELECT * FROM tournaments
WHERE id = $1 AND tenant_id = $2;

-- name: ListTournaments :many
SELECT * FROM tournaments
WHERE tenant_id = $1
ORDER BY start_date DESC;
