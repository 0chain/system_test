#!/bin/bash
# test_swap_image.sh — Automated swap-image validation
#
# Swaps each service to staging, verifies health, swaps back to the configured
# branch, verifies health again, then runs targeted tests to confirm functionality.
#
# Usage: bash scripts/test_swap_image.sh [service...]
#   No args: tests all services (blobber, 0box, zauth-server, zvault)
#   With args: tests only specified services

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

export PATH=$PATH:/usr/local/go/bin:/root/go/bin

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAILURES=$((FAILURES + 1)); }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

FAILURES=0

get_branch() {
    local repo="$1"
    python3 -c "
import yaml
with open('scripts/deploy_config.yaml') as f:
    d = yaml.safe_load(f)
repos = d.get('repositories', {})
print(repos.get('$repo', {}).get('branch', 'staging'))
" 2>/dev/null || echo "staging"
}

check_chain() {
    local round
    round=$(curl -s "http://198.18.0.81:7171/v1/chain/get/stats" -m 5 2>/dev/null | \
        python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")
    [ "$round" -gt 0 ] 2>/dev/null
}

check_blobber() {
    cd tests/cli_tests
    local alloc
    alloc=$(./zbox newallocation --lock 1 --size 1048576 --data 2 --parity 2 \
        --configDir /root/.zcn --config local.yaml --wallet local.json --silent 2>&1 | \
        grep -o "[a-f0-9]\{64\}" | head -1)
    if [ -z "$alloc" ]; then cd ../..; return 1; fi
    dd if=/dev/urandom of=/tmp/swap_test.bin bs=1K count=10 2>/dev/null
    ./zbox upload --allocation "$alloc" --localpath /tmp/swap_test.bin --remotepath /test.bin \
        --configDir /root/.zcn --config local.yaml --wallet local.json --silent 2>&1 >/dev/null
    rm -f /tmp/swap_dl.bin
    ./zbox download --allocation "$alloc" --remotepath /test.bin --localpath /tmp/swap_dl.bin \
        --configDir /root/.zcn --config local.yaml --wallet local.json --silent 2>&1 >/dev/null
    cd ../..
    [ -f /tmp/swap_dl.bin ]
}

check_0box() {
    local domain
    domain=$(grep 'domain:' scripts/deploy_config.yaml | head -1 | awk '{print $2}' | tr -d '"')
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "https://${domain}/" -k 2>/dev/null)
    [ "$status" = "200" ]
}

test_swap() {
    local service="$1"
    local original_branch
    original_branch=$(get_branch "$service")

    info "Testing swap-image for: $service (current: $original_branch)"

    # Swap to staging
    info "  Swapping $service -> staging..."
    bash scripts/deploy_local.sh swap-image "$service" staging 2>&1 | tail -3
    sleep 30

    if check_chain; then pass "$service: chain alive after swap to staging"
    else fail "$service: chain dead after swap to staging"; fi

    [ "$service" = "blobber" ] && { check_blobber && pass "$service: blobber ops work on staging" || fail "$service: blobber ops failed on staging"; }
    [ "$service" = "0box" ] && { check_0box && pass "$service: 0box responds on staging" || fail "$service: 0box dead on staging"; }

    # Swap back
    info "  Swapping $service -> $original_branch..."
    bash scripts/deploy_local.sh swap-image "$service" "$original_branch" 2>&1 | tail -3
    sleep 30

    if check_chain; then pass "$service: chain alive after swap back"
    else fail "$service: chain dead after swap back"; fi

    [ "$service" = "blobber" ] && { check_blobber && pass "$service: blobber ops work on $original_branch" || fail "$service: blobber ops failed on $original_branch"; }
    [ "$service" = "0box" ] && { check_0box && pass "$service: 0box responds on $original_branch" || fail "$service: 0box dead on $original_branch"; }

    pass "$service: swap round-trip complete"
}

SERVICES=("${@:-blobber 0box zauth-server zvault}")
[ $# -eq 0 ] && SERVICES=(blobber 0box zauth-server zvault)

echo "========================================"
echo "  Swap-Image Validation Test"
echo "  Services: ${SERVICES[*]}"
echo "========================================"

for svc in "${SERVICES[@]}"; do
    test_swap "$svc"
    echo ""
done

echo "========================================"
[ "$FAILURES" -eq 0 ] && echo -e "${GREEN}  ALL SWAP TESTS PASSED${NC}" || echo -e "${RED}  $FAILURES SWAP TESTS FAILED${NC}"
echo "========================================"
exit $FAILURES
