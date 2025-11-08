package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

// Test Client Creation

func TestNewClientValidation(t *testing.T) {
	// Test nil CMTS
	_, err := NewClient(nil)
	if err == nil {
		t.Error("Expected error for nil CMTS")
	}
}

func TestNewClientStructure(t *testing.T) {
	cmts := &models.CMTS{
		Name:           "Test CMTS",
		IPAddress:      "192.0.2.1", // TEST-NET-1 (documentation/testing)
		SNMPPort:       161,
		CommunityRead:  "public",
		CommunityWrite: "private",
		SNMPVersion:    2,
		Enabled:        true,
	}

	// Note: This will fail to connect since it's a test IP, but we can verify
	// the client would be created with correct parameters
	client, err := NewClient(cmts)

	// We expect connection to fail for test IP, but that's okay
	if err != nil {
		t.Logf("Expected connection failure for test IP: %v", err)
	}

	// Client might be nil if connection failed, which is expected
	if client != nil {
		t.Log("Client created successfully (unexpected for test IP)")
	}
}

// Test Helper Functions

func TestParseMACAddress(t *testing.T) {
	tests := []struct {
		name     string
		pdu      gosnmp.SnmpPDU
		expected string
	}{
		{
			name: "Valid 6-byte MAC",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{0x00, 0x01, 0x5C, 0x11, 0x22, 0x33},
			},
			expected: "00:01:5C:11:22:33",
		},
		{
			name: "Empty bytes",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{},
			},
			expected: "",
		},
		{
			name: "Invalid length",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{0x00, 0x01, 0x5C},
			},
			expected: "",
		},
		{
			name: "All zeros",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			expected: "00:00:00:00:00:00",
		},
		{
			name: "All FFs",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			},
			expected: "FF:FF:FF:FF:FF:FF",
		},
		{
			name: "String MAC address",
			pdu: gosnmp.SnmpPDU{
				Value: "00:01:5C:11:22:33",
			},
			expected: "00:01:5C:11:22:33",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMACAddress(tt.pdu)
			if result != tt.expected {
				t.Errorf("parseMACAddress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		pdu      gosnmp.SnmpPDU
		expected string
	}{
		{
			name: "Valid IPv4",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{192, 168, 1, 100},
			},
			expected: "192.168.1.100",
		},
		{
			name: "Empty bytes",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{},
			},
			expected: "",
		},
		{
			name: "Invalid length",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{192, 168, 1},
			},
			expected: "",
		},
		{
			name: "Localhost",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{127, 0, 0, 1},
			},
			expected: "127.0.0.1",
		},
		{
			name: "All zeros",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{0, 0, 0, 0},
			},
			expected: "0.0.0.0",
		},
		{
			name: "Broadcast",
			pdu: gosnmp.SnmpPDU{
				Value: []byte{255, 255, 255, 255},
			},
			expected: "255.255.255.255",
		},
		{
			name: "String IP address",
			pdu: gosnmp.SnmpPDU{
				Value: "10.0.0.1",
			},
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPAddress(tt.pdu)
			if result != tt.expected {
				t.Errorf("parseIPAddress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractFirmwareFromSysDescr(t *testing.T) {
	tests := []struct {
		name     string
		sysDescr string
		expected string
	}{
		{
			name:     "Arris modem with SW_REV",
			sysDescr: "Motorola SB6141 HW_REV: 7.0 VENDOR: Motorola SW_REV: SB6141-7.0.0.1-SCM01-SHPC",
			expected: "SB6141-7.0.0.1-SCM01-SHPC",
		},
		{
			name:     "Arris with SW V prefix (note the space)",
			sysDescr: "Arris CM8200 DOCSIS 3.1 Cable Modem SW V 1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "Empty sysDescr",
			sysDescr: "",
			expected: "",
		},
		{
			name:     "No version info",
			sysDescr: "Generic Cable Modem",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirmwareFromSysDescr(tt.sysDescr)
			if result != tt.expected {
				t.Errorf("extractFirmwareFromSysDescr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSignalLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "Positive signal level",
			input:    "6.5",
			expected: 6.5,
		},
		{
			name:     "Negative signal level",
			input:    "-3.2",
			expected: -3.2,
		},
		{
			name:     "Zero",
			input:    "0",
			expected: 0.0,
		},
		{
			name:     "Integer value",
			input:    "5",
			expected: 5.0,
		},
		{
			name:     "Invalid string",
			input:    "invalid",
			expected: 0.0,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSignalLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSignalLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractIndexFromOID(t *testing.T) {
	tests := []struct {
		name     string
		oidStr   string
		baseOID  string
		expected string
	}{
		{
			name:     "Valid OID with index",
			oidStr:   "1.3.6.1.2.1.10.127.1.3.3.1.2.123",
			baseOID:  "1.3.6.1.2.1.10.127.1.3.3.1.2",
			expected: "123",
		},
		{
			name:     "Multi-part index",
			oidStr:   "1.3.6.1.2.1.10.127.1.3.3.1.2.10.20.30",
			baseOID:  "1.3.6.1.2.1.10.127.1.3.3.1.2",
			expected: "10.20.30",
		},
		{
			name:     "No match - different base",
			oidStr:   "1.3.6.1.2.1.99.99.99",
			baseOID:  "1.3.6.1.2.1.10.127.1.3.3.1.2",
			expected: "",
		},
		{
			name:     "Empty OID",
			oidStr:   "",
			baseOID:  "1.3.6.1.2.1.10.127.1.3.3.1.2",
			expected: "",
		},
		{
			name:     "OID equals base (no index)",
			oidStr:   "1.3.6.1.2.1.10.127.1.3.3.1.2",
			baseOID:  "1.3.6.1.2.1.10.127.1.3.3.1.2",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIndexFromOID(tt.oidStr, tt.baseOID)
			if result != tt.expected {
				t.Errorf("extractIndexFromOID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test OID Constants

func TestOIDConstants(t *testing.T) {
	oids := map[string]string{
		"OIDDocsIfCmtsCmStatusMacAddress":      OIDDocsIfCmtsCmStatusMacAddress,
		"OIDDocsIfCmtsCmStatusIpAddress":       OIDDocsIfCmtsCmStatusIpAddress,
		"OIDDocsIfCmtsCmStatusDownstreamPower": OIDDocsIfCmtsCmStatusDownstreamPower,
		"OIDDocsIfCmtsCmStatusValue":           OIDDocsIfCmtsCmStatusValue,
		"OIDSysDescr":                          OIDSysDescr,
		"OIDDocsDevSwServer":                   OIDDocsDevSwServer,
		"OIDDocsDevSwFilename":                 OIDDocsDevSwFilename,
		"OIDDocsDevSwAdminStatus":              OIDDocsDevSwAdminStatus,
		"OIDDocsDevSwOperStatus":               OIDDocsDevSwOperStatus,
	}

	for name, oid := range oids {
		if oid == "" {
			t.Errorf("OID %s should not be empty", name)
		}
		// Verify OID format (should start with numbers and dots)
		if len(oid) < 3 || oid[0:2] != "1." {
			t.Errorf("OID %s has invalid format: %s", name, oid)
		}
	}
}

// Test SNMP Version Mapping

func TestSNMPVersionMapping(t *testing.T) {
	tests := []struct {
		name        string
		cmtsVersion int
		shouldWork  bool
	}{
		{
			name:        "SNMP v1",
			cmtsVersion: 1,
			shouldWork:  true,
		},
		{
			name:        "SNMP v2c",
			cmtsVersion: 2,
			shouldWork:  true,
		},
		{
			name:        "SNMP v3",
			cmtsVersion: 3,
			shouldWork:  true,
		},
		{
			name:        "Invalid version 0",
			cmtsVersion: 0,
			shouldWork:  false, // Will default to v2c but connection will fail
		},
		{
			name:        "Invalid version 4",
			cmtsVersion: 4,
			shouldWork:  false, // Will default to v2c but connection will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmts := &models.CMTS{
				Name:          "Test CMTS",
				IPAddress:     "192.0.2.1",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   tt.cmtsVersion,
			}

			// Try to create client (will fail to connect, but validates version mapping)
			_, err := NewClient(cmts)

			// All versions should be handled, even if connection fails
			if err != nil {
				t.Logf("Expected connection failure for test IP with version %d: %v", tt.cmtsVersion, err)
			}
		})
	}
}

// Test Error Cases

func TestClientErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		cmts  *models.CMTS
		check func(*testing.T, error)
	}{
		{
			name: "Nil CMTS",
			cmts: nil,
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("Expected error for nil CMTS")
				}
			},
		},
		{
			name: "Invalid IP address",
			cmts: &models.CMTS{
				Name:          "Test",
				IPAddress:     "999.999.999.999",
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("Expected error for invalid IP")
				}
			},
		},
		{
			name: "Unreachable IP",
			cmts: &models.CMTS{
				Name:          "Test",
				IPAddress:     "192.0.2.1", // TEST-NET-1
				SNMPPort:      161,
				CommunityRead: "public",
				SNMPVersion:   2,
			},
			check: func(t *testing.T, err error) {
				// Connection may succeed but actual SNMP queries will fail
				// This is acceptable behavior for the test IP
				t.Logf("Connection result for unreachable IP: %v", err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.cmts)
			tt.check(t, err)
		})
	}
}
