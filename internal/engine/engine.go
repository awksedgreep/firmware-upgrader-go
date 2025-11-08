package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/database"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
	"github.com/awksedgreep/firmware-upgrader/internal/snmp"
	"github.com/rs/zerolog/log"
)

// Config holds engine configuration
type Config struct {
	Workers       int
	RetryAttempts int
	PollInterval  time.Duration
	JobTimeout    time.Duration
	MaxPerCMTS    int
}

// Engine manages firmware upgrade operations
type Engine struct {
	db           *database.DB
	config       Config
	jobs         chan *models.UpgradeJob
	matcher      *Matcher
	cmtsLimits   map[int]*semaphore
	cmtsLimitsMu sync.RWMutex
}

// semaphore implements a simple counting semaphore
type semaphore struct {
	ch chan struct{}
}

func newSemaphore(max int) *semaphore {
	return &semaphore{
		ch: make(chan struct{}, max),
	}
}

func (s *semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *semaphore) Release() {
	<-s.ch
}

// New creates a new upgrade engine
func New(db *database.DB, config Config) *Engine {
	if config.MaxPerCMTS <= 0 {
		config.MaxPerCMTS = 10 // Default limit
	}
	return &Engine{
		db:         db,
		config:     config,
		jobs:       make(chan *models.UpgradeJob, 100),
		matcher:    NewMatcher(),
		cmtsLimits: make(map[int]*semaphore),
	}
}

// getCMTSSemaphore gets or creates a semaphore for a CMTS
func (e *Engine) getCMTSSemaphore(cmtsID int) *semaphore {
	e.cmtsLimitsMu.RLock()
	sem, exists := e.cmtsLimits[cmtsID]
	e.cmtsLimitsMu.RUnlock()

	if exists {
		return sem
	}

	e.cmtsLimitsMu.Lock()
	defer e.cmtsLimitsMu.Unlock()

	// Double-check after acquiring write lock
	if sem, exists := e.cmtsLimits[cmtsID]; exists {
		return sem
	}

	sem = newSemaphore(e.config.MaxPerCMTS)
	e.cmtsLimits[cmtsID] = sem

	log.Debug().
		Int("cmts_id", cmtsID).
		Int("max_concurrent", e.config.MaxPerCMTS).
		Msg("Created rate limiter for CMTS")

	return sem
}

// Start begins the upgrade engine
func (e *Engine) Start(ctx context.Context) error {
	log.Info().
		Int("workers", e.config.Workers).
		Dur("poll_interval", e.config.PollInterval).
		Msg("Starting upgrade engine")

	// Start worker goroutines
	for i := 0; i < e.config.Workers; i++ {
		go e.worker(ctx, i)
	}

	// Start job scheduler
	go e.scheduler(ctx)

	// Start discovery scheduler
	go e.discoveryScheduler(ctx)

	// Start rule evaluation scheduler
	go e.ruleEvaluationScheduler(ctx)

	// Start cleanup scheduler
	go e.cleanupScheduler(ctx)

	<-ctx.Done()
	log.Info().Msg("Upgrade engine shutting down")
	close(e.jobs)
	return nil
}

// worker processes upgrade jobs
func (e *Engine) worker(ctx context.Context, id int) {
	log.Debug().Int("worker_id", id).Msg("Worker started")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("worker_id", id).Msg("Worker stopped")
			return
		case job := <-e.jobs:
			if err := e.processJob(ctx, job); err != nil {
				log.Error().
					Err(err).
					Int("worker_id", id).
					Int("job_id", job.ID).
					Msg("Failed to process job")
			}
		}
	}
}

// scheduler periodically checks for pending jobs
func (e *Engine) scheduler(ctx context.Context) {
	ticker := time.NewTicker(e.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.checkPendingJobs(); err != nil {
				log.Error().Err(err).Msg("Failed to check pending jobs")
			}
		}
	}
}

