#!/bin/bash
# Crawler Blobber Discovery Test Script
#
# Tests the crawler's ability to discover uncovered blobbers and manage allocations
# when max_blobbers_per_allocation limits are reached.
#
# Usage:
#   bash scripts/test_crawler_discovery.sh [server_ip]

set -o pipefail

SERVER="${1:-37.27.65.188}"
PASS_ENV="${SERVER_PASS:-***REDACTED***}"
CRAWLER_API="http://localhost:3030"
ZCN_CONFIG_DIR="/root/.zcn"
WALLET="owner.json"
CONFIG="local.yaml"
ZBOX="/root/Code/zboxcli/zbox"
ZWALLET="/root/Code/zwalletcli/zwallet"
SHARDER="http://198.18.0.81:7171"
STORAGE_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0

pass() { echo -e "  ${GREEN}PASS${NC}: $1"; PASS_COUNT=$((PASS_COUNT+1)); }
fail() { echo -e "  ${RED}FAIL${NC}: $1"; FAIL_COUNT=$((FAIL_COUNT+1)); }
info() { echo -e "  ${YELLOW}INFO${NC}: $1"; }
header() { echo -e "\n${BLUE}========================================${NC}"; echo -e "${BLUE}  $1${NC}"; echo -e "${BLUE}========================================${NC}"; }

run_remote() {
    sshpass -p "$PASS_ENV" ssh -o StrictHostKeyChecking=no root@"${SERVER}" "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin; $1" 2>/dev/null
}

get_max_blobbers() {
    run_remote "curl -s '${SHARDER}/v1/screst/${STORAGE_SC}/storage-config' | python3 -c \"import json,sys; d=json.load(sys.stdin); f=d.get('fields',d); print(int(float(f.get('max_blobbers_per_allocation','40'))))\""
}

get_alloc_blobber_count() {
    run_remote "curl -s '${SHARDER}/v1/screst/${STORAGE_SC}/allocation?allocation=$1' | python3 -c \"import json,sys; d=json.load(sys.stdin); print(len(d.get('blobbers',[])))\""
}

get_active_regular_count() {
    run_remote "curl -s '${SHARDER}/v1/screst/${STORAGE_SC}/getblobbers' | python3 -c \"import json,sys; d=json.load(sys.stdin); print(len([b for b in d.get('Nodes',[]) if not b.get('is_killed') and not b.get('is_shutdown') and not b.get('is_enterprise')]))\""
}

trigger_discover() {
    run_remote "curl -s -X POST ${CRAWLER_API}/v1/discover"
}

crawler_health() {
    run_remote "curl -s ${CRAWLER_API}/v1/health"
}

set_sc_config() {
    run_remote "${ZWALLET} sc-update-config --keys $1 --values $2 --wallet ${WALLET} --configDir ${ZCN_CONFIG_DIR} --config ${CONFIG} --silent 2>&1"
}

header "Crawler Blobber Discovery Test"

# Pre-check
info "Checking crawler API..."
health=$(crawler_health)
if ! echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d['status']=='ok'" 2>/dev/null; then
    echo -e "${RED}FATAL${NC}: Crawler API not reachable at ${CRAWLER_API} on ${SERVER}"
    exit 1
fi
alloc_count=$(echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('allocations',0))" 2>/dev/null)
info "Crawler healthy, ${alloc_count} allocation(s)"

ORIGINAL_MAX=$(get_max_blobbers)
ACTIVE_COUNT=$(get_active_regular_count)
info "max_blobbers_per_allocation: ${ORIGINAL_MAX}, active regular blobbers: ${ACTIVE_COUNT}"

# Get current crawler allocation
CRAWLER_ALLOC=$(run_remote "grep -A 1 'allocations:' /root/Code/crawler/docker.local/config/crawler.yaml | tail -1 | sed 's/.*- //' | tr -d ' \"'")
CURRENT_BLOBBERS=$(get_alloc_blobber_count "$CRAWLER_ALLOC")
info "Crawler allocation ${CRAWLER_ALLOC:0:16}... has ${CURRENT_BLOBBERS} blobbers"

