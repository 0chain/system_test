#!/bin/bash
# 0Chain System Test Runner
#
# Two execution modes:
#   Default  — All suites run IN PARALLEL simultaneously (fast, ~30-120 min)
#   --long   — All suites run SEQUENTIALLY one after another (thorough, ~2-3 h)
#   --smoke  — Smoke subtests only, parallel (fast ~30 min; default suites: sdk+api+cli)
#
# Within each suite, Go's test framework handles internal parallelism:
#   - Tests WITH t.Parallel() run concurrently (unique wallets, no global state)
#   - Tests WITHOUT t.Parallel() run sequentially (SC config, kill tests, etc.)
#
# DATA POLICY: Re-runs and --rerun-failed NEVER delete any data.
# Result files accumulate (suite_run1.txt, suite_run2.txt, ...).
# Only --full-retest clears result files for the selected suites.
# Only 'deploy_local.sh redeploy' wipes chain data.
#
# Usage:
#   ./run_tests.sh                    # All suites in parallel (default)
#   ./run_tests.sh --smoke            # Smoke run: sdk+api+cli parallel, ~30 min
#   ./run_tests.sh --smoke api        # API smoke subtests only
#   ./run_tests.sh --long             # All suites sequential (thorough, ~2-3 h)
#   ./run_tests.sh api                # Run API suite only (parallel)
#   ./run_tests.sh api cli            # Run API + CLI in parallel
#   ./run_tests.sh --long api cli     # Run API then CLI sequentially
#   ./run_tests.sh --retries 3        # Retry failed tests 3 times (default: 3)
#   ./run_tests.sh --timeout 60m      # Set per-suite timeout (default: 120m)
#   ./run_tests.sh --filter TestName  # Run only tests matching filter
#   ./run_tests.sh --rerun-failed     # Re-run only tests that failed in the last run
#   ./run_tests.sh --full-retest      # Clear result files then run full suite from scratch

set -o pipefail

# Ensure common binary paths are in PATH (needed for tmux/non-login shells)
export PATH="$PATH:/usr/local/go/bin:/root/go/bin:/usr/local/bin"

# Set up automatic logging: all stdout/stderr goes to both terminal AND log file.
RUN_TESTS_LOG="${RUN_TESTS_LOG:-/tmp/run_tests.log}"
if [ -z "$_RUN_TESTS_LOGGING_SET" ]; then
    export _RUN_TESTS_LOGGING_SET=1
    exec > >(stdbuf -oL tee -a "$RUN_TESTS_LOG") 2>&1
    echo "" > "$RUN_TESTS_LOG"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYSTEM_TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESULTS_DIR="${SYSTEM_TEST_DIR}/test_results"

# Defaults
MAX_RETRIES=0
SUITE_TIMEOUT="90m"
TEST_TIMEOUT=""
TEST_FILTER=""
SUITES=()
RERUN_FAILED=false
FULL_RETEST=false
SMOKE_TEST=false   # --smoke: run only smoke subtests (SMOKE_TEST_MODE=true)
LONG_MODE=false    # --long: run suites sequentially instead of in parallel
LONG_TESTS=false   # --long-tests: run ONLY long-running tests (excluded from standard runs)

# Long-running tests excluded from standard runs (regex patterns for go test -skip).
# These tests take 30+ minutes each and are run separately via --long-tests.
# Format: associative array keyed by suite name, values are '|'-separated regex patterns.
declare -A LONG_TEST_PATTERNS=(
    [api]="Test0boxGraphAndTotalEndpoints|Test0boxGraphBlobberEndpoints|TestProtocolChallengeTimings|Test1ChimneyBlobberRewards|TestMultiOperation/Multi_upload_operations_of_(single|multiple)_format"
    [cli]="TestBlobberStakedCapacity"
    [tokenomics]="TestAddOrReplace|TestBlobberSlashPenalty|TestBlobberChallengeReward|TestBlobberRewardOnDownload"
)

# Tests permanently excluded from all runs. These require infrastructure not present
# in local/dev environments (Tenderly bridge, external cloud storage credentials,
# Firebase auth) or are destructive (kill tests destroy providers on-chain).
declare -A EXCLUDED_PATTERNS=(
    [api]="Test0BoxNFT|TestFileReferencePath|Test0BoxJWT|Test0BoxTransactions|TestZauthOperations"
    [cli]="TestKillBlobber|TestKillSharder|TestKillMiner|Test0Dropbox|Test0Gdrive|Test0S3Migration|TestLivestreamDownload|TestStreamUploadDownload|TestMaxFileSize|TestLFBSharderSync|TestRestrictedBlobbers|TestUpdateGlobalConfig|TestSharderFeeRewards"
    [tokenomics]="TestBlobberReadReward"
    [zs3]="TestWarpAnalysis"
)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $(date +%H:%M:%S) $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $(date +%H:%M:%S) $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $(date +%H:%M:%S) $1"; }
log_header() {
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --retries)
                MAX_RETRIES="$2"; shift 2 ;;
            --timeout)
                SUITE_TIMEOUT="$2"; shift 2 ;;
            --test-timeout)
                TEST_TIMEOUT="$2"; shift 2 ;;
            --filter)
                TEST_FILTER="$2"; shift 2 ;;
            --rerun-failed)
                RERUN_FAILED=true; shift ;;
            --full-retest)
                FULL_RETEST=true; shift ;;
            --smoke)
                # Smoke mode: run only smoke subtests (fast ~30 min).
                # If no explicit suites given, defaults to sdk+api+cli.
                SMOKE_TEST=true
                SUITE_TIMEOUT="35m"
                shift ;;
            --long)
                # Sequential mode: suites run one after another.
                # Still skips long-running tests (use --long-tests for those).
                LONG_MODE=true
                shift ;;
            --long-tests)
                # Run ONLY the long-running tests (excluded from standard runs).
                LONG_TESTS=true
                SUITE_TIMEOUT="180m"
                LONG_MODE=true   # long tests must run sequentially
                shift ;;
            --help|-h)
                usage; exit 0 ;;
            api|cli|tokenomics|sdk|zs3|mc|rclone|cypress)
                SUITES+=("$1"); shift ;;
            *)
                log_error "Unknown argument: $1"; usage; exit 1 ;;
        esac
    done

    # Default suite selection
    if [ ${#SUITES[@]} -eq 0 ]; then
        if $SMOKE_TEST; then
            # --smoke with no explicit suites → fast coverage: sdk + api + cli
            SUITES=("sdk" "api" "cli")
        else
            # Full run: all suites
            SUITES=("sdk" "mc" "rclone" "zs3" "cli" "api")
        fi
    fi
}

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] [SUITES...]

