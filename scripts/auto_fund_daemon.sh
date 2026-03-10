#!/bin/bash
# Auto-funding and staking daemon for 0Chain test network service providers
# - Funds blobbers/validators/miners/sharders/assigner wallets when balance is low
# - Fixes circular dependency: funds blobbers from containers BEFORE they register on chain
# - Stakes blobbers and validators that are registered but have no stake yet
# Uses nonce management for reliable transactions (like vc.sh)
# Detects chain health and pauses when chain is down
#
# Usage: ./auto_fund_daemon.sh [interval_seconds] [min_balance_zcn] [topup_amount_zcn]
#        ./auto_fund_daemon.sh --stop     # Stop daemon
#        ./auto_fund_daemon.sh --status   # Check status
#        ./auto_fund_daemon.sh --once     # Single cycle

set -o pipefail

INTERVAL=${1:-120}          # Check every 2 minutes
MIN_BALANCE=${2:-5}         # Minimum balance in ZCN before top-up
TOP_UP=${3:-20}             # Amount to send when low

ZWALLET="${ZWALLET:-/root/Code/zwalletcli/zwallet}"
ZBOX="${ZBOX:-/root/Code/zboxcli/zbox}"
CONFIG_DIR="${CONFIG_DIR:-/root/.zcn}"
CONFIG_FILE="${CONFIG_FILE:-config.yaml}"
WALLET_FILE="${WALLET_FILE:-wallet.json}"
LOG_FILE="/tmp/auto_fund_daemon.log"
PID_FILE="/tmp/auto_fund_daemon.pid"

# Blobber keys directory (contains b0bnode1_keys.txt through b0bnode15_keys.txt)
BLOBBER_KEYS_DIR="${BLOBBER_KEYS_DIR:-/root/Code/blobber/docker.local/keys_config}"

# 0box free storage assigner wallet (from 0box config)
ASSIGNER_WALLET="65b32a635cffb6b6f3c73f09da617c29569a5f690662b5be57ed0d994f234335"
ASSIGNER_MIN_BALANCE=50    # Assigner needs more tokens (funds user allocations)
ASSIGNER_TOP_UP=100

# Staking config
BLOBBER_STAKE_TOKENS=10   # ZCN to stake per blobber
VALIDATOR_STAKE_TOKENS=20 # ZCN to stake per validator

# Sharder URLs for balance/round queries
SHARDER_URLS=("http://localhost:7171" "http://localhost:7172")

# Chain health
CHAIN_CHECK_INTERVAL=10
CHAIN_RECOVERY_MAX_WAIT=300  # 5 minutes max wait for chain recovery

# Nonce management (like vc.sh)
CURRENT_NONCE=0
NONCE_INITIALIZED=false

