#!/bin/bash
# test_swap_image.sh — Automated swap-image validation for ALL repos
#
# For each service: swaps to an earlier commit on the same branch,
# verifies health, swaps back to HEAD, verifies health again.
# This tests the full swap-image pipeline (checkout, build, restart, config).
#
# Usage: bash scripts/test_swap_image.sh [service...]
#   No args: tests all services
#   With args: tests only specified services
#
# All supported services:
#   0chain blobber eblobber 0box zauth-server zvault zs3server crawler 0dns gosdk rclone_zus

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

# Get the configured branch for a repo from deploy_config.yaml
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

# Get the previous commit hash on the current branch
get_prev_commit() {
    local repo="$1"
    local repo_path
    repo_path=$(python3 -c "
import yaml
with open('scripts/deploy_config.yaml') as f:
    d = yaml.safe_load(f)
repos = d.get('repositories', {})
print(repos.get('$repo', {}).get('path', '$repo'))
" 2>/dev/null || echo "$repo")
    local dir="$HOME/Code/${repo_path}"
    [ -d "$dir/.git" ] || dir="$HOME/Code/${repo}"
    if [ -d "$dir/.git" ]; then
        git -C "$dir" log --oneline -2 2>/dev/null | tail -1 | awk '{print $1}'
    else
        echo ""
    fi
}

# Health checks
check_chain() {
    local round
    round=$(curl -s "http://198.18.0.81:7171/v1/chain/get/stats" -m 5 2>/dev/null | \
        python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")
    [ "$round" -gt 0 ] 2>/dev/null
}

check_service() {
    local url="$1"
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url" -k -m 5 2>/dev/null || echo "000")
    [ "$status" != "000" ] && [ "$status" != "502" ] && [ "$status" != "503" ]
}

check_0dns() { check_service "http://198.18.0.100:9091/dns"; }
check_0box() {
    local domain
    domain=$(grep 'domain:' scripts/deploy_config.yaml | grep -v '#' | head -1 | awk '{print $2}' | tr -d '"')
    check_service "https://${domain}/" || check_service "http://localhost:9081/"
}
check_zauth() { check_service "http://localhost:8080/"; }
check_zvault() { check_service "http://localhost:8090/"; }
check_zs3() { check_service "http://localhost:9100/minio/health/live"; }

# Service-specific health check dispatch
service_health() {
    local svc="$1"
    case "$svc" in
        0chain)     check_chain ;;
        blobber)    check_chain ;;
        eblobber)   check_chain ;;
        0box)       check_0box ;;
        zauth-server) check_zauth ;;
        zvault)     check_zvault ;;
        zs3server)  check_zs3 ;;
        0dns)       check_0dns ;;
        gosdk)      return 0 ;; # gosdk is a library, no health check
        crawler)    docker ps --format '{{.Names}}' 2>/dev/null | grep -q crawler ;;
        rclone_zus) return 0 ;; # rclone is a binary, no health check
        *)          return 0 ;;
    esac
}

test_swap() {
    local service="$1"
    local branch
    branch=$(get_branch "$service")
    local prev_commit
    prev_commit=$(get_prev_commit "$service")

    if [ -z "$prev_commit" ]; then
        info "Skipping $service: cannot determine previous commit (repo not found?)"
        return 0
    fi

    info "Testing swap-image for: $service"
    info "  Branch: $branch, HEAD will swap to prev commit: $prev_commit"

    # Step 1: Swap to previous commit
    info "  [1/4] Swapping $service -> $prev_commit..."
    if ! bash scripts/deploy_local.sh swap-image "$service" "$prev_commit" >/dev/null 2>&1; then
        fail "$service: swap to $prev_commit failed"
        return
    fi
    sleep 10

    # Step 2: Verify health after swap
    if service_health "$service"; then
        pass "$service: healthy after swap to $prev_commit"
    else
        fail "$service: unhealthy after swap to $prev_commit"
    fi

    # Step 3: Swap back to HEAD
    info "  [3/4] Swapping $service -> $branch (HEAD)..."
    if ! bash scripts/deploy_local.sh swap-image "$service" "$branch" >/dev/null 2>&1; then
        fail "$service: swap back to $branch failed"
        return
    fi
    sleep 10

    # Step 4: Verify health after swap back
    if service_health "$service"; then
        pass "$service: healthy after swap back to $branch"
    else
        fail "$service: unhealthy after swap back to $branch"
    fi

    pass "$service: swap round-trip complete"
}

# Default: all services that have Docker images
ALL_SERVICES=(zvault zauth-server 0dns 0box blobber eblobber zs3server crawler gosdk rclone_zus 0chain)

if [ $# -gt 0 ]; then
    SERVICES=("$@")
else
    SERVICES=("${ALL_SERVICES[@]}")
fi

echo "========================================"
echo "  Swap-Image Validation Test"
echo "  Services: ${SERVICES[*]}"
echo "  $(date)"
echo "========================================"

for svc in "${SERVICES[@]}"; do
    test_swap "$svc"
    echo ""
done

echo "========================================"
if [ "$FAILURES" -eq 0 ]; then
    echo -e "${GREEN}  ALL SWAP TESTS PASSED${NC}"
else
    echo -e "${RED}  $FAILURES SWAP TESTS FAILED${NC}"
fi
echo "========================================"
exit $FAILURES
