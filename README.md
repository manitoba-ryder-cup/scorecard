# Scorecard

A REST API for golf tournament scoring and management, built in Go. It powers the Manitoba
Ryder Cup: live match-play scoring, team standings, and player records.

This is a ground-up rewrite of the original Python/Flask application.

---

## Features

- **Live match-play scoring** — hole-by-hole state (leader, lead, holes remaining, closed out)
- **Best-ball and one-ball formats** — singles and fourball score per player; alt shot,
  scramble, and scotch score per team
- **Materialized results** — every score write recomputes the match outcome, so team points,
  standings, and player records derive from one small table instead of rescanning scores
- **Player records** — all-time wins/losses/ties and cups won, derived on read (never stored,
  so they cannot go stale)
- **Course management** — courses, tee sets, and per-tee hole configuration
- **Public reads** — anonymous spectators can read a configured tenant; writes require a
  scoped token
- **Multi-tenancy** — PostgreSQL Row-Level Security behind explicit tenant predicates
- **JWT authentication** — integrated with the Heimdall authentication service
- **JSON API** — snake_case, with a typed Go client in `sdk/`

Handicapped play is **not** implemented. The schema carries `matches.handicapped` and per-hole
stroke indexes so it can be added later, but all scoring is currently gross.

## Quick Start

### Prerequisites

- Go 1.26+
- PostgreSQL 16+
- Docker (for the test stack, linting, and code generation)
- Make

### Installation

```bash
git clone https://github.com/manitoba-ryder-cup/scorecard.git
cd scorecard
go mod download
make dev
```

### Database Setup

```bash
docker run -d \
  --name scorecard-postgres \
  -e POSTGRES_USER=scorecard \
  -e POSTGRES_PASSWORD=scorecard_dev \
  -e POSTGRES_DB=scorecard \
  -p 5432:5432 \
  postgres:16-alpine
```

Migrations run automatically on server startup, or apply them manually:

```bash
./bin/scorecard migrate up --database-url "postgres://scorecard:scorecard_dev@localhost:5432/scorecard?sslmode=disable"
```

The application role must **not** be a superuser and must not hold `BYPASSRLS` — either would
bypass tenant isolation.

### Running the Service

```bash
./bin/scorecard start \
  --http-address ":5000" \
  --database-url "postgres://scorecard:scorecard_dev@localhost:5432/scorecard?sslmode=disable" \
  --jwt-public-key "/path/to/heimdall.pub" \
  --environment "development"
```

## Project Structure

```txt
.
├── cmd/scorecard/              # CLI entry point (urfave/cli)
│   ├── main.go                 # CLI setup and logging
│   ├── start.go                # Server start command
│   ├── migrate.go              # Migration commands
│   ├── seed.go                 # seed-tournament command
│   ├── version.go
│   ├── flags.go
│   └── config.go
├── internal/
│   ├── app/                    # Lifecycle and dependency wiring
│   │   ├── server.go           # HTTP + DB coordination
│   │   ├── services.go         # Repository and service graph
│   │   └── seed.go             # Tournament seeding
│   ├── api/rest/               # HTTP layer (stdlib http.ServeMux)
│   │   ├── server.go           # Routes, middleware, optional auth
│   │   ├── dto.go              # Domain -> SDK DTO mapping
│   │   ├── json.go             # Response and error helpers
│   │   └── ...                 # One file per resource
│   ├── golf/                   # Domain layer
│   │   ├── scoring.go          # Match-play engine (pure functions)
│   │   ├── models.go           # Domain entities
│   │   ├── dependencies.go     # Repository interfaces
│   │   └── ...                 # One file per service
│   └── db/postgres/            # Data access layer
│       ├── db.go               # Connection pool (knowhere wrapper)
│       ├── tenant.go           # Tenant-scoped transaction helpers
│       ├── errors.go           # SQLSTATE -> domain error mapping
│       ├── migrations/         # SQL schema
│       ├── queries/            # SQL queries (sqlc input)
│       └── internal/sqlc/      # Generated code
├── sdk/                        # Public API contract + Go client
├── test/                       # Integration suite
│   └── isolation/              # Cross-tenant isolation suite
├── Dockerfile
├── Makefile
└── sqlc.yaml
```

## Golf Domain Model

- **Player** — stable identity; all-time record and cups won are derived on read
- **Tournament** — the event, created together with both teams
- **Team** — Red or Blue, with an optional captain
- **TournamentPlayer** — a player entered in a tournament, with that year's tier, biography,
  and handicap
