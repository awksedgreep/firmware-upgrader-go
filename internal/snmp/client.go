package snmp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/models"
	"github.com/gosnmp/gosnmp"
	"github.com/rs/zerolog/log"
)

// DOCSIS OIDs for modem discovery
const (
	// Cable modem MAC address table
	OIDDocsIfCmtsCmStatusMacAddress = "1.3.6.1.2.1.10.127.1.3.3.1.2"
	// Cable modem IP address
	OIDDocsIfCmtsCmStatusIpAddress = "1.3.6.1.2.1.10.127.1.3.3.1.3"
	// Cable modem downstream power level
	OIDDocsIfCmtsCmStatusDownstreamPower = "1.3.6.1.2.1.10.127.1.3.3.1.6"
	// Cable modem status
	OIDDocsIfCmtsCmStatusValue = "1.3.6.1.2.1.10.127.1.3.3.1.9"
	// System description (for firmware matching)
	OIDSysDescr = "1.3.6.1.2.1.1.1.0"
	// TFTP server address for firmware upgrades
	OIDDocsDevSwServer = "1.3.6.1.2.1.69.1.1.3.0"
	// Firmware filename
	OIDDocsDevSwFilename = "1.3.6.1.2.1.69.1.1.4.0"
	// Admin status to trigger upgrade
	OIDDocsDevSwAdminStatus = "1.3.6.1.2.1.69.1.1.5.0"
	// Operational status of upgrade
	OIDDocsDevSwOperStatus = "1.3.6.1.2.1.69.1.1.6.0"
)

// Client handles SNMP operations
type Client struct {
	conn *gosnmp.GoSNMP
}

// NewClient creates a new SNMP client
func NewClient(cmts *models.CMTS) (*Client, error) {
	if cmts == nil {
		return nil, fmt.Errorf("CMTS cannot be nil")
	}

	// Determine SNMP version
	var version gosnmp.SnmpVersion
	switch cmts.SNMPVersion {
	case 1:
		version = gosnmp.Version1
	case 2:
		version = gosnmp.Version2c
	case 3:
		version = gosnmp.Version3
	default:
		version = gosnmp.Version2c
	}

	conn := &gosnmp.GoSNMP{
		Target:    cmts.IPAddress,
		Port:      uint16(cmts.SNMPPort),
		Community: cmts.CommunityRead,
		Version:   version,
		Timeout:   time.Duration(10) * time.Second,
		Retries:   3,
		MaxOids:   60, // Max OIDs per GET request
	}

	// Set connection timeout with context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt connection with timeout
	connectErr := make(chan error, 1)
	go func() {
		connectErr <- conn.Connect()
	}()

	select {
	case err := <-connectErr:
		if err != nil {
			return nil, fmt.Errorf("failed to connect to CMTS %s:%d (version: v%d, community: %s): %w",
				cmts.IPAddress, cmts.SNMPPort, cmts.SNMPVersion, cmts.CommunityRead, err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("connection timeout to CMTS %s:%d after 15 seconds", cmts.IPAddress, cmts.SNMPPort)
	}

	log.Debug().
		Str("cmts", cmts.Name).
		Str("ip", cmts.IPAddress).
		Msg("SNMP connection established")

	return &Client{conn: conn}, nil
}

// Close closes the SNMP connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Conn.Close()
	}
	return nil
}

// modemInfo holds basic modem info from CMTS walk
type modemInfo struct {
	ifIndex string
	mac     string
}

