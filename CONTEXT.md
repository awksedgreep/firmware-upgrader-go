# Firmware Upgrader - Current State & Context

## Project Overview
Go-based firmware upgrade management system for DOCSIS cable modems via SNMP. Runs on MikroTik routers or Linux servers.

## Current Architecture (BROKEN - NEEDS REFACTOR)

### The Problem
Built like a JavaScript SPA instead of a proper Go web app:
- **Static HTML files** in `/web` directory served directly
- **Duplicated header/footer** in every HTML file (10+ files)
- **JavaScript frontend** makes AJAX calls to `/api/*` endpoints
- Menu changes require editing every single HTML file

### What It Should Be
A proper Go web application with:
- **Go templates** with shared layouts
- **Server-side rendering** with header/footer defined ONCE
- **Template handlers** that render HTML
- Still keep JavaScript for dynamic updates

## Current Stats
- **Lines of Code**: ~7,896 total (3,555 production, 4,341 tests)
- **Test Coverage**: 66%
- **Binary Size**: 2.8MB (ARM64), 3.2MB (AMD64) - UPX compressed
- **Version**: v0.3.1

## System Flow

### 1. Discovery (every 60s)
- Connect to CMTS via SNMP
- BulkWalk MAC address table
- **Poll modem details CONCURRENTLY** (50 worker pool, 200 q/s rate limit)
- For each modem: IP, signal level, status, sysDescr
- Upsert modems to database

### 2. Cleanup (every hour)
- Mark modems offline if not seen in 10 minutes
- Delete offline modems after 7 days

### 3. Rule Evaluation (every 120s)
- Get enabled rules (by priority)
- Get online modems from DB
- Match modems by MAC range or sysDescr regex
- Create upgrade jobs for matches (no duplicates)

### 4. Job Processing (continuous workers)
- Workers pull jobs from queue
- Acquire CMTS semaphore (rate limit per CMTS)
- Send SNMP SET to modem (point to TFTP server)
- Poll for completion
- Update job status
- Release semaphore
- Retry with exponential backoff if failed

## Technology Stack
- **Language**: Go 1.24
- **Database**: SQLite with WAL mode (concurrent reads/writes)
- **SNMP**: gosnmp library
- **Web**: gorilla/mux router
- **Logging**: zerolog
- **UI**: Static HTML/CSS/JavaScript (NEEDS REFACTOR)

## Repository
- **GitHub**: awksedgreep/firmware-upgrader-go
- **GHCR**: ghcr.io/awksedgreep/firmware-upgrader

## Key Files

### Backend (Go)
- `cmd/firmware-upgrader/main.go` - Entry point
- `internal/api/server.go` - HTTP API server (serves static files + JSON API)
- `internal/engine/engine.go` - Discovery, rule eval, job processing
- `internal/snmp/client.go` - SNMP client with concurrent polling
- `internal/database/database.go` - SQLite with WAL mode
- `internal/models/models.go` - Data models

### Frontend (Static - BAD)
- `web/*.html` - 10+ HTML files with DUPLICATED headers/footers
- `web/shared.css` - Shared styles
- `web/shared.js` - Shared JavaScript functions

### Configuration
- `Makefile` - Build targets (mikrotik, ghcr-push, etc)
- `Dockerfile` - Multi-stage build with UPX compression
- Environment variables: BIND_ADDRESS, PORT, DB_PATH, LOG_LEVEL, WORKERS

## Deployment
- **Container**: Podman (not Docker)
- **Target**: MikroTik RouterOS 7.x containers
- **Binary**: Direct deployment to Linux/ARM64/AMD64

## CRITICAL ISSUE: UI Architecture

### Current State
```
web/
├── index.html        (Dashboard - has 6 menu items)
├── cmts.html         (CMTS list - has 6 menu items)
├── rules.html        (Rules - has 6 menu items)
├── activity.html     (Activity - has 6 menu items)
├── settings.html     (Settings - has 6 menu items)
├── docs.html         (Docs - has 6 menu items)
├── api.html          (API ref - has 6 menu items)
├── shared.css        (Global styles)
└── shared.js         (Global JS)
```

**Problem**: Header/footer/navigation duplicated in EVERY file. Any menu change = edit 10+ files.

### Solution (See proper_go_app.md)
```
templates/
├── layouts/
│   └── base.html     (Header, footer, nav - ONCE)
├── index.html        (Dashboard content only)
├── cmts.html         (CMTS content only)
└── ...

internal/api/server.go - Add template rendering handlers
```

**Refactor Plan**: See `proper_go_app.md` - 5 phases, ~65 minutes

## Recent Changes (Last Session)

### Added
- ✅ Concurrent modem polling (50 workers, 200 q/s rate limit)
- ✅ Stale modem cleanup (mark offline after 10min, delete after 7 days)
- ✅ SQLite WAL mode for concurrency
- ✅ Database locked retry logic (3 attempts with backoff)
- ✅ Standardized navigation menu (6 items everywhere)
- ✅ Documentation pages (docs.html, api.html)
- ✅ Settings page redesign (card-based UI)

### Known Issues
- ❌ UI built like JS SPA instead of Go templates (NEEDS REFACTOR)
- ❌ Duplicated header/footer code everywhere
- ❌ Menu inconsistencies fixed but via duplication

