# Firmware Upgrader (Go)

A lightweight, high-performance firmware upgrade manager for cable modems built with Go. Discovers modems via SNMP, matches them against upgrade rules (MAC ranges or regex patterns), and orchestrates firmware upgrades via SNMP with persistent logging.

## Why Go Instead of C++?

- **Memory Safety**: No segfaults, no undefined behavior, no manual memory management
- **Concurrency**: Built-in goroutines and channels make concurrent operations trivial
- **Error Handling**: Explicit error returns instead of exceptions and crashes
- **Fast Compilation**: Rebuilds in seconds, not minutes
- **Small Binaries**: Single static binary with no external dependencies
- **Better Tooling**: Built-in formatter, linter, test runner, dependency management
- **Easier Debugging**: Stack traces that actually help, no pointer arithmetic nightmares

## Features

- **SNMP-Based Discovery**: Automatically discover cable modems attached to CMTS devices
- **Flexible Matching Rules**: 
  - MAC address range matching
  - Regex pattern matching on sysDescr
  - Priority-based rule evaluation
- **Persistent Storage**: SQLite database for CMTS configs, modems, rules, and upgrade logs
- **RESTful API**: Complete REST API for management operations
- **Web UI**: Browser-based interface for managing CMTS, rules, and viewing logs
- **Minimal Binary**: Single static executable (~10-15MB) with no dependencies
- **Concurrent Operations**: Goroutines handle multiple upgrades simultaneously
- **Audit Trail**: Complete activity logging for compliance and debugging
- **Graceful Shutdown**: Proper cleanup of in-flight operations

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   HTTP Server (net/http)                    │
│         REST API + Static UI via embed.FS                   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│                    Core Services                            │
├──────────────────┬──────────────────┬──────────────────────┤
│  SNMP Client     │  Upgrade Engine  │  Database (SQLite)  │
│  (gosnmp)        │  (Matcher)       │  (database/sql)     │
├──────────────────┼──────────────────┼──────────────────────┤
│ • Discovery      │ • MAC Range      │ • CMTS Config       │
│ • SNMP Queries   │ • Regex Match    │ • Cable Modem       │
│ • Device Polling │ • Job Scheduler  │ • Upgrade Rules     │
│                  │ • Upgrade Exec   │ • Upgrade Logs      │
└──────────────────┴──────────────────┴──────────────────────┘
```

## Prerequisites

**Go 1.21 or later**

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

That's it! No external C libraries, no package managers, no dependency hell.

## Building

```bash
# Clone or create your project directory
mkdir firmware-upgrader-go
cd firmware-upgrader-go

# Initialize Go module
go mod init github.com/yourusername/firmware-upgrader

# Download dependencies (automatic on first build)
go mod tidy

# Build
go build -o firmware-upgrader ./cmd/firmware-upgrader

# Or build with optimizations for smaller binary
go build -ldflags="-s -w" -o firmware-upgrader ./cmd/firmware-upgrader

# Cross-compile for MikroTik (ARM)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o firmware-upgrader-arm64 ./cmd/firmware-upgrader

# Or for MIPS (common on older routers)
GOOS=linux GOARCH=mipsle go build -ldflags="-s -w" -o firmware-upgrader-mips ./cmd/firmware-upgrader
```

Binary will be a single file - copy it anywhere and run.

## Project Structure

```
firmware-upgrader/
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── cmd/
│   └── firmware-upgrader/
│       └── main.go             # Entry point
├── internal/
│   ├── models/
│   │   └── models.go           # Data structures
│   ├── database/
│   │   ├── database.go         # Database layer
│   │   └── migrations.go       # Schema migrations
│   ├── snmp/
│   │   └── client.go           # SNMP operations
│   ├── engine/
│   │   ├── engine.go           # Upgrade logic
│   │   └── matcher.go          # Rule matching
│   ├── api/
│   │   ├── server.go           # HTTP server
│   │   ├── handlers.go         # API handlers
│   │   └── middleware.go       # Logging, CORS, etc.
│   └── logger/
│       └── logger.go           # Structured logging
├── web/                        # Static files (embedded)
│   ├── index.html
│   ├── cmts.html
│   ├── rules.html
│   └── static/
│       ├── css/
│       └── js/
└── README.md
```

## Dependencies

```go
require (
    github.com/gosnmp/gosnmp v1.37.0           // SNMP client
    github.com/mattn/go-sqlite3 v1.14.18       // SQLite driver
    github.com/gorilla/mux v1.8.1              // HTTP router
    github.com/rs/zerolog v1.31.0              // Fast structured logging
    github.com/google/uuid v1.4.0              // UUID generation
)
```

All dependencies are pure Go except sqlite3 (which is CGO but widely available).

## Running

```bash
# Run with defaults (uses config.yaml if present)
./firmware-upgrader