// DiscoverModems discovers all cable modems on the CMTS with concurrent polling
func (c *Client) DiscoverModems(cmts *models.CMTS) ([]*models.CableModem, error) {
	log.Info().
		Str("cmts", cmts.Name).
		Str("ip", cmts.IPAddress).
		Msg("Starting modem discovery via SNMP")

	// Walk the MAC address table with timeout
	startTime := time.Now()
	macResults, err := c.conn.BulkWalkAll(OIDDocsIfCmtsCmStatusMacAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to walk MAC table on %s (%s): %w", cmts.Name, cmts.IPAddress, err)
	}

	log.Debug().
		Str("cmts", cmts.Name).
		Dur("duration", time.Since(startTime)).
		Int("results", len(macResults)).
		Msg("MAC table walk completed")

	// Extract basic modem info from walk results
	modemInfos := make([]modemInfo, 0, len(macResults))
	for _, result := range macResults {
		ifIndex := extractIndexFromOID(result.Name, OIDDocsIfCmtsCmStatusMacAddress)
		if ifIndex == "" {
			continue
		}

		mac := parseMACAddress(result)
		if mac == "" {
			log.Warn().Str("oid", result.Name).Msg("Failed to parse MAC address")
			continue
		}

		modemInfos = append(modemInfos, modemInfo{
			ifIndex: ifIndex,
			mac:     mac,
		})
	}

	log.Info().
		Str("cmts", cmts.Name).
		Int("modems", len(modemInfos)).
		Msg("Starting concurrent modem detail polling")

	// Poll modem details concurrently with rate limiting
	modems := c.pollModemsDetails(cmts, modemInfos)

	log.Info().
		Str("cmts", cmts.Name).
		Int("discovered", len(modems)).
		Dur("total_duration", time.Since(startTime)).
		Msg("Modem discovery completed")

	return modems, nil
}

// pollModemsDetails polls modem details concurrently with rate limiting
func (c *Client) pollModemsDetails(cmts *models.CMTS, modemInfos []modemInfo) []*models.CableModem {
	// Use worker pool to avoid UDP buffer overruns
	// Limit concurrent SNMP queries to avoid overwhelming the CMTS
	const maxWorkers = 50
	const maxQueriesPerSecond = 200

	workers := maxWorkers
	if len(modemInfos) < workers {
		workers = len(modemInfos)
	}

	// Create work queue and results channel
	workQueue := make(chan modemInfo, len(modemInfos))
	results := make(chan *models.CableModem, len(modemInfos))

	// Rate limiter - allow burst but limit sustained rate
	ticker := time.NewTicker(time.Second / time.Duration(maxQueriesPerSecond))
	defer ticker.Stop()

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for info := range workQueue {
				// Rate limit
				<-ticker.C

				// Poll modem details
				modem := c.pollSingleModem(cmts, info)
				if modem != nil {
					results <- modem
				}
			}
		}(i)
	}

	// Queue all work
	for _, info := range modemInfos {
		workQueue <- info
	}
	close(workQueue)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	modems := make([]*models.CableModem, 0, len(modemInfos))
	for modem := range results {
		modems = append(modems, modem)
	}

	return modems
}

// pollSingleModem polls details for a single modem
func (c *Client) pollSingleModem(cmts *models.CMTS, info modemInfo) *models.CableModem {
	// Get IP address
	ipAddress := c.getModemIP(info.ifIndex)

	// Get signal level
	signalLevel := c.getSignalLevel(info.ifIndex)

	// Get status
	status := c.getModemStatus(info.ifIndex)

	// Get sysDescr (for modem-specific queries, we'd need the CM community string)
	sysDescr := c.getModemSysDescr(cmts, info.mac)

	return &models.CableModem{
		CMTSID:          cmts.ID,
		MACAddress:      info.mac,
		IPAddress:       ipAddress,
		SysDescr:        sysDescr,
		CurrentFirmware: extractFirmwareFromSysDescr(sysDescr),
		SignalLevel:     signalLevel,
		Status:          status,
		LastSeen:        time.Now(),
	}
}

// getModemIP retrieves the IP address for a modem by interface index
func (c *Client) getModemIP(ifIndex string) string {
	oid := fmt.Sprintf("%s.%s", OIDDocsIfCmtsCmStatusIpAddress, ifIndex)
	result, err := c.conn.Get([]string{oid})
	if err != nil {
		return ""
	}

	if len(result.Variables) == 0 {
		return ""
	}

	return parseIPAddress(result.Variables[0])
}

