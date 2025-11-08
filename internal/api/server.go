package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/database"
	"github.com/awksedgreep/firmware-upgrader/internal/engine"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// Config holds API server configuration
type Config struct {
	Bind    string
	Port    int
	WebRoot string
}

// Server represents the HTTP API server
type Server struct {
	db        *database.DB
	engine    *engine.Engine
	config    Config
	router    *mux.Router
	server    *http.Server
	templates map[string]*template.Template
}

// NewServer creates a new API server
func NewServer(db *database.DB, eng *engine.Engine, config Config) *Server {
	s := &Server{
		db:     db,
		engine: eng,
		config: config,
		router: mux.NewRouter(),
	}

	// Load templates
	if err := s.loadTemplates(); err != nil {
		log.Warn().Err(err).Msg("Failed to load templates, template rendering will be disabled")
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Bind, config.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// loadTemplates loads all HTML templates
func (s *Server) loadTemplates() error {
	s.templates = make(map[string]*template.Template)

	// List of pages to load
	pages := []string{
		"index",
		"cmts",
		"rules",
		"activity",
		"settings",
		"docs",
		"api",
		"edit-cmts",
		"edit-rule",
	}

	// Load each page with the base layout
	for _, page := range pages {
		tmpl := template.New(page)

		// Parse base layout first
		tmpl, err := tmpl.ParseFiles("templates/layouts/base.gohtml")
		if err != nil {
			log.Warn().Err(err).Str("page", page).Msg("Failed to parse base layout")
			continue
		}

		// Parse the specific page template
		tmpl, err = tmpl.ParseFiles("templates/" + page + ".gohtml")
		if err != nil {
			log.Warn().Err(err).Str("page", page).Msg("Failed to parse page template")
			continue
		}

		s.templates[page] = tmpl
	}

	log.Info().Int("count", len(s.templates)).Msg("Templates loaded successfully")
	return nil
}

// renderTemplate renders a template with data
func (s *Server) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := s.templates[name]
	if !ok {
		log.Error().Str("template", name).Msg("Template not found")
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	err := tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Error().Err(err).Str("template", name).Msg("Failed to render template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Info().Int("port", s.config.Port).Msg("Starting HTTP server")
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)

	// UI Pages (server-side rendered)
	s.router.HandleFunc("/", s.handleIndexPage).Methods("GET")
	s.router.HandleFunc("/cmts", s.handleCMTSPage).Methods("GET")
	s.router.HandleFunc("/rules", s.handleRulesPage).Methods("GET")
	s.router.HandleFunc("/activity", s.handleActivityPage).Methods("GET")
	s.router.HandleFunc("/settings", s.handleSettingsPage).Methods("GET")
	s.router.HandleFunc("/docs", s.handleDocsPage).Methods("GET")
	s.router.HandleFunc("/api-docs", s.handleAPIPage).Methods("GET")
	s.router.HandleFunc("/edit-cmts", s.handleEditCMTSPage).Methods("GET")
	s.router.HandleFunc("/edit-rule", s.handleEditRulePage).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// CMTS routes
	api.HandleFunc("/cmts", s.handleListCMTS).Methods("GET")
	api.HandleFunc("/cmts", s.handleCreateCMTS).Methods("POST")
	api.HandleFunc("/cmts/{id:[0-9]+}", s.handleGetCMTS).Methods("GET")
	api.HandleFunc("/cmts/{id:[0-9]+}", s.handleUpdateCMTS).Methods("PUT")
	api.HandleFunc("/cmts/{id:[0-9]+}", s.handleDeleteCMTS).Methods("DELETE")
	api.HandleFunc("/cmts/{id:[0-9]+}/discover", s.handleDiscoverModems).Methods("POST")
	api.HandleFunc("/discovery/trigger", s.handleTriggerAllDiscovery).Methods("POST")

	// Modem routes
	api.HandleFunc("/modems", s.handleListModems).Methods("GET")
	api.HandleFunc("/modems/{id:[0-9]+}", s.handleGetModem).Methods("GET")

	// Rule routes
	api.HandleFunc("/rules", s.handleListRules).Methods("GET")
	api.HandleFunc("/rules", s.handleCreateRule).Methods("POST")
	api.HandleFunc("/rules/{id:[0-9]+}", s.handleGetRule).Methods("GET")
	api.HandleFunc("/rules/{id:[0-9]+}", s.handleUpdateRule).Methods("PUT")
	api.HandleFunc("/rules/{id:[0-9]+}", s.handleDeleteRule).Methods("DELETE")
	api.HandleFunc("/rules/evaluate", s.handleEvaluateRules).Methods("POST")

	// Job routes
	api.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	api.HandleFunc("/jobs/{id:[0-9]+}", s.handleGetJob).Methods("GET")
	api.HandleFunc("/jobs/{id:[0-9]+}/retry", s.handleRetryJob).Methods("POST")

	// Activity log routes
	api.HandleFunc("/activity-log", s.handleListActivityLogs).Methods("GET")

	// Settings routes
	api.HandleFunc("/settings", s.handleListSettings).Methods("GET")
	api.HandleFunc("/settings/{key}", s.handleGetSetting).Methods("GET")
	api.HandleFunc("/settings/{key}", s.handleUpdateSetting).Methods("PUT")

	// Health and metrics routes
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	api.HandleFunc("/dashboard", s.handleDashboard).Methods("GET")

	// Static assets (CSS, JS)
	if s.config.WebRoot != "" {
		s.router.PathPrefix("/").Handler(http.FileServer(http.Dir(s.config.WebRoot)))
	}
}

// Page handlers
func (s *Server) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Dashboard"}
	s.renderTemplate(w, "index", data)
}

func (s *Server) handleCMTSPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "CMTS Management"}
	s.renderTemplate(w, "cmts", data)
}

