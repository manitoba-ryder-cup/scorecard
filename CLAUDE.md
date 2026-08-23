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

- `make unit` — unit tests with the race detector (excludes `./test`, which needs infrastructure)
- `make test-setup` — build and start postgres + scorecard test infrastructure
- `make integration` — run `./test/...` (needs `make test-setup` first)
- `make test` — test-setup + unit + integration
- `make test-teardown` — stop the test stack and drop its volumes
- `make lint` — golangci-lint via Docker
- `make fmt` — gofmt, goimports, then gci to enforce import grouping
- `make coverage` — HTML coverage report
- `make help` — list every target

Run `make fmt` before `make build` to resolve imports.

CI runs eight jobs on every push and PR: format, lint, build, test (unit + integration),
govulncheck, sqlc-drift, docker build, and a migrate up/down/up cycle against a live Postgres.
The Go minor is pinned in the workflow, not read from `go.mod` — bumping Go means editing both.

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
  ├── server.go          # Builds the http.Server; shutdown ordering
  ├── services.go        # Builds repositories and domain services
  └── seed.go            # Tournament seeding logic

internal/api/rest/       # HTTP layer (stdlib http.ServeMux, method-prefixed patterns)
  ├── router.go          # Router, its Handler, the middleware chain, optional auth
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
- **Tournament** — the event; created with both teams in one transaction, and carries its
  location. It has no timezone — that moved to the course. Its `phase` (upcoming / live /
  finished) is derived on read from its match outcomes, never stored — live from the first
  score recorded until the last match is final, so a cup whose start date has arrived is
  still upcoming until someone scores. It also picks the endpoint's cache tier
- **Team** — Red or Blue, with an optional captain
- **TournamentPlayer** — a player entered in a tournament, carrying the per-tournament tier,
  biography, and handicap (these are *not* on Player)
- **TeamMember** — the draft outcome: an entered player assigned to a team
- **Match / MatchParticipant / Score** — a match, its two sides, and hole scores
- **MatchResult** — the materialized per-match outcome; team points, standings, and player
  records all derive from it
- **Course / Hole / TeeSet / TeeColor / MatchFormat** — course setup and reference data. A
  course owns the IANA time zone its tee times are entered in (`sdk.DefaultTimeZone`,
  `America/Winnipeg`, when unnamed); tee times are stored and served as UTC instants
- **PlayerStats** — per-format and per-partner/opponent records, cup points, and how matches
  end (went the distance vs. closed out early, plus the heaviest result each way). Derived on
  read like the all-time record. A halved match can only land in `last_hole`, since a half
  requires playing the 18th

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

### The Scoring Window

Scores are accepted only from `scoringOpensBefore` (2h) before a match's tee time until
`scoringClosesAfter` (12h) after it — both in `internal/golf/match.go`, both measured from the
tee time, which is an instant and so needs no timezone. Outside the window a write is
`ErrConflict` (409), and the error names the window's bounds.

The window is a scorer's guard, not the tournament's rule: it stops a phone on a fairway
recording against the wrong match. A token carrying `tournaments:write` is exempt, decided in
the handler from the token's scopes and passed down as `SubmitScoresInput.IgnoreScoringWindow`
— the domain never learns who the caller is. Correcting a card the next morning is the case,
and the alternative was moving the tee time, which is published on the match and derives the
window, so it left the match misscheduled unless it was moved back.

`Match.TeeTime` is required for exactly this reason. `MatchResult` also carries the window
itself as `scoring_opens_at`/`scoring_closes_at`, derived by `golf.ScoringWindow` — the one
place either bound is computed, including for the 409 message. The web client gates its UI on
those fields rather than keeping its own copy of the constants, which is what it used to do:
the rule then lived in two repos with nothing keeping them equal, and a client stricter than
this side would silently offer no way to record a legitimate score.

Published as two instants rather than a boolean, because a yes/no is only true when it was
computed and score entry is left open on a fairway for hours. Changing a constant here now
changes the client's behaviour on its next read, with no release on that side.

## Transactions

A write that spans several tables belongs in **one repository method, inside one
`withTenant` closure** — that closure is the transaction. `CreateTournamentWithTeams` and
`SeedDB.SeedTournament` are the examples: the caller asks for a tournament or a seeded
event and never learns a transaction was involved.

