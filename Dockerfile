# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /build

# Install build dependencies for UPX
RUN apk add --no-cache upx git

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

# Copy web UI
COPY --from=builder /build/web ./web

# Expose port
EXPOSE 8080

# Run as non-root
USER nonroot:nonroot

# Set default command
ENTRYPOINT ["./firmware-upgrader"]
CMD ["-port", "8080"]
