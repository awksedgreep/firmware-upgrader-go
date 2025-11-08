# Docker/Podman Deployment Guide

## Image Size

**Total Image Size: 39.8 MB**

- Base OS (distroless): ~32.6 MB
- Application binary: ~6.5 MB
- Web UI assets: ~0.7 MB

This is **20% smaller** than the 50MB Elixir version and perfect for resource-constrained environments like MikroTik routers.

## Building the Image

### Using Podman (recommended)

```bash
cd go-version
podman build -t firmware-upgrader:latest .
```

### Using Docker

```bash
cd go-version
docker build -t firmware-upgrader:latest .
```

Build time: ~2-3 minutes on first build (cached afterward)

## Running the Container

### Basic Run

```bash
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db
```

### With Custom Configuration

```bash
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db \
  -port 8080 \
  -workers 10 \
  -log-level debug
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  firmware-upgrader:
    image: firmware-upgrader:latest
    container_name: firmware-upgrader
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    command: ["-db", "/data/upgrader.db", "-port", "8080", "-workers", "5"]
    restart: unless-stopped
    user: "65532:65532"  # nonroot user
```

Run with:
```bash
docker-compose up -d
```

## Configuration Options

All configuration is via command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-db` | `upgrader.db` | Path to SQLite database file |
| `-port` | `8080` | HTTP server port |
| `-workers` | `5` | Number of concurrent upgrade workers |
| `-log-level` | `info` | Log level (debug, info, warn, error) |

## Persistent Storage

The container needs a persistent volume for the database:

```bash
# Create a named volume
podman volume create fw-data

# Run with named volume
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v fw-data:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db
```

## Accessing the UI

Once running, access the web UI at:
- http://localhost:8080

API endpoints:
- http://localhost:8080/api/cmts
- http://localhost:8080/api/rules
- http://localhost:8080/api/modems
- http://localhost:8080/api/jobs
- http://localhost:8080/api/activity-log

## Health Check

```bash
# Check if container is running
podman ps | grep firmware-upgrader

# View logs
podman logs firmware-upgrader

# Follow logs
podman logs -f firmware-upgrader

# Check API health
curl http://localhost:8080/api/cmts
```

## Deployment to MikroTik Router

### Option 1: Export and Import

```bash
# Export image to tar
podman save firmware-upgrader:latest -o firmware-upgrader.tar

# Compress for smaller size
gzip firmware-upgrader.tar
# Result: ~15-20MB compressed

# Copy to router
scp firmware-upgrader.tar.gz admin@192.168.88.1:/tmp/

# On router, import and run
ssh admin@192.168.88.1
podman load -i /tmp/firmware-upgrader.tar.gz
podman run -d --name fw -p 8080:8080 firmware-upgrader:latest
```

### Option 2: Container Registry

```bash
# Tag for registry
podman tag firmware-upgrader:latest registry.example.com/firmware-upgrader:latest

# Push to registry
podman push registry.example.com/firmware-upgrader:latest

# On router, pull and run
podman pull registry.example.com/firmware-upgrader:latest
podman run -d --name fw -p 8080:8080 \
  registry.example.com/firmware-upgrader:latest
```

## Resource Requirements

Minimum system requirements:
- **RAM**: 64 MB (typically uses 30-40 MB)
- **Disk**: 50 MB for image + database
- **CPU**: Any (tested on ARM, MIPS, x86_64)

Typical resource usage:
```bash
podman stats firmware-upgrader
```

Example output:
```
NAME               CPU %   MEM USAGE / LIMIT   MEM %
firmware-upgrader  0.5%    35MB / 8GB          0.43%
```

## Cross-Platform Builds

Build for different architectures:

### ARM64 (MikroTik RB5009, newer devices)
```bash
podman build --platform linux/arm64 -t firmware-upgrader:arm64 .
```

### ARMv7 (older MikroTik devices)
```bash
podman build --platform linux/arm/v7 -t firmware-upgrader:armv7 .
```

### MIPS (very old routers)
```bash
podman build --platform linux/mips -t firmware-upgrader:mips .
```

### Multi-arch build
```bash
podman manifest create firmware-upgrader:latest
podman build --platform linux/amd64 -t firmware-upgrader:amd64 .
podman build --platform linux/arm64 -t firmware-upgrader:arm64 .
podman manifest add firmware-upgrader:latest firmware-upgrader:amd64
podman manifest add firmware-upgrader:latest firmware-upgrader:arm64
```

## Security

The container runs as non-root user (UID 65532):
```bash
podman exec firmware-upgrader id
# Output: uid=65532(nonroot) gid=65532(nonroot) groups=65532(nonroot)
```

Security features:
- Minimal distroless base (fewer vulnerabilities)
- No shell or package manager in container
- Read-only root filesystem (add `--read-only` flag)
- Runs as unprivileged user
- No capabilities required

### Read-Only Container
```bash
podman run -d \
  --name firmware-upgrader \
  --read-only \
  -p 8080:8080 \
  -v ./data:/data:Z \
  --tmpfs /tmp \
  firmware-upgrader:latest \
  -db /data/upgrader.db