Suites: api, cli, tokenomics, sdk, zs3, mc, rclone, cypress

Execution modes:
  Default      — All suites run IN PARALLEL simultaneously (fast; ~1 hour)
                 Long-running tests (800min+) are auto-skipped.
  --smoke      — Smoke subtests only, parallel (fast ~30 min); default suites: sdk+api+cli
  --long       — All suites run SEQUENTIALLY one after another (thorough; ~2-3 h)
  --long-tests — Run ONLY the long-running tests that are skipped in standard mode

Within each suite, Go's test framework handles internal parallelism:
  • Tests WITH t.Parallel() run concurrently (unique wallets, isolated state)
  • Tests WITHOUT t.Parallel() always run sequentially (SC config, kill tests, etc.)

DATA POLICY: Re-runs and --rerun-failed NEVER delete any data — chain state
(allocations, wallets, blobbers) AND test result files are all preserved.
Result files accumulate (run1.txt, run2.txt, ...).
Only --full-retest clears the result files for the suites being run.
Only 'deploy_local.sh redeploy' wipes chain data.

Options:
  --retries N        Max retries for failed tests (default: 3)
  --timeout DURATION Per-suite timeout (default: 65m)
  --test-timeout DUR Per-test timeout override
  --filter PATTERN   Only run tests matching pattern
  --smoke            Smoke mode: smoke subtests only (SMOKE_TEST_MODE=true, 35m timeout)
  --long             Sequential mode: suites run one after another (most thorough)
  --long-tests       Run ONLY long-running tests (skipped in standard mode, 180m timeout)
  --rerun-failed     Re-run only tests that failed in the most recent run
  --full-retest      Clear result files for selected suites, then run fresh
  -h, --help         Show this help

Examples:
  $(basename "$0")                          # All suites in parallel (~1h, long tests skipped)
  $(basename "$0") --smoke                  # Smoke run: sdk+api+cli parallel, ~30 min
  $(basename "$0") --smoke api              # API smoke subtests only
  $(basename "$0") --long                   # All suites sequential, full ~2-3 h
  $(basename "$0") --long api cli           # API then CLI sequentially
  $(basename "$0") --long-tests             # Run ONLY long tests (graph, challenge, etc.)
  $(basename "$0") --long-tests api         # Run only long API tests
  $(basename "$0") api cli                  # API + CLI in parallel (no smoke)
  $(basename "$0") --retries 3 api          # API with up to 3 retries
  $(basename "$0") --filter TestCreate cli  # CLI tests matching "TestCreate"
  $(basename "$0") --rerun-failed           # Re-run all previously failed tests
  $(basename "$0") --rerun-failed api cli   # Re-run failed API and CLI tests only
  $(basename "$0") --full-retest            # Clear results then run all suites
EOF
}

# Get test directory for a suite
suite_dir() {
    case "$1" in
        api)        echo "${SYSTEM_TEST_DIR}/tests/api_tests" ;;
        cli)        echo "${SYSTEM_TEST_DIR}/tests/cli_tests" ;;
        tokenomics) echo "${SYSTEM_TEST_DIR}/tests/tokenomics_tests" ;;
        sdk)        echo "${SYSTEM_TEST_DIR}/tests/sdk_tests" ;;
        zs3)        echo "${SYSTEM_TEST_DIR}/tests/cli_tests/zs3server_tests" ;;
        mc)         echo "${SYSTEM_TEST_DIR}/tests/cli_tests/mc_tests" ;;
        rclone)     echo "${SYSTEM_TEST_DIR}/tests/cli_tests/rclone_zus_tests" ;;
        cypress)    echo "${SYSTEM_TEST_DIR}" ;; # special handling
    esac
}

# Check if a suite is a Go test suite (vs Cypress which uses npm)
is_go_suite() {
    case "$1" in
        cypress) return 1 ;;
        *) return 0 ;;
    esac
}

