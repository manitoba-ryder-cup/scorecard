# Scorecard

A modern REST API for golf tournament scoring and management built with Go. Supports live tournament scoring, match play formats, handicapped play, and comprehensive player statistics.

This is a ground-up rewrite of the original Python/Flask application, now using Go with improved performance, type safety, and maintainability.

---

## Features

- **Live Tournament Scoring**: Real-time score updates for ongoing matches
- **Match Play Support**: Automatic calculation of match status ("2 UP", "AS", "3 & 2")
- **Best Ball Format**: Team scoring with automatic best-ball calculations
- **Handicapped Play**: Per-hole handicap stroke allocation
- **Player Management**: Track player statistics (wins, losses, ties, cups)
- **Course Management**: Support for multiple courses, tee sets, and hole configurations
- **Multi-Tenancy**: PostgreSQL Row-Level Security for tenant isolation
- **JWT Authentication**: Integrated with Heimdall authentication service
- **RESTful API**: Clean HTTP API with JSON responses (camelCase)

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 16+
- Docker (for development)
- Make

### Installation

```bash
# Clone the repository
git clone https://github.com/travisbale/scorecard.git
cd scorecard

# Install dependencies
go mod download

# Build development binary
make dev
```

### Database Setup

```bash
# Start PostgreSQL with Docker
docker run -d \
  --name scorecard-postgres \
  -e POSTGRES_USER=scorecard \
  -e POSTGRES_PASSWORD=scorecard_dev \
  -e POSTGRES_DB=scorecard \
  -p 5432:5432 \
  postgres:16-alpine

# Migrations run automatically on startup
# Or run manually:
./bin/scorecard migrate up --database-url "postgres://scorecard:scorecard_dev@localhost:5432/scorecard?sslmode=disable"
```

### Running the Service

```bash
# Start the server (requires JWT public key from Heimdall)
./bin/scorecard start \
  --http-address ":5000" \
  --database-url "postgres://scorecard:scorecard_dev@localhost:5432/scorecard?sslmode=disable" \
  --jwt-public-key "/path/to/heimdall.pub" \
  --environment "development"

# The API is now available at http://localhost:5000
```

## Project Structure

```txt
.
├── cmd/scorecard/              # CLI entry point (urfave/cli)
│   ├── main.go                 # CLI app setup
│   ├── start.go                # Server start command
│   ├── migrate.go              # Migration commands
│   ├── version.go              # Version command
│   ├── flags.go                # Shared CLI flags
│   └── config.go               # Configuration struct
├── internal/
│   ├── app/                    # Server lifecycle management
│   │   └── server.go           # HTTP server initialization
│   ├── api/http/               # HTTP handlers (chi router)
│   │   ├── server.go           # Route definitions
│   │   ├── players.go          # Player endpoints
│   │   ├── tournaments.go      # Tournament endpoints
│   │   ├── matches.go          # Match endpoints
│   │   ├── scores.go           # Live scoring endpoints
│   │   └── health.go           # Health check
│   ├── domain/                 # Business logic services
│   │   ├── players.go          # Player service
│   │   ├── matches.go          # Match scoring logic
│   │   └── errors.go           # Domain errors
│   └── db/postgres/            # Data access layer
│       ├── db.go               # Database connection (knowhere wrapper)
│       ├── migrate.go          # Migration runner
│       ├── migrations/         # SQL migrations
│       ├── queries/            # SQL queries (sqlc input)
│       └── internal/sqlc/      # Generated type-safe code
├── sdk/                        # Client SDK (future)
├── Dockerfile                  # Multi-stage build
├── Makefile                    # Build automation
└── sqlc.yaml                   # Database code generation config
```

## Golf Domain Model

The API models golf tournaments with the following entities:

- **Player**: Golfer with handicap, tier, and statistics
- **Tournament**: Top-level event with start/end dates
- **Team**: Red or Blue teams within a tournament
- **TeamMember**: Links players to teams for a tournament
- **Match**: Golf match with course, tee set, and format
- **MatchParticipant**: Links players to matches with team assignment
- **Score**: Individual hole scores for participants
- **Course**: Golf course information
- **Hole**: Hole details (par, handicap, yardage)
- **TeeSet**: Tee configuration (slope, rating)
- **TeeColor**: Tee marker colors
- **MatchFormat**: Match type (stroke play, match play, etc.)

## API Endpoints

All endpoints are prefixed with `/scorecard/v1/`:

### Public

- `GET /healthz` - Service health check

### Authentication Required

- `POST /v1/players` - Create player
- `GET /v1/players` - List players
- `GET /v1/players/{id}` - Get player details
- `PUT /v1/players/{id}` - Update player
- `DELETE /v1/players/{id}` - Delete player

- `POST /v1/tournaments` - Create tournament
- `GET /v1/tournaments` - List tournaments
- `GET /v1/tournaments/{id}` - Get tournament details

- `POST /v1/matches` - Create match
- `GET /v1/matches/{id}` - Get match with live scoring
- `PUT /v1/matches/{id}` - Update match

- `POST /v1/scores` - Submit score
- `PUT /v1/scores/{id}` - Update score

_Full API documentation coming soon._