// checkPendingJobs retrieves and queues pending jobs with deduplication
func (e *Engine) checkPendingJobs() error {
	jobs, err := e.db.ListJobs(models.JobStatusPending, 100)
	if err != nil {
		return fmt.Errorf("failed to list pending jobs: %w", err)
	}

	// Track jobs already in progress to avoid duplication
	inProgressJobs, err := e.db.ListJobs(models.JobStatusInProgress, 100)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get in-progress jobs for deduplication")
	}

	inProgressMACs := make(map[string]bool)
	for _, job := range inProgressJobs {
		inProgressMACs[job.MACAddress] = true
	}

	for _, job := range jobs {
		// Skip if modem already has job in progress
		if inProgressMACs[job.MACAddress] {
			log.Debug().
				Int("job_id", job.ID).
				Str("mac", job.MACAddress).
				Msg("Skipping job - modem already being upgraded")
			continue
		}

		select {
		case e.jobs <- job:
			log.Debug().Int("job_id", job.ID).Msg("Queued pending job")
		default:
			log.Warn().Msg("Job queue full, skipping")
			return nil
		}
	}

	return nil
}

// processJob executes a single upgrade job
func (e *Engine) processJob(ctx context.Context, job *models.UpgradeJob) error {
	log.Info().
		Int("job_id", job.ID).
		Str("mac", job.MACAddress).
		Msg("Processing upgrade job")

	// Update job status to in progress
	now := time.Now()
	job.Status = models.JobStatusInProgress
	job.StartedAt = &now

	if err := e.db.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Log activity
	e.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventUpgradeStarted,
		EntityType: "job",
		EntityID:   job.ID,
		Message:    fmt.Sprintf("Started firmware upgrade for modem %s", job.MACAddress),
	})

	// Execute actual upgrade logic
	if err := e.executeUpgrade(ctx, job); err != nil {
		return e.handleJobFailure(job, err)
	}

	// Mark as completed
	completed := time.Now()
	job.Status = models.JobStatusCompleted
	job.CompletedAt = &completed

	if err := e.db.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to mark job complete: %w", err)
	}

	// Log completion
	e.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventUpgradeCompleted,
		EntityType: "job",
		EntityID:   job.ID,
		Message:    fmt.Sprintf("Completed firmware upgrade for modem %s", job.MACAddress),
	})

	log.Info().
		Int("job_id", job.ID).
		Str("mac", job.MACAddress).
		Msg("Upgrade job completed")

	return nil
}

// DiscoverModems discovers modems on a CMTS
func (e *Engine) DiscoverModems(cmtsID int) error {
	log.Info().Int("cmts_id", cmtsID).Msg("Starting modem discovery")

	// Get CMTS details
	cmts, err := e.db.GetCMTS(cmtsID)
	if err != nil {
		return fmt.Errorf("failed to get CMTS: %w", err)
	}

	if !cmts.Enabled {
		return fmt.Errorf("CMTS is disabled")
	}

	// Connect via SNMP
	client, err := snmp.NewClient(cmts)
	if err != nil {
		return fmt.Errorf("failed to connect to CMTS: %w", err)
	}
	defer client.Close()

	// Discover modems
	modems, err := client.DiscoverModems(cmts)
	if err != nil {
		return fmt.Errorf("failed to discover modems: %w", err)
	}

	// Upsert to database
	for _, modem := range modems {
		if err := e.db.UpsertModem(modem); err != nil {
			log.Error().
				Err(err).
				Str("mac", modem.MACAddress).
				Msg("Failed to upsert modem")
			continue
		}
	}

	// Log activity
	e.db.LogActivity(&models.ActivityLog{
		EventType:  "modem_discovered",
		EntityType: "cmts",
		EntityID:   cmtsID,
		Message:    fmt.Sprintf("Discovered %d modems on CMTS %s", len(modems), cmts.Name),
	})

	log.Info().
		Int("cmts_id", cmtsID).
		Int("discovered", len(modems)).
		Msg("Modem discovery completed")

	return nil
}

