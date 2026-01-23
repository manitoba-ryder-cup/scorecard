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

-- name: ListTournamentsByDateRange :many
SELECT * FROM tournaments
WHERE tenant_id = $1
  AND start_date >= $2
  AND end_date <= $3
ORDER BY start_date DESC;

-- name: UpdateTournament :one
UPDATE tournaments
SET
    name = $3,
    start_date = $4,
    end_date = $5,
    location = $6
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteTournament :exec
DELETE FROM tournaments
WHERE id = $1 AND tenant_id = $2;