- **TeamMember** — the draft outcome: an entered player assigned to a team
- **Match** — a match within a tournament, on a course/tee, in a format
- **MatchParticipant** — a player, with their team, taking part in a match
- **Score** — one hole score, per player or per team depending on the format
- **Course / Hole / TeeSet / TeeColor** — course setup
- **MatchFormat** — global seeded reference data (Singles, Fourball, Alt Shot, Scramble, Scotch)

## API Endpoints

The service serves `/v1/...`. The public site reaches it via `/api/scorecard/*` at the edge.

Reads are public: with a token they resolve to that token's tenant, without one they resolve
to the configured public tenant (401 if none is configured). Writes require the listed scope.

### Health

| Method | Path | Scope |
|---|---|---|
| GET | `/healthz` | — (also exempt from the proxy-secret check) |

### Players

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/players` | public read |
| POST | `/v1/players` | `scorecard:players:write` |
| GET | `/v1/players/{id}` | public read |
| GET | `/v1/players/{id}/tournaments` | public read |

### Courses and Reference Data

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/match-formats` | fully public (global, no tenant) |
| GET | `/v1/tee-colors` | public read |
| POST | `/v1/tee-colors` | `scorecard:courses:write` |
| GET | `/v1/courses` | public read |
| POST | `/v1/courses` | `scorecard:courses:write` |
| GET | `/v1/courses/{id}` | public read |
| GET | `/v1/courses/{id}/tees` | public read |
| POST | `/v1/courses/{id}/tees` | `scorecard:courses:write` |

### Tournaments

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/tournaments` | public read |
| POST | `/v1/tournaments` | `scorecard:tournaments:write` |
| GET | `/v1/tournaments/{id}` | public read |
| GET | `/v1/tournaments/{id}/teams` | public read |
| GET | `/v1/tournaments/{id}/results` | public read |
| GET | `/v1/tournaments/{id}/winner` | public read |
| GET | `/v1/tournaments/{id}/status` | public read |

### Roster and Draft

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/tournaments/{id}/players` | public read |
| POST | `/v1/tournaments/{id}/players` | `scorecard:tournaments:write` |
| PUT | `/v1/tournaments/{id}/players/{playerId}` | `scorecard:tournaments:write` |
| GET | `/v1/teams/{id}/members` | public read |
| POST | `/v1/teams/{id}/members` | `scorecard:tournaments:write` |
| DELETE | `/v1/teams/{id}/members/{playerId}` | `scorecard:tournaments:write` |
| PUT | `/v1/teams/{id}/captain` | `scorecard:tournaments:write` |
| DELETE | `/v1/teams/{id}/captain` | `scorecard:tournaments:write` |

