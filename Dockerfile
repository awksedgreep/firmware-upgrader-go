# Build stage - use Debian for better CGO compatibility
FROM golang:1.22-bookworm AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations for smaller binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o firmware-upgrader \
    ./cmd/firmware-upgrader

# Strip debug symbols for smaller size
RUN strip firmware-upgrader

# Final stage - minimal distroless image
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/firmware-upgrader .

# Copy web UI
COPY --from=builder /build/web ./web

# Expose port
EXPOSE 8080

# Run as non-root
USER nonroot:nonroot

# Set default command
ENTRYPOINT ["./firmware-upgrader"]
CMD ["-port", "8080"]