# Extract failed test names from go test output
extract_failed_tests() {
    local output_file="$1"
    # Match "--- FAIL: TestName" or "--- FAIL: TestParent/SubTest"
    # Use -a to handle files with ANSI escape codes, strip color codes first
    sed 's/\x1b\[[0-9;]*m//g' "$output_file" 2>/dev/null | \
        grep -oP '^--- FAIL: \K\S+' 2>/dev/null | \
        # Get top-level test names only (for -run flag)
        sed 's|/.*||' | \
        sort -u
}

# Extract test counts from output
# Counts TOP-LEVEL tests only (no "/" in name). Subtests are NOT counted separately
# to avoid double-counting (e.g. TestFoo + TestFoo/Sub1 + TestFoo/Sub2 = 1, not 3).
count_results() {
    local output_file="$1"
    local clean
    clean=$(sed 's/\x1b\[[0-9;]*m//g' "$output_file" 2>/dev/null)
    local passed; passed=$(echo "$clean" | grep -oP '^--- PASS: \K\S+' | grep -cv '/' 2>/dev/null) || passed=0
    local failed; failed=$(echo "$clean" | grep -oP '^--- FAIL: \K\S+' | grep -cv '/' 2>/dev/null) || failed=0
    local skipped; skipped=$(echo "$clean" | grep -oP '^--- SKIP: \K\S+' | grep -cv '/' 2>/dev/null) || skipped=0
    echo "$passed $failed $skipped"
}

# Append a run entry to test_results/runs_meta.json
# Usage: update_runs_meta <suite> <run_num> <mode> <output_file>
update_runs_meta() {
    local suite="$1"
    local run_num="$2"
    local mode="$3"
    local output_file="$4"
    local meta_file="${RESULTS_DIR}/runs_meta.json"
    local timestamp
    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

    local counts
    counts=$(count_results "$output_file")
    local passed; passed=$(echo "$counts" | awk '{print $1}')
    local failed; failed=$(echo "$counts" | awk '{print $2}')
    local skipped; skipped=$(echo "$counts" | awk '{print $3}')
    local total=$((passed + failed + skipped))

    # Build JSON array of failed test names
    local failed_tests_json="[]"
    local failed_names
    failed_names=$(extract_failed_tests "$output_file")
    if [ -n "$failed_names" ]; then
        failed_tests_json="["
        local first=true
        while IFS= read -r tname; do
            [ -z "$tname" ] && continue
            local escaped
            escaped=$(echo "$tname" | sed 's/"/\\"/g')
            if $first; then
                failed_tests_json="${failed_tests_json}\"${escaped}\""
                first=false
            else
                failed_tests_json="${failed_tests_json},\"${escaped}\""
            fi
        done <<< "$failed_names"
        failed_tests_json="${failed_tests_json}]"
    fi

    local entry
    entry=$(printf '{"suite":"%s","run":%d,"timestamp":"%s","mode":"%s","passed":%d,"failed":%d,"skipped":%d,"total":%d,"failed_tests":%s}' \
        "$suite" "$run_num" "$timestamp" "$mode" "$passed" "$failed" "$skipped" "$total" "$failed_tests_json")

    # Append to JSON array (read existing, append, write back)
    if [ -f "$meta_file" ]; then
        # Strip trailing ']', append new entry, close array
        local existing
        existing=$(sed '$ s/]$//' "$meta_file")
        printf '%s,%s]' "$existing" "$entry" > "$meta_file"
    else
        printf '[%s]' "$entry" > "$meta_file"
    fi
}

# Pre-suite setup hooks: called once before the first attempt of a suite.
# Ensures external dependencies (ZS3 allocation, etc.) are valid.
pre_suite_setup() {
    local suite="$1"
    local deploy_sh="${SCRIPT_DIR}/deploy_local.sh"
    case "$suite" in
        zs3|mc|rclone)
            # Ensure the ZS3 server has a valid (non-expired) allocation and is running.
            # Uses deploy_local.sh renew-zs3 command which checks expiry and restarts if needed.
            # flock prevents concurrent calls (zs3, mc, and rclone share the same ZS3 server).
            if [ -f "$deploy_sh" ]; then
                log_info "[${suite}] Checking ZS3 server allocation..."
                (
                    flock -x 200
                    bash "$deploy_sh" renew-zs3 2>&1 | while IFS= read -r line; do
                        log_info "[zs3-setup] $line"
                    done || log_warn "[${suite}] ZS3 renewal returned non-zero (tests may fail if allocation expired)"
                ) 200>"/tmp/zs3_setup.lock"
            else
                log_warn "[${suite}] deploy_local.sh not found — cannot auto-renew ZS3 allocation"
            fi
            # For rclone suite: also ensure rclone-zus binary is available
            if [ "$suite" = "rclone" ]; then
                local cli_dir="${SYSTEM_TEST_DIR}/tests/cli_tests"
                if [ ! -f "${cli_dir}/rclone-zus" ] && [ -f "$deploy_sh" ]; then
                    log_info "[rclone] Building rclone-zus binary..."
                    bash "$deploy_sh" rclone-zus 2>&1 | while IFS= read -r line; do
                        log_info "[rclone-setup] $line"
                    done || log_warn "[rclone] rclone-zus build failed — tests will skip"
                fi
            fi
            ;;
    esac
}

