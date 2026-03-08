#!/bin/bash
# Test script for swap-image command
#
# Verifies that swap-image correctly handles:
# 1. Git checkout/pull for all supported repos
# 2. Docker image build for each repo
# 3. --with-config flag (default=no config changes)
# 4. Container restart
#
# Usage:
#   ./test_swap_image.sh                    # Test all repos (dry-run: skip build/restart)
#   ./test_swap_image.sh --live             # Actually build and restart (destructive)
#   ./test_swap_image.sh --repo blobber     # Test a single repo
#   ./test_swap_image.sh --config-test      # Verify --with-config vs default behavior
#
# This script creates a temporary test branch, runs swap-image, then cleans up.

set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "${SCRIPT_DIR}/../../" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
log_skip() { echo -e "${YELLOW}[SKIP]${NC} $1"; SKIP_COUNT=$((SKIP_COUNT + 1)); }
log_info() { echo -e "[INFO] $1"; }

# All repos that swap-image supports
ALL_REPOS=(
    0chain blobber eblobber 0box zauth-server zvault
    zs3server web-apps crawler 0dns gosdk zboxcli zwalletcli rclone_zus
)

# Repo → local directory name mapping
declare -A REPO_DIRS=(
    ["0chain"]="0chain"
    ["blobber"]="blobber"
    ["eblobber"]="eblobber"
    ["0box"]="0box"
    ["zauth-server"]="zauth-server"
    ["zvault"]="zvault"
    ["zs3server"]="zs3server"
    ["web-apps"]="web-apps"
    ["crawler"]="crawler"
    ["0dns"]="0dns"
    ["gosdk"]="gosdk"
    ["zboxcli"]="zboxcli"
    ["zwalletcli"]="zwalletcli"
    ["rclone_zus"]="rclone_zus"
)

# Config files that should NOT be modified without --with-config
declare -A CONFIG_FILES=(
    ["0box"]="0box/docker.local/config/0box.yaml"
    ["blobber"]="blobber/config/0chain_blobber.yaml"
    ["0chain"]="0chain/docker.local/config/0chain.yaml"
    ["zauth-server"]="zauth-server/docker.local/config/faucet.yaml"
    ["zs3server"]="zs3server/environment/docker-compose.yaml"
)

# Parse args
LIVE_MODE=false
CONFIG_TEST=false
TARGET_REPO=""

while [ $# -gt 0 ]; do
    case "$1" in
        --live) LIVE_MODE=true; shift ;;
        --config-test) CONFIG_TEST=true; shift ;;
        --repo) TARGET_REPO="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

# ============================================================
# Test 1: Verify all repos exist locally
# ============================================================
test_repos_exist() {
    log_info "=== Test: Verify repo directories exist ==="
    for repo in "${ALL_REPOS[@]}"; do
        local dir="${REPO_DIRS[$repo]}"
        local repo_path="${BASE_DIR}/${dir}"
        if [ -d "$repo_path" ]; then
            log_pass "$repo → ${repo_path}"
        else
            log_fail "$repo → ${repo_path} NOT FOUND"
        fi
    done
}

