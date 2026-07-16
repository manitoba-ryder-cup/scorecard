-- Match formats are global, code-defined reference data (seeded, not tenant-scoped),
-- so these reads take no tenant_id and there is no create/update/delete.

-- name: GetMatchFormat :one
SELECT * FROM match_formats
WHERE id = $1;

-- name: ListMatchFormats :many
SELECT * FROM match_formats
ORDER BY id;