Do not open transactions above the repository layer. An earlier attempt threaded one down
from `internal/app` through the context, which made every repository call in the app change
behaviour based on invisible state. Where two repository methods both need the same writes,
extract a helper taking `*sqlc.Queries` and let each closure compose it — see
`createTournamentWithTeams`.

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

`ScoresDB.SaveScoresAndRecompute` is the only writer of `match_results` (`results.go` reads
only). A whole hole is one unit: it takes every score on that hole, and in one transaction
takes a `FOR UPDATE` lock on the match (`LockMatchForScoring`), re-reads the committed
scores, runs the domain's `guard`, writes, re-reads, and upserts the `recompute`d result.

`ScoresDB.ResetMatch` is the only other writer: it deletes a match's scores and its
`match_results` row together, behind the same lock, in the same order, and for the same
reason. The result row is deleted rather than zeroed — its existence is what marks a match
started.

**Nothing else may delete a score.** `scores` referenced `match_participants` with
`ON DELETE CASCADE`, which let two routes destroy a played match without anything
recomputing what was left behind — removing a participant, and undrafting a player, which
reaches participants by cascade. Either left `match_results` claiming a finished match whose
scores were gone, so one cup read as finished and never-played at once depending on the
endpoint. Both are refused now, in two places that do not cover the same ground:

- The domain guards (`MatchService.RemoveParticipant`, `RosterService.UndraftPlayer`) refuse
  with `ErrConflict`, and are the complete rule — they hold for every scoring grain.
- The `ON DELETE RESTRICT` from migration 003 is the backstop, and covers **only per-player
  scores**. A one-ball format records against the team with a null `player_id`, which the
  foreign key skips (`MATCH SIMPLE`), so alt shot and scramble rest on the guard alone.

Two things a reset does not undo. It ignores the scoring window but score entry does not,
so clearing a match played yesterday leaves it uneditable until its tee time moves — the
same `PUT /v1/matches/{id}` a group that went out late needs, and it has to be moved back
afterwards or the match is permanently misscheduled. And resetting a *settled* cup strands
the edge: `/tournaments/{id}`, `/teams` and `/results` hold the old standing for up to
`settledMaxAge` while `/v1/tournaments` reports the new phase within the minute, so the two
disagree until that expires. The service has no purge hook, which is why that tier is an
hour rather than a day.

**Do not split these steps, and keep the lock before the first read.** A single transaction
is not enough on its own: under READ COMMITTED two concurrent submissions can each recompute
from a snapshot missing the other's score, and the later write reverts the result to a
partial view. The lock-then-read order is also what stops the guard's decision being raced —
it must see the same committed set the recompute will. Both the scoring rule and the guard
arrive as callbacks, so the persistence layer owns only the transaction and the lock.
Covered by `test/concurrency_test.go`.

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
- The API layer holds the domain services concretely and is not unit-tested through fakes:
  it translates, and everything it does is visible from outside, so the integration suite
  is where its correctness is established. What stays in `rest` are tests needing no
  domain at all — status mapping, cache tiers, decode policy, DTO shape
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

The service serves `/v1/...` and `/health`. The public site reaches it through the edge at
`/api/scorecard/*`, which strips that prefix. Route constants live in `sdk/routes.go` and
`internal/api/rest/router.go` registers every route from them — there are no path
literals in the routing table, so a route change is a compile-time concern. Keep it that
way.

A match's `/winner` and `/status` are deliberately the same handler. They started as two
endpoints and were collapsed when a match got one outcome shape covering both a finished match
and one still in progress; `/status` is the name the handler kept. Both are still routed, so
neither can be dropped without a client change.

## Development Guidelines

- Keep comments minimal and focused on *why*. The code should speak for itself; do not
  narrate what a function does. One line where at all possible. Write for someone who
  never saw the change: "used to" earns its place only where it warns off a path they
  might take again, and anything else about how the code got this way is a commit message.
- No `Co-Authored-By` trailer and no generated-with footer, in commits or PR descriptions.
  The repo squash-merges with the PR body as the message, so anything in it lands in the
  log — write PR descriptions as prose for that reason.
- Not yet released — no backwards compatibility required, and existing migrations can be
  edited in place rather than adding new ones.
- The web client (sibling `web/` repository) consumes the SDK's wire types. Changing a
  response shape means updating `web/src/api/types.ts` too.
- The match format *names* seeded in `002_seed_match_formats.up.sql` are matched by string in
  the web client, so renaming one silently breaks match creation.
