.PHONY: build dev test test-keys test-setup test-teardown integration coverage-html clean sqlc protoc fmt lint tidy download docker-build migrate-up migrate-down help

# Version is derived from git tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build production binary
build: fmt
	@echo "Building production binary..."
	@CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X 'main.Version=$(VERSION)'" -o bin/scorecard ./cmd/scorecard
	@echo "Build complete: bin/scorecard"

# Development build (faster, debug symbols)
dev: fmt
	@echo "Building development binary..."
	@go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/scorecard ./cmd/scorecard
	@echo "Build complete: bin/scorecard"

# Run unit tests (excludes the integration suite in ./test, which needs infrastructure)
test:
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

# Tear down test infrastructure and its volumes
test-teardown:
	@echo "Stopping test infrastructure..."
	@docker compose -f test/docker-compose.yml down -v

# Run integration tests (requires test infrastructure to be running)
integration:
	@echo "Running integration tests..."
	@go test -count=1 -timeout 5m ./test/... || \
		(echo "Integration tests failed. Scorecard logs:" && \
		docker compose -f test/docker-compose.yml logs scorecard > test/scorecard.log 2>&1 && exit 1)

# Generate HTML coverage report
coverage-html: test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@go clean
	@rm -rf bin
	@rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@go run golang.org/x/tools/cmd/goimports@v0.38.0 -w $(shell \
		find . -type f -name '*.go' \
			-not -path './internal/pb/*' \
			-not -path './internal/db/*' )

# Lint code
lint:
	@echo "Linting code..."
	@docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.6.0 golangci-lint run

# Generate sqlc code (uses version from go.mod)
sqlc:
	@echo "Generating sqlc code..."
	@docker run --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/src -w /src sqlc/sqlc generate

# Generate protobuf code
protoc:
	@echo "Generating protobuf code..."
	@docker build -q -t go-protoc:latest -f proto/Dockerfile . > /dev/null
	@docker run --rm -v $(shell pwd):/proto --user $(shell id -u):$(shell id -g) \
		-w /proto \
		go-protoc:latest \
		-I proto \
		--go_out=internal/pb --go_opt=paths=source_relative \
		--go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative \
		proto/*.proto

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run database migrations up
migrate-up:
	@./bin/scorecard migrate up

# Run database migrations down
migrate-down:
	@./bin/scorecard migrate down

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t scorecard:dev .

# Display help
help:
	@echo "Available targets:"
	@echo "  build          - Build production binary"
	@echo "  dev            - Build with debug symbols (faster compilation)"
	@echo "  test           - Run unit tests with coverage (no infrastructure)"
	@echo "  coverage-html  - Generate HTML coverage report"
	@echo ""
	@echo "Integration testing:"
	@echo "  test-keys      - Generate test JWT RSA keys (one-time)"
	@echo "  test-setup     - Build and start postgres + scorecard test infra"
	@echo "  integration    - Run the integration suite (needs test-setup first)"
	@echo "  test-teardown  - Stop test infra and remove volumes"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install and tidy Go dependencies"
	@echo "  sqlc           - Generate sqlc code from queries"
	@echo "  protoc         - Generate protobuf/gRPC code"
	@echo "  migrate-up     - Apply database migrations"
	@echo "  migrate-down   - Roll back latest migration"
	@echo ""
	@echo "Development:"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   - Build Docker image"
	@echo ""
	@echo "  help           - Display this help message"
