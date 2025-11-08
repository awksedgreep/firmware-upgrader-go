package engine

import (
	"testing"

	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

func TestMatchMACRange(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name      string
		mac       string
		startMAC  string
		endMAC    string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "MAC in range - middle",
			mac:       "00:01:5C:12:34:56",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "MAC at start of range",
			mac:       "00:01:5C:00:00:00",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "MAC at end of range",
			mac:       "00:01:5C:FF:FF:FF",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "MAC below range",
			mac:       "00:01:5B:FF:FF:FF",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "MAC above range",
			mac:       "00:01:5D:00:00:00",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "MAC with different format - hyphens",
			mac:       "00-01-5C-12-34-56",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "MAC with different format - dots",
			mac:       "0001.5C12.3456",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "MAC with no separators",
			mac:       "00015C123456",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "Invalid MAC format",
			mac:       "invalid-mac",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Empty start MAC",
			mac:       "00:01:5C:12:34:56",
			startMAC:  "",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Empty end MAC",
			mac:       "00:01:5C:12:34:56",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Lowercase MAC",
			mac:       "00:01:5c:12:34:56",
			startMAC:  "00:01:5C:00:00:00",
			endMAC:    "00:01:5C:FF:FF:FF",
			wantMatch: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria := &models.MatchCriteria{
				StartMAC: tt.startMAC,
				EndMAC:   tt.endMAC,
			}

			match, err := matcher.matchMACRange(tt.mac, criteria)

			if (err != nil) != tt.wantErr {
				t.Errorf("matchMACRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if match != tt.wantMatch {
				t.Errorf("matchMACRange() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

func TestMatchSysDescrRegex(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name      string
		sysDescr  string
		pattern   string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "Arris modem - exact match",
			sysDescr:  "Arris SB8200 DOCSIS 3.1 Cable Modem",
			pattern:   "Arris SB8200",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "Arris modem - case insensitive",
			sysDescr:  "Arris SB8200 DOCSIS 3.1 Cable Modem",
			pattern:   "(?i)arris sb8200",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "Motorola modem - regex pattern",
			sysDescr:  "Motorola SB6141 HW_REV: 7.0 VENDOR: Motorola",
			pattern:   "Motorola.*SB6141",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "Version number pattern",
			sysDescr:  "Arris CM8200 SW_REV: 1.2.3.4",
			pattern:   "SW_REV: \\d+\\.\\d+\\.\\d+",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "No match - different vendor",
			sysDescr:  "Arris SB8200 DOCSIS 3.1 Cable Modem",
			pattern:   "Motorola",
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "Empty sysDescr",
			sysDescr:  "",
			pattern:   "Arris",
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "Empty pattern",
			sysDescr:  "Arris SB8200",
			pattern:   "",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Invalid regex pattern",
			sysDescr:  "Arris SB8200",
			pattern:   "[invalid(regex",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Complex regex - OR conditions",
			sysDescr:  "Arris SB8200 DOCSIS 3.1",
			pattern:   "(Arris|Motorola).*(SB8200|SB6141)",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "DOCSIS version matching",
			sysDescr:  "Arris CM8200 DOCSIS 3.1 Cable Modem",
			pattern:   "DOCSIS 3\\.(0|1)",
			wantMatch: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria := &models.MatchCriteria{
				Pattern: tt.pattern,
			}

			match, err := matcher.matchSysDescrRegex(tt.sysDescr, criteria)

			if (err != nil) != tt.wantErr {
				t.Errorf("matchSysDescrRegex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if match != tt.wantMatch {
				t.Errorf("matchSysDescrRegex() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

func TestFilterEligibleModems(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name           string
		modems         []*models.CableModem
		expectedCount  int
		expectedMACs   []string
		unexpectedMACs []string
	}{
		{
			name: "All modems eligible",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 5.0},
				{ID: 2, MACAddress: "00:01:5C:22:22:22", Status: "online", SignalLevel: 0.0},
				{ID: 3, MACAddress: "00:01:5C:33:33:33", Status: "online", SignalLevel: -5.0},
			},
			expectedCount: 3,
			expectedMACs:  []string{"00:01:5C:11:11:11", "00:01:5C:22:22:22", "00:01:5C:33:33:33"},
		},
		{
			name: "Filter offline modems",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 5.0},
				{ID: 2, MACAddress: "00:01:5C:22:22:22", Status: "offline", SignalLevel: 0.0},
				{ID: 3, MACAddress: "00:01:5C:33:33:33", Status: "partial", SignalLevel: 3.0},
			},
			expectedCount:  1,
			expectedMACs:   []string{"00:01:5C:11:11:11"},
			unexpectedMACs: []string{"00:01:5C:22:22:22", "00:01:5C:33:33:33"},
		},
		{
			name: "Filter poor signal - too low",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 5.0},
				{ID: 2, MACAddress: "00:01:5C:22:22:22", Status: "online", SignalLevel: -20.0},
			},
			expectedCount:  1,
			expectedMACs:   []string{"00:01:5C:11:11:11"},
			unexpectedMACs: []string{"00:01:5C:22:22:22"},
		},
		{
			name: "Filter poor signal - too high",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 5.0},
				{ID: 2, MACAddress: "00:01:5C:22:22:22", Status: "online", SignalLevel: 20.0},
			},
			expectedCount:  1,
			expectedMACs:   []string{"00:01:5C:11:11:11"},
			unexpectedMACs: []string{"00:01:5C:22:22:22"},
		},
		{
			name: "Signal at boundaries - minimum",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: -15.0},
			},
			expectedCount: 1,
			expectedMACs:  []string{"00:01:5C:11:11:11"},
		},
		{
			name: "Signal at boundaries - maximum",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 15.0},
			},
			expectedCount: 1,
			expectedMACs:  []string{"00:01:5C:11:11:11"},
		},
		{
			name:          "Empty modem list",
			modems:        []*models.CableModem{},
			expectedCount: 0,
		},
		{
			name: "Multiple filters apply",
			modems: []*models.CableModem{
				{ID: 1, MACAddress: "00:01:5C:11:11:11", Status: "online", SignalLevel: 5.0},   // OK
				{ID: 2, MACAddress: "00:01:5C:22:22:22", Status: "offline", SignalLevel: 0.0},  // Offline
				{ID: 3, MACAddress: "00:01:5C:33:33:33", Status: "online", SignalLevel: -20.0}, // Poor signal
				{ID: 4, MACAddress: "00:01:5C:44:44:44", Status: "denied", SignalLevel: 5.0},   // Denied
				{ID: 5, MACAddress: "00:01:5C:55:55:55", Status: "online", SignalLevel: 0.0},   // OK
			},
			expectedCount:  2,
			expectedMACs:   []string{"00:01:5C:11:11:11", "00:01:5C:55:55:55"},
			unexpectedMACs: []string{"00:01:5C:22:22:22", "00:01:5C:33:33:33", "00:01:5C:44:44:44"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible := matcher.FilterEligibleModems(tt.modems)

			if len(eligible) != tt.expectedCount {
				t.Errorf("FilterEligibleModems() returned %d modems, want %d", len(eligible), tt.expectedCount)
			}

			// Check expected MACs are present
			for _, expectedMAC := range tt.expectedMACs {
				found := false
				for _, modem := range eligible {
					if modem.MACAddress == expectedMAC {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected MAC %s not found in eligible modems", expectedMAC)
				}
			}

			// Check unexpected MACs are not present
			for _, unexpectedMAC := range tt.unexpectedMACs {
				for _, modem := range eligible {
					if modem.MACAddress == unexpectedMAC {
						t.Errorf("Unexpected MAC %s found in eligible modems", unexpectedMAC)
					}
				}
			}
		})
	}
}