// EvaluateRules evaluates all enabled rules against all modems
func (e *Engine) EvaluateRules() error {
	log.Info().Msg("Evaluating upgrade rules")

	// Get all enabled rules (sorted by priority)
	allRules, err := e.db.ListRules()
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	// Filter enabled rules
	var rules []*models.UpgradeRule
	for _, rule := range allRules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}

	if len(rules) == 0 {
		log.Info().Msg("No enabled rules found")
		return nil
	}

	// Get all modems
	allModems, err := e.db.ListModems(0) // 0 = all CMTS
	if err != nil {
		return fmt.Errorf("failed to list modems: %w", err)
	}

	// Filter eligible modems (online, good signal)
	modems := e.matcher.FilterEligibleModems(allModems)

	log.Info().
		Int("total_modems", len(allModems)).
		Int("eligible_modems", len(modems)).
		Int("active_rules", len(rules)).
		Msg("Starting rule evaluation")

	// Match modems to rules
	jobsCreated := 0
	for _, modem := range modems {
		rule, err := e.matcher.MatchModemToRules(modem, rules)
		if err != nil {
			log.Error().
				Err(err).
				Str("mac", modem.MACAddress).
				Msg("Failed to match modem to rules")
			continue
		}

		if rule == nil {
			continue // No matching rule
		}

		// Check if upgrade is needed
		if !e.matcher.ShouldUpgrade(modem, rule) {
			continue
		}

		// Check if job already exists (pending or in-progress)
		existingPending, err := e.db.ListJobs(models.JobStatusPending, 1000)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check existing pending jobs")
		}

		existingInProgress, err := e.db.ListJobs(models.JobStatusInProgress, 1000)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check existing in-progress jobs")
		}

		// Combine all existing jobs
		existingJobs := append(existingPending, existingInProgress...)
		skip := false
		for _, job := range existingJobs {
			if job.MACAddress == modem.MACAddress {
				log.Debug().
					Str("mac", modem.MACAddress).
					Str("status", job.Status).
					Int("job_id", job.ID).
					Msg("Job already exists for modem, skipping")
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Create upgrade job
		job := &models.UpgradeJob{
			ModemID:          modem.ID,
			RuleID:           rule.ID,
			CMTSID:           modem.CMTSID,
			MACAddress:       modem.MACAddress,
			Status:           models.JobStatusPending,
			TFTPServerIP:     rule.TFTPServerIP,
			FirmwareFilename: rule.FirmwareFilename,
			RetryCount:       0,
			MaxRetries:       3,
		}

		jobID, err := e.db.CreateJob(job)
		if err != nil {
			log.Error().
				Err(err).
				Str("mac", modem.MACAddress).
				Msg("Failed to create upgrade job")
			continue
		}

		log.Info().
			Int("job_id", jobID).
			Str("mac", modem.MACAddress).
			Str("rule", rule.Name).
			Msg("Created upgrade job")

		jobsCreated++
	}

	log.Info().
		Int("jobs_created", jobsCreated).
		Msg("Rule evaluation completed")

	return nil
}

