.PHONY: fmt lint comments unit test-keys test-setup integration test test-teardown coverage dev build sqlc deps migrate-up migrate-down docker-build clean help

# Version is derived from git tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Every target that formats or checks formatting works on this set: everything but the
# generated sqlc package.
GO_FILES = $(shell find . -type f -name '*.go' -not -path './internal/db/postgres/internal/sqlc/*')

# --- Quality ---------------------------------------------------------------------------

# gci runs after goimports because goimports treats a blank line as deliberate grouping
# and preserves it, so a stray one inside the stdlib block survives formatting. gci
# enforces the two sections instead of respecting what it finds.
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@go run golang.org/x/tools/cmd/goimports@v0.38.0 -w $(GO_FILES)
	@go run github.com/daixiang0/gci@v0.13.7 write --skip-generated -s standard -s default $(GO_FILES) >/dev/null

# Lint code
lint: comments
	@echo "Linting code..."
	@docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.11.4 golangci-lint run

# Cap comment blocks at the length the convention asks for
comments:
	@echo "Checking comment length..."
	@go run ./tools/commentcap ./cmd ./internal ./sdk ./test ./tools

# --- Tests -----------------------------------------------------------------------------

# Run unit tests (excludes the integration suite in ./test, which needs infrastructure)
unit:
	@echo "Running unit tests..."
	@go test -race -coverprofile=coverage.out -covermode=atomic $$(go list ./... | grep -v '/test')
	@echo "Unit test coverage: $$(go tool cover -func=coverage.out | grep total | awk '{print $$3}')"

# Generate RSA keys the test JWT issuer/validator share (one-time setup)
test-keys:
	@if [ ! -f test/keys/private-key.pem ]; then \
		echo "Generating test RSA keys..."; \
		mkdir -p test/keys; \
		openssl genrsa -out test/keys/private-key.pem 2048 2>/dev/null; \
		openssl rsa -in test/keys/private-key.pem -pubout -out test/keys/public-key.pem 2>/dev/null; \
		chmod 644 test/keys/*.pem; \
		echo "Test keys generated in test/keys/"; \
	else \
		echo "Test keys already exist, skipping generation"; \
	fi

# Start test infrastructure (postgres + scorecard); scorecard auto-migrates on boot
test-setup: test-keys
	@echo "Building scorecard image..."
	@docker compose -f test/docker-compose.yml build
	@echo "Starting postgres and scorecard..."
	@docker compose -f test/docker-compose.yml up -d --wait scorecard || \
		(echo "Scorecard failed to start. Logs:" && \
		docker compose -f test/docker-compose.yml logs scorecard && exit 1)
	@echo "Test infrastructure ready"

# Run integration tests (requires test infrastructure to be running)
integration:
	@echo "Running integration tests..."
	@go test -count=1 -timeout 5m ./test/... || \
		(echo "Integration tests failed. Scorecard logs:" && \
		docker compose -f test/docker-compose.yml logs scorecard > test/scorecard.log 2>&1 && exit 1)

# Run everything (unit + integration, requires Docker)
test: test-setup unit integration

# Tear down test infrastructure and its volumes
test-teardown:
	@echo "Stopping test infrastructure..."
	@docker compose -f test/docker-compose.yml down -v

# Generate HTML coverage report from the unit run
coverage: unit
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# --- Build -----------------------------------------------------------------------------

# Development build (faster, debug symbols)
dev: fmt
	@echo "Building development binary..."
	@go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/scorecard ./cmd/scorecard
	@echo "Build complete: bin/scorecard"

# Build production binary
build: fmt
	@echo "Building production binary..."
	@CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X 'main.Version=$(VERSION)'" -o bin/scorecard ./cmd/scorecard
	@echo "Build complete: bin/scorecard"

# --- Code generation -------------------------------------------------------------------

# Generate sqlc code (uses version from go.mod)
sqlc:
	@echo "Generating sqlc code..."
	@docker run --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/src -w /src sqlc/sqlc generate

# --- Database --------------------------------------------------------------------------

# Run database migrations up
migrate-up:
	@./bin/scorecard migrate up

# Run database migrations down
migrate-down:
	@./bin/scorecard migrate down

# --- Docker ----------------------------------------------------------------------------

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t scorecard:dev .

# --- Housekeeping ----------------------------------------------------------------------

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@go clean
	@rm -rf bin
	@rm -f coverage.out coverage.html

# Display help
help:
	@echo "Quality:"
	@echo "  fmt            - Format code (gofmt, goimports, gci)"
	@echo "  lint           - Lint code"
	@echo ""
	@echo "Tests:"
	@echo "  unit           - Run unit tests with coverage (no infrastructure)"
	@echo "  test-keys      - Generate test JWT RSA keys (one-time)"
	@echo "  test-setup     - Build and start postgres + scorecard test infra"
	@echo "  integration    - Run the integration suite (needs test-setup first)"
	@echo "  test           - test-setup + unit + integration"
	@echo "  test-teardown  - Stop test infra and remove volumes"
	@echo "  coverage       - HTML coverage report from the unit run"
	@echo ""
	@echo "Build:"
	@echo "  dev            - Build with debug symbols (faster compilation)"
	@echo "  build          - Build production binary"
	@echo ""
	@echo "Code generation:"
	@echo "  sqlc           - Generate sqlc code from queries"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up     - Apply database migrations"
	@echo "  migrate-down   - Roll back latest migration"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   - Build Docker image"
	@echo ""
	@echo "Housekeeping:"
	@echo "  deps           - Install and tidy Go dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Display this help message"
