# Firmware Upgrader - User Guide

**Version:** v0.5.1  
**Date:** 2024-11-08

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Dashboard Overview](#dashboard-overview)
3. [Managing CMTS](#managing-cmts)
4. [Discovering Modems](#discovering-modems)
5. [Creating Upgrade Rules](#creating-upgrade-rules)
6. [Managing Jobs](#managing-jobs)
7. [Viewing Activity](#viewing-activity)
8. [System Settings](#system-settings)
9. [Common Tasks](#common-tasks)
10. [Troubleshooting](#troubleshooting)

---

## Getting Started

### Accessing the Application

1. Open your web browser
2. Navigate to: `http://YOUR_ROUTER_IP:8080`
   - Example: `http://192.168.88.1:8080`
3. The dashboard will load automatically

### First Time Setup

**Step 1: Add Your First CMTS**

1. Click **CMTS** in the navigation menu
2. Click **Add CMTS** button
3. Fill in the form:
   - **Name:** Give it a friendly name (e.g., "Main CMTS")
   - **IP Address:** The CMTS IP address
   - **SNMP Port:** Usually 161
   - **Read Community:** SNMP community string (usually "public")
   - **Write Community:** For making changes (usually "private")
   - **Modem Community:** For querying cable modems
   - **SNMP Version:** Select 2 (most common)
   - **Enabled:** Check this box
4. Click **Create**

**Step 2: Discover Modems**

1. Click **Discover** button next to your CMTS
2. Wait 1-2 minutes for discovery to complete
3. Click **Modems** in navigation to see discovered modems

**Step 3: Create an Upgrade Rule**

1. Click **Rules** in navigation
2. Click **Add Rule** button
3. Fill in the form (see Creating Upgrade Rules section)
4. Click **Create**

**Step 4: Trigger Rule Evaluation**

1. Click **Evaluate Rules** button on Rules page
2. Check **Jobs** page to see created upgrade jobs
3. Jobs will process automatically

---

## Dashboard Overview

The dashboard shows a quick overview of your system:

### Key Metrics

- **Total CMTS:** Number of configured CMTS devices
- **Total Modems:** Number of discovered cable modems
- **Total Rules:** Number of upgrade rules
- **Pending Jobs:** Jobs waiting to be processed
- **In Progress Jobs:** Jobs currently running

### Recent Activity

Shows the last 10 system events including:
- Modem discoveries
- Upgrades started/completed
- Configuration changes
- Errors and warnings

### Quick Actions

- **Trigger Discovery:** Discover modems on all enabled CMTS
- **Evaluate Rules:** Check all modems against upgrade rules

---

## Managing CMTS

CMTS (Cable Modem Termination System) devices are the systems that manage your cable modems.

### Adding a CMTS

1. Go to **CMTS** page
2. Click **Add CMTS**
3. Enter details:
   - **Name:** Descriptive name
   - **IP Address:** CMTS management IP
   - **SNMP Port:** Usually 161
   - **Communities:** SNMP community strings
   - **Version:** SNMP version (1, 2, or 3)
4. Click **Create**

### Testing SNMP Access

Before adding a CMTS, test SNMP access from your router:

```bash
# From MikroTik
/tool fetch url="snmp://public@192.168.1.1/1.3.6.1.2.1.1.1.0"

# From Linux
snmpget -v2c -c public 192.168.1.1 1.3.6.1.2.1.1.1.0
```

### Editing a CMTS

1. Click **Edit** button next to the CMTS
2. Modify fields as needed
3. Click **Save**

### Disabling a CMTS

To temporarily stop using a CMTS without deleting it:

1. Click **Edit** on the CMTS
2. Uncheck **Enabled**
3. Click **Save**

Discovery and upgrades will skip disabled CMTS devices.

### Deleting a CMTS

⚠️ **Warning:** This will also delete all associated modems and jobs!

1. Click **Delete** button next to the CMTS
2. Confirm deletion

---

## Discovering Modems

### Automatic Discovery

Discovery runs automatically every 15 minutes (configurable in Settings).

### Manual Discovery

**Discover All CMTS:**
1. Go to Dashboard
2. Click **Trigger Discovery** button

**Discover Specific CMTS:**
1. Go to **CMTS** page
2. Click **Discover** button next to the CMTS

### Discovery Process

1. System connects to CMTS via SNMP
2. Walks DOCSIS MIB to find modems
3. Extracts:
   - MAC address
   - IP address
   - System description
   - Current firmware version
   - Signal level
   - Status
4. Saves/updates modems in database

### Viewing Modems

1. Click **Modems** in navigation
2. See list of all discovered modems
3. Filter by CMTS using dropdown

### Modem Information

Each modem shows:
- **MAC Address:** Unique identifier
- **IP Address:** Current IP
- **Firmware:** Current version running
- **Signal:** Signal strength (dBmV)
- **Status:** online, offline, etc.
- **Last Seen:** When last discovered

---

## Creating Upgrade Rules

Rules determine which modems get upgraded and to what firmware.

### Rule Types

**1. MAC Range Rule**
- Matches modems by MAC address range
- Good for vendor-specific upgrades
- Example: All Arris modems (00:01:5C:xx:xx:xx)

**2. System Description Regex Rule**
- Matches modems by model name/description
- Good for model-specific upgrades
- Example: All "Arris SB8200" modems

### Creating a MAC Range Rule

1. Go to **Rules** page
2. Click **Add Rule**
3. Fill in form:
   - **Name:** "Arris SB8200 Upgrade"
   - **Description:** "Upgrade all Arris SB8200 to v2.0"
   - **Match Type:** Select "MAC Range"
   - **Start MAC:** 00:01:5C:00:00:00
   - **End MAC:** 00:01:5C:FF:FF:FF
   - **TFTP Server:** 192.168.1.50
   - **Firmware File:** arris-sb8200-v2.0.0.bin
   - **Priority:** 100 (higher = evaluated first)
   - **Enabled:** Checked
4. Click **Create**

### Creating a Regex Rule

1. Go to **Rules** page
2. Click **Add Rule**
3. Fill in form:
   - **Name:** "Motorola MB8600 Upgrade"
   - **Match Type:** Select "Regex Pattern"
   - **Pattern:** Motorola MB8600
   - **TFTP Server:** 192.168.1.50
   - **Firmware File:** motorola-mb8600-v3.0.1.bin
   - **Priority:** 90
   - **Enabled:** Checked
4. Click **Create**

### Rule Priority

- Rules are evaluated in priority order (highest first)
- If a modem matches multiple rules, highest priority wins
- Use priority to control which upgrades happen first

### Editing Rules

1. Click **Edit** button next to rule
2. Modify fields
3. Click **Save**

### Disabling Rules

To temporarily stop a rule without deleting:

1. Click **Edit** on the rule
2. Uncheck **Enabled**
3. Click **Save**

### Deleting Rules

1. Click **Delete** button next to rule
2. Confirm deletion

⚠️ **Note:** Deleting a rule doesn't affect jobs already created.

---

## Managing Jobs

Jobs are the actual upgrade tasks that get executed.

### Job Lifecycle

1. **PENDING** - Waiting to be processed by a worker
2. **IN_PROGRESS** - Currently being upgraded
3. **COMPLETED** - Successfully upgraded
4. **FAILED** - Upgrade failed (will retry)
5. **SKIPPED** - Skipped due to conditions

### Viewing Jobs

1. Click **Jobs** in navigation
2. See all upgrade jobs
3. Use status filter to show specific job types

### Job Information

Each job shows:
- **MAC Address:** Modem being upgraded
- **Status:** Current state
- **Firmware:** Target firmware file
- **Retry Count:** How many times retried
- **Error Message:** If failed, why
- **Created:** When job was created
- **Started:** When processing began
- **Completed:** When finished

### Retry Logic

Failed jobs automatically retry with exponential backoff:
- 1st retry: Wait 30 seconds
- 2nd retry: Wait 60 seconds
- 3rd retry: Wait 120 seconds
- 4th retry: Wait 240 seconds (max)

After all retries exhausted, job marked as FAILED.

### Manual Retry

To retry a failed job:

1. Go to **Jobs** page
2. Filter by **Status:** FAILED
3. Click **Retry** button next to job
4. Job will be reset to PENDING

### Monitoring Progress

**Dashboard:**
- Shows count of pending and in-progress jobs

**Activity Log:**
- Shows upgrade started/completed events

**Jobs Page:**
- Real-time view of all jobs
- Refresh page to see updates

---

## Viewing Activity

The activity log shows all system events.

### Accessing Activity Log

1. Click **Activity** in navigation
2. See chronological list of events

### Event Types

- **MODEM_DISCOVERED:** New modem found
- **UPGRADE_STARTED:** Firmware upgrade began
- **UPGRADE_COMPLETED:** Upgrade finished successfully
- **UPGRADE_FAILED:** Upgrade failed
- **RULE_CREATED/UPDATED/DELETED:** Rule changes
- **CMTS_ADDED/UPDATED/DELETED:** CMTS changes
- **SYSTEM_EVENT:** General system events

### Using Activity Log

**Troubleshooting:**
- Look for ERROR events
- Check upgrade failure messages
- Verify discovery is running

**Monitoring:**
- Confirm upgrades completing
- Check discovery frequency
- Verify rule evaluations

**Auditing:**
- See who made changes (future feature)
- Track when modems were upgraded
- Review system history

---

## System Settings

Configure system behavior via Settings page.

### Accessing Settings

1. Click **Settings** in navigation
2. See all configurable parameters
3. Edit values and click **Save** for each

### Available Settings

**Engine Settings:**

- **Workers:** Number of concurrent upgrade jobs (default: 4)
  - More workers = faster upgrades
  - Too many = may overwhelm CMTS
  - Recommended: 2-8 depending on hardware

- **Poll Interval:** How often to check for new jobs (default: 30s)
  - Lower = faster job pickup
  - Higher = less CPU usage

- **Discovery Interval:** Auto-discovery frequency (default: 900s / 15min)
  - Lower = more frequent updates
  - Higher = less SNMP traffic

- **Evaluation Interval:** Auto-evaluate rules (default: 1800s / 30min)
  - Lower = faster rule application
  - Higher = less processing

- **Job Timeout:** Max time for upgrade (default: 300s / 5min)
  - Increase for slow modems
  - Decrease to fail fast

- **Retry Attempts:** Max retries for failed jobs (default: 3)
  - More = more persistent
  - Less = fail faster

**Modem Settings:**

- **Signal Level Min:** Minimum acceptable signal (default: -15.0 dBmV)
  - Only upgrade modems with signal >= this

- **Signal Level Max:** Maximum acceptable signal (default: 15.0 dBmV)
  - Only upgrade modems with signal <= this

- **Max Upgrades Per CMTS:** Concurrent upgrades per CMTS (default: 10)
  - Prevents overloading CMTS
  - Adjust based on CMTS capacity

### Applying Settings

⚠️ **Important:** Some settings require application restart:
- Workers
- Poll Interval

Other settings take effect immediately.

### Recommended Settings

**Small Network (< 50 modems):**
```
Workers: 2
Discovery Interval: 1800s (30min)
Evaluation Interval: 3600s (60min)
Max Upgrades Per CMTS: 5
```

**Medium Network (50-200 modems):**
```
Workers: 4
Discovery Interval: 900s (15min)
Evaluation Interval: 1800s (30min)
Max Upgrades Per CMTS: 10
```

**Large Network (200+ modems):**
```
Workers: 8
Discovery Interval: 600s (10min)
Evaluation Interval: 900s (15min)
Max Upgrades Per CMTS: 15
```

---

## Common Tasks

### Task: Upgrade All Arris SB8200 Modems

1. **Create Rule:**
   - Go to Rules → Add Rule
   - Name: "Arris SB8200 Upgrade"
   - Match Type: Regex Pattern
   - Pattern: `Arris SB8200`
   - TFTP Server: Your TFTP server IP
   - Firmware File: `arris-sb8200-v2.0.0.bin`
   - Click Create

2. **Trigger Evaluation:**
   - Click "Evaluate Rules" button
   - Wait 30 seconds

3. **Monitor Progress:**
   - Go to Jobs page
   - Watch jobs process
   - Check Activity log for completion

### Task: Test with One Modem First

1. **Find Test Modem MAC:**
   - Go to Modems page
   - Note MAC address (e.g., 00:01:5C:11:22:33)

2. **Create Specific Rule:**
   - Use MAC Range with exact MAC
   - Start MAC: 00:01:5C:11:22:33
   - End MAC: 00:01:5C:11:22:33

3. **Evaluate and Monitor:**
   - Trigger evaluation
   - Watch this single job complete
   - Verify modem upgrades successfully

4. **Expand to All:**
   - Edit rule to full MAC range
   - Evaluate rules again

### Task: Schedule Upgrades for Off-Hours

Currently manual. Best practice:

1. Create rule but leave **Disabled**
2. At scheduled time:
   - Enable the rule
   - Trigger evaluation
3. Monitor progress
4. When complete, disable rule

### Task: Rollback a Failed Upgrade

1. **Create Rollback Rule:**
   - Same match criteria as original
   - Point to previous firmware version
   - Higher priority than original rule

2. **Disable Original Rule**

3. **Trigger Evaluation:**
   - Jobs created for downgrade
   - Monitor completion

### Task: Find Modems Not Upgraded

1. Go to **Modems** page
2. Look at "Firmware" column
3. Sort or search for old version
4. Check if they match any rules
5. Check Activity log for failures

---

## Troubleshooting

### Problem: No Modems Discovered

**Possible Causes:**
- SNMP community string incorrect
- CMTS IP address wrong
- Network connectivity issue
- Firewall blocking SNMP

**Solutions:**
1. Test SNMP access from router:
   ```
   /tool fetch url="snmp://public@192.168.1.1/1.3.6.1.2.1.1.1.0"
   ```
2. Verify CMTS is enabled
3. Check CMTS IP address
4. Try manual discovery
5. Check Activity log for errors

---

### Problem: Jobs Stuck in PENDING

**Possible Causes:**
- All workers busy
- System overloaded
- Database locked

**Solutions:**
1. Check Dashboard for in-progress count
2. Increase workers in Settings
3. Restart application
4. Check system resources

---

### Problem: Jobs Keep Failing

**Possible Causes:**
- TFTP server unreachable
- Firmware file missing/wrong
- Modem not responding
- SNMP write community wrong

**Solutions:**
1. Check error message in job details
2. Verify TFTP server running:
   ```
   curl tftp://192.168.1.50/firmware.bin
   ```
3. Verify firmware file exists on TFTP server
4. Test SNMP write access
5. Check modem signal levels

---

### Problem: Upgrades Too Slow

**Solutions:**
1. Increase workers in Settings
2. Increase max upgrades per CMTS
3. Check network bandwidth
4. Verify TFTP server performance

---

### Problem: Wrong Modems Being Upgraded

**Causes:**
- Rule match criteria too broad
- Multiple rules conflicting

**Solutions:**
1. Review rule match criteria
2. Test rule against single modem first
3. Check rule priority
4. Disable other rules temporarily

---

### Problem: Can't Access Web UI

**Solutions:**
1. Check service is running:
   ```
   curl http://127.0.0.1:8080/api/health
   ```
2. Check firewall rules
3. Verify correct port (default 8080)
4. Try from router itself first

---

## Best Practices

### 1. Test First

Always test with one modem before mass upgrade:
- Create narrow rule (single MAC)
- Verify upgrade works
- Then expand to all

### 2. Monitor Signal Levels

Configure signal level thresholds:
- Min: -15.0 dBmV (or stricter: -10.0)
- Max: +15.0 dBmV (or stricter: +10.0)

Only upgrade modems with good signal.

### 3. Stagger Large Upgrades

For 100+ modems:
- Use priority to upgrade in batches
- Create multiple rules with different priorities
- Monitor each batch

### 4. Keep Activity Log

Don't delete activity log - it's your audit trail:
- Shows when upgrades happened
- Tracks failures
- Documents changes

### 5. Regular Discovery

Run discovery regularly:
- Every 15 minutes minimum
- More frequent for large networks
- Catches new modems quickly

### 6. Backup Database

Backup upgrader.db regularly:
- Contains all configuration
- Tracks all history
- Easy to restore

### 7. Label Rules Clearly

Use descriptive names and descriptions:
- "Arris SB8200 → v2.0.0 (Nov 2024)"
- Include version and date
- Easier to track what's active

---

## Keyboard Shortcuts

- **Refresh Page:** F5 or Ctrl+R
- **Navigate Back:** Alt+← or Ctrl+[
- **Navigate Forward:** Alt+→ or Ctrl+]

---

## Getting Help

### Check Health

Visit: `http://your-router:8080/api/health`

Should return:
```json
{
  "status": "healthy",
  "version": "v0.5.1",
  "database": "connected"
}
```

### Review Logs

Check Activity page for recent events and errors.

### Verify Settings

Go to Settings page and verify all values are reasonable.

### API Documentation

For automation and troubleshooting, see: `API_GUIDE.md`

### MikroTik Deployment

For router-specific issues, see: `MIKROTIK_DEPLOYMENT.md`

---

## Quick Reference

### Common URLs

- Dashboard: `http://router:8080/`
- CMTS Management: `http://router:8080/cmts.html`
- Modems: `http://router:8080/modems.html`
- Rules: `http://router:8080/rules.html`
- Jobs: `http://router:8080/jobs.html`
- Activity: `http://router:8080/activity.html`
- Settings: `http://router:8080/settings.html`
- API Health: `http://router:8080/api/health`

### Default Values

- Port: 8080
- Workers: 4
- Discovery Interval: 900s (15min)
- Evaluation Interval: 1800s (30min)
- Job Timeout: 300s (5min)
- Max Retries: 3
- Signal Range: -15.0 to +15.0 dBmV

---

**Document Version:** 1.0  
**Last Updated:** 2024-11-08  
**For Version:** v0.5.1