func (s *Server) handleRulesPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Rules Management"}
	s.renderTemplate(w, "rules", data)
}

func (s *Server) handleActivityPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Activity Log"}
	s.renderTemplate(w, "activity", data)
}

func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Settings"}
	s.renderTemplate(w, "settings", data)
}

func (s *Server) handleDocsPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Documentation"}
	s.renderTemplate(w, "docs", data)
}

func (s *Server) handleAPIPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "API Reference"}
	s.renderTemplate(w, "api", data)
}

func (s *Server) handleEditCMTSPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Edit CMTS"}
	s.renderTemplate(w, "edit-cmts", data)
}

func (s *Server) handleEditRulePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{"Title": "Edit Rule"}
	s.renderTemplate(w, "edit-rule", data)
}

// Middleware

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("duration", time.Since(start)).
			Msg("HTTP request")
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper functions

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) respondError(w http.ResponseWriter, statusCode int, message string) {
	s.respondJSON(w, statusCode, map[string]string{"error": message})
}

// handleHealth returns service health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	cmtsList, err := s.db.ListCMTS()
	if err != nil {
		s.respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "unhealthy",
			"error":   "database connection failed",
			"details": err.Error(),
		})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "healthy",
		"version":    "v0.5.0",
		"database":   "connected",
		"total_cmts": len(cmtsList),
	})
}

// handleMetrics returns system metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get counts
	cmtsList, _ := s.db.ListCMTS()
	modems, _ := s.db.ListModems(0)
	rules, _ := s.db.ListRules()
	allJobs, _ := s.db.ListJobs("", 0)
	pendingJobs, _ := s.db.ListJobs(models.JobStatusPending, 0)
	inProgressJobs, _ := s.db.ListJobs(models.JobStatusInProgress, 0)
	completedJobs, _ := s.db.ListJobs(models.JobStatusCompleted, 0)
	failedJobs, _ := s.db.ListJobs(models.JobStatusFailed, 0)

	// Count enabled CMTS
	enabledCMTS := 0
	for _, cmts := range cmtsList {
		if cmts.Enabled {
			enabledCMTS++
		}
	}

	// Count enabled rules
	enabledRules := 0
	for _, rule := range rules {
		if rule.Enabled {
			enabledRules++
		}
	}

	metrics := map[string]interface{}{
		"cmts": map[string]int{
			"total":   len(cmtsList),
			"enabled": enabledCMTS,
		},
		"modems": map[string]int{
			"total": len(modems),
		},
		"rules": map[string]int{
			"total":   len(rules),
			"enabled": enabledRules,
		},
		"jobs": map[string]int{
			"total":       len(allJobs),
			"pending":     len(pendingJobs),
			"in_progress": len(inProgressJobs),
			"completed":   len(completedJobs),
			"failed":      len(failedJobs),
		},
	}

	s.respondJSON(w, http.StatusOK, metrics)
}