```

## Troubleshooting

### Container won't start
```bash
# Check logs
podman logs firmware-upgrader

# Common issues:
# 1. Port already in use - change port mapping
# 2. Volume permissions - add :Z flag for SELinux
# 3. Database locked - ensure only one instance
```

### Database locked error
```bash
# Stop all containers using the database
podman stop firmware-upgrader

# Remove lock file if exists
rm ./data/upgrader.db-wal ./data/upgrader.db-shm

# Restart container
podman start firmware-upgrader
```

### Can't access web UI
```bash
# Check if port is exposed correctly
podman port firmware-upgrader

# Test from inside container
podman exec firmware-upgrader wget -O- http://localhost:8080/api/cmts

# Check firewall rules on host
```

### High memory usage
```bash
# Reduce worker count
podman stop firmware-upgrader
podman rm firmware-upgrader
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db \
  -workers 2  # Reduced from default 5
```

## Updating

### Pull new version
```bash
# Stop and remove old container
podman stop firmware-upgrader
podman rm firmware-upgrader

# Pull new image (or rebuild)
podman pull firmware-upgrader:latest  # if using registry
# OR
podman build -t firmware-upgrader:latest .

# Run new version (database will auto-migrate)
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db
```

### Backup database before update
```bash
# Backup
cp ./data/upgrader.db ./data/upgrader.db.backup

# If update fails, restore
cp ./data/upgrader.db.backup ./data/upgrader.db
```

## Production Deployment

### systemd service (if supported on router)

Create `/etc/systemd/system/firmware-upgrader.service`:

```ini
[Unit]
Description=Firmware Upgrader Container
After=network.target

[Service]
Type=simple
ExecStartPre=-/usr/bin/podman stop firmware-upgrader
ExecStartPre=-/usr/bin/podman rm firmware-upgrader
ExecStart=/usr/bin/podman run --rm \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v /var/lib/firmware-upgrader:/data:Z \
  firmware-upgrader:latest \
  -db /data/upgrader.db \
  -workers 5
ExecStop=/usr/bin/podman stop firmware-upgrader
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
systemctl enable firmware-upgrader
systemctl start firmware-upgrader
systemctl status firmware-upgrader
```

### Monitoring

```bash
# Watch resource usage
watch podman stats firmware-upgrader

# Watch logs in real-time
podman logs -f firmware-upgrader | grep -E "ERROR|WARN|discovered"

# Check upgrade job status
watch "curl -s http://localhost:8080/api/jobs | jq '.[] | {mac, status}'"
```

## Performance Tuning

### For high-traffic deployments
```bash
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  --memory=256m \
  --cpus=2 \
  firmware-upgrader:latest \
  -db /data/upgrader.db \
  -workers 20  # More workers for parallel upgrades
```

### For low-resource environments
```bash
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/data:Z \
  --memory=64m \
  --cpus=0.5 \
  firmware-upgrader:latest \
  -db /data/upgrader.db \
  -workers 2  # Fewer workers to save RAM
```

## Comparison with Native Binary

| Aspect | Container | Native Binary |
|--------|-----------|---------------|
| Size | 39.8 MB | 12 MB |
| Deployment | Portable | Platform-specific |
| Updates | Easy (pull image) | Manual copy |
| Isolation | Full | None |
| Overhead | Minimal (~5 MB RAM) | None |

For MikroTik routers, **native binary** might be preferred if space is extremely tight, but the container provides better isolation and easier updates.

## License

MIT License - See LICENSE file for details