# Refactor to Proper Go Web App with Templates

## Current State (Bad)
- Static HTML files served from `/web` directory
- Duplicated header/footer in every file (10+ files)
- JavaScript makes API calls to `/api/*` endpoints
- Changes to menu require editing every HTML file

## Target State (Good)
- Go templates with shared layouts
- Header/footer/navigation defined once
- Server-side rendering + JavaScript for dynamic updates
- Single source of truth for UI components

---

## Phase 1: Create Template Structure (15 min)

### 1.1 Create Base Layout Template
**File**: `templates/layouts/base.html`
- Define `<!DOCTYPE html>`, `<head>`, shared CSS
- Define header with navigation (ONE place)
- Define footer (ONE place)
- Use `{{template "content" .}}` for page-specific content

### 1.2 Create Page Templates
Convert existing HTML to templates:
- `templates/index.html` - Dashboard
- `templates/cmts.html` - CMTS list
- `templates/rules.html` - Rules list
- `templates/activity.html` - Activity log
- `templates/settings.html` - Settings
- `templates/docs.html` - Documentation
- `templates/api.html` - API reference

Each template:
1. Define `{{define "content"}}`
2. Move page-specific HTML inside
3. Keep existing `<style>` and `<script>` sections
4. Remove duplicated header/footer

---

## Phase 2: Update API Server (20 min)

### 2.1 Add Template Rendering
**File**: `internal/api/server.go`

Add template loading:
```go
type Server struct {
    templates *template.Template
    // ... existing fields
}

func (s *Server) loadTemplates() error {
    s.templates = template.Must(template.ParseGlob("templates/**/*.html"))
    return nil
}
```

### 2.2 Add Page Handlers
Replace static file serving with template handlers:
```go
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
    s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleCMTS(w http.ResponseWriter, r *http.Request) {
    s.templates.ExecuteTemplate(w, "cmts.html", nil)
}
// ... etc for each page
```

### 2.3 Update Routes
**File**: `internal/api/server.go` in `setupRoutes()`
```go
// UI Pages (server-side rendered)
s.router.HandleFunc("/", s.handleIndex)
s.router.HandleFunc("/cmts.html", s.handleCMTS)
s.router.HandleFunc("/rules.html", s.handleRules)
s.router.HandleFunc("/activity.html", s.handleActivity)
s.router.HandleFunc("/settings.html", s.handleSettings)
s.router.HandleFunc("/docs.html", s.handleDocs)
s.router.HandleFunc("/api.html", s.handleAPIReference)

// Static assets (CSS, JS)
s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web")))

// API endpoints (keep as-is)
api := s.router.PathPrefix("/api").Subrouter()
// ... existing API routes
```

---

## Phase 3: Move Static Assets (5 min)

### 3.1 Keep Static Files
`web/` directory keeps:
- `shared.css` - Global styles
- `shared.js` - Shared JavaScript functions
- Any images/icons

### 3.2 Remove Old HTML
Delete old HTML files after templates working:
- `web/index.html`
- `web/cmts.html`
- etc.

---

## Phase 4: Testing (20 min)

### 4.1 Build and Run
```bash
go build -o firmware-upgrader ./cmd/firmware-upgrader
./firmware-upgrader
```

### 4.2 Test Each Page
- [ ] http://localhost:8080/ - Dashboard loads
- [ ] http://localhost:8080/cmts.html - CMTS page loads
- [ ] http://localhost:8080/rules.html - Rules page loads
- [ ] http://localhost:8080/activity.html - Activity page loads
- [ ] http://localhost:8080/settings.html - Settings page loads
- [ ] http://localhost:8080/docs.html - Docs page loads
- [ ] http://localhost:8080/api.html - API reference loads

### 4.3 Test Functionality
- [ ] Navigation menu works (same on all pages)
- [ ] CSS loads correctly
- [ ] JavaScript API calls still work
- [ ] Forms submit properly
- [ ] Tables populate with data

### 4.4 Test API Endpoints
```bash
curl http://localhost:8080/api/cmts
curl http://localhost:8080/api/modems
curl http://localhost:8080/api/rules
```

---

## Phase 5: Cleanup (5 min)

### 5.1 Update Config
**File**: `cmd/firmware-upgrader/main.go`
```go
srv := api.NewServer(db, eng, api.Config{
    Port:         *port,
    WebRoot:      "./web",      // For static assets only
    TemplateDir:  "./templates", // For Go templates
})
```

### 5.2 Update Documentation
- Update BUILD.md to mention templates
- Update deployment docs (copy templates/ directory)

---

## Rollback Plan (if broken)

```bash
git stash  # Save changes
git checkout HEAD~1  # Go back to static HTML version
```

---

## Benefits After Refactor

✅ **Single source of truth** - Header/footer/nav in ONE place  
✅ **Proper Go architecture** - Server-side rendering  
✅ **Easy maintenance** - Change menu once, updates everywhere  
✅ **Better performance** - Server can cache rendered templates  
✅ **Consistent UI** - No more menu differences between pages  

---

## Estimated Time

- Phase 1: 15 minutes (create templates)
- Phase 2: 20 minutes (update server)
- Phase 3: 5 minutes (move assets)
- Phase 4: 20 minutes (testing)
- Phase 5: 5 minutes (cleanup)

**Total: ~65 minutes** (1 hour if no issues)

---

## Risk Assessment

**Medium Risk** - Could break UI rendering if templates wrong

**Mitigation**:
1. Do it on a branch (`git checkout -b refactor-templates`)
2. Test thoroughly before merging
3. Keep static HTML as backup
4. Can rollback if broken

---

## Next Steps

1. Review this plan
2. Create branch: `git checkout -b refactor-templates`
3. Execute Phase 1
4. Test after each phase
5. Commit after each working phase
6. Merge when all tests pass