// handleDashboard returns dashboard summary data
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Get counts
	cmtsList, _ := s.db.ListCMTS()
	modems, _ := s.db.ListModems(0)
	rules, _ := s.db.ListRules()
	pendingJobs, _ := s.db.ListJobs(models.JobStatusPending, 0)
	inProgressJobs, _ := s.db.ListJobs(models.JobStatusInProgress, 0)
	recentActivity, _ := s.db.ListActivityLogs(10, 0)

	// Count enabled items
	enabledCMTS := 0
	for _, cmts := range cmtsList {
		if cmts.Enabled {
			enabledCMTS++
		}
	}

	enabledRules := 0
	for _, rule := range rules {
		if rule.Enabled {
			enabledRules++
		}
	}

	dashboard := map[string]interface{}{
		"total_cmts":       len(cmtsList),
		"enabled_cmts":     enabledCMTS,
		"total_modems":     len(modems),
		"total_rules":      len(rules),
		"enabled_rules":    enabledRules,
		"pending_jobs":     len(pendingJobs),
		"in_progress_jobs": len(inProgressJobs),
		"recent_activity":  recentActivity,
	}

	s.respondJSON(w, http.StatusOK, dashboard)
}

// CMTS Handlers

func (s *Server) handleListCMTS(w http.ResponseWriter, r *http.Request) {
	cmtsList, err := s.db.ListCMTS()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list CMTS")
		s.respondError(w, http.StatusInternalServerError, "Failed to list CMTS")
		return
	}

	if cmtsList == nil {
		cmtsList = []*models.CMTS{}
	}

	s.respondJSON(w, http.StatusOK, cmtsList)
}

func (s *Server) handleCreateCMTS(w http.ResponseWriter, r *http.Request) {
	var cmts models.CMTS
	if err := json.NewDecoder(r.Body).Decode(&cmts); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	id, err := s.db.CreateCMTS(&cmts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create CMTS")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventCMTSAdded,
		EntityType: "cmts",
		EntityID:   id,
		Message:    fmt.Sprintf("Added CMTS: %s", cmts.Name),
	})

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (s *Server) handleGetCMTS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	cmts, err := s.db.GetCMTS(id)
	if err == models.ErrNotFound {
		s.respondError(w, http.StatusNotFound, "CMTS not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to get CMTS")
		s.respondError(w, http.StatusInternalServerError, "Failed to get CMTS")
		return
	}

	s.respondJSON(w, http.StatusOK, cmts)
}

func (s *Server) handleUpdateCMTS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var cmts models.CMTS
	if err := json.NewDecoder(r.Body).Decode(&cmts); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	cmts.ID = id

	if err := s.db.UpdateCMTS(&cmts); err != nil {
		log.Error().Err(err).Msg("Failed to update CMTS")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventCMTSUpdated,
		EntityType: "cmts",
		EntityID:   id,
		Message:    fmt.Sprintf("Updated CMTS: %s", cmts.Name),
	})

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDeleteCMTS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := s.db.DeleteCMTS(id); err != nil {
		if err == models.ErrNotFound {
			s.respondError(w, http.StatusNotFound, "CMTS not found")
			return
		}
		log.Error().Err(err).Msg("Failed to delete CMTS")
		s.respondError(w, http.StatusInternalServerError, "Failed to delete CMTS")
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventCMTSDeleted,
		EntityType: "cmts",
		EntityID:   id,
		Message:    fmt.Sprintf("Deleted CMTS ID: %d", id),
	})

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDiscoverModems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	go func() {
		if err := s.engine.DiscoverModems(id); err != nil {
			log.Error().Err(err).Int("cmts_id", id).Msg("Discovery failed")
		}
	}()

	s.respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Discovery started",
	})
}

func (s *Server) handleTriggerAllDiscovery(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Manual trigger: discovery for all CMTS")

	cmtsList, err := s.db.ListCMTS()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list CMTS")
		s.respondError(w, http.StatusInternalServerError, "Failed to list CMTS")
		return
	}

	triggeredCount := 0
	for _, cmts := range cmtsList {
		if !cmts.Enabled {
			continue
		}

		go func(id int) {
			if err := s.engine.DiscoverModems(id); err != nil {
				log.Error().Err(err).Int("cmts_id", id).Msg("Discovery failed")
			}
		}(cmts.ID)

		triggeredCount++
	}

	s.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":        "Discovery started for all enabled CMTS",
		"cmts_triggered": triggeredCount,
	})
}

// Modem Handlers

func (s *Server) handleListModems(w http.ResponseWriter, r *http.Request) {
	cmtsID := 0
	if id := r.URL.Query().Get("cmts_id"); id != "" {
		cmtsID, _ = strconv.Atoi(id)
	}

	modems, err := s.db.ListModems(cmtsID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list modems")
		s.respondError(w, http.StatusInternalServerError, "Failed to list modems")
		return
	}

	if modems == nil {
		modems = []*models.CableModem{}
	}

	s.respondJSON(w, http.StatusOK, modems)
}