func TestShouldUpgrade(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name            string
		currentFirmware string
		targetFilename  string
		wantUpgrade     bool
	}{
		{
			name:            "Needs upgrade - different versions",
			currentFirmware: "1.0.0",
			targetFilename:  "firmware-v1.2.3.bin",
			wantUpgrade:     true,
		},
		{
			name:            "No upgrade needed - same version",
			currentFirmware: "1.2.3",
			targetFilename:  "firmware-v1.2.3.bin",
			wantUpgrade:     false,
		},
		{
			name:            "Unknown current firmware",
			currentFirmware: "",
			targetFilename:  "firmware-v1.2.3.bin",
			wantUpgrade:     true,
		},
		{
			name:            "Cannot extract target version",
			currentFirmware: "1.2.3",
			targetFilename:  "firmware.bin",
			wantUpgrade:     true,
		},
		{
			name:            "Different filename format",
			currentFirmware: "2.0.1",
			targetFilename:  "arris-sb8200-v2.0.1.bin",
			wantUpgrade:     false,
		},
		{
			name:            "Version in filename without v prefix",
			currentFirmware: "3.5.2",
			targetFilename:  "modem-3.5.2-release.bin",
			wantUpgrade:     false,
		},
		{
			name:            "Needs upgrade - older version",
			currentFirmware: "1.0.0",
			targetFilename:  "firmware-v2.0.0.bin",
			wantUpgrade:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modem := &models.CableModem{
				MACAddress:      "00:01:5C:11:11:11",
				CurrentFirmware: tt.currentFirmware,
			}

			rule := &models.UpgradeRule{
				FirmwareFilename: tt.targetFilename,
			}

			result := matcher.ShouldUpgrade(modem, rule)

			if result != tt.wantUpgrade {
				t.Errorf("ShouldUpgrade() = %v, want %v (current: %s, target: %s)",
					result, tt.wantUpgrade, tt.currentFirmware, tt.targetFilename)
			}
		})
	}
}

