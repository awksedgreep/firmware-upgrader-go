package engine

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/awksedgreep/firmware-upgrader/internal/models"
)

// Matcher handles matching modems to upgrade rules
type Matcher struct{}

// NewMatcher creates a new matcher
func NewMatcher() *Matcher {
	return &Matcher{}
}

// MatchModemToRules finds the best matching rule for a modem
func (m *Matcher) MatchModemToRules(modem *models.CableModem, rules []*models.UpgradeRule) (*models.UpgradeRule, error) {
	if modem == nil {
		return nil, fmt.Errorf("modem cannot be nil")
	}

	// Rules should already be sorted by priority (descending)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		match, err := m.matchRule(modem, rule)
		if err != nil {
			log.Warn().
				Err(err).
				Int("rule_id", rule.ID).
				Str("rule_name", rule.Name).
				Msg("Error evaluating rule")
			continue
		}

		if match {
			log.Debug().
				Str("mac", modem.MACAddress).
				Int("rule_id", rule.ID).
				Str("rule_name", rule.Name).
				Msg("Modem matched to rule")
			return rule, nil
		}
	}

	return nil, nil // No matching rule found
}

// matchRule evaluates if a modem matches a specific rule
func (m *Matcher) matchRule(modem *models.CableModem, rule *models.UpgradeRule) (bool, error) {
	criteria, err := rule.ParseMatchCriteria()
	if err != nil {
		return false, fmt.Errorf("failed to parse match criteria: %w", err)
	}

	switch rule.MatchType {
	case "MAC_RANGE":
		return m.matchMACRange(modem.MACAddress, criteria)
	case "SYSDESCR_REGEX":
		return m.matchSysDescrRegex(modem.SysDescr, criteria)
	default:
		return false, fmt.Errorf("unknown match type: %s", rule.MatchType)
	}
}

// matchMACRange checks if a MAC address falls within a range
func (m *Matcher) matchMACRange(mac string, criteria *models.MatchCriteria) (bool, error) {
	if criteria.StartMAC == "" || criteria.EndMAC == "" {
		return false, fmt.Errorf("MAC range criteria missing start_mac or end_mac")
	}

	// Parse MAC addresses
	modemMAC, err := parseMAC(mac)
	if err != nil {
		return false, fmt.Errorf("invalid modem MAC: %w", err)
	}

	startMAC, err := parseMAC(criteria.StartMAC)
	if err != nil {
		return false, fmt.Errorf("invalid start MAC: %w", err)
	}

	endMAC, err := parseMAC(criteria.EndMAC)
	if err != nil {
		return false, fmt.Errorf("invalid end MAC: %w", err)
	}

	// Compare as uint64 for easy range checking
	modemInt := macToUint64(modemMAC)
	startInt := macToUint64(startMAC)
	endInt := macToUint64(endMAC)

	inRange := modemInt >= startInt && modemInt <= endInt

	log.Debug().
		Str("modem_mac", mac).
		Str("start_mac", criteria.StartMAC).
		Str("end_mac", criteria.EndMAC).
		Bool("in_range", inRange).
		Msg("MAC range check")

	return inRange, nil
}

