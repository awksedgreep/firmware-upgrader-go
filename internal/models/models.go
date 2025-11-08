package models

import (
	"encoding/json"
	"time"
)

// CMTS represents a Cable Modem Termination System
type CMTS struct {
	ID                int       `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	IPAddress         string    `json:"ip_address" db:"ip_address"`
	SNMPPort          int       `json:"snmp_port" db:"snmp_port"`
	CommunityRead     string    `json:"community_read" db:"community_read"`
	CommunityWrite    string    `json:"community_write" db:"community_write"`
	CMCommunityString string    `json:"cm_community_string" db:"cm_community_string"`
	SNMPVersion       int       `json:"snmp_version" db:"snmp_version"`
	Enabled           bool      `json:"enabled" db:"enabled"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// CableModem represents a discovered cable modem
type CableModem struct {
	ID              int       `json:"id" db:"id"`
	CMTSID          int       `json:"cmts_id" db:"cmts_id"`
	MACAddress      string    `json:"mac_address" db:"mac_address"`
	IPAddress       string    `json:"ip_address" db:"ip_address"`
	SysDescr        string    `json:"sysdescr" db:"sysdescr"`
	CurrentFirmware string    `json:"current_firmware" db:"current_firmware"`
	SignalLevel     float64   `json:"signal_level" db:"signal_level"`
	Status          string    `json:"status" db:"status"`
	LastSeen        time.Time `json:"last_seen" db:"last_seen"`
}

// UpgradeRule represents a firmware upgrade rule
type UpgradeRule struct {
	ID               int       `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	MatchType        string    `json:"match_type" db:"match_type"`         // "MAC_RANGE" or "SYSDESCR_REGEX"
	MatchCriteria    string    `json:"match_criteria" db:"match_criteria"` // JSON string
	TFTPServerIP     string    `json:"tftp_server_ip" db:"tftp_server_ip"`
	FirmwareFilename string    `json:"firmware_filename" db:"firmware_filename"`
	Enabled          bool      `json:"enabled" db:"enabled"`
	Priority         int       `json:"priority" db:"priority"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// MatchCriteria represents the criteria for matching modems
type MatchCriteria struct {
	StartMAC string `json:"start_mac,omitempty"`
	EndMAC   string `json:"end_mac,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
}

// ParseMatchCriteria parses the JSON match criteria
func (r *UpgradeRule) ParseMatchCriteria() (*MatchCriteria, error) {
	var criteria MatchCriteria
	if err := json.Unmarshal([]byte(r.MatchCriteria), &criteria); err != nil {
		return nil, err
	}
	return &criteria, nil
}

// UpgradeJob represents a firmware upgrade job
type UpgradeJob struct {
	ID               int        `json:"id" db:"id"`
	ModemID          int        `json:"modem_id" db:"modem_id"`
	RuleID           int        `json:"rule_id" db:"rule_id"`
	CMTSID           int        `json:"cmts_id" db:"cmts_id"`
	MACAddress       string     `json:"mac_address" db:"mac_address"`
	Status           string     `json:"status" db:"status"` // PENDING, IN_PROGRESS, COMPLETED, FAILED, SKIPPED
	TFTPServerIP     string     `json:"tftp_server_ip" db:"tftp_server_ip"`
	FirmwareFilename string     `json:"firmware_filename" db:"firmware_filename"`
	RetryCount       int        `json:"retry_count" db:"retry_count"`
	MaxRetries       int        `json:"max_retries" db:"max_retries"`
	ErrorMessage     *string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	StartedAt        *time.Time `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at" db:"completed_at"`
}

// Job status constants
const (
	JobStatusPending    = "PENDING"
	JobStatusInProgress = "IN_PROGRESS"
	JobStatusCompleted  = "COMPLETED"
	JobStatusFailed     = "FAILED"
	JobStatusSkipped    = "SKIPPED"
)

// ActivityLog represents a system activity log entry
type ActivityLog struct {
	ID         int       `json:"id" db:"id"`
	EventType  string    `json:"event_type" db:"event_type"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   int       `json:"entity_id" db:"entity_id"`
	Message    string    `json:"message" db:"message"`
	Details    string    `json:"details" db:"details"` // JSON string
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Event type constants
const (
	EventModemDiscovered  = "MODEM_DISCOVERED"
	EventModemLost        = "MODEM_LOST"
	EventUpgradeStarted   = "UPGRADE_STARTED"
	EventUpgradeCompleted = "UPGRADE_COMPLETED"
	EventUpgradeFailed    = "UPGRADE_FAILED"
	EventRuleCreated      = "RULE_CREATED"
	EventRuleUpdated      = "RULE_UPDATED"
	EventRuleDeleted      = "RULE_DELETED"
	EventCMTSAdded        = "CMTS_ADDED"
	EventCMTSUpdated      = "CMTS_UPDATED"
	EventCMTSDeleted      = "CMTS_DELETED"
	EventSystemEvent      = "SYSTEM_EVENT"
)

// Validate validates a CMTS configuration
func (c *CMTS) Validate() error {
	if c.Name == "" {
		return ErrInvalidName
	}
	if c.IPAddress == "" {
		return ErrInvalidIPAddress
	}
	if c.SNMPPort < 1 || c.SNMPPort > 65535 {
		return ErrInvalidPort
	}
	if c.CommunityRead == "" {
		return ErrInvalidCommunity
	}
	if c.SNMPVersion < 1 || c.SNMPVersion > 3 {
		return ErrInvalidSNMPVersion
	}
	return nil
}

// Validate validates an upgrade rule
func (r *UpgradeRule) Validate() error {
	if r.Name == "" {
		return ErrInvalidName
	}
	if r.MatchType != "MAC_RANGE" && r.MatchType != "SYSDESCR_REGEX" {
		return ErrInvalidMatchType
	}
	if r.TFTPServerIP == "" {
		return ErrInvalidTFTPServer
	}
	if r.FirmwareFilename == "" {
		return ErrInvalidFirmware
	}

	// Validate match criteria JSON
	_, err := r.ParseMatchCriteria()
	if err != nil {
		return ErrInvalidMatchCriteria
	}

	return nil
}

// Common errors
var (
	ErrInvalidName          = &ValidationError{Field: "name", Message: "name is required"}
	ErrInvalidIPAddress     = &ValidationError{Field: "ip_address", Message: "valid IP address is required"}
	ErrInvalidPort          = &ValidationError{Field: "port", Message: "port must be between 1 and 65535"}
	ErrInvalidCommunity     = &ValidationError{Field: "community", Message: "SNMP community string is required"}
	ErrInvalidSNMPVersion   = &ValidationError{Field: "snmp_version", Message: "SNMP version must be 1, 2, or 3"}
	ErrInvalidMatchType     = &ValidationError{Field: "match_type", Message: "match_type must be MAC_RANGE or SYSDESCR_REGEX"}
	ErrInvalidTFTPServer    = &ValidationError{Field: "tftp_server_ip", Message: "TFTP server IP is required"}
	ErrInvalidFirmware      = &ValidationError{Field: "firmware_filename", Message: "firmware filename is required"}
	ErrInvalidMatchCriteria = &ValidationError{Field: "match_criteria", Message: "invalid match criteria JSON"}
	ErrNotFound             = &AppError{Code: "NOT_FOUND", Message: "resource not found"}
	ErrDuplicate            = &AppError{Code: "DUPLICATE", Message: "resource already exists"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}