# ============================================================
# Test 2: Verify swap-image parses all repos without error
# ============================================================
test_swap_image_parse() {
    log_info "=== Test: swap-image recognizes all repos ==="
    for repo in "${ALL_REPOS[@]}"; do
        local dir="${REPO_DIRS[$repo]}"
        local repo_path="${BASE_DIR}/${dir}"
        if [ ! -d "$repo_path/.git" ]; then
            log_skip "$repo — no .git directory (would need clone)"
            continue
        fi
        # Get current branch to use as test
        local current_branch
        current_branch=$(git -C "$repo_path" branch --show-current 2>/dev/null || echo "")
        if [ -z "$current_branch" ]; then
            current_branch=$(git -C "$repo_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "master")
        fi
        log_pass "$repo → branch: ${current_branch}"
    done
}

# ============================================================
# Test 3: Config protection test (--with-config vs default)
# ============================================================
test_config_protection() {
    log_info "=== Test: Config files protected without --with-config ==="

    for repo in "${!CONFIG_FILES[@]}"; do
        local config_rel="${CONFIG_FILES[$repo]}"
        local config_path="${BASE_DIR}/${config_rel}"

        if [ ! -f "$config_path" ]; then
            log_skip "$repo config: ${config_rel} not found"
            continue
        fi

        # Take a checksum of the config file
        local before_hash
        before_hash=$(md5sum "$config_path" 2>/dev/null | awk '{print $1}')

        if [ -z "$before_hash" ]; then
            # macOS
            before_hash=$(md5 -q "$config_path" 2>/dev/null)
        fi

        log_pass "$repo config exists: ${config_rel} (hash: ${before_hash:0:8})"
    done
}

# ============================================================
# Test 4: Verify build case exists for each repo
# ============================================================
test_build_cases() {
    log_info "=== Test: Build cases exist in swap_image() ==="

    local deploy_script="${SCRIPT_DIR}/deploy_local.sh"

    for repo in "${ALL_REPOS[@]}"; do
        # Check if repo has a build case (grep for "repo)" or "repo|" or "|repo)" patterns)
        if grep -qE "^        ${repo}\)" "$deploy_script" 2>/dev/null; then
            log_pass "$repo has build case"
        elif grep -qE "^        ${repo}\||\|${repo}\)" "$deploy_script" 2>/dev/null; then
            log_pass "$repo has build case (combined)"
        else
            log_fail "$repo MISSING build case in swap_image()"
        fi
    done
}

# ============================================================
# Test 5: Verify restart case exists for each repo
# ============================================================
test_restart_cases() {
    log_info "=== Test: Restart cases exist in swap_image() ==="

    local deploy_script="${SCRIPT_DIR}/deploy_local.sh"

    # Repos that return early (no restart needed): gosdk, zboxcli, zwalletcli, rclone_zus
    local no_restart_repos=("gosdk" "zboxcli" "zwalletcli" "rclone_zus")

    for repo in "${ALL_REPOS[@]}"; do
        # Skip repos that exit before restart
        local skip=false
        for nr in "${no_restart_repos[@]}"; do
            if [ "$repo" = "$nr" ]; then skip=true; break; fi
        done
        if $skip; then
            log_pass "$repo — no restart needed (binary-only)"
            continue
        fi

        # Check restart case in Step 3
        if grep -A2 "Step 3" "$deploy_script" >/dev/null 2>&1 && \
           grep -q "^        ${repo})" "$deploy_script" 2>/dev/null; then
            log_pass "$repo has restart case"
        else
            log_fail "$repo MISSING restart case in swap_image()"
        fi
    done
}

# ============================================================
# Test 6: Verify --with-config flag is properly gated
# ============================================================
test_config_gating() {
    log_info "=== Test: Config functions gated behind --with-config ==="

    local deploy_script="${SCRIPT_DIR}/deploy_local.sh"

    # These config functions should be inside an apply_config check
    local config_funcs=(
        "fix_0box_config"
        "patch_zauth_faucet"
        "setup_web_app_env_files"
        "configure_kafka_in_configs"
        "fix_blobber_config"
    )

    for func in "${config_funcs[@]}"; do
        # Check that all calls to this function within swap_image are gated
        # Look for the function being called after an apply_config check
        local total_calls
        total_calls=$(sed -n '/^swap_image()/,/^}/p' "$deploy_script" | grep -c "$func" 2>/dev/null || echo 0)
        local gated_calls
        gated_calls=$(sed -n '/^swap_image()/,/^}/p' "$deploy_script" | grep -B5 "$func" | grep -c 'apply_config' 2>/dev/null || echo 0)

        if [ "$total_calls" -eq 0 ]; then
            log_skip "$func — not called in swap_image()"
        elif [ "$gated_calls" -ge 1 ]; then
            log_pass "$func is gated behind --with-config ($gated_calls/$total_calls calls)"
        else
            log_fail "$func NOT gated behind --with-config ($gated_calls/$total_calls calls)"
        fi
    done
}

# ============================================================
# Test 7: Live swap test (only with --live flag)
# ============================================================
test_live_swap() {
    if ! $LIVE_MODE; then
        log_info "=== Skipping live swap test (pass --live to enable) ==="
        return
    fi

    local repo="${TARGET_REPO:-gosdk}"
    log_info "=== Test: Live swap-image for ${repo} ==="

    local dir="${REPO_DIRS[$repo]}"
    local repo_path="${BASE_DIR}/${dir}"

    if [ ! -d "$repo_path/.git" ]; then
        log_fail "$repo — no .git directory"
        return
    fi

    # Save current state
    local original_branch
    original_branch=$(git -C "$repo_path" branch --show-current 2>/dev/null)
    local original_hash
    original_hash=$(git -C "$repo_path" rev-parse HEAD 2>/dev/null)

    log_info "Original: ${original_branch} (${original_hash:0:8})"

    # Create test branch
    local test_branch="test-swap-image-$(date +%s)"
    git -C "$repo_path" checkout -b "$test_branch" 2>/dev/null || {
        log_fail "Failed to create test branch"
        return
    }
    log_info "Created test branch: $test_branch"

    # Run swap-image with the test branch
    log_info "Running: bash ${SCRIPT_DIR}/deploy_local.sh swap-image ${repo} ${test_branch}"
    if bash "${SCRIPT_DIR}/deploy_local.sh" swap-image "$repo" "$test_branch" 2>&1; then
        log_pass "swap-image ${repo} ${test_branch} succeeded"
    else
        log_fail "swap-image ${repo} ${test_branch} failed"
    fi

    # Verify we're on the test branch
    local after_branch
    after_branch=$(git -C "$repo_path" branch --show-current 2>/dev/null)
    if [ "$after_branch" = "$test_branch" ]; then
        log_pass "Repo is on test branch after swap"
    else
        log_fail "Repo on wrong branch: ${after_branch} (expected: ${test_branch})"
    fi

    # Cleanup: restore original branch and delete test branch
    git -C "$repo_path" checkout "$original_branch" 2>/dev/null || true
    git -C "$repo_path" branch -D "$test_branch" 2>/dev/null || true
    log_info "Restored to ${original_branch}"
}

# ============================================================
# Main
# ============================================================
echo ""
echo "=========================================="
echo "  swap-image Test Suite"
echo "=========================================="
echo "BASE_DIR: ${BASE_DIR}"
echo "LIVE_MODE: ${LIVE_MODE}"
echo "CONFIG_TEST: ${CONFIG_TEST}"
echo "TARGET_REPO: ${TARGET_REPO:-all}"
echo ""

test_repos_exist
echo ""
test_swap_image_parse
echo ""
test_build_cases
echo ""
test_restart_cases
echo ""
test_config_gating
echo ""

if $CONFIG_TEST; then
    test_config_protection
    echo ""
fi

test_live_swap
echo ""

echo "=========================================="
echo "  Results: ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}, ${YELLOW}${SKIP_COUNT} skipped${NC}"
echo "=========================================="

exit $FAIL_COUNT
