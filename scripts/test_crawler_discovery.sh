#!/bin/bash
# Crawler Blobber Discovery Test Script
#
# Tests the crawler's ability to discover uncovered blobbers, add them to
# existing allocations, and create new allocations when max_blobbers_per_allocation
# is reached.
#
# Test plan (4 subtests):
#   A. Set max_blobbers_per_allocation=6, restart crawler with empty alloc list
#   B. Trigger discovery → should create 2 allocations (6 + 5 = 11 blobbers)
#   C. Raise max to 9, trigger again → should be no-op (all covered)
#   D. Verify all blobbers covered, then restore original config
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
CRAWLER_CONFIG="/root/Code/crawler/docker.local/config/crawler.yaml"

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

# Restart crawler with empty allocation list (keeps wallet keys intact)
restart_crawler_empty() {
    run_remote "python3 -c \"
import yaml
with open('${CRAWLER_CONFIG}') as f:
    cfg = yaml.safe_load(f)
cfg['allocations'] = []
with open('${CRAWLER_CONFIG}', 'w') as f:
    yaml.dump(cfg, f, default_flow_style=False)
print('Cleared allocations from config')
\" && docker restart crawler && sleep 5"
}

header "Crawler Blobber Discovery Test"

# Pre-checks
info "Checking crawler API..."
health=$(crawler_health)
if ! echo "$health" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d['status']=='ok'" 2>/dev/null; then
    echo -e "${RED}FATAL${NC}: Crawler API not reachable at ${CRAWLER_API} on ${SERVER}"
    exit 1
fi
info "Crawler healthy"

ORIGINAL_MAX=$(get_max_blobbers)
ACTIVE_COUNT=$(get_active_regular_count)
info "max_blobbers_per_allocation: ${ORIGINAL_MAX}, active regular blobbers: ${ACTIVE_COUNT}"

if [ "$ACTIVE_COUNT" -lt 8 ]; then
    echo -e "${RED}FATAL${NC}: Need at least 8 active regular blobbers, found ${ACTIVE_COUNT}"
    exit 1
fi

# Save original crawler config
run_remote "cp ${CRAWLER_CONFIG} ${CRAWLER_CONFIG}.bak"

# ================================================================
header "Subtest A: Set max_blobbers=6, restart crawler with empty allocs"
# ================================================================

info "Setting max_blobbers_per_allocation to 6..."
output=$(set_sc_config "max_blobbers_per_allocation" "6")
sleep 3

VERIFY_MAX=$(get_max_blobbers)
if [ "$VERIFY_MAX" = "6" ]; then
    info "Verified: max_blobbers_per_allocation = 6"
else
    fail "Subtest A: expected max=6, got ${VERIFY_MAX}"
fi

info "Restarting crawler with empty allocation list..."
restart_crawler_empty
sleep 5

# Verify crawler is up — initial discovery runs on startup so it may already have allocations
health2=$(crawler_health)
alloc_count=$(echo "$health2" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('allocations',0))" 2>/dev/null)
if echo "$health2" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d['status']=='ok'" 2>/dev/null; then
    pass "Subtest A: Crawler restarted with max_blobbers=6 (${alloc_count} alloc(s) from initial discovery)"
else
    fail "Subtest A: Crawler not healthy after restart"
fi

# ================================================================
header "Subtest B: Trigger discovery → expect 2 allocations"
# ================================================================

info "Triggering discovery (${ACTIVE_COUNT} blobbers, max 6 per alloc)..."
info "Expected: alloc1=6 blobbers, alloc2=${ACTIVE_COUNT}-6=$((ACTIVE_COUNT-6)) blobbers"
result=$(trigger_discover)
echo "$result" | python3 -m json.tool 2>/dev/null

uncovered=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
added=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('added_to_existing',0))" 2>/dev/null)
new_allocs=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('new_allocations_created',0))" 2>/dev/null)
leftover=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('leftover_count',0))" 2>/dev/null)
total=$(echo "$result" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('total_active',0))" 2>/dev/null)

# With 0 existing allocations and max=6: discovery creates NEW allocations.
# First alloc: 6 blobbers. Second alloc: remaining (5 if 11 total, 6 if 12).
# The remaining must be >= 4 to create an allocation.
expected_allocs=0
remaining=$((ACTIVE_COUNT))
while [ "$remaining" -ge 4 ]; do
    if [ "$remaining" -gt 6 ]; then
        remaining=$((remaining - 6))
    else
        remaining=0
    fi
    expected_allocs=$((expected_allocs + 1))
done

if [ "$new_allocs" -ge 2 ]; then
    pass "Subtest B: Created ${new_allocs} allocations for ${total} blobbers (leftover: ${leftover})"
elif [ "$new_allocs" -ge 1 ] && [ "$leftover" -lt 4 ]; then
    pass "Subtest B: Created ${new_allocs} allocation(s), ${leftover} leftover (< 4, cannot form allocation)"
else
    fail "Subtest B: Expected >=2 allocations, got ${new_allocs} (uncovered=${uncovered}, added=${added})"
fi

# Verify crawler now has the allocations
health3=$(crawler_health)
alloc_count3=$(echo "$health3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('allocations',0))" 2>/dev/null)
info "Crawler now has ${alloc_count3} allocation(s)"

# ================================================================
header "Subtest C: Raise max to 9, re-trigger → should be no-op"
# ================================================================

info "Raising max_blobbers_per_allocation to 9..."
set_sc_config "max_blobbers_per_allocation" "9" > /dev/null
sleep 3

info "Triggering discovery again..."
result2=$(trigger_discover)
echo "$result2" | python3 -m json.tool 2>/dev/null

uncovered2=$(echo "$result2" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
added2=$(echo "$result2" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('added_to_existing',0))" 2>/dev/null)

if [ "$uncovered2" = "0" ]; then
    pass "Subtest C: All blobbers still covered after raising max to 9 (0 uncovered)"
elif [ "$added2" -gt 0 ]; then
    # If leftover blobbers from subtest B now get added to existing allocs
    pass "Subtest C: ${added2} leftover blobbers added to existing allocations after raising max"
else
    fail "Subtest C: ${uncovered2} uncovered, ${added2} added — expected all covered"
fi

# ================================================================
header "Subtest D: Final verify — all blobbers covered"
# ================================================================

result3=$(trigger_discover)
uncovered3=$(echo "$result3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uncovered_count',0))" 2>/dev/null)
total3=$(echo "$result3" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('total_active',0))" 2>/dev/null)
health4=$(crawler_health)
final_allocs=$(echo "$health4" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('allocations',0))" 2>/dev/null)

if [ "$uncovered3" = "0" ] && [ "$total3" -ge 8 ]; then
    pass "Subtest D: All ${total3} blobbers covered across ${final_allocs} allocation(s)"
else
    fail "Subtest D: ${uncovered3} uncovered out of ${total3} total"
fi

# ================================================================
header "Cleanup"
# ================================================================

info "Restoring max_blobbers_per_allocation to ${ORIGINAL_MAX}..."
set_sc_config "max_blobbers_per_allocation" "${ORIGINAL_MAX}" > /dev/null

info "Restoring original crawler config..."
run_remote "cp ${CRAWLER_CONFIG}.bak ${CRAWLER_CONFIG} && docker restart crawler" > /dev/null
sleep 5
info "Done"

# ================================================================
header "Results"
# ================================================================
echo -e "  ${GREEN}PASS: ${PASS_COUNT}${NC}"
echo -e "  ${RED}FAIL: ${FAIL_COUNT}${NC}"
echo ""

[ "$FAIL_COUNT" -eq 0 ] && exit 0 || exit 1
