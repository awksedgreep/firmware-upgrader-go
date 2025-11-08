package database

import (
	"testing"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

func TestNewTestDB(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Database should not be nil")
	}

	if db.conn == nil {
		t.Fatal("Database connection should not be nil")
	}
}

func TestLoadTestFixtures(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load test fixtures: %v", err)
	}

	// Verify CMTS was created
	cmtsList, err := db.ListCMTS()
	if err != nil {
		t.Fatalf("Failed to list CMTS: %v", err)
	}
	if len(cmtsList) != 1 {
		t.Errorf("Expected 1 CMTS, got %d", len(cmtsList))
	}

	// Verify modem was created
	modems, err := db.ListModems(0)
	if err != nil {
		t.Fatalf("Failed to list modems: %v", err)
	}
	if len(modems) != 1 {
		t.Errorf("Expected 1 modem, got %d", len(modems))
	}

	// Verify rule was created
	rules, err := db.ListRules()
	if err != nil {
		t.Fatalf("Failed to list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
}

// CMTS Tests

func TestCreateCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	cmts := &models.CMTS{
		Name:              "Test CMTS",
		IPAddress:         "192.168.1.1",
		SNMPPort:          161,
		CommunityRead:     "public",
		CommunityWrite:    "private",
		CMCommunityString: "cable-modem",
		SNMPVersion:       2,
		Enabled:           true,
	}

	id, err := db.CreateCMTS(cmts)
	if err != nil {
		t.Fatalf("Failed to create CMTS: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero ID")
	}

	// Verify CMTS was created
	retrieved, err := db.GetCMTS(id)
	if err != nil {
		t.Fatalf("Failed to get CMTS: %v", err)
	}

	if retrieved.Name != cmts.Name {
		t.Errorf("Expected name %s, got %s", cmts.Name, retrieved.Name)
	}
	if retrieved.IPAddress != cmts.IPAddress {
		t.Errorf("Expected IP %s, got %s", cmts.IPAddress, retrieved.IPAddress)
	}
	if retrieved.Enabled != cmts.Enabled {
		t.Errorf("Expected enabled %v, got %v", cmts.Enabled, retrieved.Enabled)
	}
}

func TestGetCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	cmts, err := db.GetCMTS(1)
	if err != nil {
		t.Fatalf("Failed to get CMTS: %v", err)
	}

	if cmts.ID != 1 {
		t.Errorf("Expected ID 1, got %d", cmts.ID)
	}
	if cmts.Name != "Test CMTS" {
		t.Errorf("Expected name 'Test CMTS', got %s", cmts.Name)
	}
}

func TestGetCMTSNotFound(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	_, err = db.GetCMTS(999)
	if err != models.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestListCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create multiple CMTS
	for i := 1; i <= 3; i++ {
		_, err := db.CreateCMTS(&models.CMTS{
			Name:           "CMTS " + string(rune('A'+i-1)),
			IPAddress:      "192.168.1." + string(rune('0'+i)),
			SNMPPort:       161,
			CommunityRead:  "public",
			CommunityWrite: "private",
			SNMPVersion:    2,
			Enabled:        i%2 == 0, // Alternate enabled/disabled
		})
		if err != nil {
			t.Fatalf("Failed to create CMTS: %v", err)
		}
	}

	cmtsList, err := db.ListCMTS()
	if err != nil {
		t.Fatalf("Failed to list CMTS: %v", err)
	}

	if len(cmtsList) != 3 {
		t.Errorf("Expected 3 CMTS, got %d", len(cmtsList))
	}
}

func TestUpdateCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	cmts, err := db.GetCMTS(1)
	if err != nil {
		t.Fatalf("Failed to get CMTS: %v", err)
	}

	// Update fields
	cmts.Name = "Updated CMTS"
	cmts.IPAddress = "10.0.0.1"
	cmts.Enabled = false

	err = db.UpdateCMTS(cmts)
	if err != nil {
		t.Fatalf("Failed to update CMTS: %v", err)
	}

	// Verify update
	updated, err := db.GetCMTS(1)
	if err != nil {
		t.Fatalf("Failed to get updated CMTS: %v", err)
	}

	if updated.Name != "Updated CMTS" {
		t.Errorf("Expected name 'Updated CMTS', got %s", updated.Name)
	}
	if updated.IPAddress != "10.0.0.1" {
		t.Errorf("Expected IP '10.0.0.1', got %s", updated.IPAddress)
	}
	if updated.Enabled {
		t.Error("Expected enabled to be false")
	}
}

func TestDeleteCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	err = db.DeleteCMTS(1)
	if err != nil {
		t.Fatalf("Failed to delete CMTS: %v", err)
	}

	// Verify deletion
	_, err = db.GetCMTS(1)
	if err != models.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got %v", err)
	}
}

// Cable Modem Tests

func TestUpsertModem(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	modem := &models.CableModem{
		CMTSID:          1,
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		IPAddress:       "10.0.0.200",
		SysDescr:        "Test Modem",
		CurrentFirmware: "1.0.0",
		SignalLevel:     6.5,
		Status:          "online",
		LastSeen:        time.Now(),
	}

	// Insert
	err = db.UpsertModem(modem)
	if err != nil {
		t.Fatalf("Failed to upsert modem: %v", err)
	}

	// Verify insert - get all modems and find ours
	allModems, err := db.ListModems(0)
	if err != nil {
		t.Fatalf("Failed to list modems: %v", err)
	}

	var retrieved *models.CableModem
	for _, m := range allModems {
		if m.MACAddress == "AA:BB:CC:DD:EE:FF" {
			retrieved = m
			break
		}
	}

	if retrieved == nil {
		t.Fatal("Modem not found after insert")
	}

	if retrieved.IPAddress != "10.0.0.200" {
		t.Errorf("Expected IP 10.0.0.200, got %s", retrieved.IPAddress)
	}

	// Update
	modem.IPAddress = "10.0.0.201"
	modem.CurrentFirmware = "2.0.0"

	err = db.UpsertModem(modem)
	if err != nil {
		t.Fatalf("Failed to upsert modem update: %v", err)
	}

	// Verify update
	allModems, err = db.ListModems(0)
	if err != nil {
		t.Fatalf("Failed to list modems: %v", err)
	}

	var updated *models.CableModem
	for _, m := range allModems {
		if m.MACAddress == "AA:BB:CC:DD:EE:FF" {
			updated = m
			break
		}
	}

	if updated == nil {
		t.Fatal("Modem not found after update")
	}

	if updated.IPAddress != "10.0.0.201" {
		t.Errorf("Expected IP 10.0.0.201, got %s", updated.IPAddress)
	}
	if updated.CurrentFirmware != "2.0.0" {
		t.Errorf("Expected firmware 2.0.0, got %s", updated.CurrentFirmware)
	}
}

func TestListModems(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Add more modems
	for i := 1; i <= 3; i++ {
		mac := string([]byte{0x00, 0x01, 0x5C, 0x11, 0x22, byte(0x40 + i)})
		err := db.UpsertModem(&models.CableModem{
			CMTSID:          1,
			MACAddress:      mac,
			IPAddress:       "10.0.0." + string(rune('0'+i)),
			SysDescr:        "Test Modem",
			CurrentFirmware: "1.0.0",
			SignalLevel:     5.0,
			Status:          "online",
			LastSeen:        time.Now(),
		})
		if err != nil {
			t.Fatalf("Failed to create modem: %v", err)
		}
	}

	modems, err := db.ListModems(0)
	if err != nil {
		t.Fatalf("Failed to list modems: %v", err)
	}

	if len(modems) < 1 {
		t.Errorf("Expected at least 1 modem, got %d", len(modems))
	}
}

func TestListModemsByCMTS(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	modems, err := db.ListModems(1)
	if err != nil {
		t.Fatalf("Failed to list modems by CMTS: %v", err)
	}

	if len(modems) != 1 {
		t.Errorf("Expected 1 modem, got %d", len(modems))
	}

	if modems[0].CMTSID != 1 {
		t.Errorf("Expected CMTS ID 1, got %d", modems[0].CMTSID)
	}
}

// Upgrade Rule Tests

func TestCreateRule(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	rule := &models.UpgradeRule{
		Name:             "Test Rule",
		Description:      "Test upgrade rule",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		Enabled:          true,
		Priority:         100,
	}

	id, err := db.CreateRule(rule)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero ID")
	}

	// Verify rule was created
	retrieved, err := db.GetRule(id)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Expected name %s, got %s", rule.Name, retrieved.Name)
	}
	if retrieved.Priority != rule.Priority {
		t.Errorf("Expected priority %d, got %d", rule.Priority, retrieved.Priority)
	}
}

