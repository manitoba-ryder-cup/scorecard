# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Scorecard is a Go REST API for the Manitoba Ryder Cup: live tournament scoring, match play
formats, and player records. It uses PostgreSQL, JWT authentication issued by Heimdall, and
runs in Docker.

This is a port of the Python/Flask application in the sibling `scorecardpy/` repository.

## Common Commands

### Building

- `make dev` — development binary (debug symbols, faster)
- `make build` — production binary (optimized, stripped)

### Code Generation

- `make sqlc` — regenerate type-safe Go from `internal/db/postgres/queries/*.sql`

### Testing and Quality

- `make test` — unit tests with the race detector (excludes `./test`, which needs infrastructure)
- `make test-setup` — build and start postgres + scorecard test infrastructure
- `make integration` — run `./test/...` (needs `make test-setup` first)
- `make test-teardown` — stop the test stack and drop its volumes
- `make lint` — golangci-lint via Docker
- `make fmt` — gofmt + goimports

Run `make fmt` before `make build` to resolve imports.

### Database Migrations

```bash
./bin/scorecard migrate up --database-url "postgres://..."
./bin/scorecard migrate down --database-url "postgres://..."
./bin/scorecard migrate version --database-url "postgres://..."
```

The server also runs pending migrations on startup.

### Running the Service

```bash
./bin/scorecard start \
  --database-url "postgres://scorecard:password@localhost:5432/scorecard?sslmode=disable" \
  --http-address ":5000" \
  --jwt-public-key "/path/to/heimdall.pub" \
  --environment "development"
```

### Seeding a Tournament

`./bin/scorecard seed-tournament --tenant-id <uuid> --file setup.json` creates a tournament,
its roster and captains, and the match schedule from JSON. It does not draft the field or
assign match participants — those happen live at the event.

## Architecture Overview

### Multi-Tenancy with Row-Level Security

Tenant isolation has two independent layers, and both must stay intact:

1. Every query carries an explicit `tenant_id = $n` predicate.
2. PostgreSQL RLS policies filter by `app.current_tenant_id`, set per transaction by
   knowhere's `WithTenantContext`.

**`ALTER TABLE ... FORCE ROW LEVEL SECURITY` in `001_init.up.sql` is load-bearing — do not
remove it.** Postgres exempts a table's owner from its policies, and the application role
owns every table because it runs the migrations. Without `FORCE`, `ENABLE` alone is silently
skipped and layer 2 does nothing. `test/isolation/rls_test.go` guards this by querying as the
application role with no tenant predicate; note that the API-level tests in
`test/isolation/api_test.go` pass either way, because layer 1 already filtered the rows.

The application role must never be a superuser or hold `BYPASSRLS`.

### Layered Architecture

```txt
cmd/scorecard/           # CLI entry point (urfave/cli)
  ├── main.go            # CLI app setup, slog configuration
  ├── start.go           # Server start command
  ├── migrate.go         # Migration commands
  ├── seed.go            # seed-tournament command
  ├── version.go
  ├── flags.go
  └── config.go

internal/app/            # Server lifecycle and dependency wiring
  ├── server.go          # Coordinates HTTP + DB; shutdown ordering
  ├── services.go        # Builds repositories and domain services
  └── seed.go            # Tournament seeding logic

internal/api/rest/       # HTTP layer (stdlib http.ServeMux, method-prefixed patterns)
  ├── server.go          # Routes, middleware chain, optional auth
  ├── dto.go             # Domain -> SDK DTO mapping
  ├── json.go            # Response and error helpers
  ├── players.go, tournaments.go, matches.go, courses.go,
  │   roster.go, teams.go, formats.go, health_handler.go

internal/golf/           # Domain layer (business logic, no tenant awareness)
  ├── models.go          # Domain entities
  ├── scoring.go         # Match-play scoring engine (pure functions)
  ├── match.go, tournament.go, roster.go, player.go, team.go, course.go, format.go
  ├── dependencies.go    # Repository interfaces the services depend on
  └── errors.go          # Sentinel errors

internal/db/postgres/    # Data access layer
  ├── db.go              # Connection pool (knowhere wrapper)
  ├── tenant.go          # withTenant / withTenantExec helpers
  ├── errors.go          # SQLSTATE -> domain sentinel mapping
  ├── migrations/        # SQL schema (golang-migrate)
  ├── queries/           # SQL queries (sqlc input)
  └── internal/sqlc/     # Generated code — never edit by hand

sdk/                     # Public API contract: DTOs, routes, scopes, HTTP client
test/                    # Integration suite (needs the docker-compose stack)
  └── isolation/         # Cross-tenant isolation suite
```

### Golf Domain Models

- **Player** — stable identity (name, optional heimdall `user_id`, optional email) plus an
  all-time record and cups won, both derived on read from `match_results`
- **Tournament** — the event; created with both teams in one transaction
- **Team** — Red or Blue, with an optional captain
- **TournamentPlayer** — a player entered in a tournament, carrying the per-tournament tier,
  biography, and handicap (these are *not* on Player)
- **TeamMember** — the draft outcome: an entered player assigned to a team
- **Match / MatchParticipant / Score** — a match, its two sides, and hole scores
- **MatchResult** — the materialized per-match outcome; team points, standings, and player
  records all derive from it
- **Course / Hole / TeeSet / TeeColor / MatchFormat** — course setup and reference data

### Match Scoring

`internal/golf/scoring.go` holds the scoring engine as pure functions:

- A team's gross score on a hole is the minimum strokes recorded for that team, which covers
  singles (one score), fourball (best of two), and one-ball formats (a single team score)
