package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/models"
	_ "modernc.org/sqlite"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes schema
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// NewTestDB creates an in-memory database for testing with fixtures
func NewTestDB() (*DB, error) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open test database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate test database: %w", err)
	}

	return db, nil
}

// LoadTestFixtures loads test data into the database
func (db *DB) LoadTestFixtures() error {
	// Add test CMTS
	_, err := db.CreateCMTS(&models.CMTS{
		Name:              "Test CMTS",
		IPAddress:         "192.168.1.1",
		SNMPPort:          161,
		CommunityRead:     "public",
		CommunityWrite:    "private",
		CMCommunityString: "cable-modem",
		SNMPVersion:       2,
		Enabled:           true,
	})
	if err != nil {
		return fmt.Errorf("failed to create test CMTS: %w", err)
	}

	// Add test modem
	err = db.UpsertModem(&models.CableModem{
		CMTSID:          1,
		MACAddress:      "00:01:5C:11:22:33",
		IPAddress:       "10.0.0.100",
		SysDescr:        "Arris SB8200 DOCSIS 3.1",
		CurrentFirmware: "1.0.0",
		SignalLevel:     5.0,
		Status:          "online",
		LastSeen:        time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to create test modem: %w", err)
	}

	// Add test rule
	_, err = db.CreateRule(&models.UpgradeRule{
		Name:             "Test Rule",
		Description:      "Test upgrade rule",
		MatchType:        "MAC_RANGE",
		MatchCriteria:    `{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}`,
		TFTPServerIP:     "192.168.1.50",
		FirmwareFilename: "firmware-v2.0.0.bin",
		Enabled:          true,
		Priority:         100,
	})
	if err != nil {
		return fmt.Errorf("failed to create test rule: %w", err)
	}

	return nil
}

// migrate creates or updates the database schema
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cmts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		snmp_port INTEGER DEFAULT 161,
		community_read TEXT NOT NULL,
		community_write TEXT,
		cm_community_string TEXT,
		snmp_version INTEGER DEFAULT 2,
		enabled BOOLEAN DEFAULT 1,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cable_modem (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cmts_id INTEGER NOT NULL,
		mac_address TEXT UNIQUE NOT NULL,
		ip_address TEXT,
		sysdescr TEXT,
		current_firmware TEXT,
		signal_level REAL,
		status TEXT,
		last_seen INTEGER,
		FOREIGN KEY (cmts_id) REFERENCES cmts(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_cable_modem_mac ON cable_modem(mac_address);
	CREATE INDEX IF NOT EXISTS idx_cable_modem_cmts ON cable_modem(cmts_id);

	CREATE TABLE IF NOT EXISTS upgrade_rule (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		match_type TEXT NOT NULL,
		match_criteria TEXT NOT NULL,
		tftp_server_ip TEXT NOT NULL,
		firmware_filename TEXT NOT NULL,
		enabled BOOLEAN DEFAULT 1,
		priority INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_upgrade_rule_enabled ON upgrade_rule(enabled);
	CREATE INDEX IF NOT EXISTS idx_upgrade_rule_priority ON upgrade_rule(priority DESC);

	CREATE TABLE IF NOT EXISTS upgrade_job (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		modem_id INTEGER NOT NULL,
		rule_id INTEGER NOT NULL,
		cmts_id INTEGER NOT NULL,
		mac_address TEXT NOT NULL,
		status TEXT NOT NULL,
		tftp_server_ip TEXT,
		firmware_filename TEXT,
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		error_message TEXT,
		created_at INTEGER NOT NULL,
		started_at INTEGER,
		completed_at INTEGER,
		FOREIGN KEY (modem_id) REFERENCES cable_modem(id),
		FOREIGN KEY (rule_id) REFERENCES upgrade_rule(id),
		FOREIGN KEY (cmts_id) REFERENCES cmts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_upgrade_job_status ON upgrade_job(status);
	CREATE INDEX IF NOT EXISTS idx_upgrade_job_mac ON upgrade_job(mac_address);

	CREATE TABLE IF NOT EXISTS activity_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		entity_type TEXT,
		entity_id INTEGER,
		message TEXT NOT NULL,
		details TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_activity_log_created ON activity_log(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_activity_log_type ON activity_log(event_type);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Initialize default settings
	defaults := map[string]string{
		"workers":               "5",
		"discovery_interval":    "60",
		"evaluation_interval":   "120",
		"job_timeout":           "300",
		"retry_attempts":        "3",
		"signal_level_min":      "-15.0",
		"signal_level_max":      "15.0",
		"max_upgrades_per_cmts": "10",
		"log_level":             "info",
	}

	for key, value := range defaults {
		_, err := db.conn.Exec(`
			INSERT INTO settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO NOTHING
		`, key, value, time.Now().Unix())
		if err != nil {
			return fmt.Errorf("failed to initialize setting %s: %w", key, err)
		}
	}

	return nil
}

// CMTS operations

// CreateCMTS creates a new CMTS
func (db *DB) CreateCMTS(cmts *models.CMTS) (int, error) {
	if err := cmts.Validate(); err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	result, err := db.conn.Exec(`
		INSERT INTO cmts (name, ip_address, snmp_port, community_read, community_write,
			cm_community_string, snmp_version, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmts.Name, cmts.IPAddress, cmts.SNMPPort, cmts.CommunityRead, cmts.CommunityWrite,
		cmts.CMCommunityString, cmts.SNMPVersion, cmts.Enabled, now, now)

	if err != nil {
		return 0, fmt.Errorf("failed to create CMTS: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetCMTS retrieves a CMTS by ID
func (db *DB) GetCMTS(id int) (*models.CMTS, error) {
	var cmts models.CMTS
	var createdAt, updatedAt int64

	err := db.conn.QueryRow(`
		SELECT id, name, ip_address, snmp_port, community_read, community_write,
			cm_community_string, snmp_version, enabled, created_at, updated_at
		FROM cmts WHERE id = ?`, id).Scan(
		&cmts.ID, &cmts.Name, &cmts.IPAddress, &cmts.SNMPPort, &cmts.CommunityRead,
		&cmts.CommunityWrite, &cmts.CMCommunityString, &cmts.SNMPVersion,
		&cmts.Enabled, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get CMTS: %w", err)
	}

	cmts.CreatedAt = time.Unix(createdAt, 0)
	cmts.UpdatedAt = time.Unix(updatedAt, 0)

	return &cmts, nil
}

// ListCMTS retrieves all CMTS devices
func (db *DB) ListCMTS() ([]*models.CMTS, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, ip_address, snmp_port, community_read, community_write,
			cm_community_string, snmp_version, enabled, created_at, updated_at
		FROM cmts ORDER BY name`)

	if err != nil {
		return nil, fmt.Errorf("failed to list CMTS: %w", err)
	}
	defer rows.Close()

	var cmtsList []*models.CMTS
	for rows.Next() {
		var cmts models.CMTS
		var createdAt, updatedAt int64

		err := rows.Scan(&cmts.ID, &cmts.Name, &cmts.IPAddress, &cmts.SNMPPort,
			&cmts.CommunityRead, &cmts.CommunityWrite, &cmts.CMCommunityString,
			&cmts.SNMPVersion, &cmts.Enabled, &createdAt, &updatedAt)

		if err != nil {
			return nil, err
		}

		cmts.CreatedAt = time.Unix(createdAt, 0)
		cmts.UpdatedAt = time.Unix(updatedAt, 0)
		cmtsList = append(cmtsList, &cmts)
	}

	return cmtsList, nil
}

// UpdateCMTS updates a CMTS
func (db *DB) UpdateCMTS(cmts *models.CMTS) error {
	if err := cmts.Validate(); err != nil {
		return err
	}

	now := time.Now().Unix()
	result, err := db.conn.Exec(`
		UPDATE cmts SET name = ?, ip_address = ?, snmp_port = ?, community_read = ?,
			community_write = ?, cm_community_string = ?, snmp_version = ?, enabled = ?,
			updated_at = ?
		WHERE id = ?`,
		cmts.Name, cmts.IPAddress, cmts.SNMPPort, cmts.CommunityRead, cmts.CommunityWrite,
		cmts.CMCommunityString, cmts.SNMPVersion, cmts.Enabled, now, cmts.ID)

	if err != nil {
		return fmt.Errorf("failed to update CMTS: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrNotFound
	}

	return nil
}

// DeleteCMTS deletes a CMTS
func (db *DB) DeleteCMTS(id int) error {
	result, err := db.conn.Exec("DELETE FROM cmts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete CMTS: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Cable Modem operations

// UpsertModem inserts or updates a cable modem
func (db *DB) UpsertModem(modem *models.CableModem) error {
	now := time.Now().Unix()

	_, err := db.conn.Exec(`
		INSERT INTO cable_modem (cmts_id, mac_address, ip_address, sysdescr,
			current_firmware, signal_level, status, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			cmts_id = excluded.cmts_id,
			ip_address = excluded.ip_address,
			sysdescr = excluded.sysdescr,
			current_firmware = excluded.current_firmware,
			signal_level = excluded.signal_level,
			status = excluded.status,
			last_seen = excluded.last_seen`,
		modem.CMTSID, modem.MACAddress, modem.IPAddress, modem.SysDescr,
		modem.CurrentFirmware, modem.SignalLevel, modem.Status, now)

	if err != nil {
		return fmt.Errorf("failed to upsert modem: %w", err)
	}

	return nil
}

// GetModem retrieves a modem by ID
func (db *DB) GetModem(id int) (*models.CableModem, error) {
	var modem models.CableModem
	var lastSeen int64

	err := db.conn.QueryRow(`
		SELECT id, cmts_id, mac_address, ip_address, sysdescr, current_firmware,
			signal_level, status, last_seen
		FROM cable_modem WHERE id = ?`, id).Scan(
		&modem.ID, &modem.CMTSID, &modem.MACAddress, &modem.IPAddress, &modem.SysDescr,
		&modem.CurrentFirmware, &modem.SignalLevel, &modem.Status, &lastSeen)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get modem: %w", err)
	}

	modem.LastSeen = time.Unix(lastSeen, 0)
	return &modem, nil
}

// ListModems retrieves all modems, optionally filtered by CMTS
func (db *DB) ListModems(cmtsID int) ([]*models.CableModem, error) {
	query := `
		SELECT id, cmts_id, mac_address, ip_address, sysdescr, current_firmware,
			signal_level, status, last_seen
		FROM cable_modem`

	var rows *sql.Rows
	var err error

	if cmtsID > 0 {
		query += " WHERE cmts_id = ? ORDER BY last_seen DESC"
		rows, err = db.conn.Query(query, cmtsID)
	} else {
		query += " ORDER BY last_seen DESC"
		rows, err = db.conn.Query(query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list modems: %w", err)
	}
	defer rows.Close()

	var modems []*models.CableModem
	for rows.Next() {
		var modem models.CableModem
		var lastSeen int64

		err := rows.Scan(&modem.ID, &modem.CMTSID, &modem.MACAddress, &modem.IPAddress,
			&modem.SysDescr, &modem.CurrentFirmware, &modem.SignalLevel,
			&modem.Status, &lastSeen)

		if err != nil {
			return nil, err
		}

		modem.LastSeen = time.Unix(lastSeen, 0)
		modems = append(modems, &modem)
	}

	return modems, nil
}

// Upgrade Rule operations

// CreateRule creates a new upgrade rule
func (db *DB) CreateRule(rule *models.UpgradeRule) (int, error) {
	if err := rule.Validate(); err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	result, err := db.conn.Exec(`
		INSERT INTO upgrade_rule (name, description, match_type, match_criteria,
			tftp_server_ip, firmware_filename, enabled, priority, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.Name, rule.Description, rule.MatchType, rule.MatchCriteria,
		rule.TFTPServerIP, rule.FirmwareFilename, rule.Enabled, rule.Priority, now, now)

	if err != nil {
		return 0, fmt.Errorf("failed to create rule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetRule retrieves a rule by ID
func (db *DB) GetRule(id int) (*models.UpgradeRule, error) {
	var rule models.UpgradeRule
	var createdAt, updatedAt int64

	err := db.conn.QueryRow(`
		SELECT id, name, description, match_type, match_criteria, tftp_server_ip,
			firmware_filename, enabled, priority, created_at, updated_at
		FROM upgrade_rule WHERE id = ?`, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.MatchType, &rule.MatchCriteria,
		&rule.TFTPServerIP, &rule.FirmwareFilename, &rule.Enabled, &rule.Priority,
		&createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	rule.CreatedAt = time.Unix(createdAt, 0)
	rule.UpdatedAt = time.Unix(updatedAt, 0)

	return &rule, nil
}

// ListRules retrieves all upgrade rules
func (db *DB) ListRules() ([]*models.UpgradeRule, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, description, match_type, match_criteria, tftp_server_ip,
			firmware_filename, enabled, priority, created_at, updated_at
		FROM upgrade_rule ORDER BY priority DESC, name`)

	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.UpgradeRule
	for rows.Next() {
		var rule models.UpgradeRule
		var createdAt, updatedAt int64

		err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &rule.MatchType,
			&rule.MatchCriteria, &rule.TFTPServerIP, &rule.FirmwareFilename,
			&rule.Enabled, &rule.Priority, &createdAt, &updatedAt)

		if err != nil {
			return nil, err
		}

		rule.CreatedAt = time.Unix(createdAt, 0)
		rule.UpdatedAt = time.Unix(updatedAt, 0)
		rules = append(rules, &rule)
	}

	return rules, nil
}

// UpdateRule updates an upgrade rule
func (db *DB) UpdateRule(rule *models.UpgradeRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}

	now := time.Now().Unix()
	result, err := db.conn.Exec(`
		UPDATE upgrade_rule SET name = ?, description = ?, match_type = ?,
			match_criteria = ?, tftp_server_ip = ?, firmware_filename = ?,
			enabled = ?, priority = ?, updated_at = ?
		WHERE id = ?`,
		rule.Name, rule.Description, rule.MatchType, rule.MatchCriteria,
		rule.TFTPServerIP, rule.FirmwareFilename, rule.Enabled, rule.Priority,
		now, rule.ID)

	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrNotFound
	}

	return nil
}

// DeleteRule deletes an upgrade rule
func (db *DB) DeleteRule(id int) error {
	result, err := db.conn.Exec("DELETE FROM upgrade_rule WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Upgrade Job operations

// CreateJob creates a new upgrade job
func (db *DB) CreateJob(job *models.UpgradeJob) (int, error) {
	now := time.Now().Unix()
	result, err := db.conn.Exec(`
		INSERT INTO upgrade_job (modem_id, rule_id, cmts_id, mac_address, status,
			tftp_server_ip, firmware_filename, retry_count, max_retries, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ModemID, job.RuleID, job.CMTSID, job.MACAddress, job.Status,
		job.TFTPServerIP, job.FirmwareFilename, job.RetryCount, job.MaxRetries, now)

	if err != nil {
		return 0, fmt.Errorf("failed to create job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetJob retrieves a job by ID
func (db *DB) GetJob(id int) (*models.UpgradeJob, error) {
	var job models.UpgradeJob
	var createdAt int64
	var startedAt, completedAt sql.NullInt64

	err := db.conn.QueryRow(`
		SELECT id, modem_id, rule_id, cmts_id, mac_address, status, tftp_server_ip,
			firmware_filename, retry_count, max_retries, error_message,
			created_at, started_at, completed_at
		FROM upgrade_job WHERE id = ?`, id).Scan(
		&job.ID, &job.ModemID, &job.RuleID, &job.CMTSID, &job.MACAddress, &job.Status,
		&job.TFTPServerIP, &job.FirmwareFilename, &job.RetryCount, &job.MaxRetries,
		&job.ErrorMessage, &createdAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	job.CreatedAt = time.Unix(createdAt, 0)
	if startedAt.Valid {
		t := time.Unix(startedAt.Int64, 0)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		job.CompletedAt = &t
	}

	return &job, nil
}

// ListJobs retrieves jobs, optionally filtered by status
func (db *DB) ListJobs(status string, limit int) ([]*models.UpgradeJob, error) {
	query := `
		SELECT id, modem_id, rule_id, cmts_id, mac_address, status, tftp_server_ip,
			firmware_filename, retry_count, max_retries, error_message,
			created_at, started_at, completed_at
		FROM upgrade_job`

	var rows *sql.Rows
	var err error

	if status != "" {
		query += " WHERE status = ? ORDER BY created_at DESC"
		if limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", limit)
		}
		rows, err = db.conn.Query(query, status)
	} else {
		query += " ORDER BY created_at DESC"
		if limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", limit)
		}
		rows, err = db.conn.Query(query)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.UpgradeJob
	for rows.Next() {
		var job models.UpgradeJob
		var createdAt int64
		var startedAt, completedAt sql.NullInt64

		err := rows.Scan(&job.ID, &job.ModemID, &job.RuleID, &job.CMTSID, &job.MACAddress,
			&job.Status, &job.TFTPServerIP, &job.FirmwareFilename, &job.RetryCount,
			&job.MaxRetries, &job.ErrorMessage, &createdAt, &startedAt, &completedAt)

		if err != nil {
			return nil, err
		}

		job.CreatedAt = time.Unix(createdAt, 0)
		if startedAt.Valid {
			t := time.Unix(startedAt.Int64, 0)
			job.StartedAt = &t
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			job.CompletedAt = &t
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// UpdateJob updates a job
func (db *DB) UpdateJob(job *models.UpgradeJob) error {
	var startedAt, completedAt interface{}
	if job.StartedAt != nil {
		startedAt = job.StartedAt.Unix()
	}
	if job.CompletedAt != nil {
		completedAt = job.CompletedAt.Unix()
	}

	result, err := db.conn.Exec(`
		UPDATE upgrade_job SET status = ?, retry_count = ?, error_message = ?,
			started_at = ?, completed_at = ?
		WHERE id = ?`,
		job.Status, job.RetryCount, job.ErrorMessage, startedAt, completedAt, job.ID)

	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return models.ErrNotFound
	}

	return nil
}

// Activity Log operations

// LogActivity creates an activity log entry
func (db *DB) LogActivity(log *models.ActivityLog) error {
	now := time.Now().Unix()
	_, err := db.conn.Exec(`
		INSERT INTO activity_log (event_type, entity_type, entity_id, message, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		log.EventType, log.EntityType, log.EntityID, log.Message, log.Details, now)

	if err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}

	return nil
}

// ListActivityLogs retrieves recent activity logs
func (db *DB) ListActivityLogs(limit, offset int) ([]*models.ActivityLog, error) {
	rows, err := db.conn.Query(`
		SELECT id, event_type, entity_type, entity_id, message, details, created_at
		FROM activity_log ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to list activity logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.ActivityLog
	for rows.Next() {
		var log models.ActivityLog
		var createdAt int64

		err := rows.Scan(&log.ID, &log.EventType, &log.EntityType, &log.EntityID,
			&log.Message, &log.Details, &createdAt)

		if err != nil {
			return nil, err
		}

		log.CreatedAt = time.Unix(createdAt, 0)
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetSetting retrieves a setting value by key
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("setting not found: %s", key)
		}
		return "", err
	}
	return value, nil
}

// SetSetting updates or creates a setting
func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?
	`, key, value, time.Now().Unix(), value, time.Now().Unix())
	return err
}

// ListSettings retrieves all settings
func (db *DB) ListSettings() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}