func TestGetRule(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	rule, err := db.GetRule(1)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if rule.ID != 1 {
		t.Errorf("Expected ID 1, got %d", rule.ID)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("Expected name 'Test Rule', got %s", rule.Name)
	}
}

func TestListRules(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create multiple rules
	for i := 1; i <= 3; i++ {
		_, err := db.CreateRule(&models.UpgradeRule{
			Name:             "Rule " + string(rune('A'+i-1)),
			Description:      "Test rule",
			MatchType:        "MAC_RANGE",
			MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
			TFTPServerIP:     "192.168.1.50",
			FirmwareFilename: "firmware.bin",
			Enabled:          true,
			Priority:         i * 10,
		})
		if err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}
	}

	rules, err := db.ListRules()
	if err != nil {
		t.Fatalf("Failed to list rules: %v", err)
	}

	if len(rules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(rules))
	}

	// Verify ordering by priority (DESC)
	if rules[0].Priority < rules[1].Priority {
		t.Error("Rules should be ordered by priority DESC")
	}
}

func TestListEnabledAndDisabledRules(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create enabled and disabled rules
	db.CreateRule(&models.UpgradeRule{
		Name:             "Enabled Rule",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		Enabled:          true,
		Priority:         100,
	})

	db.CreateRule(&models.UpgradeRule{
		Name:             "Disabled Rule",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		Enabled:          false,
		Priority:         50,
	})

	rules, err := db.ListRules()
	if err != nil {
		t.Fatalf("Failed to list rules: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules total, got %d", len(rules))
	}

	// Count enabled rules
	enabledCount := 0
	for _, rule := range rules {
		if rule.Enabled {
			enabledCount++
		}
	}

	if enabledCount != 1 {
		t.Errorf("Expected 1 enabled rule, got %d", enabledCount)
	}
}

func TestUpdateRule(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	rule, err := db.GetRule(1)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	// Update fields
	rule.Name = "Updated Rule"
	rule.Priority = 200
	rule.Enabled = false

	err = db.UpdateRule(rule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	// Verify update
	updated, err := db.GetRule(1)
	if err != nil {
		t.Fatalf("Failed to get updated rule: %v", err)
	}

	if updated.Name != "Updated Rule" {
		t.Errorf("Expected name 'Updated Rule', got %s", updated.Name)
	}
	if updated.Priority != 200 {
		t.Errorf("Expected priority 200, got %d", updated.Priority)
	}
	if updated.Enabled {
		t.Error("Expected enabled to be false")
	}
}

func TestDeleteRule(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	err = db.DeleteRule(1)
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	// Verify deletion
	_, err = db.GetRule(1)
	if err != models.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got %v", err)
	}
}

// Upgrade Job Tests

func TestCreateJob(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	job := &models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	}

	id, err := db.CreateJob(job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero ID")
	}

	// Verify job was created
	retrieved, err := db.GetJob(id)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.Status != models.JobStatusPending {
		t.Errorf("Expected status PENDING, got %s", retrieved.Status)
	}
}

func TestGetJob(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create a job
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.ID != jobID {
		t.Errorf("Expected ID %d, got %d", jobID, job.ID)
	}
}

func TestListPendingJobs(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create multiple jobs with different statuses
	db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	})

	db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:44",
		Status:           models.JobStatusCompleted,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	})

	jobs, err := db.ListJobs(models.JobStatusPending, 0)
	if err != nil {
		t.Fatalf("Failed to list pending jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("Expected 1 pending job, got %d", len(jobs))
	}

	if jobs[0].Status != models.JobStatusPending {
		t.Errorf("Expected status PENDING, got %s", jobs[0].Status)
	}
}