// executeUpgrade performs the actual firmware upgrade via SNMP
func (e *Engine) executeUpgrade(ctx context.Context, job *models.UpgradeJob) error {
	// Acquire CMTS rate limit semaphore
	sem := e.getCMTSSemaphore(job.CMTSID)
	sem.Acquire()
	defer sem.Release()

	log.Debug().
		Int("job_id", job.ID).
		Int("cmts_id", job.CMTSID).
		Str("mac", job.MACAddress).
		Msg("Acquired CMTS rate limit slot")

	// 1. Get modem details from database
	modem, err := e.db.GetModem(job.ModemID)
	if err != nil {
		return fmt.Errorf("failed to get modem details: %w", err)
	}

	// Verify modem has IP address
	if modem.IPAddress == "" {
		return fmt.Errorf("modem has no IP address")
	}

	// 2. Get CMTS details for CM community string
	cmts, err := e.db.GetCMTS(job.CMTSID)
	if err != nil {
		return fmt.Errorf("failed to get CMTS details: %w", err)
	}

	// Use CM community string if available, fallback to write community
	community := cmts.CMCommunityString
	if community == "" {
		community = cmts.CommunityWrite
	}

	if community == "" {
		return fmt.Errorf("no SNMP write community string available")
	}

	log.Info().
		Str("modem_ip", modem.IPAddress).
		Str("mac", job.MACAddress).
		Msg("Connecting to cable modem via SNMP")

	// 3. Connect to cable modem via SNMP
	client, err := snmp.ConnectToModem(modem.IPAddress, community, 161)
	if err != nil {
		return fmt.Errorf("failed to connect to modem: %w", err)
	}
	defer client.Close()

	// 4. Trigger firmware upgrade
	log.Info().
		Str("mac", job.MACAddress).
		Str("tftp_server", job.TFTPServerIP).
		Str("firmware", job.FirmwareFilename).
		Msg("Triggering firmware upgrade")

	err = client.TriggerFirmwareUpgrade(
		modem.IPAddress,
		job.TFTPServerIP,
		job.FirmwareFilename,
	)
	if err != nil {
		return fmt.Errorf("failed to trigger upgrade: %w", err)
	}

	// 5. Monitor upgrade progress with timeout
	timeout := time.After(e.config.JobTimeout)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Info().
		Str("mac", job.MACAddress).
		Dur("timeout", e.config.JobTimeout).
		Msg("Monitoring upgrade progress")

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during upgrade")

		case <-timeout:
			return fmt.Errorf("upgrade timeout after %v", e.config.JobTimeout)

		case <-ticker.C:
			status, err := client.CheckUpgradeStatus()
			if err != nil {
				log.Warn().
					Err(err).
					Str("mac", job.MACAddress).
					Msg("Failed to check upgrade status, will retry")
				continue
			}

			log.Debug().
				Str("mac", job.MACAddress).
				Str("status", status).
				Msg("Upgrade status check")

			switch status {
			case "completed":
				log.Info().
					Str("mac", job.MACAddress).
					Msg("Firmware upgrade completed successfully")
				return nil

			case "failed":
				return fmt.Errorf("firmware upgrade failed on device")

			case "in_progress":
				log.Debug().
					Str("mac", job.MACAddress).
					Msg("Upgrade still in progress")
				// Continue monitoring

			default:
				log.Warn().
					Str("mac", job.MACAddress).
					Str("status", status).
					Msg("Unknown upgrade status")
			}
		}
	}
}

// handleJobFailure handles job failures with exponential backoff retry logic
func (e *Engine) handleJobFailure(job *models.UpgradeJob, err error) error {
	log.Error().
		Err(err).
		Int("job_id", job.ID).
		Str("mac", job.MACAddress).
		Int("retry_count", job.RetryCount).
		Int("max_retries", job.MaxRetries).
		Msg("Job failed")

	errMsg := err.Error()
	job.ErrorMessage = &errMsg
	job.RetryCount++

	// Check if we should retry
	if job.RetryCount < job.MaxRetries {
		// Calculate exponential backoff delay: 30s, 60s, 120s, 240s...
		backoffSeconds := 30 * (1 << uint(job.RetryCount-1))
		if backoffSeconds > 300 {
			backoffSeconds = 300 // Cap at 5 minutes
		}
		retryAfter := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

		// Reset to pending for retry
		job.Status = models.JobStatusPending
		job.StartedAt = nil

		if updateErr := e.db.UpdateJob(job); updateErr != nil {
			log.Error().Err(updateErr).Msg("Failed to update job for retry")
		}

		// Log retry attempt with backoff time
		e.db.LogActivity(&models.ActivityLog{
			EventType:  models.EventUpgradeFailed,
			EntityType: "job",
			EntityID:   job.ID,
			Message:    fmt.Sprintf("Upgrade failed for modem %s, will retry in %ds (attempt %d/%d): %v", job.MACAddress, backoffSeconds, job.RetryCount, job.MaxRetries, err),
		})

		log.Info().
			Int("job_id", job.ID).
			Str("mac", job.MACAddress).
			Int("retry_count", job.RetryCount).
			Int("backoff_seconds", backoffSeconds).
			Time("retry_after", retryAfter).
			Msg("Job will be retried with exponential backoff")

		// Job will be picked up again by worker when it polls pending jobs
		// No sleep needed here - this allows other jobs to be processed
		return nil
	}

	// Max retries exceeded, mark as failed
	failed := time.Now()
	job.Status = models.JobStatusFailed
	job.CompletedAt = &failed

	if updateErr := e.db.UpdateJob(job); updateErr != nil {
		return fmt.Errorf("failed to mark job as failed: %w", updateErr)
	}

	// Log final failure
	e.db.LogActivity(&models.ActivityLog{
		EventType:  models.EventUpgradeFailed,
		EntityType: "job",
		EntityID:   job.ID,
		Message:    fmt.Sprintf("Upgrade permanently failed for modem %s after %d attempts: %v", job.MACAddress, job.RetryCount, err),
	})

	return fmt.Errorf("job failed after %d retries: %w", job.RetryCount, err)
}

