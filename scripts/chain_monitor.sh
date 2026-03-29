#!/bin/bash
# Chain Monitor — detects stuck chain and restarts all miners to fix split-DKG deadlock.
# Run via cron every hour: 0 * * * * /usr/local/bin/chain_monitor.sh >> /var/log/0chain/chain_monitor.log 2>&1

LOG="/var/log/0chain/chain_monitor.log"
SHARDER_URL="http://198.18.0.81:7171"

get_round() {
    curl -s "${SHARDER_URL}/v1/chain/get/stats" -m 5 2>/dev/null | \
        python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0"
}

echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') Chain monitor check started"

R1=$(get_round)
sleep 10
R2=$(get_round)

if [ "$R1" = "0" ] || [ "$R2" = "0" ]; then
    echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') WARNING: Could not get chain round (sharder not responding)"
    exit 1
fi

DIFF=$((R2 - R1))

if [ "$DIFF" -le 0 ]; then
    echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') STUCK: Chain at round $R1 (no progress in 10s). Restarting all miners..."
    docker restart miner-1 miner-2 miner-3 miner-4
    sleep 15
    R3=$(get_round)
    echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') After restart: round $R3 (was $R1)"
    if [ "$((R3 - R1))" -gt 0 ]; then
        echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') RECOVERED: Chain advancing again"
    else
        echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') CRITICAL: Chain still stuck after miner restart"
    fi
else
    echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') OK: Chain advancing ($R1 → $R2, +$DIFF blocks/10s)"
fi
