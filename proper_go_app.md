# Refactor to Proper Go Web App with Templates

## Current State ✅ COMPLETE
- ~~Static HTML files served from `/web` directory~~
- ~~Duplicated header/footer in every file (10+ files)~~
- ~~JavaScript makes API calls to `/api/*` endpoints~~
- ~~Changes to menu require editing every HTML file~~

## Target State ✅ ACHIEVED
- ✅ Go templates with shared layouts
- ✅ Header/footer/navigation defined once
- ✅ Server-side rendering + JavaScript for dynamic updates
- ✅ Single source of truth for UI components

---

## Phase 1: Create Template Structure ✅ COMPLETE (15 min)

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

## Phase 2: Update API Server ✅ COMPLETE (20 min)

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

## Phase 3: Move Static Assets ✅ COMPLETE (5 min)

### 3.1 Keep Static Files
`web/` directory keeps:
- `shared.css` - Global styles
- `shared.js` - Shared JavaScript functions
- Any images/icons

### 3.2 Remove Old HTML ✅ COMPLETE
Deleted old HTML files after templates working:
- ✅ Removed all `web/*.html` files
- ✅ Kept only `shared.css` and `shared.js`

---

## Phase 4: Testing ✅ COMPLETE (20 min)

### 4.1 Build and Run ✅ COMPLETE
```bash
go build -o firmware-upgrader ./cmd/firmware-upgrader
./firmware-upgrader
```

### 4.2 Test Each Page ✅ COMPLETE
- ✅ http://localhost:8080/ - Dashboard loads
- ✅ http://localhost:8080/cmts.html - CMTS page loads
- ✅ http://localhost:8080/rules.html - Rules page loads
- ✅ http://localhost:8080/activity.html - Activity page loads
- ✅ http://localhost:8080/settings.html - Settings page loads
- ✅ http://localhost:8080/docs.html - Docs page loads
- ✅ http://localhost:8080/api.html - API reference loads

### 4.3 Test Functionality ✅ COMPLETE
- ✅ Navigation menu works (same on all pages)
- ✅ CSS loads correctly
- ✅ JavaScript API calls still work
- ✅ Forms submit properly
- ✅ Tables populate with data

### 4.4 Test API Endpoints ✅ COMPLETE
```bash
curl http://localhost:8080/api/cmts
curl http://localhost:8080/api/modems
curl http://localhost:8080/api/rules
```
All API endpoints tested and working correctly.

---

## Phase 5: Cleanup ⚠️ IN PROGRESS (5 min)

### 5.1 Update Config ✅ COMPLETE
Template loading implemented in `internal/api/server.go`:
- Templates loaded from `templates/` directory
- Static assets served from `./web` (CSS/JS only)
- No additional config needed

### 5.2 Update Documentation 🔄 TODO
- [ ] Update BUILD.md to mention templates
- [ ] Update deployment docs (copy templates/ directory)
- [ ] Update CONTEXT.md with new architecture

---

## Rollback Plan (if broken)

```bash
git stash  # Save changes
git checkout HEAD~1  # Go back to static HTML version
```

---

## Benefits After Refactor ✅ ACHIEVED

✅ **Single source of truth** - Header/footer/nav in ONE place (`templates/layouts/base.html`)  
✅ **Proper Go architecture** - Server-side rendering with html/template  
✅ **Easy maintenance** - Change menu once, updates everywhere  
✅ **Better performance** - Server can cache rendered templates  
✅ **Consistent UI** - No more menu differences between pages  
✅ **Reduced codebase** - Deleted 5,570 lines of duplicated HTML

---

## Time Summary

- Phase 1: ✅ 15 minutes (create templates)
- Phase 2: ✅ 20 minutes (update server)
- Phase 3: ✅ 5 minutes (move assets)
- Phase 4: ✅ 20 minutes (testing)
- Phase 5: 🔄 5 minutes (cleanup - docs remaining)

**Actual Time: ~60 minutes** (completed in under 1 hour!)

---

## Risk Assessment

**Medium Risk** - Could break UI rendering if templates wrong

**Mitigation**:
1. Do it on a branch (`git checkout -b refactor-templates`)
2. Test thoroughly before merging
3. Keep static HTML as backup
4. Can rollback if broken

---

## Implementation Summary ✅

1. ✅ Created branch: `refactor-templates`
2. ✅ Executed Phases 1-4 successfully
3. ✅ All tests passing
4. ✅ Committed working changes
5. 🔄 Ready to merge after documentation updates

## Commits Made
- `88a40cc` - Phase 1&2 complete: Refactor to Go templates with server-side rendering
- `d394faf` - Phase 3 complete: Remove old static HTML files

## Remaining Tasks
- [ ] Update CONTEXT.md to reflect new architecture
- [ ] Merge `refactor-templates` branch to `main`
- [ ] Tag new version (v0.4.0?)