func (s *Server) handleGetModem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	modem, err := s.db.GetModem(id)
	if err == models.ErrNotFound {
		s.respondError(w, http.StatusNotFound, "Modem not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to get modem")
		s.respondError(w, http.StatusInternalServerError, "Failed to get modem")
		return
	}

	s.respondJSON(w, http.StatusOK, modem)
}

// Rule Handlers

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.db.ListRules()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list rules")
		s.respondError(w, http.StatusInternalServerError, "Failed to list rules")
		return
	}

	if rules == nil {
		rules = []*models.UpgradeRule{}
	}

	s.respondJSON(w, http.StatusOK, rules)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.UpgradeRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	id, err := s.db.CreateRule(&rule)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create rule")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventRuleCreated,
		EntityType: "rule",
		EntityID:   id,
		Message:    fmt.Sprintf("Created rule: %s", rule.Name),
	})

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	rule, err := s.db.GetRule(id)
	if err == models.ErrNotFound {
		s.respondError(w, http.StatusNotFound, "Rule not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to get rule")
		s.respondError(w, http.StatusInternalServerError, "Failed to get rule")
		return
	}

	s.respondJSON(w, http.StatusOK, rule)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var rule models.UpgradeRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	rule.ID = id

	if err := s.db.UpdateRule(&rule); err != nil {
		log.Error().Err(err).Msg("Failed to update rule")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventRuleUpdated,
		EntityType: "rule",
		EntityID:   id,
		Message:    fmt.Sprintf("Updated rule: %s", rule.Name),
	})

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := s.db.DeleteRule(id); err != nil {
		if err == models.ErrNotFound {
			s.respondError(w, http.StatusNotFound, "Rule not found")
			return
		}
		log.Error().Err(err).Msg("Failed to delete rule")
		s.respondError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventRuleDeleted,
		EntityType: "rule",
		EntityID:   id,
		Message:    fmt.Sprintf("Deleted rule ID: %d", id),
	})

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Job Handlers

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	jobs, err := s.db.ListJobs(status, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list jobs")
		s.respondError(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}

	if jobs == nil {
		jobs = []*models.UpgradeJob{}
	}

	s.respondJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	job, err := s.db.GetJob(id)
	if err == models.ErrNotFound {
		s.respondError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to get job")
		s.respondError(w, http.StatusInternalServerError, "Failed to get job")
		return
	}

	s.respondJSON(w, http.StatusOK, job)
}

func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	job, err := s.db.GetJob(id)
	if err == models.ErrNotFound {
		s.respondError(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to get job")
		s.respondError(w, http.StatusInternalServerError, "Failed to get job")
		return
	}

	// Reset job status
	job.Status = models.JobStatusPending
	job.RetryCount = 0
	job.ErrorMessage = nil
	job.StartedAt = nil
	job.CompletedAt = nil

	if err := s.db.UpdateJob(job); err != nil {
		log.Error().Err(err).Msg("Failed to retry job")
		s.respondError(w, http.StatusInternalServerError, "Failed to retry job")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleEvaluateRules(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Manual trigger: rule evaluation")

	go func() {
		if err := s.engine.EvaluateRules(); err != nil {
			log.Error().Err(err).Msg("Rule evaluation failed")
		}
	}()

	s.respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Rule evaluation started",
	})
}

// Activity Log Handlers

func (s *Server) handleListActivityLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	logs, err := s.db.ListActivityLogs(limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list activity logs")
		s.respondError(w, http.StatusInternalServerError, "Failed to list activity logs")
		return
	}

	if logs == nil {
		logs = []*models.ActivityLog{}
	}

	s.respondJSON(w, http.StatusOK, logs)
}

// Settings Handlers

func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.ListSettings()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list settings")
		s.respondError(w, http.StatusInternalServerError, "Failed to list settings")
		return
	}

	s.respondJSON(w, http.StatusOK, settings)
}

func (s *Server) handleGetSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	value, err := s.db.GetSetting(key)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Setting not found")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"key":   key,
		"value": value,
	})
}

func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.db.SetSetting(key, req.Value); err != nil {
		log.Error().Err(err).Msg("Failed to update setting")
		s.respondError(w, http.StatusInternalServerError, "Failed to update setting")
		return
	}

	// Log activity
	s.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventSystemEvent,
		EntityType: "setting",
		EntityID:   0,
		Message:    fmt.Sprintf("Updated setting: %s = %s", key, req.Value),
	})

	s.respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}