## Build & Deploy

### Local Development
```bash
make build              # Build for current platform
make test               # Run tests
make coverage           # Generate coverage report
```

### MikroTik Deployment
```bash
make mikrotik           # Build ARM64 + AMD64 binaries
# Creates build/firmware-upgrader-linux-{arm64,amd64}
```

### Container Build & Push
```bash
make ghcr-login         # Login with gh CLI
make ghcr-build         # Build multi-platform (AMD64 + ARM64)
make ghcr-push          # Push to GHCR
```

### Run Locally (Container)
```bash
podman run -it --rm -p 8080:8080 \
  -v $PWD/data:/app/data:z \
  -e BIND_ADDRESS=0.0.0.0 \
  ghcr.io/awksedgreep/firmware-upgrader:latest
```

## Priority #1: REFACTOR TO PROPER GO TEMPLATES

**WHY**: This is a Go app. It should use Go templates, not static HTML with duplicated code.

**PLAN**: See `proper_go_app.md` for detailed 5-phase refactoring plan

**PHASES**:
1. Create base layout template (15 min)
2. Update API server with template handlers (20 min)  
3. Move static assets (5 min)
4. Test everything (20 min)
5. Cleanup (5 min)

**TOTAL**: ~65 minutes

**BRANCH**: Create `refactor-templates` branch

## Testing

### Run Tests
```bash
make test
```

### Test Results
- API: 31 passed, 2 skipped (61.3% coverage)
- Database: 27 passed (72.5% coverage)
- Engine: 12 passed, 1 skipped (64.9% coverage)
- Models: 14 passed (100% coverage)
- SNMP: 3 passed (32.5% coverage)

### Manual Testing
1. Start server: `./firmware-upgrader -bind 127.0.0.1 -port 8080`
2. Open: http://localhost:8080
3. Test: Add CMTS, create rules, view activity
4. Check: All menu items appear consistently

## Environment Variables (Container)
```bash
BIND_ADDRESS=0.0.0.0      # Bind to specific IP (security)
PORT=8080                  # HTTP port
DB_PATH=/app/data/upgrader.db  # Database location
LOG_LEVEL=info             # debug, info, warn, error
WORKERS=5                  # Concurrent upgrade workers
```

## Database Schema

### Tables
- `cmts` - CMTS devices
- `cable_modem` - Discovered modems (with last_seen for cleanup)
- `upgrade_rule` - Firmware upgrade rules
- `upgrade_job` - Upgrade job queue
- `activity_log` - Event log
- `settings` - System settings

### Cleanup Logic
- **last_seen < 10 minutes**: Mark as offline
- **offline for 7 days**: Delete from database

## API Endpoints

### CMTS
- GET /api/cmts - List all
- POST /api/cmts - Create
- GET /api/cmts/{id} - Get one
- PUT /api/cmts/{id} - Update
- DELETE /api/cmts/{id} - Delete

### Modems
- GET /api/modems - List (filter by cmts_id, firmware)
- GET /api/modems/{id} - Get one

### Rules
- GET /api/rules - List
- POST /api/rules - Create
- GET /api/rules/{id} - Get one
- PUT /api/rules/{id} - Update
- DELETE /api/rules/{id} - Delete

### Jobs
- GET /api/jobs - List (filter by status, rule_id)
- GET /api/jobs/{id} - Get one
- POST /api/jobs/{id}/retry - Retry failed job
- DELETE /api/jobs/{id} - Cancel job

### System
- GET /api/settings - Get all settings
- PUT /api/settings - Update settings
- GET /api/stats - System statistics
- GET /health - Health check
- GET /metrics - Prometheus metrics

## Next Session Goals

1. **REFACTOR UI TO GO TEMPLATES** (see proper_go_app.md)
   - Create branch: `git checkout -b refactor-templates`
   - Follow 5-phase plan
   - Test thoroughly
   - Merge when working

2. **Test with Real Hardware**
   - Deploy to lab MikroTik
   - Connect to real CMTS
   - Test discovery and upgrades

3. **Production Readiness**
   - Load testing (100+ modems)
   - Performance monitoring
   - Error handling edge cases

## Documentation Files
- `README.md` - Project overview
- `BUILD.md` - Build instructions and binary optimization
- `API_GUIDE.md` - API endpoint documentation (outdated - see /api.html)
- `USER_GUIDE.md` - User documentation (outdated - see /docs.html)
- `MIKROTIK_DEPLOYMENT.md` - MikroTik deployment guide
- `proper_go_app.md` - **REFACTORING PLAN** ⭐
- `CONTEXT.md` - This file

## Git State
- **Branch**: main
- **Last Tag**: v0.3.1
- **Commits Ahead**: Check with `git status`
- **Origin**: https://github.com/awksedgreep/firmware-upgrader-go.git

## Cost Concerns
Previous session had issues due to:
- Building wrong architecture (JS SPA instead of Go app)
- Duplicating code instead of using templates
- Making mistakes without asking first

**RULES FOR NEXT SESSION**:
1. This is a Go app - use Go patterns (templates, not static HTML)
2. Ask before major architectural decisions
3. Don't duplicate code - use templates/functions
4. Test after each change
5. Commit working code frequently