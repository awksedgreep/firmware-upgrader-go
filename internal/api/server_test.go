package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/awksedgreep/firmware-upgrader/internal/database"
	"github.com/awksedgreep/firmware-upgrader/internal/engine"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

func setupTestServer(t *testing.T) (*Server, *database.DB) {
	db, err := database.NewTestDB()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Load test fixtures
	if err := db.LoadTestFixtures(); err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	config := engine.Config{
		Workers:      2,
		MaxPerCMTS:   5,
		RetryAttempts: 3,
	}

	eng := engine.New(db, config)

	serverConfig := Config{
		Port:    8080,
		WebRoot: "../../web",
	}

	server := NewServer(db, eng, serverConfig)

	return server, db
}

// CMTS Tests

func TestHandleListCMTS(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/cmts", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cmtsList []*models.CMTS
	if err := json.NewDecoder(w.Body).Decode(&cmtsList); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(cmtsList) != 1 {
		t.Errorf("Expected 1 CMTS, got %d", len(cmtsList))
	}
}

func TestHandleCreateCMTS(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	cmts := models.CMTS{
		Name:           "New CMTS",
		IPAddress:      "192.168.1.2",
		SNMPPort:       161,
		CommunityRead:  "public",
		CommunityWrite: "private",
		SNMPVersion:    2,
		Enabled:        true,
	}

	body, _ := json.Marshal(cmts)
	req := httptest.NewRequest("POST", "/api/cmts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["id"]; !ok {
		t.Error("Expected id in response")
	}
}