// getSignalLevel retrieves the downstream power level for a modem
func (c *Client) getSignalLevel(ifIndex string) float64 {
	oid := fmt.Sprintf("%s.%s", OIDDocsIfCmtsCmStatusDownstreamPower, ifIndex)
	result, err := c.conn.Get([]string{oid})
	if err != nil {
		return 0.0
	}

	if len(result.Variables) == 0 {
		return 0.0
	}

	// Power is typically in tenths of dBmV
	switch v := result.Variables[0].Value.(type) {
	case int:
		return float64(v) / 10.0
	case int64:
		return float64(v) / 10.0
	case uint:
		return float64(v) / 10.0
	case uint64:
		return float64(v) / 10.0
	}

	return 0.0
}

// getModemStatus retrieves the operational status of a modem
func (c *Client) getModemStatus(ifIndex string) string {
	oid := fmt.Sprintf("%s.%s", OIDDocsIfCmtsCmStatusValue, ifIndex)
	result, err := c.conn.Get([]string{oid})
	if err != nil {
		return "unknown"
	}

	if len(result.Variables) == 0 {
		return "unknown"
	}

	// DOCSIS status values: 1=other, 2=notReady, 3=notSynchronized,
	// 4=phySynchronized, 5=usParametersAcquired, 6=rangingComplete,
	// 7=ipComplete, 8=todEstablished, 9=securityEstablished,
	// 10=paramTransferComplete, 11=registrationComplete, 12=operational,
	// 13=accessDenied
	switch result.Variables[0].Value {
	case 12:
		return "online"
	case 13:
		return "denied"
	case 1, 2, 3:
		return "offline"
	default:
		return "partial"
	}
}

// getModemSysDescr retrieves sysDescr from the cable modem itself
func (c *Client) getModemSysDescr(cmts *models.CMTS, mac string) string {
	// This requires connecting directly to the cable modem
	// Skip if we don't have a CM community string
	if cmts.CMCommunityString == "" {
		return ""
	}

	// Try to connect to the modem (this is simplified)
	// In production, you'd want to cache these connections
	return ""
}

// TriggerFirmwareUpgrade triggers a firmware upgrade on a cable modem
func (c *Client) TriggerFirmwareUpgrade(modemIP, tftpServer, filename string) error {
	log.Info().
		Str("modem_ip", modemIP).
		Str("tftp_server", tftpServer).
		Str("firmware", filename).
		Msg("Triggering firmware upgrade")

	// Verify connectivity before attempting upgrade
	result, err := c.conn.Get([]string{OIDSysDescr})
	if err != nil {
		return fmt.Errorf("failed to verify modem connectivity at %s: %w", modemIP, err)
	}
	if len(result.Variables) == 0 {
		return fmt.Errorf("modem at %s not responding to SNMP queries", modemIP)
	}

	// Set TFTP server address
	if err := c.setOID(OIDDocsDevSwServer, tftpServer, gosnmp.IPAddress); err != nil {
		return fmt.Errorf("failed to set TFTP server %s on modem %s: %w", tftpServer, modemIP, err)
	}
	log.Debug().Str("modem_ip", modemIP).Str("tftp_server", tftpServer).Msg("TFTP server set")

	// Set firmware filename
	if err := c.setOID(OIDDocsDevSwFilename, filename, gosnmp.OctetString); err != nil {
		return fmt.Errorf("failed to set firmware filename %s on modem %s: %w", filename, modemIP, err)
	}
	log.Debug().Str("modem_ip", modemIP).Str("filename", filename).Msg("Firmware filename set")

	// Trigger upgrade (set admin status to upgradeFromMgt(1))
	if err := c.setOID(OIDDocsDevSwAdminStatus, 1, gosnmp.Integer); err != nil {
		return fmt.Errorf("failed to trigger upgrade on modem %s: %w", modemIP, err)
	}

	log.Info().
		Str("modem_ip", modemIP).
		Str("tftp_server", tftpServer).
		Str("firmware", filename).
		Msg("Firmware upgrade triggered successfully")

	return nil
}

// CheckUpgradeStatus checks the status of an ongoing firmware upgrade
func (c *Client) CheckUpgradeStatus() (string, error) {
	result, err := c.conn.Get([]string{OIDDocsDevSwOperStatus})
	if err != nil {
		return "", fmt.Errorf("failed to get upgrade status: %w", err)
	}

	if len(result.Variables) == 0 {
		return "unknown", nil
	}

	// Operational status: 1=inProgress, 2=completeFromProvisioning,
	// 3=completeFromMgt, 4=failed, 5=other
	switch result.Variables[0].Value {
	case 1:
		return "in_progress", nil
	case 2, 3:
		return "completed", nil
	case 4:
		return "failed", nil
	default:
		return "unknown", nil
	}
}