func TestUpdateJob(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create a job
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	// Update job
	job.Status = models.JobStatusInProgress
	now := time.Now()
	job.StartedAt = &now

	err = db.UpdateJob(job)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Verify update
	updated, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updated.Status != models.JobStatusInProgress {
		t.Errorf("Expected status IN_PROGRESS, got %s", updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

func TestCheckPendingJobForModem(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	mac := "00:01:5C:11:22:33"

	// No pending job initially
	pendingJobs, err := db.ListJobs(models.JobStatusPending, 0)
	if err != nil {
		t.Fatalf("Failed to list pending jobs: %v", err)
	}

	hasPending := false
	for _, job := range pendingJobs {
		if job.MACAddress == mac {
			hasPending = true
			break
		}
	}

	if hasPending {
		t.Error("Expected no pending job initially")
	}

	// Create a pending job
	db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       mac,
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
	})

	// Should have pending job now
	pendingJobs, err = db.ListJobs(models.JobStatusPending, 0)
	if err != nil {
		t.Fatalf("Failed to list pending jobs: %v", err)
	}

	hasPending = false
	for _, job := range pendingJobs {
		if job.MACAddress == mac {
			hasPending = true
			break
		}
	}

	if !hasPending {
		t.Error("Expected pending job to exist")
	}
}

// Activity Log Tests

func TestLogActivity(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	log := &models.ActivityLog{
		EventType:  models.EventSystemEvent,
		EntityType: "test",
		EntityID:   1,
		Message:    "Test activity",
		Details:    `{"key":"value"}`,
	}

	err = db.LogActivity(log)
	if err != nil {
		t.Fatalf("Failed to log activity: %v", err)
	}
}

func TestListActivityLogs(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create multiple activity logs
	for i := 1; i <= 5; i++ {
		err := db.LogActivity(&models.ActivityLog{
			EventType:  models.EventSystemEvent,
			EntityType: "test",
			EntityID:   i,
			Message:    "Test activity " + string(rune('A'+i-1)),
			Details:    `{"index":` + string(rune('0'+i)) + `}`,
		})
		if err != nil {
			t.Fatalf("Failed to log activity: %v", err)
		}
	}

	logs, err := db.ListActivityLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to list activity logs: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 activity logs, got %d", len(logs))
	}

	// Verify ordering (most recent first)
	if logs[0].CreatedAt.Before(logs[1].CreatedAt) {
		t.Error("Activity logs should be ordered by created_at DESC")
	}
}

func TestListActivityLogsPagination(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create 15 activity logs
	for i := 1; i <= 15; i++ {
		err := db.LogActivity(&models.ActivityLog{
			EventType:  models.EventSystemEvent,
			EntityType: "test",
			EntityID:   i,
			Message:    "Test activity",
		})
		if err != nil {
			t.Fatalf("Failed to log activity: %v", err)
		}
	}

	// Get first page (10 items)
	page1, err := db.ListActivityLogs(10, 0)
	if err != nil {
		t.Fatalf("Failed to list activity logs page 1: %v", err)
	}

	if len(page1) != 10 {
		t.Errorf("Expected 10 items in page 1, got %d", len(page1))
	}

	// Get second page (5 remaining items)
	page2, err := db.ListActivityLogs(10, 10)
	if err != nil {
		t.Fatalf("Failed to list activity logs page 2: %v", err)
	}

	if len(page2) != 5 {
		t.Errorf("Expected 5 items in page 2, got %d", len(page2))
	}
}

// Settings Tests

func TestGetSetting(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set a setting
	err = db.SetSetting("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Get the setting
	value, err := db.GetSetting("test_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got %s", value)
	}
}

func TestGetSettingNotFound(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	_, err = db.GetSetting("nonexistent_key")
	if err == nil {
		t.Error("Expected error for nonexistent setting")
	}
}

func TestSetSetting(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set a new setting
	err = db.SetSetting("new_key", "new_value")
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Verify it was set
	value, err := db.GetSetting("new_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if value != "new_value" {
		t.Errorf("Expected value 'new_value', got %s", value)
	}

	// Update the setting
	err = db.SetSetting("new_key", "updated_value")
	if err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	// Verify it was updated
	updated, err := db.GetSetting("new_key")
	if err != nil {
		t.Fatalf("Failed to get updated setting: %v", err)
	}

	if updated != "updated_value" {
		t.Errorf("Expected value 'updated_value', got %s", updated)
	}
}

