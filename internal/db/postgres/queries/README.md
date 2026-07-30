# SQL Queries

sqlc input. One file per table or resource; `make sqlc` compiles these into type-safe Go in
`internal/db/postgres/internal/sqlc/`, checking them against the schema in `../migrations/`.

Never hand-edit the generated package — CI regenerates it and fails on drift.

## Conventions

**Every tenant-scoped query must filter on `tenant_id`.** That predicate is the first of the
two independent isolation layers (RLS is the second); a query that omits it leans entirely on
RLS and will pass tests that only exercise the API. Use the named form:

```sql
-- name: GetPlayer :one
SELECT * FROM players WHERE id = @id AND tenant_id = @tenant_id;
```

Named parameters (`@name`) over positional `$1` for anything with more than a couple of
arguments — the generated struct fields read better and reordering can't silently swap them.
`sqlc.narg('x')` marks a parameter nullable, which is how one query serves both a list and a
single lookup:

```sql
WHERE p.tenant_id = @tenant_id
  AND (sqlc.narg('id')::uuid IS NULL OR p.id = sqlc.narg('id'))
```

Comment *why* a non-obvious query is shaped the way it is, not what it selects — see
`PlayerRecords` in `players.sql`.

## What the config already handles

Set in `../../../../sqlc.yaml`, so don't work around these:

- `uuid` maps to `google/uuid.UUID` (and `*uuid.UUID` when nullable) — no `pgtype` conversion
  in the repositories
- date/timestamp/timestamptz map to `time.Time` / `*time.Time`
- `emit_empty_slices` — a `:many` with no rows returns `[]`, not `nil`

## Where the rules live

Retrieval belongs here; scoring rules don't. Which side won a Cup, how points are apportioned,
and how a match's result is computed live in `internal/golf/`, even where SQL could do it —
that split is deliberate and was moved this way on purpose.

See the [sqlc docs](https://docs.sqlc.dev/) for annotation reference.