# ============================================================================
# Logging
# ============================================================================
log() { echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_FILE"; }
log_console() { echo "$(date '+%Y-%m-%d %H:%M:%S') $1" | tee -a "$LOG_FILE"; }

# ============================================================================
# Chain health detection (like vc.sh)
# ============================================================================
get_current_round() {
    local best=0
    for url in "${SHARDER_URLS[@]}"; do
        local r
        r=$(curl -s --connect-timeout 3 --max-time 5 "${url}/v1/chain/get/stats" 2>/dev/null | \
            python3 -c 'import json,sys; print(json.load(sys.stdin).get("latest_finalized_round",0))' 2>/dev/null) || continue
        if [ -n "$r" ] && [ "$r" -gt "$best" ] 2>/dev/null; then
            best=$r
        fi
    done
    echo "${best:-0}"
}

is_chain_healthy() {
    local round1
    round1=$(get_current_round)
    if [ "$round1" = "0" ]; then
        return 1
    fi
    sleep 3
    local round2
    round2=$(get_current_round)
    if [ "$round2" -gt "$round1" ] 2>/dev/null; then
        return 0
    fi
    return 1
}

wait_for_chain_recovery() {
    local start_time
    start_time=$(date +%s)
    log "Chain appears stuck — waiting for recovery (max ${CHAIN_RECOVERY_MAX_WAIT}s)..."

    while true; do
        local elapsed=$(( $(date +%s) - start_time ))
        local round1
        round1=$(get_current_round)
        sleep "$CHAIN_CHECK_INTERVAL"
        local round2
        round2=$(get_current_round)

        if [ "$round2" -gt "$round1" ] 2>/dev/null; then
            log "Chain recovered! Round advancing: $round1 -> $round2"
            return 0
        fi

        if [ "$elapsed" -ge "$CHAIN_RECOVERY_MAX_WAIT" ]; then
            log "Chain still stuck after ${elapsed}s — continuing cycle anyway"
            return 1
        fi

        log "Still waiting for chain... round=$round2 (${elapsed}s/${CHAIN_RECOVERY_MAX_WAIT}s)"
    done
}

# ============================================================================
# Balance helpers
# ============================================================================
get_balance_raw() {
    local client_id="$1"
    for url in "${SHARDER_URLS[@]}"; do
        local result
        result=$(curl -s --connect-timeout 3 --max-time 5 "${url}/v1/client/get/balance?client_id=${client_id}" 2>/dev/null | \
            python3 -c 'import json,sys; print(json.load(sys.stdin).get("balance",0))' 2>/dev/null) || continue
        if [ -n "$result" ] && [ "$result" != "0" ]; then
            echo "$result"
            return
        fi
    done
    echo "0"
}

zcn() { python3 -c "print(round(${1:-0}/1e10, 4))" 2>/dev/null || echo "0"; }

is_low() { python3 -c "print('yes' if float('${1:-0}') < float('${2:-5}') else 'no')" 2>/dev/null || echo "no"; }

# ============================================================================
# Nonce management (like vc.sh)
# ============================================================================
get_funder_client_id() {
    python3 -c "import json; print(json.load(open('${CONFIG_DIR}/${WALLET_FILE}'))['client_id'])" 2>/dev/null
}

get_nonce_from_sharder() {
    local url="$1"
    local client_id
    client_id=$(get_funder_client_id)
    [ -z "$client_id" ] && { echo "0"; return; }
    local nonce
    nonce=$(curl -s --connect-timeout 3 --max-time 5 "${url}/v1/client/get/balance?client_id=${client_id}" 2>/dev/null | \
        python3 -c 'import json,sys; print(json.load(sys.stdin).get("nonce",0))' 2>/dev/null) || echo "0"
    echo "${nonce:-0}"
}

get_current_nonce() {
    local best=0
    for url in "${SHARDER_URLS[@]}"; do
        local n
        n=$(get_nonce_from_sharder "$url") || continue
        if [ "$n" -gt "$best" ] 2>/dev/null; then
            best=$n
        fi
    done
    echo "$best"
}

initialize_nonce() {
    if [ "$NONCE_INITIALIZED" = true ]; then
        return
    fi
    local sharder_nonce
    sharder_nonce=$(get_current_nonce)
    if [ "$sharder_nonce" -gt 0 ] 2>/dev/null; then
        CURRENT_NONCE=$sharder_nonce
        log "Initialized nonce from sharder: $CURRENT_NONCE"
    else
        CURRENT_NONCE=0
        log "Starting with nonce: 0"
    fi
    NONCE_INITIALIZED=true
}

advance_nonce() {
    local sharder_nonce
    sharder_nonce=$(get_current_nonce)
    if [ "$sharder_nonce" -gt "$CURRENT_NONCE" ] 2>/dev/null; then
        CURRENT_NONCE=$sharder_nonce
    fi
    CURRENT_NONCE=$((CURRENT_NONCE + 1))
}

# ============================================================================
# Transaction execution with nonce + retry (like vc.sh run_zwallet_cmd)
# ============================================================================
run_zwallet_send() {
    local to_id="$1"
    local amount="$2"
    local desc="$3"
    local max_retries=3
    local retry=0

    while [ $retry -lt $max_retries ]; do
        advance_nonce
        local nonce=$CURRENT_NONCE

        local output
        output=$($ZWALLET send --to_client_id "$to_id" --tokens "$amount" --desc "$desc" \
            --wallet "$WALLET_FILE" --configDir "$CONFIG_DIR" --config "$CONFIG_FILE" \
            --withNonce "$nonce" --silent 2>&1)
        local exit_code=$?

        # Success
        if echo "$output" | grep -qi "success"; then
            log "  Funded ${to_id:0:16}... with ${amount} ZCN (nonce=$nonce)"
            return 0
        fi

        # Nonce error — re-sync
        if echo "$output" | grep -qiE "nonce"; then
            retry=$((retry + 1))
            local fresh_nonce
            fresh_nonce=$(get_current_nonce)
            log "  Nonce error for ${to_id:0:16}..., refreshing: $CURRENT_NONCE -> $fresh_nonce (retry $retry/$max_retries)"
            if [ "$fresh_nonce" -gt "$CURRENT_NONCE" ] 2>/dev/null; then
                CURRENT_NONCE=$fresh_nonce
            else
                CURRENT_NONCE=$((CURRENT_NONCE + 1))
            fi
            sleep 2
            continue
        fi

        # Insufficient balance on funder — pour from faucet
        if echo "$output" | grep -qiE "insufficient balance"; then
            retry=$((retry + 1))
            log "  Funder insufficient balance, pouring from faucet (retry $retry/$max_retries)"
            pour_faucet 5
            sleep 2
            continue
        fi

        # Network error — wait and retry
        if echo "$output" | grep -qiE "connection refused|timeout|too less sharders|unexpected end"; then
            retry=$((retry + 1))
            log "  Network error for ${to_id:0:16}..., retrying in 5s ($retry/$max_retries)"
            sleep 5
            continue
        fi

        # Unknown error
        retry=$((retry + 1))
        log "  Send failed for ${to_id:0:16}...: $output (retry $retry/$max_retries)"
        sleep 3
    done

    log "  FAILED to fund ${to_id:0:16}... after $max_retries retries"
    return 1
}

pour_faucet() {
    local count=${1:-5}
    for i in $(seq 1 "$count"); do
        advance_nonce
        $ZWALLET faucet --methodName pour --tokens 100 --input "{}" \
            --wallet "$WALLET_FILE" --configDir "$CONFIG_DIR" --config "$CONFIG_FILE" \
            --withNonce "$CURRENT_NONCE" --silent 2>/dev/null || true
        sleep 1
    done
}

# ============================================================================
# Funder balance check
# ============================================================================
ensure_funder_balance() {
    local funder_id
    funder_id=$(get_funder_client_id)
    local bal_raw
    bal_raw=$(get_balance_raw "$funder_id")
    local bal
    bal=$(zcn "$bal_raw")

    if [ "$(is_low "$bal" 100)" = "yes" ]; then
        log "Funder balance low (${bal} ZCN), pouring from faucet..."
        pour_faucet 10
        sleep 3
        bal_raw=$(get_balance_raw "$funder_id")
        bal=$(zcn "$bal_raw")
        log "Funder balance after faucet: ${bal} ZCN"
    fi
}

# ============================================================================
# Provider funding
# ============================================================================
fund_if_low() {
    local id="$1"
    local name="$2"
    local min_bal="$3"
    local topup_amount="$4"

    local bal_raw
    bal_raw=$(get_balance_raw "$id")
    local bal
    bal=$(zcn "$bal_raw")

    if [ "$(is_low "$bal" "$min_bal")" = "yes" ]; then
        log "  ${name} ${id:0:16}... LOW: ${bal} ZCN (min: ${min_bal})"
        ensure_funder_balance
        run_zwallet_send "$id" "$topup_amount" "Auto-fund ${name}"
        return 0  # funded
    fi
    return 1  # not needed
}

# ============================================================================
# Bootstrap: fund blobbers from running containers
# Solves circular dependency: blobbers need ZCN to register on chain,
# but daemon only funds chain-registered blobbers.
# This function funds ALL running blobber containers regardless of chain state.
# ============================================================================
fund_container_blobbers() {
    if [ ! -d "$BLOBBER_KEYS_DIR" ]; then
        return 0
    fi
    local funded_any=0
    for i in $(seq 1 15); do
        # Only process running containers
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^blobber-${i}$" || continue

        local key_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt"
        [ -f "$key_file" ] || continue

        local pub_key
        pub_key=$(awk 'NR==1' "$key_file" 2>/dev/null | tr -d '[:space:]')
        [ -z "$pub_key" ] && continue

        local blobber_id
        blobber_id=$(python3 -c "import hashlib; print(hashlib.sha3_256(bytes.fromhex('${pub_key}')).hexdigest())" 2>/dev/null) || continue
        [ ${#blobber_id} -eq 64 ] || continue

        if fund_if_low "$blobber_id" "blobber-${i}(container)" "$MIN_BALANCE" "$TOP_UP"; then
            funded_any=$((funded_any + 1))
        fi
    done
    return $funded_any
}

# ============================================================================
# Staking: stake blobbers and validators that are registered but have no stake
# ============================================================================
stake_unstaked_providers() {
    local W="--wallet ${WALLET_FILE} --configDir ${CONFIG_DIR} --config ${CONFIG_FILE} --silent"
    local staked=0

    # --- Stake unstaked blobbers ---
    local blobber_json
    blobber_json=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers?limit=50" 2>/dev/null) || true

    if [ -n "$blobber_json" ]; then
        local unstaked_blobbers
        unstaked_blobbers=$(echo "$blobber_json" | python3 -c "
import sys,json
try:
    data = json.load(sys.stdin)
    nodes = data if isinstance(data, list) else data.get('Nodes', data.get('nodes', []))
    for n in (nodes or []):
        url = n.get('base_url', n.get('url', ''))
        # Only stake local blobbers (skip fake/external)
        if url and '.com/' not in url:
            stake = n.get('total_stake', 0)
            if stake == 0:
                print(n['id'])
except: pass
" 2>/dev/null) || true

        for bid in $unstaked_blobbers; do
            log "  Staking unstaked blobber ${bid:0:16}... (${BLOBBER_STAKE_TOKENS} ZCN)"
            ensure_funder_balance
            advance_nonce
            local stake_out
            stake_out=$($ZBOX sp-lock --blobber_id "$bid" --tokens "$BLOBBER_STAKE_TOKENS" \
                $W --withNonce "$CURRENT_NONCE" 2>&1) || true
            if echo "$stake_out" | grep -qi "success\|locked\|already"; then
                log "    Staked blobber ${bid:0:16}..."
                staked=$((staked + 1))
            else
                log "    Stake failed for blobber ${bid:0:16}...: ${stake_out##*$'\n'}"
            fi
            sleep 1
        done
    fi

    # --- Stake unstaked validators ---
    local validator_json
    validator_json=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators?limit=50" 2>/dev/null) || true

    if [ -n "$validator_json" ]; then
        local unstaked_validators
        unstaked_validators=$(echo "$validator_json" | python3 -c "
import sys,json
try:
    data = json.load(sys.stdin)
    nodes = data if isinstance(data, dict) else []
    if isinstance(data, list):
        nodes = data
    elif isinstance(data, dict):
        nodes = data.get('Nodes', data.get('nodes', []))
    for n in (nodes or []):
        stake = n.get('total_stake', n.get('stake_total', 0))
        if stake == 0:
            vid = n.get('validator_id', n.get('id', ''))
            if vid:
                print(vid)
except: pass
" 2>/dev/null) || true

        for vid in $unstaked_validators; do
            log "  Staking unstaked validator ${vid:0:16}... (${VALIDATOR_STAKE_TOKENS} ZCN)"
            ensure_funder_balance
            advance_nonce
            local stake_out
            stake_out=$($ZBOX sp-lock --validator_id "$vid" --tokens "$VALIDATOR_STAKE_TOKENS" \
                $W --withNonce "$CURRENT_NONCE" 2>&1) || true
            if echo "$stake_out" | grep -qi "success\|locked\|already"; then
                log "    Staked validator ${vid:0:16}..."
                staked=$((staked + 1))
            else
                log "    Stake failed for validator ${vid:0:16}...: ${stake_out##*$'\n'}"
            fi
            sleep 1
        done
    fi

    [ $staked -gt 0 ] && log "  Staking cycle: staked $staked providers"
}

# ============================================================================
# Main funding cycle
# ============================================================================
check_and_fund_cycle() {
    log "=== Funding check cycle started ==="

    # Check chain health
    local current_round
    current_round=$(get_current_round)
    if [ "$current_round" = "0" ]; then
        log "Chain not responding — waiting for recovery..."
        wait_for_chain_recovery
        current_round=$(get_current_round)
        if [ "$current_round" = "0" ]; then
            log "Chain still not responding, skipping cycle"
            return
        fi
    fi

    # Verify chain is actually progressing
    sleep 3
    local new_round
    new_round=$(get_current_round)
    if [ "$new_round" -le "$current_round" ] 2>/dev/null; then
        log "Chain stuck at round $current_round — waiting for recovery..."
        wait_for_chain_recovery
    fi

    log "Chain healthy at round $(get_current_round)"

    # Initialize nonce on first run
    initialize_nonce

    local funded=0
    local checked=0

    # --- 0. Bootstrap: fund blobber containers regardless of chain registration ---
    # This fixes the circular dependency: new blobbers need ZCN before they can register,
    # but the chain-registered blobber list only shows registered blobbers.
    fund_container_blobbers
    # Note: fund_container_blobbers returns the funded count but we track separately

    # --- 1. Fund 0box free storage assigner ---
    checked=$((checked + 1))
    if fund_if_low "$ASSIGNER_WALLET" "assigner" "$ASSIGNER_MIN_BALANCE" "$ASSIGNER_TOP_UP"; then
        funded=$((funded + 1))
    fi

    # --- 2. Fund registered blobbers (from chain) ---
    local blobber_ids
    blobber_ids=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers?limit=20" 2>/dev/null | \
        python3 -c "
import sys,json
data = json.load(sys.stdin)
nodes = data if isinstance(data, list) else data.get('Nodes', data.get('nodes', []))
for n in (nodes or []):
    url = n.get('base_url', n.get('url', ''))
    if url and '.com/' not in url:  # skip fake blobbers
        print(n['id'])
" 2>/dev/null) || true

    for id in $blobber_ids; do
        checked=$((checked + 1))
        if fund_if_low "$id" "blobber" "$MIN_BALANCE" "$TOP_UP"; then
            funded=$((funded + 1))
        fi
    done

    # --- 3. Fund validators (from chain) ---
    local validator_ids
    validator_ids=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators" 2>/dev/null | \
        python3 -c "
import sys,json
data = json.load(sys.stdin)
nodes = data.get('Nodes', data) if isinstance(data, dict) else data
for n in nodes:
    vid = n.get('validator_id', n.get('id', ''))
    if vid:
        print(vid)
" 2>/dev/null) || true

    for id in $validator_ids; do
        checked=$((checked + 1))
        if fund_if_low "$id" "validator" 2 5; then
            funded=$((funded + 1))
        fi
    done

    # --- 4. Fund miners (from chain) ---
    local miner_ids
    miner_ids=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList?active=true" 2>/dev/null | \
        python3 -c "
import sys,json
data = json.load(sys.stdin)
for n in data.get('Nodes',[]):
    sm = n.get('simple_miner', n)
    nid = sm.get('id', n.get('id', ''))
    if nid:
        print(nid)
" 2>/dev/null) || true

    for id in $miner_ids; do
        checked=$((checked + 1))
        if fund_if_low "$id" "miner" 2 10; then
            funded=$((funded + 1))
        fi
    done

    # --- 5. Fund sharders (from chain) ---
    local sharder_ids
    sharder_ids=$(curl -s --max-time 10 "${SHARDER_URLS[0]}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList?active=true" 2>/dev/null | \
        python3 -c "
import sys,json
data = json.load(sys.stdin)
for n in data.get('Nodes',[]):
    sm = n.get('simple_miner', n)
    nid = sm.get('id', n.get('id', ''))
    if nid:
        print(nid)
" 2>/dev/null) || true

    for id in $sharder_ids; do
        checked=$((checked + 1))
        if fund_if_low "$id" "sharder" 2 10; then
            funded=$((funded + 1))
        fi
    done

    # --- 6. Stake any unstaked blobbers and validators ---
    stake_unstaked_providers

    log "=== Cycle complete: checked ${checked} providers, funded ${funded} ==="
}

# ============================================================================
# CLI commands
# ============================================================================
case "${1}" in
    --stop)
        if [ -f "$PID_FILE" ]; then
            pid=$(cat "$PID_FILE")
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null
                echo "Daemon stopped (PID: $pid)"
            else
                echo "Daemon not running (stale PID: $pid)"
            fi
            rm -f "$PID_FILE"
        else
            echo "No PID file found"
        fi
        exit 0
        ;;
    --status)
        if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
            echo "Daemon running (PID: $(cat "$PID_FILE"))"
            echo "---"
            tail -10 "$LOG_FILE" 2>/dev/null
        else
            echo "Daemon NOT running"
            if [ -f "$LOG_FILE" ]; then
                echo "Last log entries:"
                tail -5 "$LOG_FILE" 2>/dev/null
            fi
        fi
        exit 0
        ;;
    --once)
        check_and_fund_cycle
        exit 0
        ;;
esac

# ============================================================================
# Daemon mode
# ============================================================================

# Prevent double-start
if [ -f "$PID_FILE" ]; then
    old_pid=$(cat "$PID_FILE")
    if kill -0 "$old_pid" 2>/dev/null; then
        echo "Daemon already running (PID: $old_pid)"
        exit 1
    else
        log "Removing stale PID file (old PID: $old_pid)"
        rm -f "$PID_FILE"
    fi
fi

echo $$ > "$PID_FILE"
log_console "Starting auto-funding daemon (interval: ${INTERVAL}s, min: ${MIN_BALANCE} ZCN, topup: ${TOP_UP} ZCN, assigner_min: ${ASSIGNER_MIN_BALANCE} ZCN)"

trap 'rm -f "$PID_FILE"; log "Daemon stopped (signal)"; exit 0' INT TERM
trap 'log "Daemon exiting unexpectedly (EXIT trap)"; rm -f "$PID_FILE"' EXIT

while true; do
    # Wrap cycle in subshell-safe error handling
    check_and_fund_cycle || {
        log "WARNING: Funding cycle failed with error, continuing..."
    }
    sleep "$INTERVAL"
done