# Run a single test suite
run_suite() {
    local suite="$1"
    local attempt="$2"
    local test_filter="$3"
    local dir=$(suite_dir "$suite")
    local output_file="${RESULTS_DIR}/${suite}_run${attempt}.txt"
    # Also write to /tmp for nginx serving
    local live_log="/tmp/test_${suite}.log"

    if [ ! -d "$dir" ]; then
        log_error "Suite directory not found: $dir"
        return 1
    fi

    # Run pre-suite setup on first attempt; for zs3/mc also on retries (fresh allocation needed
    # so retry tests don't fail due to stale allocation root from previous passing tests).
    if [ "$attempt" -eq 1 ] || [[ "$suite" =~ ^(zs3|mc|rclone)$ ]]; then
        pre_suite_setup "$suite"
    fi

    if ! is_go_suite "$suite"; then
        # Cypress suite - special handling
        # Call directly (not via $(...)) so log_info output goes to terminal, not captured as return value
        run_cypress_suite "$attempt" "$test_filter" "$output_file"
        return $?
    fi

    # Build go test command
    local cmd="go test -v -timeout ${SUITE_TIMEOUT} -count=1"

    if [ -n "$test_filter" ]; then
        cmd="$cmd -run '${test_filter}'"
    fi

    # Build combined skip pattern from long tests + permanently excluded tests.
    local long_pattern="${LONG_TEST_PATTERNS[$suite]:-}"
    local excluded_pattern="${EXCLUDED_PATTERNS[$suite]:-}"
    local skip_pattern=""

    # Always skip excluded tests (even on retries — they can never pass).
    if [ -n "$excluded_pattern" ]; then
        skip_pattern="$excluded_pattern"
    fi

    # Long test handling: skip or run-only based on mode.
    # When retrying specific failed tests (test_filter set), don't add -skip
    # since the retry already targets only the specific tests that failed.
    if [ -n "$long_pattern" ]; then
        if $LONG_TESTS; then
            # --long-tests mode: run ONLY the long tests (unless already filtered by retry)
            if [ -z "$test_filter" ]; then
                cmd="$cmd -run '${long_pattern}'"
            fi
        elif [ -z "$test_filter" ]; then
            # Standard mode: also skip long tests
            if [ -n "$skip_pattern" ]; then
                skip_pattern="${skip_pattern}|${long_pattern}"
            else
                skip_pattern="$long_pattern"
            fi
        fi
    fi

    if [ -n "$skip_pattern" ]; then
        cmd="$cmd -skip '${skip_pattern}'"
    fi

    if [ -n "$TEST_TIMEOUT" ]; then
        cmd="$cmd -test.timeout ${TEST_TIMEOUT}"
    fi

    # Use '.' (current package only) for main suites to avoid running sub-packages
    # (e.g., cli_tests has zs3server_tests/, mc_tests/, rclone_zus_tests/ sub-dirs)
    cmd="$cmd ."

    # Smoke mode: set env var so tests can skip non-smoke subtests
    local env_prefix=""
    $SMOKE_TEST && env_prefix="SMOKE_TEST_MODE=true "

    log_info "[${suite}] Run #${attempt}: ${env_prefix}${cmd}"

    # Execute (write to both results dir and live log)
    cd "$dir"
    eval "${env_prefix}${cmd}" 2>&1 | tee "$output_file" > "$live_log"
    local exit_code=${PIPESTATUS[0]}

    # Parse results
    local counts
    counts=$(count_results "$output_file")
    local passed=$(echo "$counts" | awk '{print $1}')
    local failed=$(echo "$counts" | awk '{print $2}')
    local skipped=$(echo "$counts" | awk '{print $3}')

    if [ $exit_code -eq 0 ]; then
        log_info "[${suite}] Run #${attempt}: ${GREEN}ALL PASSED${NC} (${passed} passed, ${skipped} skipped)"
    else
        log_warn "[${suite}] Run #${attempt}: ${passed} passed, ${RED}${failed} failed${NC}, ${skipped} skipped"
    fi

    # Record metadata for this run (mode is passed via RUN_MODE env var set by caller)
    update_runs_meta "$suite" "$attempt" "${RUN_MODE:-initial}" "$output_file"

    return $exit_code
}

# Run Cypress test suite
run_cypress_suite() {
    local attempt="$1"
    local test_filter="$2"
    local output_file="$3"
    local live_log="/tmp/test_cypress.log"
    local cypress_dir="${SYSTEM_TEST_DIR}/../web-apps/packages/test"

    if [ ! -d "$cypress_dir" ]; then
        log_error "[cypress] Test directory not found: $cypress_dir"
        return 1
    fi

    local cmd="npx cypress run --browser chrome --headless"
    if [ -n "$test_filter" ]; then
        cmd="$cmd --spec '**/*${test_filter}*'"
    fi

    log_info "[cypress] Run #${attempt}: ${cmd}"

    cd "$cypress_dir"
    eval "$cmd" 2>&1 | tee "$output_file" > "$live_log"
    return ${PIPESTATUS[0]}
}

