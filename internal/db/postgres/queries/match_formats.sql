-- Match formats are global seeded reference data (not tenant-scoped), so these reads take
-- no tenant_id and there is no create/update/delete.

-- Sorted by name: the ids are gen_random_uuid(), so ordering by them shuffles
-- differently in every database.
-- name: ListMatchFormats :many
SELECT * FROM match_formats
ORDER BY name;

-- name: GetMatchFormat :one
SELECT * FROM match_formats
WHERE id = $1;