### Matches and Scoring

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/tournaments/{id}/matches` | public read |
| POST | `/v1/tournaments/{id}/matches` | `scorecard:tournaments:write` |
| GET | `/v1/matches/{id}/participants` | public read |
| POST | `/v1/matches/{id}/participants` | `scorecard:tournaments:write` |
| DELETE | `/v1/matches/{id}/participants/{playerId}` | `scorecard:tournaments:write` |
| GET | `/v1/matches/{id}/holes` | public read |
| GET | `/v1/matches/{id}/scores` | public read |
| POST | `/v1/matches/{id}/scores` | `scorecard:scores:write` |
| GET | `/v1/matches/{id}/winner` | public read |
| GET | `/v1/matches/{id}/status` | public read |

## Development

### Common Commands

```bash
make build          # Production binary
make dev            # Development binary
make test           # Unit tests with the race detector
make coverage-html  # HTML coverage report
make fmt            # Format
make lint           # Lint (golangci-lint via Docker)
make sqlc           # Regenerate database code
make clean          # Remove build artifacts
```

### Integration Testing

The integration suite runs against a real stack (postgres + scorecard in Docker):

```bash
make test-setup     # Build and start the stack (generates test JWT keys on first run)
make integration    # Run ./test/...
make test-teardown  # Stop the stack and drop its volumes
```

The suite includes `test/isolation/`, which verifies cross-tenant isolation at both layers —
through the API, and directly against the database as the application role.

### Code Generation

Queries in `internal/db/postgres/queries/*.sql` compile to type-safe Go via
[sqlc](https://sqlc.dev/):

```sql
-- name: GetPlayer :one
SELECT * FROM players WHERE id = $1 AND tenant_id = $2;
```

Run `make sqlc`, then use the generated code. Never edit `internal/db/postgres/internal/sqlc/`
by hand — CI regenerates it and fails on drift.

### Database Migrations

golang-migrate format, in `internal/db/postgres/migrations/`:

```bash
./bin/scorecard migrate up
./bin/scorecard migrate down
./bin/scorecard migrate version
```

### Seeding a Tournament

```bash
./bin/scorecard seed-tournament --tenant-id <uuid> --file setup.json
```

Creates the tournament, its roster with tiers, both captains, and the match schedule. The
draft and match lineups are assigned live at the event, so it seeds neither. The course, tee
color, and formats are referenced by name and must already exist.

## Configuration

| Variable | Flag | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | `--database-url` | local dev URL | PostgreSQL connection string |
| `HTTP_ADDRESS` | `--http-address` | `:5000` | HTTP bind address |
| `JWT_PUBLIC_KEY_PATH` | `--jwt-public-key` | _required_ | Heimdall RSA public key (PEM) |
| `ENVIRONMENT` | `--environment` | `development` | development / staging / production |
| `DEBUG` | `--debug` | `false` | Debug logging |
| `LOG_FORMAT` | `--log-format` | `json` | `json` or `text` |
| `TRUSTED_PROXY_MODE` | `--trusted-proxy-mode` | `false` | Trust `X-Forwarded-For` |
| `PROXY_SECRET` | `--proxy-secret` | _empty_ | Require `X-Proxy-Secret` on all non-health requests |
| `SCORECARD_PUBLIC_TENANT_ID` | `--public-tenant-id` | _empty_ | Tenant for anonymous reads; empty requires auth for every request |

## Multi-Tenancy

Tenant isolation is enforced twice, independently:

1. **Explicit predicates** — every query filters on `tenant_id`
2. **Row-Level Security** — policies filter on `app.current_tenant_id`, set per transaction

The second layer requires `FORCE ROW LEVEL SECURITY`, because Postgres exempts a table's owner
from its policies and the application role owns the tables it migrates. `ENABLE` alone is
silently skipped for that role. `test/isolation/rls_test.go` guards this.

## Authentication

Scorecard is a resource server; it never issues tokens.

1. Users authenticate with [Heimdall](https://github.com/travisbale/heimdall) and receive a JWT
2. Scorecard validates it with Heimdall's RSA public key
3. Tenant and actor are put on the request context via `knowhere/identity`
4. Write endpoints require the matching scope from the token

## Docker

```bash
make docker-build

docker run -p 5000:5000 \
  -e DATABASE_URL="postgres://scorecard:password@db:5432/scorecard?sslmode=disable" \
  -e JWT_PUBLIC_KEY_PATH="/app/keys/heimdall.pub" \
  -v /path/to/heimdall.pub:/app/keys/heimdall.pub:ro \
  scorecard:latest
```

## Architecture

### Layered Design

1. **HTTP** (`internal/api/rest/`) — routing, request validation, DTO mapping
2. **Domain** (`internal/golf/`) — business logic and the scoring engine; no tenant awareness
3. **Data** (`internal/db/postgres/`) — tenant-scoped transactions and SQL

The domain depends on repository interfaces it declares itself (`golf/dependencies.go`), so
the data layer depends on the domain rather than the reverse.

### Key Patterns

- **Domain sentinels** — repositories translate SQLSTATEs into `ErrNotFound` /
  `ErrInvalidInput` / `ErrConflict`; the HTTP layer maps those to status codes in one place
- **SDK as contract** — wire types, routes, and scopes live in `sdk/`, shared by the server,
  the Go client, and the integration tests, so drift surfaces at compile time
- **Materialized results** — one serialized write path recomputes `match_results` on every
  score, keeping standings consistent under concurrent scoring
- **Type-safe SQL** — sqlc generates Go from the queries

### Dependencies

- **[pgx](https://github.com/jackc/pgx)** — PostgreSQL driver
- **[knowhere](https://github.com/travisbale/knowhere)** — JWT, identity context, tenant-scoped DB wrapper
- **[sqlc](https://sqlc.dev/)** — type-safe SQL code generation
- **[urfave/cli](https://github.com/urfave/cli)** — CLI framework
- **[golang-migrate](https://github.com/golang-migrate/migrate)** — schema migrations

HTTP routing uses the standard library's `http.ServeMux` (Go 1.22+ method and wildcard
patterns) — there is no third-party router.

## License

MIT — see [LICENSE](LICENSE).