- Only holes scored by both teams count, and the progression stops at the hole where the
  match is closed out
- Results are colour-free state (team IDs, leader, lead, holes remaining); rendering "2 UP"
  or "3 & 2" is the client's job

**Handicapping is not implemented.** `matches.handicapped` and the stroke indexes on `holes`
are stored so the schema supports it, but scoring is gross only. Do not describe the API as
supporting handicapped play.

## Important Patterns

### Authentication and Authorization

JWTs are issued by Heimdall and verified with its RSA public key.

- Writes require a scope (`sdk.Scope*`) via `jwtMiddleware.RequireScope`
- Reads use `optionalAuth`: a token's tenant when present, otherwise the configured
  `PublicTenantID` for anonymous spectators, else 401
- An anonymous request has a tenant but **no actor** — `identity.GetActor` erroring is the
  signal that a request is unauthenticated

### Database Access

Repositories use the helpers in `internal/db/postgres/tenant.go`:

```go
// Returns a value
return withTenant(ctx, db, func(q *sqlc.Queries, tenantID uuid.UUID) (*golf.Player, error) {
    ...
})

// Returns only an error
return withTenantExec(ctx, db, func(q *sqlc.Queries, tenantID uuid.UUID) error {
    ...
})
```

One `withTenant` call is one transaction. Operations that must be atomic belong in a single
closure.

### The Score Write Path

`ScoresDB.SaveScoreAndRecompute` is the only writer of `match_results`. It takes a
`FOR UPDATE` lock on the match, writes the score, re-reads every score, and upserts the
recomputed result — all in one transaction. **Do not split these steps.** A single
transaction is not enough on its own: under READ COMMITTED two concurrent submissions can
each recompute from a snapshot missing the other's score, and the later write reverts the
result to a partial view. The domain passes its scoring rule in as a callback so the
persistence layer owns only the transaction and the lock. Covered by
`test/concurrency_test.go`.

### Code Generation with sqlc

1. Write queries with sqlc annotations in `internal/db/postgres/queries/*.sql`
2. Run `make sqlc`
3. Never hand-edit `internal/db/postgres/internal/sqlc/` — CI regenerates and fails on drift

### Error Handling

- Sentinels live in `internal/golf/errors.go`: `ErrNotFound`, `ErrInvalidInput`, `ErrConflict`
- `internal/db/postgres/errors.go` maps SQLSTATEs to them: `23505` -> conflict,
  `23503` -> invalid input, `22001` -> invalid input, no-rows -> not found
- Handlers call `respondDomainError`, which maps sentinels to 404 / 400 / 409 / 500
- Wrap with context: `fmt.Errorf("failed to X: %w", err)`

### JSON Serialization

- **snake_case** field names, defined once on the SDK types in `sdk/types.go`
- Handlers must never serialize domain structs — map to SDK DTOs in `dto.go`
- Empty slices serialize as `[]`, not `null`
- **Email is write-only.** It is accepted on `CreatePlayerRequest` and kept as the seed CLI's
  key for recognising a returning player, but it appears on no response type. Reads are
  public, so returning it would publish the roster's contact details.

### Shutdown

`app.Server.Shutdown` drains HTTP before closing the pool — `http.Server.Shutdown` blocks
until in-flight handlers return and those handlers are still querying. `shutdownErr` in
`cmd/scorecard/start.go` treats `http.ErrServerClosed` and `context.Canceled` as success, so
an ordinary SIGTERM exits zero.

### Testing

- Table-driven tests for domain logic; the scoring engine is pure and needs no fakes
- Domain services take repository interfaces (`dependencies.go`), faked in `*_test.go`
- Integration tests drive the real stack through the SDK client; `test/_util/request` sends
  raw HTTP so negative tests can prove the server validates independently of the SDK
- Each integration test uses a fresh random tenant, so no inter-test cleanup is needed
- Run with `-race`

## Configuration

| Variable | Flag | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | `--database-url` | local dev URL | PostgreSQL connection string |
| `HTTP_ADDRESS` | `--http-address` | `:5000` | HTTP bind address |
| `JWT_PUBLIC_KEY_PATH` | `--jwt-public-key` | _required_ | Heimdall RSA public key (PEM) |
| `ENVIRONMENT` | `--environment` | `development` | Deployment environment |
| `DEBUG` | `--debug` | `false` | Debug logging |
| `LOG_FORMAT` | `--log-format` | `json` | `json` or `text` |
| `TRUSTED_PROXY_MODE` | `--trusted-proxy-mode` | `false` | Trust `X-Forwarded-For` |
| `PROXY_SECRET` | `--proxy-secret` | _empty_ | Require `X-Proxy-Secret` on all non-health requests |
| `SCORECARD_PUBLIC_TENANT_ID` | `--public-tenant-id` | _empty_ | Tenant for anonymous reads; empty requires auth for everything |

## URL Structure

The service serves `/v1/...` and `/healthz`. The public site reaches it through the edge at
`/api/scorecard/*`, which strips that prefix. Route constants live in `sdk/routes.go` — note
that `internal/api/rest/server.go` currently repeats most paths as literals rather than using
them.

## Development Guidelines

- Keep comments minimal and focused on *why*. The code should speak for itself; do not
  narrate what a function does.
- Not yet released — no backwards compatibility required, and existing migrations can be
  edited in place rather than adding new ones.
- The web client (sibling `web/` repository) consumes the SDK's wire types. Changing a
  response shape means updating `web/src/api/types.ts` too.
- The match format *names* seeded in `002_seed_match_formats.up.sql` are matched by string in
  the web client, so renaming one silently breaks match creation.
