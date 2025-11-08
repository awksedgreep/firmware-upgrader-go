# Deployment Guide

This guide covers deploying the Firmware Upgrader to production environments.

## Prerequisites

The Firmware Upgrader requires the following files to run:
- `firmware-upgrader` - The binary executable
- `templates/` - Go templates directory (required)
- `web/` - Static assets directory (CSS/JS)

**IMPORTANT**: The application will fail to start if `templates/` directory is missing.

## Quick Deploy

### Using Pre-built Packages

1. Download the deployment package for your platform:
   ```bash
   # For ARM64 (MikroTik RB5009, CCR2004, etc.)
   wget https://github.com/awksedgreep/firmware-upgrader-go/releases/download/v0.4.0/firmware-upgrader-v0.4.0-linux-arm64.tar.gz
   
   # For AMD64 (x86-64 servers, routers)
   wget https://github.com/awksedgreep/firmware-upgrader-go/releases/download/v0.4.0/firmware-upgrader-v0.4.0-linux-amd64.tar.gz
   ```

2. Extract:
   ```bash
   tar xzf firmware-upgrader-*.tar.gz
   cd firmware-upgrader-*
   ```

3. Run:
   ```bash
   ./firmware-upgrader
   ```

The package includes everything needed: binary, templates, and web assets.

### Building Your Own Package

```bash
# Clone the repository
git clone https://github.com/awksedgreep/firmware-upgrader-go.git
cd firmware-upgrader-go

# Build deployment packages
make package-all

# Packages created in build/ directory:
# - firmware-upgrader-VERSION-linux-arm64.tar.gz
# - firmware-upgrader-VERSION-linux-amd64.tar.gz
```

## Container Deployment

### Using Podman/Docker

```bash
podman run -d \
  --name firmware-upgrader \
  -p 8080:8080 \
  -v ./data:/app/data:z \
  -e BIND_ADDRESS=0.0.0.0 \
  -e LOG_LEVEL=info \
  ghcr.io/awksedgreep/firmware-upgrader:latest
```

The container image includes all required files (binary, templates, web assets).

### Container Environment Variables

- `BIND_ADDRESS` - Bind address (default: `0.0.0.0`)
- `PORT` - HTTP port (default: `8080`)
- `DB_PATH` - Database path (default: `/app/data/upgrader.db`)
- `LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)
- `WORKERS` - Number of concurrent workers (default: `5`)

## MikroTik Deployment

### Container (Recommended)

1. Upload the container image to your MikroTik:
   ```bash
   # Pull image on your local machine
   podman pull ghcr.io/awksedgreep/firmware-upgrader:latest
   podman save ghcr.io/awksedgreep/firmware-upgrader:latest -o firmware-upgrader.tar
   
   # Upload to MikroTik
   scp firmware-upgrader.tar admin@192.168.88.1:/
   ```

2. Load and run on MikroTik:
   ```routeros
   /container
   add file=firmware-upgrader.tar interface=veth1 root-dir=/disk1/containers/firmware-upgrader \
       envlist=firmware-upgrader-env start-on-boot=yes
   
   /container/envs
   add name=firmware-upgrader-env key=BIND_ADDRESS value=0.0.0.0
   add name=firmware-upgrader-env key=PORT value=8080
   add name=firmware-upgrader-env key=LOG_LEVEL value=info
   
   /container
   start 0
   ```

### Binary Deployment (Direct)

If you prefer to run the binary directly on MikroTik without containers:

1. Create deployment directory structure:
   ```bash
   # On your local machine
   mkdir -p mikrotik-deploy
   cd mikrotik-deploy
   
   # Copy binary
   cp ../build/firmware-upgrader-linux-arm64 ./firmware-upgrader
   
   # Copy templates and web assets (REQUIRED)
   cp -r ../templates ./
   cp -r ../web ./
   
   # Create tarball
   tar czf mikrotik-deploy.tar.gz *
   ```

2. Upload to MikroTik:
   ```bash
   scp mikrotik-deploy.tar.gz admin@192.168.88.1:/disk1/
   ```

3. Extract and run on MikroTik:
   ```routeros
   /file
   # Extract files (use MikroTik's file manager or SSH)
   
   /system script
   add name=firmware-upgrader source={
     /execute script="/disk1/firmware-upgrader/firmware-upgrader"
   }
   ```

**IMPORTANT**: Always deploy the complete directory structure:
```
firmware-upgrader/
├── firmware-upgrader    # Binary
├── templates/           # REQUIRED - Go templates
│   ├── layouts/
│   │   └── base.gohtml
│   ├── index.gohtml
│   ├── cmts.gohtml
│   └── ... (other templates)
└── web/                 # Static assets
    ├── shared.css
    └── shared.js
