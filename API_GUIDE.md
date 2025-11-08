# Firmware Upgrader - API Guide

**Version:** v0.5.1  
**Base URL:** `http://localhost:8080/api`  
**Date:** 2024-11-08

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Response Format](#response-format)
4. [Error Handling](#error-handling)
5. [CMTS Endpoints](#cmts-endpoints)
6. [Modem Endpoints](#modem-endpoints)
7. [Rule Endpoints](#rule-endpoints)
8. [Job Endpoints](#job-endpoints)
9. [Activity Log Endpoints](#activity-log-endpoints)
10. [Settings Endpoints](#settings-endpoints)
11. [System Endpoints](#system-endpoints)
12. [Trigger Endpoints](#trigger-endpoints)
13. [Examples](#examples)

---

## Overview

The Firmware Upgrader REST API provides programmatic access to all system functionality including CMTS management, modem discovery, upgrade rules, job monitoring, and system configuration.

### API Features

- **RESTful Design** - Standard HTTP methods (GET, POST, PUT, DELETE)
- **JSON Format** - All requests and responses use JSON
- **CORS Enabled** - Cross-origin requests allowed
- **No Authentication** - Currently designed for internal network use
- **Consistent Errors** - Standardized error response format

### Base URL

```
http://<host>:8080/api
```

Default host: `localhost`  
Default port: `8080`

---

## Authentication

**Current Version:** No authentication required.

⚠️ **Security Note:** This application is designed for deployment on internal networks (e.g., MikroTik routers). It should NOT be exposed to the public internet without adding authentication.

**Future Versions:** API key authentication planned for production deployments.

---

## Response Format

### Success Response

```json
{
  "id": 1,
  "name": "Example CMTS",
  "ip_address": "192.168.1.1"
}
```

### List Response

```json
[
  {"id": 1, "name": "CMTS 1"},
  {"id": 2, "name": "CMTS 2"}
]
```

### Empty List

```json
[]
```

### Created Response

```json
{
  "id": 123,
  "message": "Resource created successfully"
}
```

---

## Error Handling

### Error Response Format

```json
{
  "error": "Error message describing what went wrong"
}
```

### HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request succeeded |
| 201 | Created | Resource created successfully |
| 202 | Accepted | Request accepted for async processing |
| 400 | Bad Request | Invalid request body or parameters |
| 404 | Not Found | Resource not found |
| 500 | Internal Server Error | Server error occurred |
| 503 | Service Unavailable | Service unhealthy (database issue) |

### Common Errors

**Invalid JSON:**
```json
{
  "error": "Invalid request body"
}
```

**Resource Not Found:**
```json
{
  "error": "CMTS not found"
}
```

**Validation Error:**
```json
{
  "error": "name is required"
}
```

---

## CMTS Endpoints

### List All CMTS

**GET** `/api/cmts`

Returns all configured CMTS devices.

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "name": "Main CMTS",
    "ip_address": "192.168.1.1",
    "snmp_port": 161,
    "community_read": "public",
    "community_write": "private",
    "cm_community_string": "cable-modem",
    "snmp_version": 2,
    "enabled": true,
    "created_at": "2024-11-08T10:00:00Z",
    "updated_at": "2024-11-08T10:00:00Z"
  }
]
```

---

### Get CMTS by ID

**GET** `/api/cmts/{id}`

Retrieves a specific CMTS by ID.

**Parameters:**
- `id` (path, integer) - CMTS ID

**Response:** `200 OK`
```json
{
  "id": 1,
  "name": "Main CMTS",
  "ip_address": "192.168.1.1",
  "snmp_port": 161,
  "community_read": "public",
  "community_write": "private",
  "cm_community_string": "cable-modem",
  "snmp_version": 2,
  "enabled": true,
  "created_at": "2024-11-08T10:00:00Z",
  "updated_at": "2024-11-08T10:00:00Z"
}
```

**Error:** `404 Not Found`
```json
{
  "error": "CMTS not found"
}
```

---

### Create CMTS

**POST** `/api/cmts`

Creates a new CMTS device.

**Request Body:**
```json
{
  "name": "New CMTS",
  "ip_address": "192.168.1.2",
  "snmp_port": 161,
  "community_read": "public",
  "community_write": "private",
  "cm_community_string": "cable-modem",
  "snmp_version": 2,
  "enabled": true
}
```

**Required Fields:**
- `name` - CMTS name (string)
- `ip_address` - IP address (string)
- `community_read` - SNMP read community (string)
- `snmp_version` - SNMP version: 1, 2, or 3 (integer)

**Optional Fields:**
- `snmp_port` - Default: 161
- `community_write` - Default: empty
- `cm_community_string` - Default: empty
- `enabled` - Default: true

**Response:** `201 Created`
```json
{
  "id": 2,
  "message": "CMTS created successfully"
}
```

**Error:** `400 Bad Request`
```json
{
  "error": "name is required"
}
```

---

### Update CMTS

**PUT** `/api/cmts/{id}`

Updates an existing CMTS.

**Parameters:**
- `id` (path, integer) - CMTS ID

**Request Body:** Same as Create CMTS

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

### Delete CMTS

**DELETE** `/api/cmts/{id}`

Deletes a CMTS and all associated modems.

**Parameters:**
- `id` (path, integer) - CMTS ID

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Error:** `404 Not Found`
```json
{
  "error": "CMTS not found"
}
```

---

## Modem Endpoints

### List Modems

**GET** `/api/modems`

Returns all discovered cable modems.

**Query Parameters:**
- `cmts_id` (optional, integer) - Filter by CMTS ID

**Examples:**
```
GET /api/modems
GET /api/modems?cmts_id=1
```

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "cmts_id": 1,
    "mac_address": "00:01:5C:11:22:33",
    "ip_address": "10.0.0.100",
    "sysdescr": "Arris SB8200 DOCSIS 3.1 Cable Modem",
    "current_firmware": "1.0.0",
    "signal_level": 6.5,
    "status": "online",
    "last_seen": "2024-11-08T10:30:00Z"
  }
]
```

---

### Get Modem by ID

**GET** `/api/modems/{id}`

Retrieves a specific modem by ID.

**Parameters:**
- `id` (path, integer) - Modem ID

**Response:** `200 OK`
```json
{
  "id": 1,
  "cmts_id": 1,
  "mac_address": "00:01:5C:11:22:33",
  "ip_address": "10.0.0.100",
  "sysdescr": "Arris SB8200 DOCSIS 3.1 Cable Modem",
  "current_firmware": "1.0.0",
  "signal_level": 6.5,
  "status": "online",
  "last_seen": "2024-11-08T10:30:00Z"
}
```

---

## Rule Endpoints

### List Rules

**GET** `/api/rules`

Returns all upgrade rules, ordered by priority (descending).

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "name": "Arris SB8200 Upgrade",
    "description": "Upgrade all Arris SB8200 modems to v2.0",
    "match_type": "SYSDESCR_REGEX",
    "match_criteria": "{\"pattern\":\"Arris SB8200\"}",
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "arris-sb8200-v2.0.0.bin",
    "enabled": true,
    "priority": 100,
    "created_at": "2024-11-08T09:00:00Z",
    "updated_at": "2024-11-08T09:00:00Z"
  }
]
```

---

### Get Rule by ID

**GET** `/api/rules/{id}`

Retrieves a specific rule by ID.

**Parameters:**
- `id` (path, integer) - Rule ID

**Response:** `200 OK` (same format as list item)

---

### Create Rule

**POST** `/api/rules`

Creates a new upgrade rule.

**Request Body:**
```json
{
  "name": "Motorola MB8600 Upgrade",
  "description": "Upgrade Motorola MB8600 modems",
  "match_type": "SYSDESCR_REGEX",
  "match_criteria": "{\"pattern\":\"Motorola MB8600\"}",
  "tftp_server_ip": "192.168.1.50",
  "firmware_filename": "motorola-mb8600-v3.0.1.bin",
  "enabled": true,
  "priority": 90
}
```

**Match Types:**
- `MAC_RANGE` - Match by MAC address range
- `SYSDESCR_REGEX` - Match by system description regex

**Match Criteria Examples:**

MAC Range:
```json
{
  "match_criteria": "{\"start_mac\":\"00:01:5C:00:00:00\",\"end_mac\":\"00:01:5C:FF:FF:FF\"}"
}
```

Regex Pattern:
```json
{
  "match_criteria": "{\"pattern\":\"Arris SB8200\"}"
}
```

**Required Fields:**
- `name` - Rule name
- `match_type` - "MAC_RANGE" or "SYSDESCR_REGEX"
- `match_criteria` - JSON string with criteria
- `tftp_server_ip` - TFTP server IP address
- `firmware_filename` - Firmware file name

**Optional Fields:**
- `description` - Rule description
- `enabled` - Default: true
- `priority` - Default: 0 (higher = evaluated first)

**Response:** `201 Created`
```json
{
  "id": 2,
  "message": "Rule created successfully"
}
```

**Error:** `400 Bad Request`
```json
{
  "error": "match_type must be MAC_RANGE or SYSDESCR_REGEX"
}
```

---

### Update Rule

**PUT** `/api/rules/{id}`

Updates an existing rule.

**Parameters:**
- `id` (path, integer) - Rule ID

**Request Body:** Same as Create Rule

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

### Delete Rule

**DELETE** `/api/rules/{id}`

Deletes an upgrade rule.

**Parameters:**
- `id` (path, integer) - Rule ID

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

## Job Endpoints

### List Jobs

**GET** `/api/jobs`

Returns upgrade jobs with optional filtering.

**Query Parameters:**
- `status` (optional) - Filter by status: PENDING, IN_PROGRESS, COMPLETED, FAILED, SKIPPED
- `limit` (optional, integer) - Limit results (default: 100)

**Examples:**
```
GET /api/jobs
GET /api/jobs?status=PENDING
GET /api/jobs?status=FAILED&limit=50
```

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "modem_id": 1,
    "rule_id": 1,
    "cmts_id": 1,
    "mac_address": "00:01:5C:11:22:33",
    "status": "COMPLETED",
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "arris-sb8200-v2.0.0.bin",
    "retry_count": 0,
    "max_retries": 3,
    "error_message": null,
    "created_at": "2024-11-08T10:00:00Z",
    "started_at": "2024-11-08T10:01:00Z",
    "completed_at": "2024-11-08T10:05:00Z"
  }
]
```

**Job Statuses:**
- `PENDING` - Waiting to be processed
- `IN_PROGRESS` - Currently being processed
- `COMPLETED` - Successfully completed
- `FAILED` - Failed after all retries
- `SKIPPED` - Skipped due to conditions

---

### Get Job by ID

**GET** `/api/jobs/{id}`

Retrieves a specific job by ID.

**Parameters:**
- `id` (path, integer) - Job ID

**Response:** `200 OK` (same format as list item)

---

### Retry Job

**POST** `/api/jobs/{id}/retry`

Retries a failed job by resetting it to PENDING status.

**Parameters:**
- `id` (path, integer) - Job ID

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Note:** Only failed or completed jobs can be retried.

---

## Activity Log Endpoints

### List Activity Logs

**GET** `/api/activity-log`

Returns system activity logs.

**Query Parameters:**
- `limit` (optional, integer) - Limit results (default: 50)
- `offset` (optional, integer) - Offset for pagination (default: 0)

**Examples:**
```
GET /api/activity-log
GET /api/activity-log?limit=100
GET /api/activity-log?limit=50&offset=50
```

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "event_type": "UPGRADE_COMPLETED",
    "entity_type": "job",
    "entity_id": 1,
    "message": "Firmware upgrade completed for modem 00:01:5C:11:22:33",
    "details": "{\"duration\":\"5m\"}",
    "created_at": "2024-11-08T10:05:00Z"
  }
]
```

**Event Types:**
- `MODEM_DISCOVERED` - Modem discovered via SNMP
- `MODEM_LOST` - Modem no longer responding
- `UPGRADE_STARTED` - Firmware upgrade started
- `UPGRADE_COMPLETED` - Firmware upgrade completed
- `UPGRADE_FAILED` - Firmware upgrade failed
- `RULE_CREATED` - Upgrade rule created
- `RULE_UPDATED` - Upgrade rule updated
- `RULE_DELETED` - Upgrade rule deleted
- `CMTS_ADDED` - CMTS added to system
- `CMTS_UPDATED` - CMTS updated
- `CMTS_DELETED` - CMTS deleted
- `SYSTEM_EVENT` - General system event

---

## Settings Endpoints

### List All Settings

**GET** `/api/settings`

Returns all system settings as key-value pairs.

**Response:** `200 OK`
```json
{
  "workers": "4",
  "poll_interval": "30",
  "discovery_interval": "900",
  "evaluation_interval": "1800",
  "job_timeout": "300",
  "retry_attempts": "3",
  "signal_level_min": "-15.0",
  "signal_level_max": "15.0",
  "max_upgrades_per_cmts": "10"
}
```

**Available Settings:**

| Setting | Description | Default | Unit |
|---------|-------------|---------|------|
| workers | Number of job worker threads | 4 | count |
| poll_interval | Job queue polling interval | 30 | seconds |
| discovery_interval | Auto-discovery interval | 900 | seconds |
| evaluation_interval | Rule evaluation interval | 1800 | seconds |
| job_timeout | Job timeout | 300 | seconds |
| retry_attempts | Max retry attempts | 3 | count |
| signal_level_min | Min acceptable signal level | -15.0 | dBmV |
| signal_level_max | Max acceptable signal level | 15.0 | dBmV |
| max_upgrades_per_cmts | Max concurrent upgrades per CMTS | 10 | count |

---

### Get Setting by Key

**GET** `/api/settings/{key}`

Retrieves a specific setting value.

**Parameters:**
- `key` (path, string) - Setting key

**Example:**
```
GET /api/settings/workers
```

**Response:** `200 OK`
```json
{
  "key": "workers",
  "value": "4"
}
```

**Error:** `404 Not Found`
```json
{
  "error": "setting not found: invalid_key"
}
```

---

### Update Setting

**PUT** `/api/settings/{key}`

Updates a setting value.

**Parameters:**
- `key` (path, string) - Setting key

**Request Body:**
```json
{
  "value": "8"
}
```

**Example:**
```
PUT /api/settings/workers
{
  "value": "8"
}
```

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Note:** Some settings require application restart to take effect (workers, poll_interval).

---

## System Endpoints

### Health Check

**GET** `/api/health`

Returns service health status and basic info.

**Response:** `200 OK` (Healthy)
```json
{
  "status": "healthy",
  "version": "v0.5.1",
  "database": "connected",
  "total_cmts": 3
}
```

**Response:** `503 Service Unavailable` (Unhealthy)
```json
{
  "status": "unhealthy",
  "error": "database connection failed",
  "details": "connection refused"
}
```

**Use Case:** Load balancer health checks, monitoring systems.

---

### System Metrics

**GET** `/api/metrics`

Returns comprehensive system metrics.

**Response:** `200 OK`
```json
{
  "cmts": {
    "total": 3,
    "enabled": 2
  },
  "modems": {
    "total": 150
  },
  "rules": {
    "total": 5,
    "enabled": 4
  },
  "jobs": {
    "total": 1234,
    "pending": 5,
    "in_progress": 2,
    "completed": 1200,
    "failed": 27
  }
}
```

**Use Case:** Monitoring dashboards, alerting systems, capacity planning.

---

### Dashboard Summary

**GET** `/api/dashboard`

Returns aggregated data for dashboard display (optimized single call).

**Response:** `200 OK`
```json
{
  "total_cmts": 3,
  "enabled_cmts": 2,
  "total_modems": 150,
  "total_rules": 5,
  "enabled_rules": 4,
  "pending_jobs": 5,
  "in_progress_jobs": 2,
  "recent_activity": [
    {
      "id": 100,
      "event_type": "UPGRADE_COMPLETED",
      "message": "Firmware upgrade completed",
      "created_at": "2024-11-08T10:05:00Z"
    }
  ]
}
```

**Use Case:** Web UI dashboard, mobile apps, status displays.

---

## Trigger Endpoints

### Trigger Discovery for All CMTS

**POST** `/api/discovery/trigger`

Manually triggers modem discovery for all enabled CMTS devices.

**Request Body:** None

**Response:** `202 Accepted`
```json
{
  "message": "Discovery started for all enabled CMTS",
  "cmts_triggered": 2
}
```

**Note:** Discovery runs asynchronously. Check activity log for results.

---

### Trigger Discovery for Specific CMTS

**POST** `/api/cmts/{id}/discover`

Manually triggers modem discovery for a specific CMTS.

**Parameters:**
- `id` (path, integer) - CMTS ID

**Request Body:** None

**Response:** `202 Accepted`
```json
{
  "message": "Discovery started for CMTS"
}
```

**Note:** Discovery runs asynchronously in background.

---

### Trigger Rule Evaluation

**POST** `/api/rules/evaluate`

Manually triggers rule evaluation against all discovered modems.

**Request Body:** None

**Response:** `202 Accepted`
```json
{
  "message": "Rule evaluation started"
}
```

**Note:** Evaluation runs asynchronously. New jobs will be created for eligible modems.

---

## Examples

### Example 1: Add New CMTS and Discover Modems

```bash
# 1. Create CMTS
curl -X POST http://localhost:8080/api/cmts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Lab CMTS",
    "ip_address": "192.168.1.1",
    "snmp_port": 161,
    "community_read": "public",
    "community_write": "private",
    "cm_community_string": "cable-modem",
    "snmp_version": 2,
    "enabled": true
  }'

# Response: {"id": 1, "message": "CMTS created successfully"}

# 2. Trigger discovery
curl -X POST http://localhost:8080/api/cmts/1/discover

# Response: {"message": "Discovery started for CMTS"}

# 3. Check discovered modems (wait a minute)
curl http://localhost:8080/api/modems?cmts_id=1
```

---

### Example 2: Create Upgrade Rule and Trigger Evaluation

```bash
# 1. Create upgrade rule
curl -X POST http://localhost:8080/api/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Arris SB8200 Upgrade",
    "description": "Upgrade all Arris SB8200 to v2.0",
    "match_type": "SYSDESCR_REGEX",
    "match_criteria": "{\"pattern\":\"Arris SB8200\"}",
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "arris-sb8200-v2.0.0.bin",
    "enabled": true,
    "priority": 100
  }'

# Response: {"id": 1, "message": "Rule created successfully"}

# 2. Trigger rule evaluation
curl -X POST http://localhost:8080/api/rules/evaluate

# Response: {"message": "Rule evaluation started"}

# 3. Check created jobs
curl http://localhost:8080/api/jobs?status=PENDING
```

---

### Example 3: Monitor Job Progress

```bash
# 1. List pending jobs
curl http://localhost:8080/api/jobs?status=PENDING

# 2. Get specific job details
curl http://localhost:8080/api/jobs/123

# 3. Check activity log for updates
curl http://localhost:8080/api/activity-log?limit=20

# 4. If job failed, retry it
curl -X POST http://localhost:8080/api/jobs/123/retry
```

---

### Example 4: Update System Settings

```bash
# 1. Get current settings
curl http://localhost:8080/api/settings

# 2. Update worker count
curl -X PUT http://localhost:8080/api/settings/workers \
  -H "Content-Type: application/json" \
  -d '{"value": "8"}'

# 3. Update discovery interval (15 minutes)
curl -X PUT http://localhost:8080/api/settings/discovery_interval \
  -H "Content-Type: application/json" \
  -d '{"value": "900"}'

# Note: Restart application for some settings to take effect
```

---

### Example 5: System Monitoring

```bash
# Check health
curl http://localhost:8080/api/health

# Get metrics
curl http://localhost:8080/api/metrics

# Get dashboard data
curl http://localhost:8080/api/dashboard
```

---

### Example 6: Python Integration

```python
import requests

BASE_URL = "http://localhost:8080/api"

# Create CMTS
cmts_data = {
    "name": "Production CMTS",
    "ip_address": "10.0.1.1",
    "snmp_port": 161,
    "community_read": "public",
    "snmp_version": 2,
    "enabled": True
}

response = requests.post(f"{BASE_URL}/cmts", json=cmts_data)
cmts_id = response.json()["id"]
print(f"Created CMTS ID: {cmts_id}")

# Trigger discovery
requests.post(f"{BASE_URL}/cmts/{cmts_id}/discover")
print("Discovery triggered")

# Wait and check modems
import time
time.sleep(60)

modems = requests.get(f"{BASE_URL}/modems?cmts_id={cmts_id}").json()
print(f"Discovered {len(modems)} modems")

# Create upgrade rule
rule_data = {
    "name": "Auto Upgrade",
    "match_type": "MAC_RANGE",
    "match_criteria": '{"start_mac":"00:01:5C:00:00:00","end_mac":"00:01:5C:FF:FF:FF"}',
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "firmware-v2.0.bin",
    "enabled": True,
    "priority": 100
}

response = requests.post(f"{BASE_URL}/rules", json=rule_data)
rule_id = response.json()["id"]
print(f"Created Rule ID: {rule_id}")

# Evaluate rules
requests.post(f"{BASE_URL}/rules/evaluate")
print("Rule evaluation triggered")

# Monitor jobs
jobs = requests.get(f"{BASE_URL}/jobs?status=PENDING").json()
print(f"Pending jobs: {len(jobs)}")
```

---

### Example 7: Bash Monitoring Script

```bash
#!/bin/bash

API="http://localhost:8080/api"

# Function to get metrics
get_metrics() {
    curl -s "${API}/metrics" | jq '.'
}

# Function to check failed jobs
check_failed_jobs() {
    FAILED=$(curl -s "${API}/jobs?status=FAILED" | jq '. | length')
    if [ "$FAILED" -gt 0 ]; then
        echo "WARNING: $FAILED failed jobs"
        curl -s "${API}/jobs?status=FAILED" | jq '.[] | {id, mac_address, error_message}'
    fi
}

# Function to check health
check_health() {
    STATUS=$(curl -s "${API}/health" | jq -r '.status')
    if [ "$STATUS" != "healthy" ]; then
        echo "ERROR: Service unhealthy"
        exit 1
    fi
}

# Main monitoring loop
while true; do
    echo "=== $(date) ==="
    check_health
    get_metrics
    check_failed_jobs
    echo ""
    sleep 300  # Check every 5 minutes
done
```

---

## Rate Limiting

**Current Version:** No rate limiting.

**Recommendations:**
- Use reverse proxy (nginx, traefik) for rate limiting if needed
- Monitor API usage with metrics endpoint
- Set up alerts for abnormal request patterns

---

## Best Practices

### 1. Error Handling

Always check HTTP status codes and handle errors gracefully:

```javascript
fetch('/api/cmts')
  .then(response => {
    if (!response.ok) {
      return response.json().then(err => {
        throw new Error(err.error);
      });
    }
    return response.json();
  })
  .then(data => console.log(data))
  .catch(error => console.error('API Error:', error));
```

### 2. Polling

For async operations (discovery, rule evaluation), poll the relevant endpoints:

```javascript
async function waitForDiscovery(cmtsId) {
  // Trigger discovery
  await fetch(`/api/cmts/${cmtsId}/discover`, {method: 'POST'});
  
  // Poll activity log
  let attempts = 0;
  while (attempts < 30) {
    const logs = await fetch('/api/activity-log?limit=10').then(r => r.json());
    const discoveryComplete = logs.find(log => 
      log.event_type === 'MODEM_DISCOVERED' && 
      log.entity_id === cmtsId
    );
    
    if (discoveryComplete) {
      return true;
    }
    
    await new Promise(resolve => setTimeout(resolve, 2000));
    attempts++;
  }
  
  throw new Error('Discovery timeout');
}
```

### 3. Batch Operations

Use dashboard endpoint instead of multiple individual calls:

```javascript
// ❌ Bad - Multiple calls
const cmts = await fetch('/api/cmts').then(r => r.json());
const modems = await fetch('/api/modems').then(r => r.json());
const jobs = await fetch('/api/jobs').then(r => r.json());
const activity = await fetch('/api/activity-log').then(r => r.json());

// ✅ Good - Single call
const dashboard = await fetch('/api/dashboard').then(r => r.json());
```

### 4. Settings Updates

Batch settings updates and restart once:

```bash
# Update multiple settings
curl -X PUT http://localhost:8080/api/settings/workers -d '{"value":"8"}'
curl -X PUT http://localhost:8080/api/settings/poll_interval -d '{"value":"60"}'

# Restart application once
systemctl restart firmware-upgrader
```

---

## Troubleshooting

### Problem: Connection Refused

**Cause:** Service not running or wrong port.

**Solution:**
```bash
# Check if service is running
ps aux | grep firmware-upgrader

# Check port
netstat -tlnp | grep 8080

# Start service
./firmware-upgrader -port 8080
```

---

### Problem: 503 Service Unavailable

**Cause:** Database connection issue.

**Solution:**
```bash
# Check database file
ls -lh upgrader.db

# Check permissions
chmod 644 upgrader.db

# Check disk space
df -h
```

---

### Problem: Slow Response Times

**Cause:** Large dataset or slow queries.

**Solution:**