# With custom configuration file
./firmware-upgrader -config /path/to/config.yaml

# With command-line overrides
./firmware-upgrader -port 8080 -workers 10 -log-level debug

# Show current configuration
./firmware-upgrader -show-config

# Run from source
go run ./cmd/firmware-upgrader -port 8080
```

Web UI available at `http://localhost:8080`

### Configuration

The application uses a YAML configuration file for settings. See `config.example.yaml` for a complete example with documentation.

**Configuration Priority (highest to lowest):**
1. Command-line flags
2. Environment variables (FW_* prefix)
3. Configuration file (config.yaml)
4. Built-in defaults

**Example config.yaml:**
```yaml
server:
  port: 8080
  log_level: info

engine:
  workers: 5
  discovery_interval: 60s
  evaluation_interval: 120s
  job_timeout: 5m
  retry_attempts: 3

snmp:
  timeout: 10s
  retries: 3

modem:
  signal_level_min: -15.0
  signal_level_max: 15.0
  max_upgrades_per_cmts: 10

database:
  path: upgrader.db
```

### Command Line Flags

```
-config string      Path to configuration file (default: "config.yaml")
-db string          Path to SQLite database (overrides config)
-port int           HTTP server port (overrides config)
-log-level string   Log level: debug, info, warn, error (overrides config)
-workers int        Concurrent upgrade workers (overrides config)
-show-config        Display current configuration and exit
-version            Show version and exit
-help               Show help
```

### Environment Variables

```bash
FW_SERVER_PORT=8080           # Override server.port
FW_LOG_LEVEL=debug            # Override server.log_level
FW_ENGINE_WORKERS=10          # Override engine.workers
FW_DISCOVERY_INTERVAL=5m      # Override engine.discovery_interval
FW_EVALUATION_INTERVAL=10m    # Override engine.evaluation_interval
FW_DB_PATH=/data/upgrader.db  # Override database.path
```

## Quick Start Example

```go
package main

import (
    "log"
    "github.com/yourusername/firmware-upgrader/internal/api"
    "github.com/yourusername/firmware-upgrader/internal/database"
    "github.com/yourusername/firmware-upgrader/internal/engine"
)

func main() {
    // Open database
    db, err := database.New("upgrader.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create upgrade engine
    eng := engine.New(db, engine.Config{
        Workers: 5,
        RetryAttempts: 3,
    })
    go eng.Start()

    // Start API server
    srv := api.NewServer(db, eng, api.Config{
        Port: 8080,
        WebRoot: "./web",
    })
    
    log.Fatal(srv.ListenAndServe())
}
```

## Database Schema

### Tables