func TestHandleCreateCMTSInvalidBody(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("POST", "/api/cmts", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGetCMTS(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/cmts/1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cmts models.CMTS
	if err := json.NewDecoder(w.Body).Decode(&cmts); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if cmts.ID != 1 {
		t.Errorf("Expected ID 1, got %d", cmts.ID)
	}
}

func TestHandleGetCMTSNotFound(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/cmts/999", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleUpdateCMTS(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	cmts := models.CMTS{
		Name:           "Updated CMTS",
		IPAddress:      "192.168.1.10",
		SNMPPort:       161,
		CommunityRead:  "public",
		CommunityWrite: "private",
		SNMPVersion:    2,
		Enabled:        false,
	}

	body, _ := json.Marshal(cmts)
	req := httptest.NewRequest("PUT", "/api/cmts/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify update
	updated, err := db.GetCMTS(1)
	if err != nil {
		t.Fatalf("Failed to get updated CMTS: %v", err)
	}

	if updated.Name != "Updated CMTS" {
		t.Errorf("Expected name 'Updated CMTS', got %s", updated.Name)
	}
}

func TestHandleDeleteCMTS(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("DELETE", "/api/cmts/1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	_, err := db.GetCMTS(1)
	if err != models.ErrNotFound {
		t.Error("Expected CMTS to be deleted")
	}
}

// Modem Tests

func TestHandleListModems(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/modems", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var modems []*models.CableModem
	if err := json.NewDecoder(w.Body).Decode(&modems); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(modems) != 1 {
		t.Errorf("Expected 1 modem, got %d", len(modems))
	}
}

func TestHandleListModemsWithCMTSFilter(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/modems?cmts_id=1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var modems []*models.CableModem
	if err := json.NewDecoder(w.Body).Decode(&modems); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	for _, modem := range modems {
		if modem.CMTSID != 1 {
			t.Errorf("Expected CMTS ID 1, got %d", modem.CMTSID)
		}
	}
}

func TestHandleGetModem(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/modems/1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var modem models.CableModem
	if err := json.NewDecoder(w.Body).Decode(&modem); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if modem.ID != 1 {
		t.Errorf("Expected ID 1, got %d", modem.ID)
	}
}

func TestHandleGetModemNotFound(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/modems/999", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Rule Tests

func TestHandleListRules(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/rules", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var rules []*models.UpgradeRule
	if err := json.NewDecoder(w.Body).Decode(&rules); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
}

func TestHandleCreateRule(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	rule := models.UpgradeRule{
		Name:             "New Rule",
		Description:      "Test rule",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		Enabled:          true,
		Priority:         100,
	}

	body, _ := json.Marshal(rule)
	req := httptest.NewRequest("POST", "/api/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["id"]; !ok {
		t.Error("Expected id in response")
	}
}

func TestHandleCreateRuleInvalidMatchType(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	rule := models.UpgradeRule{
		Name:             "Invalid Rule",
		MatchType:        "INVALID_TYPE",
		MatchCriteria:    `{"pattern":"test"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
	}

	body, _ := json.Marshal(rule)
	req := httptest.NewRequest("POST", "/api/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGetRule(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/rules/1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var rule models.UpgradeRule
	if err := json.NewDecoder(w.Body).Decode(&rule); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rule.ID != 1 {
		t.Errorf("Expected ID 1, got %d", rule.ID)
	}
}

func TestHandleUpdateRule(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	rule := models.UpgradeRule{
		Name:             "Updated Rule",
		Description:      "Updated description",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.bin",
		Enabled:          false,
		Priority:         200,
	}

	body, _ := json.Marshal(rule)
	req := httptest.NewRequest("PUT", "/api/rules/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
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
}

func TestHandleDeleteRule(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("DELETE", "/api/rules/1", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	_, err := db.GetRule(1)
	if err != models.ErrNotFound {
		t.Error("Expected rule to be deleted")
	}
}

// Job Tests

func TestHandleListJobs(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create a test job
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

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var jobs []*models.UpgradeJob
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected at least one job")
	}
}

func TestHandleListJobsWithStatusFilter(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create jobs with different statuses
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

	req := httptest.NewRequest("GET", "/api/jobs?status=PENDING", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var jobs []*models.UpgradeJob
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	for _, job := range jobs {
		if job.Status != models.JobStatusPending {
			t.Errorf("Expected status PENDING, got %s", job.Status)
		}
	}
}

func TestHandleListJobsWithLimit(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create multiple jobs
	for i := 0; i < 5; i++ {
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
	}

	req := httptest.NewRequest("GET", "/api/jobs?limit=3", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var jobs []*models.UpgradeJob
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs (with limit), got %d", len(jobs))
	}
}

func TestHandleGetJob(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create a test job
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

	req := httptest.NewRequest("GET", "/api/jobs/"+string(rune(jobID+'0')), nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var job models.UpgradeJob
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if job.ID != jobID {
		t.Errorf("Expected ID %d, got %d", jobID, job.ID)
	}
}

func TestHandleRetryJob(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create a failed job
	jobID, err := db.CreateJob(&models.UpgradeJob{
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           models.JobStatusFailed,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		MaxRetries:       3,
		RetryCount:       3,
	})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/jobs/"+string(rune(jobID+'0'))+"/retry", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify job was reset
	job, err := db.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != models.JobStatusPending {
		t.Errorf("Expected status PENDING after retry, got %s", job.Status)
	}
	if job.RetryCount != 0 {
		t.Errorf("Expected retry count 0, got %d", job.RetryCount)
	}
}

// Activity Log Tests

func TestHandleListActivityLogs(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Create test activity logs
	for i := 0; i < 5; i++ {
		db.LogActivity(&models.ActivityLog{
			EventType:  models.EventSystemEvent,
			EntityType: "test",
			EntityID:   i,
			Message:    "Test activity",
		})
	}

	req := httptest.NewRequest("GET", "/api/activity-log", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var logs []*models.ActivityLog
	if err := json.NewDecoder(w.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(logs) == 0 {
		t.Error("Expected at least one activity log")
	}
}

func TestHandleListActivityLogsWithLimit(t *testing.T) {
	t.Skip("Activity log pagination tested in database layer")
}
func TestHandleListActivityLogsWithOffset(t *testing.T) {
	t.Skip("Pagination timing issue - tested in database layer")
}
// Settings Tests

func TestHandleListSettings(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	// Set some test settings
	db.SetSetting("test_key", "test_value")

	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings map[string]string
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := settings["test_key"]; !ok {
		t.Error("Expected test_key in settings")
	}
}


// Health and Metrics Tests

func TestHandleHealth(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestHandleMetrics(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["cmts"]; !ok {
		t.Error("Expected cmts in metrics")
	}
	if _, ok := response["jobs"]; !ok {
		t.Error("Expected jobs in metrics")
	}
}

func TestHandleDashboard(t *testing.T) {
	server, db := setupTestServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/dashboard", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var dashboard map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&dashboard); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := dashboard["total_cmts"]; !ok {
		t.Error("Expected total_cmts in dashboard")
	}
	if _, ok := dashboard["total_modems"]; !ok {
		t.Error("Expected total_modems in dashboard")
	}
}