# Run a suite with retries
run_suite_with_retries() {
    local suite="$1"
    local dir=$(suite_dir "$suite")

    log_header "Running ${suite^^} Tests"

    # Use next available run number so we never overwrite existing results.
    # Result files accumulate: suite_run1.txt, suite_run2.txt, ...
    # Only --full-retest explicitly clears them before starting.
    local start_run
    start_run=$(next_run_number "$suite")

    # Run 1: full suite (or filtered)
    local filter="${TEST_FILTER}"
    # Use _INITIAL_MODE if set (e.g. "full_retest"), otherwise "initial"
    RUN_MODE="${_INITIAL_MODE:-initial}" run_suite "$suite" "$start_run" "$filter"
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        return 0
    fi

    # Retry failed tests
    local attempt=$((start_run + 1))
    while [ $attempt -le $((start_run + MAX_RETRIES)) ]; do
        local prev_output="${RESULTS_DIR}/${suite}_run$((attempt - 1)).txt"
        local failed_tests
        failed_tests=$(extract_failed_tests "$prev_output")

        if [ -z "$failed_tests" ]; then
            log_warn "[${suite}] No specific failed tests found to retry (possible timeout/panic)"
            # On timeout/panic, re-run the full suite (or same filter)
            log_info "[${suite}] Re-running full suite for attempt ${attempt}..."
            RUN_MODE="rerun_failed" run_suite "$suite" "$attempt" "$filter"
            exit_code=$?
            if [ $exit_code -eq 0 ]; then
                log_info "[${suite}] Full re-run passed on attempt ${attempt}"
                break
            fi
            attempt=$((attempt + 1))
            continue
        fi

        # Build regex filter for failed tests
        local retry_filter
        retry_filter=$(echo "$failed_tests" | tr '\n' '|' | sed 's/|$//')

        local fail_count
        fail_count=$(echo "$failed_tests" | wc -l | tr -d ' ')
        local retry_num=$((attempt - start_run))
        log_info "[${suite}] Retrying ${fail_count} failed test(s) (retry ${retry_num}/${MAX_RETRIES})..."

        RUN_MODE="rerun_failed" run_suite "$suite" "$attempt" "$retry_filter"
        exit_code=$?

        if [ $exit_code -eq 0 ]; then
            log_info "[${suite}] All previously failed tests now pass on retry #$((attempt - 1))"
            break
        fi

        attempt=$((attempt + 1))
    done

    return $exit_code
}

