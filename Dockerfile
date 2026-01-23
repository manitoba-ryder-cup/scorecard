# Stage 1: Build Go binary
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build


# Stage 2: Runtime
FROM alpine:latest

# Add ca-certificates for HTTPS connections and other runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy the binary from go-builder
COPY --from=go-builder /build/bin/scorecard .

# Create non-root user
RUN addgroup -g 1000 scorecard && \
    adduser -D -u 1000 -G scorecard scorecard && \
    chown -R scorecard:scorecard /app

USER scorecard

# Expose HTTP port (no gRPC for scorecard)
EXPOSE 5000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:5000/healthz || exit 1

# Run the application
ENTRYPOINT ["./scorecard"]
CMD ["start"]