**cmts**
```sql
CREATE TABLE cmts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    snmp_port INTEGER DEFAULT 161,
    community_read TEXT NOT NULL,
    community_write TEXT,
    cm_community_string TEXT,
    snmp_version INTEGER DEFAULT 2,
    enabled BOOLEAN DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

**cable_modem**
```sql
CREATE TABLE cable_modem (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cmts_id INTEGER NOT NULL,
    mac_address TEXT UNIQUE NOT NULL,
    ip_address TEXT,
    sysdescr TEXT,
    current_firmware TEXT,
    signal_level REAL,
    status TEXT,
    last_seen INTEGER,
    FOREIGN KEY (cmts_id) REFERENCES cmts(id) ON DELETE CASCADE
);
```

**upgrade_rule**
```sql
CREATE TABLE upgrade_rule (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    match_type TEXT NOT NULL, -- 'MAC_RANGE' or 'SYSDESCR_REGEX'
    match_criteria TEXT NOT NULL, -- JSON
    tftp_server_ip TEXT NOT NULL,
    firmware_filename TEXT NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    priority INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

**upgrade_job**
```sql
CREATE TABLE upgrade_job (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    modem_id INTEGER NOT NULL,
    rule_id INTEGER NOT NULL,
    cmts_id INTEGER NOT NULL,
    mac_address TEXT NOT NULL,
    status TEXT NOT NULL, -- 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED'
    tftp_server_ip TEXT,
    firmware_filename TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_message TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER,
    FOREIGN KEY (modem_id) REFERENCES cable_modem(id),
    FOREIGN KEY (rule_id) REFERENCES upgrade_rule(id),
    FOREIGN KEY (cmts_id) REFERENCES cmts(id)
);
```

**activity_log**
```sql
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    entity_type TEXT,
    entity_id INTEGER,
    message TEXT NOT NULL,
    details TEXT, -- JSON
    created_at INTEGER NOT NULL
);
```

## REST API

All endpoints return JSON. Errors follow format: `{"error": "message"}`

### CMTS Management

```http
GET    /api/cmts              # List all CMTS
POST   /api/cmts              # Create new CMTS
GET    /api/cmts/:id          # Get specific CMTS
PUT    /api/cmts/:id          # Update CMTS
DELETE /api/cmts/:id          # Delete CMTS
POST   /api/cmts/:id/discover # Trigger modem discovery
```

**Example: Create CMTS**
```bash
curl -X POST http://localhost:8080/api/cmts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CMTS-Primary",
    "ip_address": "192.168.1.100",
    "snmp_port": 161,
    "community_read": "public",
    "community_write": "private",
    "cm_community_string": "cable-modem",
    "snmp_version": 2,
    "enabled": true
  }'
```

### Cable Modems

```http
GET /api/modems                   # List all modems
GET /api/modems/:id               # Get specific modem
GET /api/modems?cmts_id=:id       # Filter by CMTS
GET /api/modems?status=online     # Filter by status
```

### Upgrade Rules

```http
GET    /api/rules                 # List all rules
POST   /api/rules                 # Create new rule
GET    /api/rules/:id             # Get specific rule
PUT    /api/rules/:id             # Update rule
DELETE /api/rules/:id             # Delete rule
```

**Example: MAC Range Rule**
```bash
curl -X POST http://localhost:8080/api/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Arris SB8200 Upgrade",
    "description": "Upgrade all Arris SB8200 modems",
    "match_type": "MAC_RANGE",
    "match_criteria": {
      "start_mac": "00:01:5C:00:00:00",
      "end_mac": "00:01:5C:FF:FF:FF"
    },
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "arris-sb8200-v1.2.3.bin",
    "enabled": true,
    "priority": 100
  }'
```

**Example: Regex Rule**
```bash
curl -X POST http://localhost:8080/api/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Casa C10G Upgrade",
    "description": "Match Casa modems by sysDescr",
    "match_type": "SYSDESCR_REGEX",
    "match_criteria": {
      "pattern": "^Casa.*C10G.*"
    },
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "casa-c10g-v2.0.1.bin",
    "enabled": true,
    "priority": 50
  }'
```

### Upgrade Jobs

```http
GET  /api/jobs                    # List all jobs
GET  /api/jobs/:id                # Get job details
POST /api/jobs/:id/retry          # Retry failed job
GET  /api/jobs?status=PENDING     # Filter by status
```

### Activity Log

```http
GET /api/activity-log?limit=50    # Recent activity
GET /api/activity-log?offset=50   # Pagination
```

## Code Examples

### Adding SNMP Discovery

```go
package snmp

import (
    "fmt"
    "time"
    "github.com/gosnmp/gosnmp"
)

type Client struct {
    conn *gosnmp.GoSNMP
}

func NewClient(host string, community string, port uint16) (*Client, error) {
    conn := &gosnmp.GoSNMP{
        Target:    host,
        Port:      port,
        Community: community,
        Version:   gosnmp.Version2c,
        Timeout:   time.Duration(5) * time.Second,
        Retries:   3,
    }
    
    if err := conn.Connect(); err != nil {
        return nil, fmt.Errorf("connect failed: %w", err)
    }
    
    return &Client{conn: conn}, nil
}

func (c *Client) DiscoverModems() ([]Modem, error) {
    oid := "1.3.6.1.2.1.10.127.1.3.3.1.2" // DOCSIS MAC table
    
    results, err := c.conn.BulkWalkAll(oid)
    if err != nil {
        return nil, fmt.Errorf("bulk walk failed: %w", err)
    }
    
    modems := make([]Modem, 0, len(results))
    for _, result := range results {
        mac := parseMACFromOID(result.Name)
        modems = append(modems, Modem{
            MAC: mac,
            // ... parse other fields
        })
    }
    
    return modems, nil
}
```

### Creating Upgrade Rules with Validation

```go
package engine

import (
    "encoding/json"
    "fmt"
    "net"
    "regexp"
)

type MatchCriteria struct {
    StartMAC string `json:"start_mac,omitempty"`
    EndMAC   string `json:"end_mac,omitempty"`
    Pattern  string `json:"pattern,omitempty"`
}

func (r *Rule) Validate() error {
    if r.Name == "" {
        return fmt.Errorf("name is required")
    }
    
    if r.MatchType != "MAC_RANGE" && r.MatchType != "SYSDESCR_REGEX" {
        return fmt.Errorf("invalid match_type: %s", r.MatchType)
    }
    
    var criteria MatchCriteria
    if err := json.Unmarshal([]byte(r.MatchCriteria), &criteria); err != nil {
        return fmt.Errorf("invalid match_criteria JSON: %w", err)
    }
    
    if r.MatchType == "MAC_RANGE" {
        if _, err := net.ParseMAC(criteria.StartMAC); err != nil {
            return fmt.Errorf("invalid start_mac: %w", err)
        }
        if _, err := net.ParseMAC(criteria.EndMAC); err != nil {
            return fmt.Errorf("invalid end_mac: %w", err)
        }
    }
    
    if r.MatchType == "SYSDESCR_REGEX" {
        if _, err := regexp.Compile(criteria.Pattern); err != nil {
            return fmt.Errorf("invalid regex pattern: %w", err)
        }
    }
    
    if net.ParseIP(r.TFTPServerIP) == nil {
        return fmt.Errorf("invalid TFTP server IP")
    }
    
    return nil
}
```

### Concurrent Job Processing

```go
package engine

import (
    "context"
    "sync"
)

type Engine struct {
    db      *database.DB
    workers int
    jobs    chan *Job
    wg      sync.WaitGroup
}

func (e *Engine) Start(ctx context.Context) {
    for i := 0; i < e.workers; i++ {
        e.wg.Add(1)
        go e.worker(ctx, i)
    }
    
    // Periodically check for pending jobs
    go e.scheduler(ctx)
    
    <-ctx.Done()
    close(e.jobs)
    e.wg.Wait()
}

func (e *Engine) worker(ctx context.Context, id int) {
    defer e.wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-e.jobs:
            if err := e.processJob(ctx, job); err != nil {
                log.Error().Err(err).Int("worker", id).Msg("job failed")
            }
        }
    }
}

func (e *Engine) processJob(ctx context.Context, job *Job) error {
    job.Status = "IN_PROGRESS"
    job.StartedAt = time.Now().Unix()
    e.db.UpdateJob(job)
    
    // Download firmware
    // Trigger SNMP upgrade
    // Wait for completion
    // Update status
    
    return nil
}
```

## Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestDiscoverModems ./internal/snmp

# Benchmark
go test -bench=. ./internal/engine
```

Example test:
```go
package database

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateCMTS(t *testing.T) {
    db, err := New(":memory:")
    assert.NoError(t, err)
    defer db.Close()
    
    cmts := &CMTS{
        Name:          "Test CMTS",
        IPAddress:     "192.168.1.1",
        CommunityRead: "public",
        SNMPPort:      161,
        SNMPVersion:   2,
        Enabled:       true,
    }
    
    id, err := db.CreateCMTS(cmts)
    assert.NoError(t, err)
    assert.Greater(t, id, 0)
    
    fetched, err := db.GetCMTS(id)
    assert.NoError(t, err)
    assert.Equal(t, cmts.Name, fetched.Name)
}
```

## Deployment

### Systemd Service (Linux)

```ini
[Unit]
Description=Firmware Upgrader
After=network.target

[Service]
Type=simple
User=upgrader
ExecStart=/usr/local/bin/firmware-upgrader -db /var/lib/upgrader/upgrader.db -port 8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable firmware-upgrader
sudo systemctl start firmware-upgrader
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o firmware-upgrader ./cmd/firmware-upgrader

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/
COPY --from=builder /app/firmware-upgrader .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./firmware-upgrader"]
```

```bash
docker build -t firmware-upgrader .
docker run -p 8080:8080 -v $(pwd)/data:/data firmware-upgrader -db /data/upgrader.db
```

### MikroTik Router Deployment

```bash
# Build for ARM/MIPS
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o firmware-upgrader-arm64 ./cmd/firmware-upgrader

# SCP to router
scp firmware-upgrader-arm64 admin@192.168.88.1:/flash/

# SSH and run
ssh admin@192.168.88.1
/flash/firmware-upgrader-arm64 -port 8080 &
```

## Performance Tuning

```go
// Adjust worker count based on load
engine := engine.New(db, engine.Config{
    Workers: 10,           // Concurrent upgrade workers
    RetryAttempts: 3,      // Retry failed jobs
    PollInterval: 60,      // Seconds between discovery
    JobTimeout: 300,       // Seconds before job timeout
})

// Database connection pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

## Monitoring

```go
// Add Prometheus metrics
import "github.com/prometheus/client_golang/prometheus"

var (
    upgradesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "firmware_upgrades_total",
            Help: "Total firmware upgrades",
        },
        []string{"status"},
    )
)

// Expose metrics
http.Handle("/metrics", promhttp.Handler())
```

## Troubleshooting

**Import errors**: Run `go mod tidy`

**SNMP timeouts**: Increase timeout in SNMP client config

**SQLite locked**: Only one writer at a time - reduce concurrent workers

**High memory**: Reduce worker count, add streaming for large responses

**Cross-compile issues**: Ensure target architecture supports CGO for sqlite3

## Why This is Better Than C++

1. **No segfaults** - Go's memory safety catches bugs at compile time
2. **Fast builds** - Recompile entire project in seconds
3. **Easy concurrency** - `go` keyword vs thread management nightmares
4. **Better errors** - Stack traces, not cryptic pointer errors
5. **Standard library** - HTTP server, JSON, testing all built-in
6. **Tooling** - `go fmt`, `go vet`, `go test` - all standardized
7. **Deployment** - Single binary, no dependencies
8. **Debugging** - Actually works, unlike gdb on macOS

## License

MIT License

## Resources

- Go Documentation: https://go.dev/doc/
- gosnmp Library: https://github.com/gosnmp/gosnmp
- Gorilla Mux: https://github.com/gorilla/mux
- SQLite3 Driver: https://github.com/mattn/go-sqlite3