# Generate final summary report
generate_report() {
    local report_file="${RESULTS_DIR}/summary.txt"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    log_header "Test Results Summary"

    {
        echo "============================================"
        echo "  0Chain System Test Report"
        echo "  Generated: ${timestamp}"
        echo "============================================"
        echo ""
    } > "$report_file"

    local total_passed=0
    local total_failed=0
    local total_skipped=0
    local all_passed=true

    for suite in "${SUITES[@]}"; do
        # Find the last run file for this suite
        local last_run
        last_run=$(ls -1 "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null | sort -V | tail -1)

        if [ -z "$last_run" ]; then
            echo -e "  ${YELLOW}${suite^^}${NC}: No results found"
            echo "  ${suite^^}: No results found" >> "$report_file"
            continue
        fi

        # Use cumulative best-result counts across ALL run files (not just the last)
        local counts
        read passed failed skipped <<< $(_cumulative_counts "$suite")
        local num_runs=$(ls -1 "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null | wc -l | tr -d ' ')

        total_passed=$((total_passed + passed))
        total_failed=$((total_failed + failed))
        total_skipped=$((total_skipped + skipped))

        # Detect failure: any test still failing after all reruns, or FAIL in final run (timeout/panic)
        local suite_failed=false
        if [ "$failed" -gt 0 ]; then
            suite_failed=true
        elif tail -5 "$last_run" | grep -q '^FAIL'; then
            suite_failed=true
            total_failed=$((total_failed + 1))
        fi

        local status_color="${GREEN}"
        local status_text="PASS"
        if $suite_failed; then
            status_color="${RED}"
            status_text="FAIL"
            all_passed=false
        fi

        echo -e "  ${status_color}${suite^^}${NC}: ${passed} passed, ${failed} failed, ${skipped} skipped (${num_runs} run(s))"
        echo "  ${suite^^} [${status_text}]: ${passed} passed, ${failed} failed, ${skipped} skipped (${num_runs} run(s))" >> "$report_file"

        # List still-failing tests or note timeout
        if $suite_failed; then
            local still_failing
            still_failing=$(_still_failing "$suite")
            if [ -n "$still_failing" ]; then
                echo -e "    ${RED}Still failing:${NC}"
                echo "    Still failing:" >> "$report_file"
                while IFS= read -r test_name; do
                    echo -e "      - ${test_name}"
                    echo "      - ${test_name}" >> "$report_file"
                done <<< "$still_failing"
            elif [ "$failed" -eq 0 ]; then
                echo -e "    ${RED}Suite failed (timeout/panic - no individual test failures detected)${NC}"
                echo "    Suite failed (timeout/panic)" >> "$report_file"
            fi
        fi
    done

    echo ""
    echo -e "  ${BOLD}Total: ${total_passed} passed, ${total_failed} failed, ${total_skipped} skipped${NC}"
    {
        echo ""
        echo "  Total: ${total_passed} passed, ${total_failed} failed, ${total_skipped} skipped"
        echo ""
        echo "  Results directory: ${RESULTS_DIR}"
        echo "============================================"
    } >> "$report_file"

    echo ""
    if $all_passed; then
        echo -e "  ${GREEN}${BOLD}ALL SUITES PASSED${NC}"
    else
        echo -e "  ${RED}${BOLD}SOME TESTS FAILED${NC} (see ${RESULTS_DIR}/ for details)"
    fi
    echo ""

    log_info "Full report: ${report_file}"
}

# Compute BEST-result counts across ALL run files for a suite (PASS > FAIL > SKIP).
# When reruns happen, a test that passed in any run counts as PASS.
# Also checks the live /tmp/test_<suite>.log for the currently-running attempt.
# Outputs: "pass fail skip" (space-separated)
_cumulative_counts() {
    local suite="$1"
    local live="/tmp/test_${suite}.log"
    python3 -c "
import re, os, glob
PRIORITY = {'PASS': 3, 'FAIL': 2, 'SKIP': 1}
results = {}
files = sorted(glob.glob('${RESULTS_DIR}/${suite}_run*.txt'))
if os.path.exists('${live}') and '${live}' not in files:
    files.append('${live}')
for fpath in files:
    try:
        with open(fpath, 'rb') as fh:
            for line in fh:
                line = line.decode('utf-8', errors='replace')
                m = re.match(r'^--- (PASS|FAIL|SKIP): (\S+)', line)
                if m:
                    st, nm = m.group(1), m.group(2)
                    if '/' in nm:
                        continue  # skip subtests — count top-level only
                    if nm not in results or PRIORITY.get(st,0) > PRIORITY.get(results.get(nm,''),0):
                        results[nm] = st
    except: pass
p = sum(1 for v in results.values() if v == 'PASS')
f = sum(1 for v in results.values() if v == 'FAIL')
s = sum(1 for v in results.values() if v == 'SKIP')
print(p, f, s)
" 2>/dev/null || echo "0 0 0"
}

# Return test names that are STILL FAILING after all runs.
# A test is "still failing" only if its best result across ALL run files is FAIL
# (i.e. it never passed in any run). Returns top-level test names for -run filtering.
_still_failing() {
    local suite="$1"
    ls -1 "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null | head -1 | grep -q . || return
    python3 -c "
import re, glob
PRIORITY = {'PASS': 3, 'FAIL': 2, 'SKIP': 1}
results = {}
for fpath in sorted(glob.glob('${RESULTS_DIR}/${suite}_run*.txt')):
    try:
        with open(fpath, 'rb') as fh:
            for line in fh:
                line = line.decode('utf-8', errors='replace')
                m = re.match(r'^--- (PASS|FAIL|SKIP): (\S+)', line)
                if m:
                    st, nm = m.group(1), m.group(2)
                    if '/' in nm:
                        continue  # skip subtests — count top-level only
                    if nm not in results or PRIORITY.get(st,0) > PRIORITY.get(results.get(nm,''),0):
                        results[nm] = st
    except: pass
for nm, st in sorted(results.items()):
    if st == 'FAIL':
        print(nm)
" 2>/dev/null
}

# Delegate HTML generation to fix_results.py (combined summary+subtests page)
RESULTS_HTML="/var/log/0chain/test_results.html"

generate_results_snapshot() {
    mkdir -p /var/log/0chain
    python3 "${SCRIPT_DIR}/fix_results.py" 2>/dev/null || true
}

generate_results_html() {
    mkdir -p /var/log/0chain
    # Kill any existing fix_results.py loop processes before starting a new one
    pkill -f "fix_results.py --loop" 2>/dev/null || true
    sleep 0.5
    # Run fix_results.py in loop mode (regenerates every 5s).
    # Redirect to /dev/null so it doesn't pollute the test log or cause
    # double-tee buffering issues that make the log appear stale.
    python3 "${SCRIPT_DIR}/fix_results.py" --loop > /dev/null 2>&1 &
}

stop_results_generator() {
    if [ -f /tmp/results_gen.pid ]; then
        kill "$(cat /tmp/results_gen.pid)" 2>/dev/null || true
        rm -f /tmp/results_gen.pid
    fi
}

# Collect failed tests from the most recent run files for the given suites.
# Prints a filter string suitable for use as -run argument, scoped per suite.
# Sets global RERUN_SUITE_FILTERS associative-style (per-suite filter stored in
# RERUN_FILTER_<SUITE> env vars so subshells can read them).
collect_rerun_filters() {
    local suites=("$@")
    local any_found=false
    for suite in "${suites[@]}"; do
        # Find the highest-numbered run file for this suite
        local last_run
        last_run=$(ls -1 "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null | sort -V | tail -1)
        if [ -z "$last_run" ]; then
            log_warn "[${suite}] No previous run file found for --rerun-failed; will run full suite"
            continue
        fi
        local failed_tests
        failed_tests=$(extract_failed_tests "$last_run")
        if [ -z "$failed_tests" ]; then
            log_info "[${suite}] No failures in last run (${last_run}); skipping suite"
            # Mark as "no failures" so we skip entirely
            export "RERUN_FILTER_${suite^^}=__SKIP__"
            continue
        fi
        local filter
        filter=$(echo "$failed_tests" | tr '\n' '|' | sed 's/|$//')
        export "RERUN_FILTER_${suite^^}=${filter}"
        local count
        count=$(echo "$failed_tests" | wc -l | tr -d ' ')
        log_info "[${suite}] --rerun-failed: ${count} failed test(s) from ${last_run}"
        any_found=true
    done
    if ! $any_found; then
        log_warn "No failed tests found in previous run files — nothing to re-run"
    fi
}

# Determine the next available run number for a suite (max existing + 1)
next_run_number() {
    local suite="$1"
    local last_num
    last_num=$(ls -1 "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null | \
        grep -oP '_run\K[0-9]+(?=\.txt)' | sort -n | tail -1)
    echo $(( ${last_num:-0} + 1 ))
}

# Run a suite starting at a specific run number, with a given filter and mode.
# Used by --rerun-failed so the new run file gets the next sequential number.
run_suite_rerun() {
    local suite="$1"
    local run_num="$2"
    local filter="$3"
    local mode="$4"
    RUN_MODE="$mode" run_suite "$suite" "$run_num" "$filter"
}

# Rotate old result .txt files: delete any that are older than 7 days.
# These are purely local test output logs — deleting them does NOT affect
# chain state, allocations, 0box data, or any database.
rotate_old_results() {
    local count
    count=$(find "${RESULTS_DIR}" -name "*_run*.txt" -mtime +7 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -gt 0 ]; then
        find "${RESULTS_DIR}" -name "*_run*.txt" -mtime +7 -delete 2>/dev/null || true
        log_info "Log rotation: removed ${count} result file(s) older than 7 days"
    fi
}

# Main
main() {
    parse_args "$@"

    log_header "0Chain System Test Runner"
    log_info "Suites: ${SUITES[*]}"
    log_info "Max retries: ${MAX_RETRIES}"
    log_info "Suite timeout: ${SUITE_TIMEOUT}"
    [ -n "$TEST_FILTER" ] && log_info "Filter: ${TEST_FILTER}"
    $SMOKE_TEST    && log_info "Mode: --smoke (smoke subtests only, SMOKE_TEST_MODE=true)"
    $LONG_TESTS    && log_info "Mode: --long-tests (ONLY long-running tests, 180m timeout)"
    $LONG_MODE && ! $LONG_TESTS && log_info "Mode: --long (sequential — suites run one after another)"
    ! $LONG_MODE   && ! $RERUN_FAILED && log_info "Mode: parallel — all suites run simultaneously (long tests skipped)"
    $RERUN_FAILED  && log_info "Mode: --rerun-failed (re-running only previously failed tests)"
    $FULL_RETEST   && log_info "Mode: --full-retest (clearing result files for: ${SUITES[*]})"

    # Create results directory
    mkdir -p "$RESULTS_DIR"

    # Rotate result files older than 7 days (purely local logs, no chain data affected)
    rotate_old_results

    if ! $RERUN_FAILED; then
        # Fresh run (normal or --full-retest): always clear old result files for the
        # suites being run so the dashboard tally resets to 0 for the new run.
        # Chain data is NEVER touched. Other suite result files are preserved.
        log_info "Resetting result files for: ${SUITES[*]}"
        for suite in "${SUITES[@]}"; do
            rm -f "${RESULTS_DIR}/${suite}_run"*.txt 2>/dev/null || true
        done
        rm -f "${RESULTS_DIR}/runs_meta.json"
    fi
    # --rerun-failed: keep old result files so the best-per-test tally accumulates
    # (a test that passed in the first run still counts as PASS after a retry).

    # Kill any leftover go test / compiled test binaries from previous interrupted runs.
    # Does NOT touch docker containers or chain data.
    local stale_pids
    stale_pids=$(pgrep -f "_tests\.test |go test " 2>/dev/null | tr '\n' ' ') || true
    if [ -n "$stale_pids" ]; then
        log_info "Killing stale test processes: $stale_pids"
        kill $stale_pids 2>/dev/null || true
        sleep 1
    fi

    # Start live HTML results generator in background
    stop_results_generator
    generate_results_html &
    echo $! > /tmp/results_gen.pid
    log_info "Live results table: /test/results (PID: $(cat /tmp/results_gen.pid))"

    # ---- --rerun-failed mode ----
    if $RERUN_FAILED; then
        collect_rerun_filters "${SUITES[@]}"

        local pids=()
        local suite_results=()
        local active_suites=()

        for suite in "${SUITES[@]}"; do
            local var="RERUN_FILTER_${suite^^}"
            local filter="${!var}"

            if [ "${filter}" = "__SKIP__" ]; then
                log_info "[${suite}] No failures in last run — skipping"
                continue
            fi

            if [ -z "$filter" ]; then
                # No previous run file at all — treat as full suite run
                filter="${TEST_FILTER}"
            fi

            local run_num
            run_num=$(next_run_number "$suite")

            (
                run_suite_rerun "$suite" "$run_num" "$filter" "rerun_failed"
                exit $?
            ) &
            pids+=($!)
            active_suites+=("$suite")
            log_info "Started ${suite} rerun (run #${run_num}, PID: ${pids[-1]})"
        done

        if [ ${#pids[@]} -eq 0 ]; then
            log_info "No suites required re-running."
            stop_results_generator
            generate_report
            exit 0
        fi

        log_info "Waiting for ${#pids[@]} suite(s) to complete..."
        local idx=0
        for pid in "${pids[@]}"; do
            wait "$pid"
            suite_results+=($?)
            local suite="${active_suites[$idx]}"
            if [ ${suite_results[-1]} -eq 0 ]; then
                log_info "${suite^^} rerun completed: ${GREEN}PASS${NC}"
            else
                log_warn "${suite^^} rerun completed: ${RED}FAIL${NC}"
            fi
            idx=$((idx + 1))
        done

        sleep 6
        stop_results_generator
        generate_report

        for result in "${suite_results[@]}"; do
            if [ "$result" -ne 0 ]; then
                exit 1
            fi
        done
        exit 0
    fi

    # ---- Normal / --full-retest mode ----
    # For --full-retest we use "full_retest" as the mode label; otherwise "initial"
    local initial_mode="initial"
    $FULL_RETEST && initial_mode="full_retest"

    local overall_rc=0

    if $LONG_MODE; then
        # ---- Long mode: suites run SEQUENTIALLY, one after another ----
        # Most thorough — avoids any cross-suite interference.
        # Within each suite, t.Parallel() tests still run concurrently.
        log_info "Long mode: running ${#SUITES[@]} suite(s) sequentially: ${SUITES[*]}"
        for suite in "${SUITES[@]}"; do
            log_info "========== Starting ${suite^^} tests =========="
            _INITIAL_MODE="$initial_mode" run_suite_with_retries "$suite" || overall_rc=1
        done
    else
        # ---- Short mode (default): two-phase parallel ----
        # Phase 1 (warm-up): lightweight suites (sdk, mc, rclone, zs3) run in parallel.
        #   These complete quickly (~5 min) and their faucet/allocation transactions warm
        #   up the chain after a fresh deploy, ensuring sharders have processed recent blocks.
        # Phase 2 (main): heavy suites (api, cli) run in parallel after Phase 1 completes.
        #   By this point the chain has settled and sharder confirmation works reliably.
        local warmup_suites=()
        local main_suites=()
        for suite in "${SUITES[@]}"; do
            case "$suite" in
                api|cli|tokenomics) main_suites+=("$suite") ;;
                *) warmup_suites+=("$suite") ;;
            esac
        done

        if [ ${#warmup_suites[@]} -gt 0 ]; then
            log_info "Phase 1 (warm-up): ${warmup_suites[*]}"
            run_parallel_group "${warmup_suites[@]}" || overall_rc=1
        fi
        if [ ${#main_suites[@]} -gt 0 ]; then
            # Run main suites SEQUENTIALLY (CLI first, then API).
            # CLI's ProtocolChallenge sets time_unit=10m on-chain temporarily.
            # If API runs in parallel, its allocations inherit the short time_unit
            # and expire in ~10 minutes, causing Test0BoxTranscoder and other failures.
            log_info "Phase 2 (main): ${main_suites[*]} (sequential — CLI before API to avoid time_unit race)"
            for suite in "${main_suites[@]}"; do
                log_info "========== Starting ${suite^^} tests =========="
                _INITIAL_MODE="$initial_mode" run_suite_with_retries "$suite" || overall_rc=1
            done
        fi
    fi

    # After all suites finish: reset time_unit to 720h (tokenomics tests may change it)
    if [[ " ${SUITES[*]} " == *" tokenomics "* ]]; then
        local zwallet_bin="${SYSTEM_TEST_DIR}/tests/cli_tests/zwallet"
        local zcn_cfg="${SYSTEM_TEST_DIR}/tests/cli_tests/config"
        if [ -x "$zwallet_bin" ]; then
            log_info "[cleanup] Resetting time_unit=720h after tokenomics run..."
            "$zwallet_bin" sc-update-config --keys time_unit --values 720h \
                --configDir "$zcn_cfg" --config zbox_config.yaml --wallet wallets/sc_owner_wallet.json \
                --silent 2>/dev/null || log_warn "[cleanup] time_unit reset failed (non-critical)"
        fi
    fi

    # Final update of results HTML, then stop generator
    sleep 6
    stop_results_generator

    # Generate report
    generate_report

    exit $overall_rc
}

# Run a list of suites in parallel, wait for all, return worst exit code.
run_parallel_group() {
    local suites=("$@")
    [ ${#suites[@]} -eq 0 ] && return 0
    [ ${#suites[@]} -eq 1 ] && {
        # Single suite: run directly (no background needed)
        _INITIAL_MODE="$initial_mode" run_suite_with_retries "${suites[0]}"
        return $?
    }

    log_info "=== Parallel group: ${suites[*]} ==="
    local pids=()
    local group_suites=()

    for suite in "${suites[@]}"; do
        # Delay API tests 60s to let Kafka/0box sync chain data (miners, sharders, blobbers).
        # Also restart crawler at this point so it generates activity for graph/challenge tests.
        local delay=0
        if [ "$suite" = "api" ]; then
            delay=60
        fi
        (
            if [ "$delay" -gt 0 ]; then
                log_info "[${suite}] Waiting ${delay}s for 0box/Kafka sync..."
                sleep "$delay"
                # Restart crawler to ensure fresh activity for API tests (graph endpoints, challenges)
                if docker ps -a --format '{{.Names}}' | grep -q '^crawler$'; then
                    docker restart crawler >/dev/null 2>&1 && log_info "[${suite}] Crawler restarted" || true
                fi
            fi
            _INITIAL_MODE="$initial_mode" run_suite_with_retries "$suite"
            exit $?
        ) &
        pids+=($!)
        group_suites+=("$suite")
        log_info "  Started ${suite^^} in background (PID: ${pids[-1]})"
    done

    local group_rc=0
    local idx=0
    for pid in "${pids[@]}"; do
        wait "$pid"
        local rc=$?
        local suite="${group_suites[$idx]}"
        if [ $rc -eq 0 ]; then
            log_info "========== ${suite^^}: ${GREEN}PASS${NC} =========="
        else
            log_warn "========== ${suite^^}: ${RED}FAIL${NC} =========="
            group_rc=1
        fi
        idx=$((idx + 1))
    done

    return $group_rc
}

# Allow sourcing this file (e.g. from deploy_local.sh) without running main()
[[ "${BASH_SOURCE[0]}" == "${0}" ]] && main "$@"