// setOID sets an SNMP OID value
func (c *Client) setOID(oid string, value interface{}, valueType gosnmp.Asn1BER) error {
	pdu := gosnmp.SnmpPDU{
		Name:  oid,
		Type:  valueType,
		Value: value,
	}

	result, err := c.conn.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return err
	}

	if result.Error != gosnmp.NoError {
		return fmt.Errorf("SNMP SET error: %s", result.Error.String())
	}

	return nil
}

// Helper functions

// extractIndexFromOID extracts the interface index from an OID
func extractIndexFromOID(oidStr, baseOID string) string {
	prefix := baseOID + "."
	if !strings.HasPrefix(oidStr, prefix) {
		return ""
	}
	return strings.TrimPrefix(oidStr, prefix)
}

// parseMACAddress converts SNMP result to MAC address string
func parseMACAddress(result gosnmp.SnmpPDU) string {
	switch v := result.Value.(type) {
	case []byte:
		if len(v) != 6 {
			return ""
		}
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
			v[0], v[1], v[2], v[3], v[4], v[5])
	case string:
		// Already a string, might need formatting
		return formatMACAddress(v)
	}
	return ""
}

// formatMACAddress ensures MAC is in standard format
func formatMACAddress(mac string) string {
	// Remove common separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	// Should be 12 hex characters
	if len(mac) != 12 {
		return ""
	}

	// Format as XX:XX:XX:XX:XX:XX
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
}

// parseIPAddress converts SNMP result to IP address string
func parseIPAddress(result gosnmp.SnmpPDU) string {
	switch v := result.Value.(type) {
	case []byte:
		if len(v) == 4 {
			return net.IP(v).String()
		}
		return ""
	case string:
		// Already a string
		if net.ParseIP(v) != nil {
			return v
		}
		return ""
	}
	return ""
}

// extractFirmwareFromSysDescr attempts to extract firmware version from sysDescr
func extractFirmwareFromSysDescr(sysDescr string) string {
	if sysDescr == "" {
		return ""
	}

	// Common patterns:
	// "Arris CM8200 DOCSIS 3.1 Cable Modem SW v1.2.3"
	// "Motorola SB6141 HW_REV: 7.0 VENDOR: Motorola SW_REV: SB6141-7.0.0.1-SCM01-SHPC"

	// Try to find version patterns using simple string matching
	if idx := strings.Index(strings.ToUpper(sysDescr), "SW_REV:"); idx != -1 {
		parts := strings.Fields(sysDescr[idx:])
		if len(parts) > 1 {
			return parts[1]
		}
	}
	if idx := strings.Index(strings.ToUpper(sysDescr), "SW V"); idx != -1 {
		parts := strings.Fields(sysDescr[idx:])
		if len(parts) > 2 {
			return parts[2]
		}
	}

	return ""
}

// ConnectToModem creates an SNMP client connected to a specific cable modem
func ConnectToModem(modemIP, community string, port int) (*Client, error) {
	conn := &gosnmp.GoSNMP{
		Target:    modemIP,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		Retries:   2,
	}

	if err := conn.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to modem %s: %w", modemIP, err)
	}

	log.Debug().Str("modem_ip", modemIP).Msg("Connected to cable modem")

	return &Client{conn: conn}, nil
}

// GetModemSysDescr retrieves sysDescr directly from a cable modem
func (c *Client) GetModemSysDescr() (string, error) {
	result, err := c.conn.Get([]string{OIDSysDescr})
	if err != nil {
		return "", fmt.Errorf("failed to get sysDescr: %w", err)
	}

	if len(result.Variables) == 0 {
		return "", fmt.Errorf("no sysDescr returned")
	}

	switch v := result.Variables[0].Value.(type) {
	case []byte:
		return string(v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("unexpected sysDescr type: %T", v)
	}
}

// ParseSignalLevel converts signal level string to float
func ParseSignalLevel(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return val
}