## Development

### Common Commands

```bash
# Build production binary (optimized)
make build

# Build development binary (debug symbols)
make dev

# Run tests with race detector
make test

# Generate HTML coverage report
make coverage-html

# Format code
make fmt

# Lint code
make lint

# Generate database code from SQL
make sqlc

# Clean build artifacts
make clean
```

### Code Generation

The project uses [sqlc](https://sqlc.dev/) for type-safe database access:

1. Write SQL queries in `internal/db/postgres/queries/*.sql`:

   ```sql
   -- name: GetPlayer :one
   SELECT * FROM players WHERE id = $1 AND tenant_id = $2;
   ```

2. Run code generation:

   ```bash
   make sqlc
   ```

3. Use generated code:

   ```go
   player, err := queries.GetPlayer(ctx, sqlc.GetPlayerParams{
       ID:       playerID,
       TenantID: tenantID,
   })
   ```

### Database Migrations

Migrations are stored in `internal/db/postgres/migrations/` using golang-migrate format:

```txt
001_init.up.sql       # Apply migration
001_init.down.sql     # Rollback migration
```

Commands:

```bash
# Apply all pending migrations
./bin/scorecard migrate up

# Rollback last migration
./bin/scorecard migrate down

# Check current version
./bin/scorecard migrate version
```

### Testing

```bash
# Run all tests
make test

# Run specific package
go test -v ./internal/domain

# Run with coverage
go test -cover ./...

# Generate coverage report
make coverage-html
```

## Configuration

Configuration can be set via environment variables or CLI flags:

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `HTTP_ADDRESS` | `--http-address` | `:5000` | HTTP server bind address |
| `DATABASE_URL` | `--database-url` | `postgres://...` | PostgreSQL connection string |
| `JWT_PUBLIC_KEY_PATH` | `--jwt-public-key` | _required_ | Path to Heimdall's RSA public key |
| `ENVIRONMENT` | `--environment` | `development` | Environment (development/staging/production) |
| `DEBUG` | `--debug` | `false` | Enable debug logging |

## Multi-Tenancy

The service uses PostgreSQL Row-Level Security (RLS) for tenant isolation:

- Tenant ID is extracted from JWT claims
- All database queries automatically filter by tenant via RLS policies
- Uses [knowhere](https://github.com/travisbale/knowhere) library for context propagation
- Single-tenant usage is supported (all data in one tenant)

## Authentication

Scorecard integrates with the [Heimdall](https://github.com/travisbale/heimdall) authentication service:

1. Users authenticate with Heimdall and receive JWT tokens
2. Scorecard validates JWTs using Heimdall's public key
3. Tenant ID and user ID are extracted from JWT claims
4. Permissions are checked from JWT claims

## Docker

### Build Image

```bash
make docker-build
# Or manually:
docker build -t scorecard:latest .
```

### Run Container

```bash
docker run -p 5000:5000 \
  -e DATABASE_URL="postgres://scorecard:password@db:5432/scorecard?sslmode=disable" \
  -e JWT_PUBLIC_KEY_PATH="/app/keys/heimdall.pub" \
  -v /path/to/heimdall.pub:/app/keys/heimdall.pub:ro \
  scorecard:latest
```

## Architecture

### Layered Design

1. **HTTP Layer** (`internal/api/http/`): Chi router, handlers, request/response validation
2. **Domain Layer** (`internal/domain/`): Business logic, match scoring algorithms
3. **Data Layer** (`internal/db/postgres/`): SQL queries, database access

### Key Patterns

- **JWT Middleware**: Validates tokens and populates context with tenant/user IDs
- **RLS Enforcement**: Automatic tenant filtering at database layer
- **Domain Errors**: Typed errors for business logic conditions
- **Type-Safe SQL**: sqlc generates Go code from SQL queries

### Dependencies

- **[chi](https://github.com/go-chi/chi)**: HTTP router
- **[pgx](https://github.com/jackc/pgx)**: PostgreSQL driver
- **[sqlc](https://sqlc.dev/)**: Type-safe SQL code generation
- **[golang-migrate](https://github.com/golang-migrate/migrate)**: Database migrations
- **[knowhere](https://github.com/travisbale/knowhere)**: Shared utilities (JWT, DB, crypto)
- **[urfave/cli](https://github.com/urfave/cli)**: CLI framework

## Migration from Python

This is a complete rewrite of the original [scorecardpy](../scorecardpy/) Flask application. Key improvements:

| Aspect | Python (Old) | Go (New) |
|--------|-------------|----------|
| **Performance** | Flask/Gunicorn | Native Go HTTP server |
| **Type Safety** | Runtime validation | Compile-time type checking |
| **Database** | SQLAlchemy ORM | Type-safe SQL with sqlc |
| **Auth** | heimdallpy (Python) | Heimdall SDK (Go) |
| **Serialization** | Marshmallow | Native JSON with struct tags |
| **Migrations** | Alembic | golang-migrate |
| **Testing** | pytest | Go testing with table-driven tests |

API endpoints remain compatible (same URL structure and JSON format).

## License

MIT

## Contributing

This is a personal project, but issues and suggestions are welcome.
