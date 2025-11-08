# Firmware Upgrader - MikroTik Deployment Guide

**Version:** v0.5.1  
**Target Platform:** MikroTik RouterOS  
**Date:** 2024-11-08

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Installation Methods](#installation-methods)
4. [Step-by-Step Deployment](#step-by-step-deployment)
5. [Configuration](#configuration)
6. [Running the Service](#running-the-service)
7. [Verification](#verification)
8. [Monitoring](#monitoring)
9. [Troubleshooting](#troubleshooting)
10. [Updating](#updating)
11. [Uninstallation](#uninstallation)

---

## Overview

The Firmware Upgrader can be deployed directly on MikroTik routers running RouterOS. This guide covers installation, configuration, and operation on MikroTik hardware.

### Why Deploy on MikroTik?

- **Network Position** - Already at the edge of your cable network
- **Always On** - Reliable 24/7 operation
- **Low Resource** - Binary uses minimal CPU/memory
- **No Extra Hardware** - Uses existing infrastructure
- **Fast Access** - Direct network access to CMTS devices

### System Requirements

**Minimum:**
- MikroTik RouterOS 6.40+ or RouterOS 7.x
- ARM, MIPS, or x86 architecture
- 64 MB free storage
- 32 MB free RAM
- Network connectivity to CMTS

**Recommended:**
- RouterOS 7.x
- 128 MB free storage (for logs)
- 64 MB free RAM
- Static IP address

---

## Prerequisites

### 1. Check RouterOS Version

```bash
/system resource print
```

Expected output:
```
version: 7.11.2
architecture-name: arm64
cpu: ARM
```

### 2. Check Available Storage

```bash
/system resource print
```

Look for:
```
free-hdd-space: 256.0MiB
```

Minimum: 64 MB free

### 3. Check Network Connectivity

Test connectivity to your CMTS:

```bash
/ping 192.168.1.1 count=5
```

All packets should succeed.

### 4. Enable SSH (for file transfer)

```bash
/ip service enable ssh
/ip service set ssh port=22
```

---

## Installation Methods

### Method 1: Direct Download (Recommended)

Download directly to the router using built-in tools.

### Method 2: SCP Upload

Upload from your computer using SCP.

### Method 3: USB Transfer

Copy binary to USB drive and mount on router.

---

## Step-by-Step Deployment

### Step 1: Create Directory Structure

Connect to your MikroTik via SSH or Terminal:

```bash
# Create application directory
/file mkdir firmware-upgrader

# Verify creation
/file print where name~"firmware-upgrader"
```

### Step 2: Upload Binary

**Option A: From Linux/Mac Computer**

```bash
# Build for ARM (most common)
GOOS=linux GOARCH=arm64 go build -o firmware-upgrader ./cmd/firmware-upgrader

# Or for MIPS
GOOS=linux GOARCH=mips go build -o firmware-upgrader ./cmd/firmware-upgrader

# Upload via SCP
scp firmware-upgrader admin@192.168.88.1:/firmware-upgrader/
```

**Option B: From Windows Computer**

```powershell
# Use WinSCP or SCP client
# Upload to: /firmware-upgrader/firmware-upgrader
```

**Option C: Using RouterOS FTP**

```bash
# On MikroTik, enable FTP temporarily
/ip service enable ftp

# From your computer
ftp 192.168.88.1
> binary
> cd /firmware-upgrader
> put firmware-upgrader
> quit

# Disable FTP after upload
/ip service disable ftp
```

### Step 3: Set Execute Permissions

```bash
# Make binary executable (RouterOS v7+)
/system script add name="set-firmware-upgrader-exec" source="/system/script/run name=set-firmware-upgrader-exec"

# Note: On RouterOS 6.x, binaries may need to be in /flash/bin/
```

### Step 4: Test Binary

```bash
# Navigate to directory
cd /firmware-upgrader

# Test execution
./firmware-upgrader -help
```

Expected output:
```
Usage of ./firmware-upgrader:
  -port int
        HTTP server port (default 8080)
  -workers int
        Number of worker threads (default 4)
```

---

## Configuration

### Default Configuration

The application uses database-backed settings (no config file needed).

**Default Settings:**
- Port: 8080
- Workers: 4
- Database: `/firmware-upgrader/upgrader.db`
- Web UI: `/firmware-upgrader/web`

### Command-Line Options

```bash
./firmware-upgrader [options]

Options:
  -port int         HTTP server port (default: 8080)
  -workers int      Number of workers (default: 4)
  -loglevel string  Log level: debug, info, warn, error (default: info)
```

### Environment Variables

```bash
# Set via RouterOS (optional)
export FW_PORT=8080
export FW_WORKERS=4
export FW_LOG_LEVEL=info
```

### Firewall Configuration

Allow access to the web interface:

```bash
# Allow HTTP access from LAN
/ip firewall filter add chain=input protocol=tcp dst-port=8080 \
  src-address=192.168.88.0/24 action=accept comment="Firmware Upgrader Web UI"

# Or allow from specific IP only
/ip firewall filter add chain=input protocol=tcp dst-port=8080 \
  src-address=192.168.88.100 action=accept comment="Admin PC only"
```

---

## Running the Service

### Method 1: Manual Start (Testing)

```bash
# Start in foreground (for testing)
cd /firmware-upgrader
./firmware-upgrader -port 8080
```

Test by accessing: `http://192.168.88.1:8080`

Press `Ctrl+C` to stop.

### Method 2: Background Process (Production)

**Create Startup Script:**

```bash
/system script add name="firmware-upgrader-start" source={
  :execute script="/firmware-upgrader/firmware-upgrader -port 8080" file=firmware-upgrader.log
}
```

**Run Script:**

```bash
/system script run firmware-upgrader-start
```

**Check Logs:**

```bash
/file print where name="firmware-upgrader.log"
/file get firmware-upgrader.log contents
```

### Method 3: System Scheduler (Auto-start)

**Create scheduler to start on boot:**

```bash
/system scheduler add name="firmware-upgrader-autostart" \
  on-event="/firmware-upgrader/firmware-upgrader -port 8080 &" \
  start-time=startup \
  interval=0 \
  comment="Auto-start Firmware Upgrader"
```

**Verify scheduler:**

```bash
/system scheduler print
```

### Method 4: Container (RouterOS 7.4+)

If your MikroTik supports containers:

```bash
# Enable container mode
/system/device-mode/update container=yes

# Reboot required
/system reboot

# After reboot, create container
/container add remote-image=<your-docker-hub>/firmware-upgrader:latest \
  interface=veth1 root-dir=/firmware-upgrader/container \
  hostname=firmware-upgrader
```

---

## Verification

### Check if Process is Running

```bash
# List running processes
/system resource monitor-traffic
```

Or check network listeners:

```bash
# On RouterOS 7+
/ip service print

# Look for port 8080 listener
```

### Test Web Interface

From your computer:

```bash
# Test HTTP access
curl http://192.168.88.1:8080/api/health

# Expected response:
# {"status":"healthy","version":"v0.5.1","database":"connected","total_cmts":0}
```

Or open in browser: `http://192.168.88.1:8080`

### Check Database

```bash
# Verify database file was created
/file print where name~"upgrader.db"
```

Expected output:
```
name: firmware-upgrader/upgrader.db
type: file
size: 49152
```

### Check Logs

```bash
# View application logs
/log print where topics~"firmware"
```

---

## Monitoring

### RouterOS Resource Monitoring

```bash
# Check CPU and memory usage
/system resource print

# Monitor specific process (if visible)
/system resource monitor-traffic
```

### Application Health Check

```bash
# Create scheduled health check
/system scheduler add name="health-check" \
  on-event="/tool fetch url=http://127.0.0.1:8080/api/health" \
  interval=5m \
  comment="Check Firmware Upgrader health"
```

### Log Monitoring

```bash
# Create script to check for errors
/system script add name="check-upgrader-errors" source={
  :local content [/file get firmware-upgrader.log contents]
  :if ([:find $content "ERROR"] >= 0) do={
    :log error "Firmware Upgrader has errors - check logs"
  }
}

# Run every hour
/system scheduler add name="error-check" \
  on-event="/system script run check-upgrader-errors" \
  interval=1h
```

### Disk Space Monitoring

```bash
# Alert if disk space low
/system script add name="check-disk-space" source={
  :local freespace [/system resource get free-hdd-space]
  :if ($freespace < 50000000) do={
    :log warning "Low disk space: $freespace bytes remaining"
  }
}

/system scheduler add name="disk-space-check" \
  on-event="/system script run check-disk-space" \
  interval=1d
```

---

## Troubleshooting

### Problem: Binary Won't Execute

**Symptoms:**
```
bad command name firmware-upgrader
```

**Solutions:**

1. Check architecture compatibility:
```bash
/system resource print
# Verify architecture matches your binary build
```

2. Re-download correct binary for your architecture:
   - ARM: `GOARCH=arm` or `GOARCH=arm64`
   - MIPS: `GOARCH=mips` or `GOARCH=mipsle`
   - x86: `GOARCH=386` or `GOARCH=amd64`

3. Check file permissions (RouterOS 7+)

---

### Problem: Port Already in Use

**Symptoms:**
```
listen tcp :8080: bind: address already in use
```

**Solutions:**

1. Check what's using the port:
```bash
/ip service print
```

2. Use different port:
```bash
./firmware-upgrader -port 8081
```

3. Stop conflicting service if found

---

### Problem: Out of Disk Space

**Symptoms:**
```
ERROR: failed to write database
```

**Solutions:**

1. Check available space:
```bash
/system resource print
```

2. Clean up old files:
```bash
/file print
/file remove firmware-upgrader.log
```

3. Rotate logs:
```bash
# Create log rotation script
/system script add name="rotate-logs" source={
  :if ([/file get firmware-upgrader.log size] > 10000000) do={
    /file remove firmware-upgrader.log
  }
}
```

---

### Problem: Can't Access Web UI

**Symptoms:**
- Browser shows "Connection refused"
- Timeout accessing `http://192.168.88.1:8080`

**Solutions:**

1. Check if service is running:
```bash
/tool fetch url=http://127.0.0.1:8080/api/health
```

2. Check firewall rules:
```bash
/ip firewall filter print where dst-port=8080
```

3. Verify port is correct:
```bash
# List all network listeners
netstat -tlnp | grep 8080  # If netstat available
```

4. Try from router itself:
```bash
/tool fetch url=http://127.0.0.1:8080/api/health mode=http
```

---

### Problem: High Memory Usage

**Symptoms:**
- Router becomes slow
- Memory usage high

**Solutions:**

1. Check memory usage:
```bash
/system resource print
```

2. Reduce worker threads:
```bash
./firmware-upgrader -workers 2
```

3. Restart service:
```bash
# Kill process
/system script run "kill-upgrader"

# Restart
/system script run "firmware-upgrader-start"
```

---

### Problem: Database Corruption

**Symptoms:**
```
ERROR: database disk image is malformed
```

**Solutions:**

1. Stop service

2. Backup database:
```bash
/file copy firmware-upgrader/upgrader.db firmware-upgrader/upgrader.db.backup
```

3. Remove corrupted database:
```bash
/file remove firmware-upgrader/upgrader.db
```

4. Restart service (will create new database):
```bash
/system script run "firmware-upgrader-start"
```

---

## Updating

### Step 1: Stop Service

```bash
# Find and kill process (method varies by RouterOS version)
# On newer versions:
/system process print
/system process kill [find name="firmware-upgrader"]
```

### Step 2: Backup Database

```bash
/file copy firmware-upgrader/upgrader.db firmware-upgrader/upgrader.db.backup
```

### Step 3: Upload New Binary

```bash
# From your computer
scp firmware-upgrader-new admin@192.168.88.1:/firmware-upgrader/firmware-upgrader.new
```

### Step 4: Replace Binary

```bash
# On MikroTik
/file remove firmware-upgrader/firmware-upgrader
/file move firmware-upgrader/firmware-upgrader.new firmware-upgrader/firmware-upgrader
```

### Step 5: Restart Service

```bash
/system script run "firmware-upgrader-start"
```

### Step 6: Verify

```bash
# Check version
/tool fetch url=http://127.0.0.1:8080/api/health mode=http
```

---

## Uninstallation

### Step 1: Stop Service

```bash
# Kill process
/system process print
/system process kill [find name="firmware-upgrader"]
```

### Step 2: Remove Scheduler

```bash
/system scheduler remove "firmware-upgrader-autostart"
/system scheduler remove "health-check"
```

### Step 3: Remove Scripts

```bash
/system script remove "firmware-upgrader-start"
/system script remove "check-upgrader-errors"
```

### Step 4: Remove Files

```bash
/file remove firmware-upgrader/firmware-upgrader
/file remove firmware-upgrader/upgrader.db
/file remove firmware-upgrader.log
```

### Step 5: Remove Firewall Rules

```bash
/ip firewall filter remove [find comment="Firmware Upgrader Web UI"]
```

### Step 6: Remove Directory

```bash
/file remove firmware-upgrader
```

---

## Performance Tuning

### For Low-End Devices (< 64 MB RAM)

```bash
# Minimal configuration
./firmware-upgrader -workers 2 -loglevel warn
```

Update settings via API:
```bash
curl -X PUT http://192.168.88.1:8080/api/settings/poll_interval \
  -H "Content-Type: application/json" \
  -d '{"value": "60"}'
```

### For High-End Devices (> 256 MB RAM)

```bash
# Optimal configuration
./firmware-upgrader -workers 8 -loglevel info
```

Update settings:
```bash
curl -X PUT http://192.168.88.1:8080/api/settings/workers \
  -d '{"value": "8"}'
```

---

## Security Best Practices

### 1. Limit Web UI Access

```bash
# Only allow from management VLAN
/ip firewall filter add chain=input protocol=tcp dst-port=8080 \
  src-address=192.168.10.0/24 action=accept

# Drop all other access
/ip firewall filter add chain=input protocol=tcp dst-port=8080 action=drop
```

### 2. Use Strong SNMP Communities

Configure unique SNMP community strings in the application, not default "public/private".

### 3. Regular Backups

```bash
# Backup script
/system script add name="backup-upgrader-db" source={
  :local date [/system clock get date]
  :local time [/system clock get time]
  /file copy firmware-upgrader/upgrader.db "firmware-upgrader/backup-$date-$time.db"
}

# Run daily
/system scheduler add name="daily-backup" \
  on-event="/system script run backup-upgrader-db" \
  interval=1d start-time=02:00:00
```

### 4. Monitor Logs

Enable log forwarding to syslog server:

```bash
/system logging action add name=remote remote=192.168.1.100:514 target=remote
/system logging add topics=error,warning action=remote
```

---

## Integration with MikroTik Features

### SNMP Traps

Configure router to send SNMP traps to upgrader:

```bash
/snmp community add name=public addresses=192.168.88.1/32
/snmp set enabled=yes trap-version=2 trap-community=public
```

### Scripts Integration

Trigger upgrades from MikroTik scripts:

```bash
/system script add name="trigger-modem-discovery" source={
  /tool fetch url="http://127.0.0.1:8080/api/discovery/trigger" \
    mode=http method=post
}
```

### DHCP Integration

Log when modems get new IPs:

```bash
/ip dhcp-server alert add on-alert={
  :local mac [/ip dhcp-server lease get $leaseActIP mac-address]
  :log info "Modem DHCP: $mac got IP $leaseActIP"
}
```

---

## Example: Complete Setup Script

Save as `setup-firmware-upgrader.rsc`:

```
# Firmware Upgrader - Complete Setup Script
# Usage: /import setup-firmware-upgrader.rsc

:log info "Starting Firmware Upgrader setup..."

# Create directory
/file mkdir firmware-upgrader

# Add firewall rule
/ip firewall filter add chain=input protocol=tcp dst-port=8080 \
  src-address=192.168.88.0/24 action=accept \
  comment="Firmware Upgrader Web UI" place-before=0

# Create start script
/system script add name="firmware-upgrader-start" source={
  :log info "Starting Firmware Upgrader service..."
  :execute script="/firmware-upgrader/firmware-upgrader -port 8080" file=firmware-upgrader.log
}

# Create auto-start scheduler
/system scheduler add name="firmware-upgrader-autostart" \
  on-event="/system script run firmware-upgrader-start" \
  start-time=startup interval=0

# Create backup script
/system script add name="backup-upgrader-db" source={
  :local date [/system clock get date]
  /file copy firmware-upgrader/upgrader.db "firmware-upgrader/backup-$date.db"
}

# Daily backup schedule
/system scheduler add name="daily-backup" \
  on-event="/system script run backup-upgrader-db" \
  interval=1d start-time=02:00:00

:log info "Firmware Upgrader setup complete!"
:log info "Next steps:"
:log info "1. Upload binary to /firmware-upgrader/"
:log info "2. Run: /system script run firmware-upgrader-start"
:log info "3. Access: http://192.168.88.1:8080"
```

---

## Support

### Getting Help

1. Check logs: `/file get firmware-upgrader.log contents`
2. Check API health: `http://192.168.88.1:8080/api/health`
3. Review this guide
4. Check project documentation

### Useful Commands Reference

```bash
# Check if running
/tool fetch url=http://127.0.0.1:8080/api/health mode=http

# View logs
/file get firmware-upgrader.log contents

# Check memory
/system resource print

# Check storage
/file print detail where name~"firmware-upgrader"

# Restart service
/system script run firmware-upgrader-start

# Remove old logs
/file remove firmware-upgrader.log
```

---

## Conclusion

You now have a complete guide for deploying the Firmware Upgrader on MikroTik RouterOS. The application runs efficiently on MikroTik hardware and provides a powerful solution for managing cable modem firmware upgrades directly from your edge network device.

**Next Steps:**
1. Complete this deployment guide
2. Access web UI to configure CMTS
3. Read the [User Guide](USER_GUIDE.md) for daily operations
4. Review [API Guide](API_GUIDE.md) for automation

---

**Document Version:** 1.0  
**Last Updated:** 2024-11-08  
**Tested On:** RouterOS 7.11.2 (ARM64), RouterOS 6.49.7 (MIPS)