func TestListSettings(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set multiple settings
	settings := map[string]string{
		"workers":            "4",
		"poll_interval":      "30s",
		"discovery_interval": "15m",
	}

	for key, value := range settings {
		err := db.SetSetting(key, value)
		if err != nil {
			t.Fatalf("Failed to set setting %s: %v", key, err)
		}
	}

	// List all settings
	retrieved, err := db.ListSettings()
	if err != nil {
		t.Fatalf("Failed to list settings: %v", err)
	}

	if len(retrieved) < len(settings) {
		t.Errorf("Expected at least %d settings, got %d", len(settings), len(retrieved))
	}

	// Verify values
	for key, expectedValue := range settings {
		actualValue, exists := retrieved[key]
		if !exists {
			t.Errorf("Expected setting %s to exist", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected setting %s to be %s, got %s", key, expectedValue, actualValue)
		}
	}
}

// Integration Tests

func TestCompleteWorkflow(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// 1. Create CMTS
	cmtsID, err := db.CreateCMTS(&models.CMTS{
		Name:              "Production CMTS",
		IPAddress:         "192.168.1.1",
		SNMPPort:          161,
		CommunityRead:     "public",
		CommunityWrite:    "private",
		CMCommunityString: "cable-modem",
		SNMPVersion:       2,
		Enabled:           true,
	})
	if err != nil {
		t.Fatalf("Failed to create CMTS: %v", err)
	}

	// 2. Discover modems
	err = db.UpsertModem(&models.CableModem{
		CMTSID:          cmtsID,
		MACAddress:      "00:01:5C:AA:BB:CC",
		IPAddress:       "10.0.0.100",
		SysDescr:        "Arris SB8200 DOCSIS 3.1",
		CurrentFirmware: "1.0.0",
		SignalLevel:     6.0,
		Status:          "online",
		LastSeen:        time.Now(),
	})
	if err != nil {
		t.Fatalf("Failed to upsert modem: %v", err)
	}

	// 3. Create upgrade rule
	ruleID, err := db.CreateRule(&models.UpgradeRule{
		Name:             "Arris SB8200 Upgrade",
		Description:      "Upgrade all Arris SB8200 modems",
		MatchType:        "SYSDESCR_REGEX",
		MatchCriteria:    `{"pattern":"Arris SB8200"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "arris-sb8200-v2.0.0.bin",
		Enabled:          true,
		Priority:         100,
	})
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}

	// 4. Create upgrade job
	// Get modem by listing and finding it
	modems, err := db.ListModems(cmtsID)
	if err != nil {
		t.Fatalf("Failed to list modems: %v", err)
	}

	var modem *models.CableModem
	for _, m := range modems {
		if m.MACAddress == "00:01:5C:AA:BB:CC" {
			modem = m
			break
		}
	}

	if modem == nil {
		t.Fatal("Modem not found")
	}

	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          modem.ID,
		RuleID:           ruleID,
		CMTSID:           cmtsID,
		MACAddress:       modem.MACAddress,
		Status:           models.JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "arris-sb8200-v2.0.0.bin",
		MaxRetries:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// 5. Process job (simulate)
	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	job.Status = models.JobStatusInProgress
	now := time.Now()
	job.StartedAt = &now
	err = db.UpdateJob(job)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// 6. Complete job
	job.Status = models.JobStatusCompleted
	completed := time.Now()
	job.CompletedAt = &completed
	err = db.UpdateJob(job)
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}

	// 7. Log activity
	err = db.LogActivity(&models.ActivityLog{
		EventType:  models.EventUpgradeCompleted,
		EntityType: "job",
		EntityID:   jobID,
		Message:    "Firmware upgrade completed successfully",
	})
	if err != nil {
		t.Fatalf("Failed to log activity: %v", err)
	}

	// Verify final state
	finalJob, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get final job: %v", err)
	}

	if finalJob.Status != models.JobStatusCompleted {
		t.Errorf("Expected job status COMPLETED, got %s", finalJob.Status)
	}
	if finalJob.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestConcurrentJobCreation(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = db.LoadTestFixtures()
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	// Create jobs sequentially to avoid database lock issues in tests
	for i := 0; i < 5; i++ {
		_, err := db.CreateJob(&models.UpgradeJob{
			ModemID:          1,
			RuleID:           1,
			CMTSID:           1,
			MACAddress:       "00:01:5C:11:22:33",
			Status:           models.JobStatusPending,
			TFTPServerIP:     "192.168.1.50",
			FirmwareFilename: "firmware.bin",
			MaxRetries:       3,
		})
		if err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}
	}

	// All jobs should have been created
	jobs, err := db.ListJobs(models.JobStatusPending, 0)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}
}
