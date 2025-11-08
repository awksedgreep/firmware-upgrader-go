# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /build

# Install build dependencies for UPX
RUN apk add --no-cache upx git

# Environment variables for runtime configuration (can be overridden)
# BIND_ADDRESS - Bind address/interface (default: 0.0.0.0)
# PORT - HTTP server port (default: 8080)
# DB_PATH - Path to SQLite database (default: upgrader.db)
# LOG_LEVEL - Log level: debug, info, warn, error (default: info)
# WORKERS - Number of concurrent workers (default: 0 = use database setting)

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations for smaller binary
# Using CGO_ENABLED=0 because modernc.org/sqlite is pure Go
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -trimpath \
    -o firmware-upgrader \
    ./cmd/firmware-upgrader

# Compress with UPX for minimal size
RUN upx --best --lzma firmware-upgrader

# Final stage - minimal distroless image
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/firmware-upgrader .

# Copy web UI and templates
COPY --from=builder /build/web ./web
COPY --from=builder /build/templates ./templates

# Expose port
EXPOSE 8080

# Run as non-root
USER nonroot:nonroot

# Environment variables (configure via container environment)
ENV BIND_ADDRESS=0.0.0.0
ENV PORT=8080
ENV DB_PATH=/app/data/upgrader.db
ENV LOG_LEVEL=info
ENV WORKERS=0

# Set default command (env vars take precedence over these defaults)
ENTRYPOINT ["./firmware-upgrader"]