```

## Server Deployment (Linux)

### Systemd Service

1. Install the binary and assets:
   ```bash
   sudo mkdir -p /opt/firmware-upgrader
   sudo tar xzf firmware-upgrader-*.tar.gz -C /opt/firmware-upgrader --strip-components=1
   sudo chmod +x /opt/firmware-upgrader/firmware-upgrader
   ```

2. Create systemd service:
   ```bash
   sudo tee /etc/systemd/system/firmware-upgrader.service << 'EOF'
   [Unit]
   Description=Firmware Upgrader
   After=network.target
   
   [Service]
   Type=simple
   User=firmware-upgrader
   Group=firmware-upgrader
   WorkingDirectory=/opt/firmware-upgrader
   ExecStart=/opt/firmware-upgrader/firmware-upgrader
   Restart=always
   RestartSec=10
   
   Environment="BIND_ADDRESS=0.0.0.0"
   Environment="PORT=8080"
   Environment="DB_PATH=/var/lib/firmware-upgrader/upgrader.db"
   Environment="LOG_LEVEL=info"
   Environment="WORKERS=5"
   
   [Install]
   WantedBy=multi-user.target
   EOF
   ```

3. Create user and data directory:
   ```bash
   sudo useradd -r -s /bin/false firmware-upgrader
   sudo mkdir -p /var/lib/firmware-upgrader
   sudo chown firmware-upgrader:firmware-upgrader /var/lib/firmware-upgrader
   ```

4. Enable and start:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable firmware-upgrader
   sudo systemctl start firmware-upgrader
   sudo systemctl status firmware-upgrader
   ```

### Verification

Check that the service is running and templates are loaded:
```bash
# Check logs
sudo journalctl -u firmware-upgrader -f

# You should see:
# Templates loaded successfully count=9

# Test web interface
curl http://localhost:8080/
```

## Configuration

### Command-line Flags

```bash
./firmware-upgrader -h

Options:
  -bind string
        Bind address (default "127.0.0.1")
  -port int
        HTTP port (default 8080)
  -db string
        Database path (default "upgrader.db")
  -log-level string
        Log level: debug, info, warn, error (default "info")
  -workers int
        Number of concurrent workers (default 5)
```

### Environment Variables

Can also be configured via environment variables:
- `BIND_ADDRESS`
- `PORT`
- `DB_PATH`
- `LOG_LEVEL`
- `WORKERS`

Command-line flags take precedence over environment variables.

## Troubleshooting

### Templates Not Found Error

**Symptom:**
```
WRN Failed to parse base layout error="open templates/layouts/base.gohtml: no such file or directory"
```

**Solution:**
Ensure the `templates/` directory is present in the working directory:
```bash
ls -la templates/
# Should show: base.gohtml and page templates

# If missing, extract from package:
tar xzf firmware-upgrader-*.tar.gz
```

### Permission Denied

**Symptom:**
```
Error: permission denied
```

**Solution:**
```bash
# Make binary executable
chmod +x firmware-upgrader

# Or run with sudo if binding to port < 1024
sudo ./firmware-upgrader -bind 0.0.0.0 -port 80
```

### Port Already in Use

**Symptom:**
```
ERR HTTP server error error="listen tcp :8080: bind: address already in use"
```

**Solution:**
```bash
# Find process using port
lsof -i :8080

# Use different port
./firmware-upgrader -port 8081

# Or stop conflicting service
sudo systemctl stop conflicting-service
```

## Directory Structure

The deployment package contains:

```
firmware-upgrader-VERSION/
├── firmware-upgrader           # Binary executable
├── templates/                  # Go templates (REQUIRED)
│   ├── layouts/
│   │   └── base.gohtml        # Base layout with header/footer/nav
│   ├── index.gohtml           # Dashboard
│   ├── cmts.gohtml            # CMTS management
│   ├── rules.gohtml           # Rules management
│   ├── activity.gohtml        # Activity log
│   ├── settings.gohtml        # Settings
│   ├── docs.gohtml            # Documentation
│   ├── api-docs.gohtml        # API reference
│   ├── edit-cmts.gohtml       # Edit CMTS
│   └── edit-rule.gohtml       # Edit rule
└── web/                       # Static assets
    ├── shared.css             # Global styles
    └── shared.js              # JavaScript utilities
```

**Never deploy just the binary alone** - it requires the `templates/` and `web/` directories.

## Security Considerations

1. **Bind Address**: Use `127.0.0.1` for local-only access, `0.0.0.0` for network access
2. **Firewall**: Restrict access to port 8080 to trusted networks
3. **SNMP Communities**: Use strong community strings, stored in database
4. **Database**: Protect `upgrader.db` file with proper permissions (600)
5. **Updates**: Keep the application updated with security patches

## Updating

### Binary Update

```bash
# Stop service
sudo systemctl stop firmware-upgrader

# Backup database
cp /var/lib/firmware-upgrader/upgrader.db /var/lib/firmware-upgrader/upgrader.db.backup

# Extract new version
sudo tar xzf firmware-upgrader-NEW-VERSION.tar.gz -C /opt/firmware-upgrader --strip-components=1

# Start service
sudo systemctl start firmware-upgrader
```

### Container Update

```bash
# Pull new image
podman pull ghcr.io/awksedgreep/firmware-upgrader:latest

# Stop old container
podman stop firmware-upgrader
podman rm firmware-upgrader

# Start new container (data persists in volume)
podman run -d --name firmware-upgrader -p 8080:8080 -v ./data:/app/data:z ghcr.io/awksedgreep/firmware-upgrader:latest
```

## Support

- GitHub: https://github.com/awksedgreep/firmware-upgrader-go
- Issues: https://github.com/awksedgreep/firmware-upgrader-go/issues
- Documentation: See README.md and docs/ directory