// discoveryScheduler periodically discovers modems on all enabled CMTS
func (e *Engine) discoveryScheduler(ctx context.Context) {
	// Use poll interval for discovery (configurable)
	ticker := time.NewTicker(e.config.PollInterval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", e.config.PollInterval).
		Msg("Discovery scheduler started")

	// Run once immediately on startup
	e.runDiscoveryForAllCMTS()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Discovery scheduler stopping")
			return
		case <-ticker.C:
			e.runDiscoveryForAllCMTS()
		}
	}
}

// ruleEvaluationScheduler periodically evaluates upgrade rules
func (e *Engine) ruleEvaluationScheduler(ctx context.Context) {
	// Evaluate rules twice as often as discovery
	interval := e.config.PollInterval * 2
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", interval).
		Msg("Rule evaluation scheduler started")

	// Wait a bit after startup to let initial discovery complete
	time.Sleep(30 * time.Second)

	// Run once after initial delay
	if err := e.EvaluateRules(); err != nil {
		log.Error().Err(err).Msg("Initial rule evaluation failed")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Rule evaluation scheduler stopping")
			return
		case <-ticker.C:
			if err := e.EvaluateRules(); err != nil {
				log.Error().Err(err).Msg("Rule evaluation failed")
			}
		}
	}
}

// runDiscoveryForAllCMTS runs discovery for all enabled CMTS devices
func (e *Engine) runDiscoveryForAllCMTS() {
	cmtsList, err := e.db.ListCMTS()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list CMTS for discovery")
		return
	}

	discoveryCount := 0
	for _, cmts := range cmtsList {
		if !cmts.Enabled {
			log.Debug().
				Str("cmts", cmts.Name).
				Msg("Skipping disabled CMTS")
			continue
		}

		// Run discovery in goroutine to avoid blocking
		go func(id int, name string) {
			log.Info().
				Int("cmts_id", id).
				Str("cmts", name).
				Msg("Starting scheduled discovery")

			if err := e.DiscoverModems(id); err != nil {
				log.Error().
					Err(err).
					Int("cmts_id", id).
					Str("cmts", name).
					Msg("Scheduled discovery failed")
			}
		}(cmts.ID, cmts.Name)

		discoveryCount++
	}

	if discoveryCount > 0 {
		log.Info().
			Int("cmts_count", discoveryCount).
			Msg("Triggered discovery for all enabled CMTS")
	}
}

// cleanupScheduler periodically cleans up stale modems
func (e *Engine) cleanupScheduler(ctx context.Context) {
	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Info().Msg("Cleanup scheduler started")

	// Run once immediately on startup
	e.runCleanup()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Cleanup scheduler stopping")
			return
		case <-ticker.C:
			e.runCleanup()
		}
	}
}

// runCleanup marks stale modems as offline and deletes very old modems
func (e *Engine) runCleanup() {
	// Mark modems offline if not seen in 10 minutes (2x discovery interval + buffer)
	// Delete modems that have been offline for 7 days
	markedOffline, deleted, err := e.db.CleanupStaleModems(10, 7)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup stale modems")
		return
	}

	if markedOffline > 0 || deleted > 0 {
		log.Info().
			Int("marked_offline", markedOffline).
			Int("deleted", deleted).
			Msg("Stale modem cleanup completed")

		// Log activity
		if markedOffline > 0 {
			e.db.LogActivity(&models.ActivityLog{
				EventType:  "modem_cleanup",
				EntityType: "system",
				EntityID:   0,
				Message:    fmt.Sprintf("Marked %d stale modems as offline", markedOffline),
			})
		}

		if deleted > 0 {
			e.db.LogActivity(&models.ActivityLog{
				EventType:  "modem_cleanup",
				EntityType: "system",
				EntityID:   0,
				Message:    fmt.Sprintf("Deleted %d old offline modems", deleted),
			})
		}
	}
}