func TestMatchModemToRules(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name        string
		modem       *models.CableModem
		rules       []*models.UpgradeRule
		wantRuleID  int
		wantNoMatch bool
		wantErr     bool
	}{
		{
			name: "Match by MAC range",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "00:01:5C:12:34:56",
				SysDescr:   "Arris SB8200",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            1,
					Name:          "Arris Range",
					MatchType:     "MAC_RANGE",
					MatchCriteria: `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
					Enabled:       true,
					Priority:      100,
				},
			},
			wantRuleID: 1,
		},
		{
			name: "Match by regex",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "AA:BB:CC:11:22:33",
				SysDescr:   "Arris SB8200 DOCSIS 3.1",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            2,
					Name:          "Arris SysDescr",
					MatchType:     "SYSDESCR_REGEX",
					MatchCriteria: `{"pattern":"Arris SB8200"}`,
					Enabled:       true,
					Priority:      90,
				},
			},
			wantRuleID: 2,
		},
		{
			name: "Multiple rules - highest priority wins",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "00:01:5C:12:34:56",
				SysDescr:   "Arris SB8200",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            2,
					Name:          "High Priority",
					MatchType:     "MAC_RANGE",
					MatchCriteria: `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
					Enabled:       true,
					Priority:      100,
				},
				{
					ID:            1,
					Name:          "Low Priority",
					MatchType:     "SYSDESCR_REGEX",
					MatchCriteria: `{"pattern":"Arris"}`,
					Enabled:       true,
					Priority:      50,
				},
			},
			wantRuleID: 2, // Higher priority evaluated first
		},
		{
			name: "Disabled rule not matched",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "00:01:5C:12:34:56",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            1,
					Name:          "Disabled Rule",
					MatchType:     "MAC_RANGE",
					MatchCriteria: `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
					Enabled:       false,
					Priority:      100,
				},
			},
			wantNoMatch: true,
		},
		{
			name: "No matching rules",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "AA:BB:CC:11:22:33",
				SysDescr:   "Unknown Modem",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            1,
					Name:          "Arris Only",
					MatchType:     "SYSDESCR_REGEX",
					MatchCriteria: `{"pattern":"Arris"}`,
					Enabled:       true,
					Priority:      100,
				},
			},
			wantNoMatch: true,
		},
		{
			name: "Invalid match criteria JSON",
			modem: &models.CableModem{
				ID:         1,
				MACAddress: "00:01:5C:12:34:56",
			},
			rules: []*models.UpgradeRule{
				{
					ID:            1,
					Name:          "Invalid JSON",
					MatchType:     "MAC_RANGE",
					MatchCriteria: `{invalid json}`,
					Enabled:       true,
					Priority:      100,
				},
			},
			wantNoMatch: true, // Rule skipped due to error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := matcher.MatchModemToRules(tt.modem, tt.rules)

			if (err != nil) != tt.wantErr {
				t.Errorf("MatchModemToRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantNoMatch {
				if rule != nil {
					t.Errorf("MatchModemToRules() returned rule %d, want no match", rule.ID)
				}
			} else {
				if rule == nil {
					t.Errorf("MatchModemToRules() returned nil, want rule %d", tt.wantRuleID)
				} else if rule.ID != tt.wantRuleID {
					t.Errorf("MatchModemToRules() returned rule %d, want rule %d", rule.ID, tt.wantRuleID)
				}
			}
		})
	}
}

func TestValidateMatchCriteria(t *testing.T) {
	matcher := NewMatcher()

	tests := []struct {
		name          string
		matchType     string
		criteriaJSON  string
		wantErr       bool
		expectedError string
	}{
		{
			name:         "Valid MAC range",
			matchType:    "MAC_RANGE",
			criteriaJSON: `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
			wantErr:      false,
		},
		{
			name:         "Valid regex pattern",
			matchType:    "SYSDESCR_REGEX",
			criteriaJSON: `{"pattern":"Arris.*SB8200"}`,
			wantErr:      false,
		},
		{
			name:          "MAC range - missing start_mac",
			matchType:     "MAC_RANGE",
			criteriaJSON:  `{"end_mac":"00:01:5C:FF:FF:FF"}`,
			wantErr:       true,
			expectedError: "start_mac is required",
		},
		{
			name:          "MAC range - missing end_mac",
			matchType:     "MAC_RANGE",
			criteriaJSON:  `{"start_mac":"00:01:5C:00:00:00"}`,
			wantErr:       true,
			expectedError: "end_mac is required",
		},
		{
			name:          "MAC range - invalid start_mac",
			matchType:     "MAC_RANGE",
			criteriaJSON:  `{"start_mac":"invalid","end_mac":"00:01:5C:FF:FF:FF"}`,
			wantErr:       true,
			expectedError: "invalid start_mac",
		},
		{
			name:          "MAC range - start > end",
			matchType:     "MAC_RANGE",
			criteriaJSON:  `{"start_mac":"00:01:5C:FF:FF:FF","end_mac":"00:01:5C:00:00:00"}`,
			wantErr:       true,
			expectedError: "start_mac must be less than or equal to end_mac",
		},
		{
			name:          "Regex - missing pattern",
			matchType:     "SYSDESCR_REGEX",
			criteriaJSON:  `{}`,
			wantErr:       true,
			expectedError: "pattern is required",
		},
		{
			name:          "Regex - invalid pattern",
			matchType:     "SYSDESCR_REGEX",
			criteriaJSON:  `{"pattern":"[invalid(regex"}`,
			wantErr:       true,
			expectedError: "invalid regex pattern",
		},
		{
			name:          "Unknown match type",
			matchType:     "UNKNOWN_TYPE",
			criteriaJSON:  `{"pattern":"test"}`,
			wantErr:       true,
			expectedError: "unknown match type",
		},
		{
			name:          "Invalid JSON",
			matchType:     "MAC_RANGE",
			criteriaJSON:  `{invalid json}`,
			wantErr:       true,
			expectedError: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matcher.ValidateMatchCriteria(tt.matchType, tt.criteriaJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMatchCriteria() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedError != "" {
				if err == nil {
					t.Errorf("ValidateMatchCriteria() expected error containing '%s', got nil", tt.expectedError)
				}
			}
		})
	}
}
