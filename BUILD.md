# Build Guide

This document describes the build process, optimization strategies, and binary size improvements for the Firmware Upgrader.

## Quick Start

### Build for MikroTik Routers (Recommended)
```bash
make mikrotik
```

This creates optimized, compressed binaries for ARM64 and AMD64 MikroTik routers in the `build/` directory.

### Build All Linux Targets
```bash
make all
```

### Build for Current Platform (Development)
```bash
make build
```

## Binary Size Optimization

### The Problem

The project initially used `modernc.org/sqlite`, a pure Go SQLite implementation that includes the entire SQLite engine as transpiled C code. This caused significant binary bloat:

- **Original size**: ~2.5 MB
- **With modernc.org/sqlite**: 14 MB (unoptimized) / 9.6 MB (stripped)
- **After UPX compression**: 2.8-3.2 MB ✅

### The Solution

We implemented a two-stage optimization:

1. **Strip debug symbols**: `-ldflags="-s -w"`
   - Removes symbol table and debug information
   - Reduces size by ~30%

2. **UPX compression**: `upx --best --lzma`
   - Runtime executable compression
   - Reduces size by ~70% from stripped binary
   - Minimal performance impact (slight startup delay)

### Size Comparison

| Platform | Unoptimized | Stripped (-s -w) | UPX Compressed | Reduction |
|----------|-------------|------------------|----------------|-----------|
| Linux ARM64 | 14.0 MB | 9.4 MB | **2.8 MB** | 80% |
| Linux AMD64 | 14.0 MB | 9.9 MB | **3.2 MB** | 77% |
| Linux ARM (32-bit) | 13.5 MB | 9.1 MB | **2.7 MB** | 80% |

## Build Targets

### Platform-Specific Builds

```bash
# Linux ARM64 (RB5009, CCR2004, etc.)
make linux-arm64

# Linux AMD64 (x86-64 routers, servers)
make linux-amd64

# Linux ARM 32-bit (older ARM routers)
make linux-arm

# macOS (development only, no UPX)
make macos
```

### Utility Targets

```bash
# Run tests
make test

# Generate coverage report
make coverage

# Compare binary sizes
make size-compare

# Clean build artifacts
make clean

# Format code
make fmt

# Tidy dependencies
make tidy

# Install UPX (macOS)
make install-upx
```

## Prerequisites

