# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Scorecard is a Go-based REST API for golf score tracking that supports live tournament scoring, individual rounds, and various golf formats. The service uses JWT authentication integrated with Heimdall and runs in Docker containers with PostgreSQL.

This is a migration from a Python/Flask application (`scorecardpy/`) to Go.

## Common Commands

### Building

- `make dev` - Build development binary with debug symbols (faster compilation)
- `make build` - Build production binary (optimized, stripped symbols)

### Code Generation

- `make sqlc` - Generate type-safe Go code from SQL queries (uses sqlc via Docker)

### Testing and Quality

- `make test` - Run all tests with race detector
- `make lint` - Run golangci-lint (via Docker)
- `make fmt` - Format code with gofmt and goimports

### Database Migrations

- `./bin/scorecard migrate up --database-url "postgres://..."` - Apply migrations
- `./bin/scorecard migrate down --database-url "postgres://..."` - Rollback last migration
- `./bin/scorecard migrate version --database-url "postgres://..."` - Check current migration version

### Running the Service

```bash
./bin/scorecard start \
  --database-url "postgres://scorecard:password@localhost:5432/scorecard?sslmode=disable" \
  --http-address ":5000" \
  --jwt-public-key "/path/to/heimdall.pub" \
  --environment "development"
```

## Architecture Overview

### Multi-Tenancy with Row-Level Security (RLS)

The service uses PostgreSQL Row-Level Security for tenant isolation. While scorecard typically operates with a single tenant, the multi-tenant architecture provides flexibility for future expansion.

- Tenant ID is extracted from JWT claims and propagated via context
- All database queries automatically filter by tenant via RLS policies
- Use `knowhere/db` package's `WithTenantContext` for tenant-scoped operations

### Layered Architecture

```txt
cmd/scorecard/           # CLI entry point (urfave/cli)
  ├── main.go            # CLI app setup
  ├── start.go           # Server start command
  ├── migrate.go         # Database migration commands
  ├── version.go         # Version command
  ├── flags.go           # Shared CLI flags
  └── config.go          # Configuration struct

internal/app/            # Server lifecycle
  └── server.go          # Coordinates HTTP, DB initialization

internal/api/http/       # HTTP layer (chi router)
  ├── server.go          # Server setup, route definitions
  ├── players.go         # Player endpoints
  ├── tournaments.go     # Tournament endpoints
  ├── matches.go         # Match endpoints (with live scoring)
  ├── scores.go          # Score endpoints
  ├── courses.go         # Course/hole/teeset endpoints
  ├── health.go          # Health check endpoint
  └── util.go            # JSON response utilities

internal/domain/         # Business logic services
  ├── players.go         # Player service
  ├── matches.go         # Match scoring logic
  ├── tournaments.go     # Tournament service
  └── errors.go          # Domain errors

internal/db/postgres/    # Data access layer
  ├── db.go              # Connection pool
  ├── migrate.go         # Migration runner (golang-migrate)
  ├── migrations/        # SQL schema files
  ├── queries/           # SQL queries (input for sqlc)
  └── internal/sqlc/     # Generated type-safe Go code

sdk/                     # Client SDK (future)
```

### Golf Domain Models

The core domain models represent golf tournament structure:

- **Player** - Golfer with handicap, tier, and statistics (wins/losses/ties/cups)
- **Tournament** - Top-level event with start/end dates
- **Team** - Red or Blue teams within a tournament
- **TeamMember** - Links players to teams for a specific tournament
- **Match** - Individual golf match within a tournament
- **MatchParticipant** - Links players to matches with team assignment
- **Score** - Individual hole scores for a participant in a match
- **Course** - Golf course
- **Hole** - Individual hole with par, handicap rating, and yardage
- **TeeSet** - Set of tees with slope and rating for a course
- **TeeColor** - Color of tee markers
- **MatchFormat** - Type of match (e.g., stroke play, match play)

### Match Scoring Logic

The match scoring system calculates live match status for golf tournaments:

- Supports handicapped and non-handicapped scoring
- Calculates team scores per hole using best ball format
- Determines match status (e.g., "2 UP", "AS", "3 & 2")
- Tracks match completion and winners

## Important Patterns

### Authentication and Authorization

The API uses JWT cookies for authentication with RS256 algorithm. JWTs are issued by Heimdall and verified using the public key.

- Use `knowhere/jwt` middleware for token validation
- Tenant ID and user ID are extracted from JWT claims
- Context is populated using `knowhere/identity.WithActor`
- Protected endpoints check permissions from JWT claims

### Database Access

Always use the knowhere database wrapper for tenant-scoped operations:

```go
import "github.com/travisbale/knowhere/db/postgres"

// For authenticated operations (tenant context available)
err := db.WithTenantContext(ctx, func(q *sqlc.Queries) error {
    player, err := q.GetPlayer(ctx, playerID)
    return err
})

// For pre-authentication or non-tenant operations
err := db.WithTransaction(ctx, func(q *sqlc.Queries) error {
    // Operations that don't require tenant scoping
    return err
})
```

### Code Generation with sqlc

SQL queries in `internal/db/postgres/queries/*.sql` are compiled to type-safe Go code:

1. Write SQL queries with sqlc annotations:

   ```sql
   -- name: GetPlayer :one
   SELECT * FROM players WHERE id = $1;
   ```

2. Run `make sqlc` to generate Go code in `internal/db/postgres/internal/sqlc/`

3. Custom type mappings are configured in `sqlc.yaml`

### Error Handling

- Use typed errors for known conditions (define in `internal/domain/errors.go`)
- Return wrapped errors with context: `fmt.Errorf("failed to X: %w", err)`
- HTTP handlers should map domain errors to appropriate status codes
- Use switch statements with `errors.Is()` for error checking

### JSON Serialization

- Use camelCase for JSON field names (matching Python API behavior)
- Request/response structs use `json` tags
- Empty slices should serialize as `[]` not `null`

### Testing

- Use table-driven tests for domain logic
- Database tests should use transactions for isolation
- Mock database interfaces for service-layer tests
- Run tests with `-race` flag to detect race conditions

## Configuration

Environment variables (can also be passed as CLI flags):

- `HTTP_ADDRESS` - HTTP server bind address (default: `:5000`)
- `DATABASE_URL` - PostgreSQL connection string (required)
- `JWT_PUBLIC_KEY_PATH` - Path to RSA public key PEM file from Heimdall (required)
- `ENVIRONMENT` - Environment name (default: `development`)
- `DEBUG` - Enable debug logging (default: `false`)

## Migration from Python

This service is a direct port from the Python/Flask `scorecardpy` application. Key differences:

1. **Authentication**: Now uses Heimdall (Go) instead of heimdallpy (Python)
2. **ORM**: sqlc (type-safe SQL) instead of SQLAlchemy
3. **Routing**: chi router instead of Flask blueprints
4. **Serialization**: Manual JSON marshaling instead of Marshmallow schemas
5. **Multi-tenancy**: RLS enforcement via knowhere library

### URL Structure

All API endpoints maintain the same prefix `/scorecard/v1/` for compatibility with existing clients.

## Development Guidelines

- Follow the code annotation style from root CLAUDE.md (focus on "why", not "what")
- No need for backwards compatibility (not yet released)
- Run `make fmt` before `make build` to resolve import issues
- Existing database migrations can be edited (no need to create new ones during development)
