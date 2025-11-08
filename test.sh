#!/bin/bash

echo "Starting server..."
./firmware-upgrader -port 8090 &
SERVER_PID=$!
sleep 2

echo ""
echo "=== Testing API ==="
echo ""

# Test GET CMTS (empty)
echo "1. Getting empty CMTS list..."
curl -s http://localhost:8090/api/cmts | jq '.'

# Test POST CMTS
echo ""
echo "2. Creating test CMTS..."
curl -s -X POST http://localhost:8090/api/cmts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test CMTS",
    "ip_address": "192.168.1.1",
    "snmp_port": 161,
    "community_read": "public",
    "community_write": "private",
    "cm_community_string": "cable-modem",
    "snmp_version": 2,
    "enabled": true
  }' | jq '.'

# Test GET CMTS (with data)
echo ""
echo "3. Getting CMTS list (should have 1 entry)..."
curl -s http://localhost:8090/api/cmts | jq '.'

# Test POST Rule
echo ""
echo "4. Creating test upgrade rule..."
curl -s -X POST http://localhost:8090/api/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Arris SB8200 Upgrade",
    "description": "Upgrade all Arris SB8200 modems",
    "match_type": "MAC_RANGE",
    "match_criteria": {"start_mac": "00:01:5C:00:00:00", "end_mac": "00:01:5C:FF:FF:FF"},
    "tftp_server_ip": "192.168.1.50",
    "firmware_filename": "arris-sb8200-v1.2.3.bin",
    "enabled": true,
    "priority": 100
  }' | jq '.'

# Test GET Rules
echo ""
echo "5. Getting rules list..."
curl -s http://localhost:8090/api/rules | jq '.'

# Test GET Activity Log
echo ""
echo "6. Getting activity log..."
curl -s http://localhost:8090/api/activity-log | jq '.'

echo ""
echo "=== All tests completed! ==="
echo ""
echo "Server is still running on http://localhost:8090"
echo "Press Ctrl+C to stop"
wait $SERVER_PID