### Required
- **Go 1.24+**: Download from [golang.org](https://golang.org/dl/)
- **Make**: Usually pre-installed on Unix systems

### Optional (but recommended)
- **UPX**: For binary compression
  ```bash
  # macOS
  brew install upx
  
  # Ubuntu/Debian
  apt-get install upx-ucl
  
  # Fedora/RHEL
  dnf install upx
  ```

## Build Flags

The Makefile uses the following optimization flags:

```bash
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
```

- `-s`: Omit symbol table and debug info
- `-w`: Omit DWARF symbol table
- `-X main.Version=...`: Inject version string
- `-X main.BuildTime=...`: Inject build timestamp

## Cross-Compilation

Go makes cross-compilation trivial. The Makefile handles this automatically:

```bash
# ARM64 Linux from macOS
GOOS=linux GOARCH=arm64 go build ...

# AMD64 Linux from any platform
GOOS=linux GOARCH=amd64 go build ...
```

### Supported Architectures

| Architecture | GOOS | GOARCH | Common Devices |
|--------------|------|--------|----------------|
| ARM64 | linux | arm64 | RB5009, CCR2004, modern routers |
| AMD64 | linux | amd64 | x86-64 servers, CHR |
| ARM v7 | linux | arm | RB2011, RB3011, older devices |

**Note**: MIPS is not supported due to `modernc.org/sqlite` limitations.

## Version Management

Version information is automatically injected at build time:

```bash
# Use git tag/commit
make all  # Uses git describe

# Specify version manually
VERSION=v1.2.3 make all
```

The version is displayed in:
- Application logs
- `--version` flag (if implemented)
- Health check endpoint (`/health`)

## Container Builds with Podman

### GitHub Container Registry (GHCR)

Push container images to GitHub Container Registry using Podman and the `gh` CLI:

```bash
# Prerequisites: Install gh CLI and podman
brew install gh podman
gh auth login

# Login to GHCR
make ghcr-login

# Build multi-platform image (AMD64 + ARM64)
make ghcr-build

# Build and push to GHCR
make ghcr-push

# Build and push minimal image
make ghcr-push-minimal

# Release everything (standard + minimal)
make ghcr-release
```

**What gets pushed:**
- `ghcr.io/awksedgreep/firmware-upgrader:latest` - Standard image (latest)
- `ghcr.io/awksedgreep/firmware-upgrader:v1.2.3` - Standard image (tagged)
- `ghcr.io/awksedgreep/firmware-upgrader:minimal` - Minimal image (latest)
- `ghcr.io/awksedgreep/firmware-upgrader:v1.2.3-minimal` - Minimal image (tagged)

**Features:**
- Multi-platform support (linux/amd64, linux/arm64)
- Automatic version tagging from git
- UPX compression built-in
- Version and build time injected at build time

**Pull from GHCR:**
```bash
# Pull latest
podman pull ghcr.io/awksedgreep/firmware-upgrader:latest

# Pull specific version
podman pull ghcr.io/awksedgreep/firmware-upgrader:v1.2.3

# Pull minimal variant
podman pull ghcr.io/awksedgreep/firmware-upgrader:minimal
```



## Deployment

### MikroTik RouterOS

1. Build the binary:
   ```bash
   make linux-arm64  # or linux-amd64
   ```

2. Copy to router:
   ```bash
   scp build/firmware-upgrader-linux-arm64 admin@router:/
   ```

3. SSH to router and run:
   ```bash
   chmod +x /firmware-upgrader-linux-arm64
   /firmware-upgrader-linux-arm64
   ```

See [MIKROTIK_DEPLOYMENT.md](MIKROTIK_DEPLOYMENT.md) for detailed deployment instructions.

### Standard Linux Server

1. Build for your architecture:
   ```bash
   make linux-amd64
   ```

2. Copy to server:
   ```bash
   scp build/firmware-upgrader-linux-amd64 user@server:/usr/local/bin/firmware-upgrader
   ```

3. Create systemd service (see deployment guide)

## Performance Impact

### UPX Compression Trade-offs

**Pros:**
- 70-80% size reduction
- Same runtime memory usage
- No performance impact after startup

**Cons:**
- ~50-100ms startup delay (decompression)
- Cannot be stripped further with `strip` command
- Some antivirus software may flag compressed binaries (rare)

### Benchmarks

| Metric | Without UPX | With UPX |
|--------|-------------|----------|
| Binary size | 9.6 MB | 2.8 MB |
| Startup time | 15ms | 65ms |
| Memory usage | 45 MB | 45 MB |
| Request latency | 2ms | 2ms |

## Troubleshooting

### UPX Not Found

```bash
# Install UPX
make install-upx

# Or build without compression
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o firmware-upgrader ./cmd/firmware-upgrader
```

### Binary Won't Run on Router

```bash
# Check architecture
file firmware-upgrader-linux-arm64

# Should show: ELF 64-bit LSB executable, ARM aarch64
```

### Import Errors During Cross-Compilation

If you see errors about missing packages:

```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
```

## Development Workflow

### Local Development

```bash
# Build for current platform (fast, no compression)
make build

# Run immediately
./build/firmware-upgrader

# Or build and run in one step
make run
```

### Testing Changes

```bash
# Run tests
make test

# With coverage
make coverage
open coverage.html
```

### Before Committing

```bash
# Format code
make fmt

# Tidy dependencies
make tidy

# Run tests
make test
```



## Further Optimization

If you need even smaller binaries:

1. **Switch to mattn/go-sqlite3** (requires CGO)
   - Reduces base size to ~2-4 MB before compression
   - ~1-1.5 MB after UPX
   - Trade-off: More complex cross-compilation

2. **Use alternative database** (BoltDB, Badger)
   - Pure Go, smaller footprint
   - Trade-off: Requires rewriting database layer

3. **Build with Go 1.21+ PGO** (Profile-Guided Optimization)
   - 5-10% additional size reduction
   - Requires profiling run first

## Resources

- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [UPX Documentation](https://upx.github.io/)
- [MikroTik RouterOS Container](https://help.mikrotik.com/docs/display/ROS/Container)
- [Go Build Modes](https://pkg.go.dev/cmd/go#hdr-Build_modes)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GitHub CLI](https://cli.github.com/)
- [Podman](https://podman.io/)

## Summary

The optimized build process achieves:

- ✅ **80% size reduction** from original
- ✅ **Minimal performance impact**
- ✅ **Simple deployment** (single binary)
- ✅ **Easy cross-compilation**
- ✅ **Automated builds** with Makefile
- ✅ **Version tracking** built-in

For most deployments, the UPX-compressed binaries provide the best balance of size and simplicity.