# ================================================================
header "Subtest A: Set max_blobbers_per_allocation to 14"
# ================================================================

NEW_MAX=14
info "Setting max_blobbers_per_allocation to ${NEW_MAX}..."
output=$(set_sc_config "max_blobbers_per_allocation" "${NEW_MAX}")
sleep 3  # wait for chain to process

VERIFY_MAX=$(get_max_blobbers)
if [ "$VERIFY_MAX" = "$NEW_MAX" ]; then
    pass "max_blobbers_per_allocation set to ${VERIFY_MAX}"
else
    fail "expected ${NEW_MAX}, got ${VERIFY_MAX}"
fi

# ================================================================
header "Subtest B: Trigger discovery — all blobbers should be covered"
# ================================================================

info "Triggering crawler discovery..."
result=$(trigger_discover)
echo "$result" | python3 -m json.tool 2>/dev/null

uncovered=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
total=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('total_active',0))" 2>/dev/null)

if [ "$uncovered" = "0" ] && [ "$total" -gt 0 ]; then
    pass "All ${total} blobbers covered, 0 uncovered"
else
    # Some uncovered — discovery should add them
    added=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('added_to_existing',0))" 2>/dev/null)
    new_allocs=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('new_allocations_created',0))" 2>/dev/null)
    if [ "$added" -gt 0 ] || [ "$new_allocs" -gt 0 ]; then
        pass "Discovery found ${uncovered} uncovered, added ${added}, created ${new_allocs} new alloc(s)"
    else
        fail "Discovery found ${uncovered} uncovered but added none"
    fi
fi

# ================================================================
header "Subtest C: Verify allocation blobber count matches chain"
# ================================================================

AFTER_BLOBBERS=$(get_alloc_blobber_count "$CRAWLER_ALLOC")
info "Crawler allocation now has ${AFTER_BLOBBERS} blobbers (chain has ${ACTIVE_COUNT} regular)"

health_after=$(crawler_health)
alloc_count_after=$(echo "$health_after" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('allocations',0))" 2>/dev/null)

if [ "$AFTER_BLOBBERS" -ge "$ACTIVE_COUNT" ] || [ "$alloc_count_after" -gt 1 ]; then
    pass "All blobbers accounted for: ${AFTER_BLOBBERS} in primary alloc, ${alloc_count_after} total alloc(s)"
else
    # Check if all are covered across multiple allocations
    result2=$(trigger_discover)
    uncovered2=$(echo "$result2" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
    if [ "$uncovered2" = "0" ]; then
        pass "All blobbers covered across ${alloc_count_after} allocation(s)"
    else
        fail "Still ${uncovered2} uncovered blobbers"
    fi
fi

# ================================================================
header "Subtest D: Re-trigger — verify idempotent (no changes)"
# ================================================================

info "Triggering discovery again (should be idempotent)..."
result3=$(trigger_discover)
echo "$result3" | python3 -m json.tool 2>/dev/null

uncovered3=$(echo "$result3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
added3=$(echo "$result3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('added_to_existing',0))" 2>/dev/null)
new_allocs3=$(echo "$result3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('new_allocations_created',0))" 2>/dev/null)

if [ "$uncovered3" = "0" ] && [ "$added3" = "0" ] && [ "$new_allocs3" = "0" ]; then
    pass "Idempotent: 0 uncovered, 0 added, 0 new allocations"
else
    fail "Not idempotent: uncovered=${uncovered3} added=${added3} new_allocs=${new_allocs3}"
fi

# ================================================================
header "Cleanup: Restore max_blobbers_per_allocation"
# ================================================================

info "Restoring max_blobbers_per_allocation to ${ORIGINAL_MAX}..."
set_sc_config "max_blobbers_per_allocation" "${ORIGINAL_MAX}" > /dev/null
info "Restored"

# ================================================================
header "Results"
# ================================================================
echo -e "  ${GREEN}PASS: ${PASS_COUNT}${NC}"
echo -e "  ${RED}FAIL: ${FAIL_COUNT}${NC}"
echo ""

[ "$FAIL_COUNT" -eq 0 ] && exit 0 || exit 1