// matchSysDescrRegex checks if sysDescr matches a regex pattern
func (m *Matcher) matchSysDescrRegex(sysDescr string, criteria *models.MatchCriteria) (bool, error) {
	if criteria.Pattern == "" {
		return false, fmt.Errorf("regex pattern is empty")
	}

	// Compile regex
	re, err := regexp.Compile(criteria.Pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	match := re.MatchString(sysDescr)

	log.Debug().
		Str("sysdescr", sysDescr).
		Str("pattern", criteria.Pattern).
		Bool("match", match).
		Msg("SysDescr regex check")

	return match, nil
}

// BatchMatchModems matches multiple modems to rules
func (m *Matcher) BatchMatchModems(modems []*models.CableModem, rules []*models.UpgradeRule) map[int]*models.UpgradeRule {
	matches := make(map[int]*models.UpgradeRule)

	for _, modem := range modems {
		rule, err := m.MatchModemToRules(modem, rules)
		if err != nil {
			log.Error().
				Err(err).
				Int("modem_id", modem.ID).
				Str("mac", modem.MACAddress).
				Msg("Failed to match modem")
			continue
		}

		if rule != nil {
			matches[modem.ID] = rule
		}
	}

	log.Info().
		Int("total_modems", len(modems)).
		Int("matched", len(matches)).
		Msg("Batch matching completed")

	return matches
}

// ValidateMatchCriteria validates match criteria JSON
func (m *Matcher) ValidateMatchCriteria(matchType, criteriaJSON string) error {
	var criteria models.MatchCriteria
	if err := json.Unmarshal([]byte(criteriaJSON), &criteria); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	switch matchType {
	case "MAC_RANGE":
		if criteria.StartMAC == "" {
			return fmt.Errorf("start_mac is required for MAC_RANGE")
		}
		if criteria.EndMAC == "" {
			return fmt.Errorf("end_mac is required for MAC_RANGE")
		}

		// Validate MAC addresses
		if _, err := parseMAC(criteria.StartMAC); err != nil {
			return fmt.Errorf("invalid start_mac: %w", err)
		}
		if _, err := parseMAC(criteria.EndMAC); err != nil {
			return fmt.Errorf("invalid end_mac: %w", err)
		}

		// Ensure start < end
		start, _ := parseMAC(criteria.StartMAC)
		end, _ := parseMAC(criteria.EndMAC)
		if macToUint64(start) > macToUint64(end) {
			return fmt.Errorf("start_mac must be less than or equal to end_mac")
		}

	case "SYSDESCR_REGEX":
		if criteria.Pattern == "" {
			return fmt.Errorf("pattern is required for SYSDESCR_REGEX")
		}

		// Validate regex compiles
		if _, err := regexp.Compile(criteria.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}

	default:
		return fmt.Errorf("unknown match type: %s", matchType)
	}

	return nil
}

// Helper functions

// parseMAC parses a MAC address string to net.HardwareAddr
func parseMAC(mac string) (net.HardwareAddr, error) {
	// Normalize MAC address format
	mac = strings.ToUpper(mac)

	// Handle Cisco dot notation (0001.5C12.3456)
	if strings.Count(mac, ".") == 2 {
		parts := strings.Split(mac, ".")
		if len(parts) == 3 && len(parts[0]) == 4 && len(parts[1]) == 4 && len(parts[2]) == 4 {
			// Convert to standard format
			mac = fmt.Sprintf("%s:%s:%s:%s:%s:%s",
				parts[0][0:2], parts[0][2:4],
				parts[1][0:2], parts[1][2:4],
				parts[2][0:2], parts[2][2:4])
		}
	} else {
		// Handle hyphen and other dots
		mac = strings.ReplaceAll(mac, "-", ":")
		mac = strings.ReplaceAll(mac, ".", ":")
	}

	// Handle formats like AABBCCDDEEFF
	if !strings.Contains(mac, ":") && len(mac) == 12 {
		mac = fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
	}

	hw, err := net.ParseMAC(mac)
	if err != nil {
		return nil, err
	}

	return hw, nil
}

// macToUint64 converts a MAC address to uint64 for comparison
func macToUint64(mac net.HardwareAddr) uint64 {
	if len(mac) != 6 {
		return 0
	}

	return uint64(mac[0])<<40 |
		uint64(mac[1])<<32 |
		uint64(mac[2])<<24 |
		uint64(mac[3])<<16 |
		uint64(mac[4])<<8 |
		uint64(mac[5])
}

// ShouldUpgrade determines if a modem needs upgrading based on current firmware
func (m *Matcher) ShouldUpgrade(modem *models.CableModem, rule *models.UpgradeRule) bool {
	// If we don't know current firmware, assume upgrade is needed
	if modem.CurrentFirmware == "" {
		log.Debug().
			Str("mac", modem.MACAddress).
			Msg("Current firmware unknown, assuming upgrade needed")
		return true
	}

	// Check if already running target firmware
	targetFirmware := extractFirmwareVersion(rule.FirmwareFilename)
	currentFirmware := modem.CurrentFirmware

	if targetFirmware == "" {
		// Can't determine target version, proceed with upgrade
		return true
	}

	if currentFirmware == targetFirmware {
		log.Debug().
			Str("mac", modem.MACAddress).
			Str("firmware", currentFirmware).
			Msg("Modem already running target firmware")
		return false
	}

	log.Debug().
		Str("mac", modem.MACAddress).
		Str("current", currentFirmware).
		Str("target", targetFirmware).
		Msg("Upgrade needed")

	return true
}

// extractFirmwareVersion extracts version from firmware filename
func extractFirmwareVersion(filename string) string {
	// Common patterns:
	// "arris-sb8200-v1.2.3.bin"
	// "firmware-1.2.3.bin"
	// "firmware-v1.2.3.bin"
	// "CM_v2.0.1_release.bin"

	// Try to find version pattern (with or without 'v' prefix)
	re := regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:\.\d+)*)`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		// Return the version without 'v' prefix
		return matches[1]
	}

	return ""
}

// FilterEligibleModems filters modems that are eligible for upgrade
func (m *Matcher) FilterEligibleModems(modems []*models.CableModem) []*models.CableModem {
	eligible := make([]*models.CableModem, 0, len(modems))

	for _, modem := range modems {
		// Only upgrade modems that are online
		if modem.Status != "online" {
			log.Debug().
				Str("mac", modem.MACAddress).
				Str("status", modem.Status).
				Msg("Skipping modem - not online")
			continue
		}

		// Check signal level (optional - could make this configurable)
		if modem.SignalLevel < -15.0 || modem.SignalLevel > 15.0 {
			log.Debug().
				Str("mac", modem.MACAddress).
				Float64("signal", modem.SignalLevel).
				Msg("Skipping modem - poor signal level")
			continue
		}

		eligible = append(eligible, modem)
	}

	log.Debug().
		Int("total", len(modems)).
		Int("eligible", len(eligible)).
		Msg("Filtered eligible modems")

	return eligible
}
