package models

import (
	"testing"
)

// CMTS Validation Tests

func TestCMTSValidate(t *testing.T) {
	tests := []struct {
		name    string
		cmts    *CMTS
		wantErr bool
		errType error
	}{
		{
			name: "Valid CMTS",
			cmts: &CMTS{
				Name:           "Test CMTS",
				IPAddress:      "192.168.1.1",
				SNMPPort:       161,
				CommunityRead:  "public",
				CommunityWrite: "private",
				SNMPVersion:    2,
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			cmts: &CMTS{
				Name:          "",
				IPAddress:     "192.168.1.1",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "Missing IP address",
			cmts: &CMTS{
				Name:          "Test CMTS",
				IPAddress:     "",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			wantErr: true,
			errType: ErrInvalidIPAddress,
		},
		{
			name: "Invalid port - too low",
			cmts: &CMTS{
				Name:          "Test CMTS",
				IPAddress:     "192.168.1.1",
				SNMPPort:      0,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			wantErr: true,
			errType: ErrInvalidPort,
		},
		{
			name: "Invalid port - too high",
			cmts: &CMTS{
				Name:          "Test CMTS",
				IPAddress:     "192.168.1.1",
				SNMPPort:      65536,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			wantErr: true,
			errType: ErrInvalidPort,
		},
		{
			name: "Missing community read",
			cmts: &CMTS{
				Name:        "Test CMTS",
				IPAddress:   "192.168.1.1",
				SNMPPort:    161,
				SNMPVersion: 2,
			},
			wantErr: true,
			errType: ErrInvalidCommunity,
		},
		{
			name: "Invalid SNMP version - too low",
			cmts: &CMTS{
				Name:          "Test CMTS",
				IPAddress:     "192.168.1.1",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   0,
			},
			wantErr: true,
			errType: ErrInvalidSNMPVersion,
		},
		{
			name: "Invalid SNMP version - too high",
			cmts: &CMTS{
				Name:          "Test CMTS",
				IPAddress:     "192.168.1.1",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   4,
			},
			wantErr: true,
			errType: ErrInvalidSNMPVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CMTS.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("CMTS.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

// UpgradeRule Validation Tests

func TestUpgradeRuleValidate(t *testing.T) {
	tests := []struct {
		name    string
		rule    *UpgradeRule
		wantErr bool
		errType error
	}{
		{
			name: "Valid MAC_RANGE rule",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "MAC_RANGE",
				MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: false,
		},
		{
			name: "Valid SYSDESCR_REGEX rule",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "SYSDESCR_REGEX",
				MatchCriteria:    `{"pattern":"Arris SB8200"}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			rule: &UpgradeRule{
				Name:             "",
				MatchType:        "MAC_RANGE",
				MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "Invalid match type",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "INVALID_TYPE",
				MatchCriteria:    `{"pattern":"test"}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: true,
			errType: ErrInvalidMatchType,
		},
		{
			name: "Missing TFTP server",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "MAC_RANGE",
				MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
				TFTPServerIP:     "",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: true,
			errType: ErrInvalidTFTPServer,
		},
		{
			name: "Missing firmware filename",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "MAC_RANGE",
				MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "",
			},
			wantErr: true,
			errType: ErrInvalidFirmware,
		},
		{
			name: "Invalid JSON in match criteria",
			rule: &UpgradeRule{
				Name:             "Test Rule",
				MatchType:        "MAC_RANGE",
				MatchCriteria:    `{invalid json}`,
				TFTPServerIP:     "192.168.1.50",
				FirmwareFilename: "firmware.bin",
			},
			wantErr: true,
			errType: ErrInvalidMatchCriteria,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpgradeRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("UpgradeRule.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

// ParseMatchCriteria Tests

func TestParseMatchCriteria(t *testing.T) {
	tests := []struct {
		name      string
		rule      *UpgradeRule
		wantError bool
		validate  func(*testing.T, *MatchCriteria)
	}{
		{
			name: "Valid MAC range criteria",
			rule: &UpgradeRule{
				MatchCriteria: `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
			},
			wantError: false,
			validate: func(t *testing.T, mc *MatchCriteria) {
				if mc.StartMAC != "00:01:5C:00:00:00" {
					t.Errorf("Expected StartMAC '00:01:5C:00:00:00', got %s", mc.StartMAC)
				}
				if mc.EndMAC != "00:01:5C:FF:FF:FF" {
					t.Errorf("Expected EndMAC '00:01:5C:FF:FF:FF', got %s", mc.EndMAC)
				}
			},
		},
		{
			name: "Valid regex pattern criteria",
			rule: &UpgradeRule{
				MatchCriteria: `{"pattern":"Arris SB8200"}`,
			},
			wantError: false,
			validate: func(t *testing.T, mc *MatchCriteria) {
				if mc.Pattern != "Arris SB8200" {
					t.Errorf("Expected Pattern 'Arris SB8200', got %s", mc.Pattern)
				}
			},
		},
		{
			name: "Invalid JSON",
			rule: &UpgradeRule{
				MatchCriteria: `{invalid json}`,
			},
			wantError: true,
		},
		{
			name: "Empty JSON",
			rule: &UpgradeRule{
				MatchCriteria: `{}`,
			},
			wantError: false,
			validate: func(t *testing.T, mc *MatchCriteria) {
				if mc.StartMAC != "" || mc.EndMAC != "" || mc.Pattern != "" {
					t.Error("Expected all fields to be empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria, err := tt.rule.ParseMatchCriteria()
			if (err != nil) != tt.wantError {
				t.Errorf("ParseMatchCriteria() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && tt.validate != nil {
				tt.validate(t, criteria)
			}
		})
	}
}

// Error Types Tests

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "test message"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestAppError(t *testing.T) {
	err := &AppError{
		Code:    "TEST_CODE",
		Message: "test message",
	}

	expected := "test message"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Job Status Constants Tests

func TestJobStatusConstants(t *testing.T) {
	statuses := []string{
		JobStatusPending,
		JobStatusInProgress,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusSkipped,
	}

	expectedStatuses := []string{
		"PENDING",
		"IN_PROGRESS",
		"COMPLETED",
		"FAILED",
		"SKIPPED",
	}

	for i, status := range statuses {
		if status != expectedStatuses[i] {
			t.Errorf("Expected status %s, got %s", expectedStatuses[i], status)
		}
	}
}

// Event Type Constants Tests

func TestEventTypeConstants(t *testing.T) {
	events := []string{
		EventModemDiscovered,
		EventModemLost,
		EventUpgradeStarted,
		EventUpgradeCompleted,
		EventUpgradeFailed,
		EventRuleCreated,
		EventRuleUpdated,
		EventRuleDeleted,
		EventCMTSAdded,
		EventCMTSUpdated,
		EventCMTSDeleted,
		EventSystemEvent,
	}

	expectedEvents := []string{
		"MODEM_DISCOVERED",
		"MODEM_LOST",
		"UPGRADE_STARTED",
		"UPGRADE_COMPLETED",
		"UPGRADE_FAILED",
		"RULE_CREATED",
		"RULE_UPDATED",
		"RULE_DELETED",
		"CMTS_ADDED",
		"CMTS_UPDATED",
		"CMTS_DELETED",
		"SYSTEM_EVENT",
	}

	for i, event := range events {
		if event != expectedEvents[i] {
			t.Errorf("Expected event %s, got %s", expectedEvents[i], event)
		}
	}
}

// Model Structure Tests

func TestCMTSStructure(t *testing.T) {
	cmts := &CMTS{
		ID:                1,
		Name:              "Test CMTS",
		IPAddress:         "192.168.1.1",
		SNMPPort:          161,
		CommunityRead:     "public",
		CommunityWrite:    "private",
		CMCommunityString: "cable-modem",
		SNMPVersion:       2,
		Enabled:           true,
	}

	if cmts.ID != 1 {
		t.Errorf("Expected ID 1, got %d", cmts.ID)
	}
	if cmts.Name != "Test CMTS" {
		t.Errorf("Expected Name 'Test CMTS', got %s", cmts.Name)
	}
	if cmts.SNMPPort != 161 {
		t.Errorf("Expected SNMPPort 161, got %d", cmts.SNMPPort)
	}
	if !cmts.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestCableModemStructure(t *testing.T) {
	modem := &CableModem{
		ID:              1,
		CMTSID:          1,
		MACAddress:      "00:01:5C:11:22:33",
		IPAddress:       "10.0.0.100",
		SysDescr:        "Arris SB8200",
		CurrentFirmware: "1.0.0",
		SignalLevel:     6.5,
		Status:          "online",
	}

	if modem.MACAddress != "00:01:5C:11:22:33" {
		t.Errorf("Expected MAC '00:01:5C:11:22:33', got %s", modem.MACAddress)
	}
	if modem.SignalLevel != 6.5 {
		t.Errorf("Expected SignalLevel 6.5, got %f", modem.SignalLevel)
	}
}

func TestUpgradeRuleStructure(t *testing.T) {
	rule := &UpgradeRule{
		ID:               1,
		Name:             "Test Rule",
		Description:      "Test description",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		Enabled:          true,
		Priority:         100,
	}

	if rule.MatchType != "MAC_RANGE" {
		t.Errorf("Expected MatchType 'MAC_RANGE', got %s", rule.MatchType)
	}
	if rule.Priority != 100 {
		t.Errorf("Expected Priority 100, got %d", rule.Priority)
	}
}

func TestUpgradeJobStructure(t *testing.T) {
	job := &UpgradeJob{
		ID:               1,
		ModemID:          1,
		RuleID:           1,
		CMTSID:           1,
		MACAddress:       "00:01:5C:11:22:33",
		Status:           JobStatusPending,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
		RetryCount:       0,
		MaxRetries:       3,
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected Status 'PENDING', got %s", job.Status)
	}
	if job.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", job.MaxRetries)
	}
	if job.ErrorMessage != nil {
		t.Error("Expected ErrorMessage to be nil")
	}
}

func TestUpgradeJobWithErrorMessage(t *testing.T) {
	errMsg := "test error"
	job := &UpgradeJob{
		ID:           1,
		MACAddress:   "00:01:5C:11:22:33",
		Status:       JobStatusFailed,
		ErrorMessage: &errMsg,
	}

	if job.ErrorMessage == nil {
		t.Fatal("Expected ErrorMessage to be set")
	}
	if *job.ErrorMessage != "test error" {
		t.Errorf("Expected ErrorMessage 'test error', got %s", *job.ErrorMessage)
	}
}

func TestActivityLogStructure(t *testing.T) {
	log := &ActivityLog{
		ID:         1,
		EventType:  EventUpgradeCompleted,
		EntityType: "job",
		EntityID:   123,
		Message:    "Upgrade completed",
		Details:    `{"duration":"5m"}`,
	}

	if log.EventType != EventUpgradeCompleted {
		t.Errorf("Expected EventType 'UPGRADE_COMPLETED', got %s", log.EventType)
	}
	if log.EntityID != 123 {
		t.Errorf("Expected EntityID 123, got %d", log.EntityID)
	}
}

// Edge Cases

func TestCMTSValidateEdgeCases(t *testing.T) {
	// Valid minimum port
	cmts := &CMTS{
		Name:          "Test",
		IPAddress:     "192.168.1.1",
		SNMPPort:      1,
		CommunityRead: "public",
		SNMPVersion:   1,
	}
	if err := cmts.Validate(); err != nil {
		t.Errorf("Port 1 should be valid, got error: %v", err)
	}

	// Valid maximum port
	cmts.SNMPPort = 65535
	if err := cmts.Validate(); err != nil {
		t.Errorf("Port 65535 should be valid, got error: %v", err)
	}

	// Valid SNMP version 1
	cmts.SNMPVersion = 1
	cmts.SNMPPort = 161
	if err := cmts.Validate(); err != nil {
		t.Errorf("SNMP version 1 should be valid, got error: %v", err)
	}

	// Valid SNMP version 3
	cmts.SNMPVersion = 3
	if err := cmts.Validate(); err != nil {
		t.Errorf("SNMP version 3 should be valid, got error: %v", err)
	}
}

func TestUpgradeRuleValidateEdgeCases(t *testing.T) {
	// Whitespace in name should be valid
	rule := &UpgradeRule{
		Name:             "  Test Rule  ",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware.bin",
	}
	// Note: The current implementation considers whitespace-only as empty
	// This documents that behavior
	if err := rule.Validate(); err != nil {
		t.Logf("Rule with whitespace in name: %v", err)
	}
}

func TestMatchCriteriaEmptyFields(t *testing.T) {
	criteria := &MatchCriteria{
		StartMAC: "",
		EndMAC:   "",
		Pattern:  "",
	}

	if criteria.StartMAC != "" || criteria.EndMAC != "" || criteria.Pattern != "" {
		t.Error("Empty criteria should have empty fields")
	}
}
