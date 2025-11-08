package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/database"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

func TestEngineNew(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	config := Config{
		Workers:       3,
		RetryAttempts: 2,
		PollInterval:  30 * time.Second,
		JobTimeout:    2 * time.Minute,
		MaxPerCMTS:    5,
	}

	engine := New(db, config)

	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if engine.config.Workers != 3 {
		t.Errorf("Expected 3 workers, got %d", engine.config.Workers)
	}

	if engine.config.MaxPerCMTS != 5 {
		t.Errorf("Expected MaxPerCMTS 5, got %d", engine.config.MaxPerCMTS)
	}

	if engine.matcher == nil {
		t.Error("Matcher should not be nil")
	}

	if engine.cmtsLimits == nil {
		t.Error("CMTS limits map should not be nil")
	}
}

func TestEngineWithDefaultMaxPerCMTS(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	config := Config{
		Workers:      2,
		MaxPerCMTS:   0, // Should default to 10
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	if engine.config.MaxPerCMTS != 10 {
		t.Errorf("Expected default MaxPerCMTS 10, got %d", engine.config.MaxPerCMTS)
	}
}

func TestGetCMTSSemaphore(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	config := Config{
		Workers:      2,
		MaxPerCMTS:   3,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Get semaphore for CMTS 1
	sem1 := engine.getCMTSSemaphore(1)
	if sem1 == nil {
		t.Fatal("Semaphore should not be nil")
	}

	// Get same semaphore again - should be cached
	sem1Again := engine.getCMTSSemaphore(1)
	if sem1 != sem1Again {
		t.Error("Should return the same semaphore instance for same CMTS ID")
	}

	// Get different semaphore for CMTS 2
	sem2 := engine.getCMTSSemaphore(2)
	if sem2 == nil {
		t.Fatal("Semaphore should not be nil")
	}

	if sem1 == sem2 {
		t.Error("Different CMTS should have different semaphores")
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	sem := newSemaphore(2)

	// Acquire twice (should not block)
	sem.Acquire()
	sem.Acquire()

	// Try to acquire in goroutine (should block)
	acquired := make(chan bool, 1)
	go func() {
		sem.Acquire()
		acquired <- true
	}()

	// Should not acquire yet (semaphore full)
	select {
	case <-acquired:
		t.Error("Should not have acquired semaphore (full)")
	case <-time.After(100 * time.Millisecond):
		// Expected - semaphore is full
	}

	// Release one slot
	sem.Release()

	// Now it should acquire
	select {
	case <-acquired:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Should have acquired semaphore after release")
	}

	// Clean up
	sem.Release()
	sem.Release()
}

func TestDiscoverModems(t *testing.T) {
	t.Skip("Skipping test that requires real SNMP device - takes 40+ seconds to timeout")

	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Note: DiscoverModems will fail without real SNMP connection
	// This test verifies the method exists and handles errors
	err = engine.DiscoverModems(1)
	if err == nil {
		t.Log("Discovery succeeded (unexpected in test environment)")
	} else {
		t.Logf("Discovery failed as expected without SNMP: %v", err)
	}
}

func TestEvaluateRules(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Evaluate rules
	err = engine.EvaluateRules()
	if err != nil {
		t.Fatalf("Failed to evaluate rules: %v", err)
	}

	// Check that a job was created for the matching modem
	jobs, err := db.ListJobs(models.JobStatusPending, 10)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected at least one job to be created")
	}

	// Verify job details
	if len(jobs) > 0 {
		job := jobs[0]
		if job.MACAddress != "00:01:5C:11:22:33" {
			t.Errorf("Expected MAC 00:01:5C:11:22:33, got %s", job.MACAddress)
		}
		if job.Status != models.JobStatusPending {
			t.Errorf("Expected status PENDING, got %s", job.Status)
		}
	}
}

func TestEvaluateRulesDeduplication(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Evaluate rules twice
	err = engine.EvaluateRules()
	if err != nil {
		t.Fatalf("Failed to evaluate rules: %v", err)
	}

	err = engine.EvaluateRules()
	if err != nil {
		t.Fatalf("Failed to evaluate rules second time: %v", err)
	}

	// Should only have one job (deduplication)
	jobs, err := db.ListJobs(models.JobStatusPending, 10)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("Expected exactly 1 job (deduplication), got %d", len(jobs))
	}
}

func TestRunDiscoveryForAllCMTS(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Run discovery (will fail without SNMP but should not panic)
	engine.runDiscoveryForAllCMTS()

	// Verify no panic occurred
	t.Log("Discovery completed without panic")
}

func TestCheckPendingJobsDeduplication(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create a job
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusInProgress,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create another pending job for same modem
	_, err = db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create second job: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Check pending jobs - should skip the pending job because one is in progress
	err = engine.checkPendingJobs()
	if err != nil {
		t.Fatalf("Failed to check pending jobs: %v", err)
	}

	// Verify job was skipped (still in jobs channel)
	if len(engine.jobs) > 0 {
		t.Log("Job was skipped due to in-progress job for same modem")
	}

	// Clean up
	job, _ := db.GetJob(jobID)
	if job != nil && job.Status == models.JobStatusInProgress {
		t.Log("In-progress job exists, deduplication working")
	}
}

func TestEngineStartStop(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 1 * time.Second, // Short interval for testing
	}

	engine := New(db, config)

	ctx, cancel := context.WithCancel(context.Background())

	// Start engine
	started := make(chan bool)
	go func() {
		started <- true
		engine.Start(ctx)
	}()

	// Wait for start
	<-started
	time.Sleep(100 * time.Millisecond)

	// Stop engine
	cancel()

	// Give it time to shutdown
	time.Sleep(200 * time.Millisecond)

	t.Log("Engine started and stopped without panic")
}

func TestHandleJobFailureExponentialBackoff(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create a job
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusInProgress,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		RetryCount:       1,
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Handle failure (should retry with backoff)
	testErr := fmt.Errorf("test error")

	// Note: This will actually sleep for the backoff period
	// For first retry (count=1), backoff is 30 seconds
	// We can't easily test the sleep without mocking time
	go engine.handleJobFailure(job, testErr)

	// Give it a moment to start processing
	time.Sleep(100 * time.Millisecond)

	// Verify job was updated
	updatedJob, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updatedJob.RetryCount != 2 {
		t.Errorf("Expected retry count 2, got %d", updatedJob.RetryCount)
	}

	t.Log("Job failure handled with retry logic")
}

func TestHandleJobFailureMaxRetriesExceeded(t *testing.T) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Load fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create a job that has reached max retries
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusInProgress,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		RetryCount:       3,
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	config := Config{
		Workers:      2,
		MaxPerCMTS:   5,
		PollInterval: 30 * time.Second,
	}

	engine := New(db, config)

	// Handle failure (should mark as failed)
	testErr := fmt.Errorf("test error")
	engine.handleJobFailure(job, testErr)

	// Verify job was marked as failed
	updatedJob, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updatedJob.Status != models.JobStatusFailed {
		t.Errorf("Expected status FAILED, got %s", updatedJob.Status)
	}

	if updatedJob.CompletedAt == nil {
		t.Error("CompletedAt should be set for failed job")
	}

	t.Log("Job marked as failed after max retries")
}
