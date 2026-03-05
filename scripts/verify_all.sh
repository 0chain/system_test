#!/bin/bash
# 0Chain Comprehensive Verification Script
# Checks infrastructure, Vult/Blimp/Bolt web apps, Atlus data, CORS, blobber _stats,
# stake/rewards/challenges, crawler, and all test suites.
#
# ALL FIXES go into deploy_local.sh or test code — this script never makes
# manual one-off changes. It calls known deploy_local.sh commands to remediate.
#
# Usage:
#   bash scripts/verify_all.sh               # All checks (no test suites)
#   bash scripts/verify_all.sh --tests       # All checks + run all test suites
#   bash scripts/verify_all.sh --tests api   # All checks + run API tests only
#   bash scripts/verify_all.sh --fix         # Auto-remediate known issues then verify
#   bash scripts/verify_all.sh --section chain|services|cors|webapps|vult|blimp|bolt|atlus|blobbers
#
# Exit codes: 0 = all pass, 1 = one or more checks failed

set -o pipefail
export PATH="$PATH:/usr/local/go/bin:/root/go/bin:/usr/local/bin"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYSTEM_TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ── Shared config (mirrors deploy_local.sh) ───────────────────────────────────
BASE_DIR="${HOME}/Code"
ZCN_CONFIG_DIR="${HOME}/.zcn"
ZCN_CONFIG_FILE="local.yaml"
ZCN_WALLET_FILE="owner.json"
ZWALLET="${BASE_DIR}/zwalletcli/zwallet"
ZBOX="${BASE_DIR}/zboxcli/zbox"
SHARDER_URL="http://127.0.0.1:7171"
# Auto-select the sharder with the highest round (in case one is stale)
_s1r=$(curl -s --max-time 3 "http://127.0.0.1:7171/v1/chain/get/stats" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("current_round",0))' 2>/dev/null || echo 0)
_s2r=$(curl -s --max-time 3 "http://127.0.0.1:7172/v1/chain/get/stats" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("current_round",0))' 2>/dev/null || echo 0)
if [ "${_s2r:-0}" -gt "${_s1r:-0}" ]; then SHARDER_URL="http://127.0.0.1:7172"; fi
STORAGE_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
MINER_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
FAUCET_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"

# Determine domain + app prefix from local config (mirrors deploy_local.sh logic)
LOCAL_CONFIG="${SCRIPT_DIR}/deploy_config.local.yaml"
NGINX_DOMAIN="test.zus.network"
APP_DOMAIN_PREFIX="test"
if [ -f "$LOCAL_CONFIG" ]; then
    _d=$(grep "^  domain:" "$LOCAL_CONFIG" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    [ -n "$_d" ] && NGINX_DOMAIN="$_d"
    _p=$(grep "^  domain_prefix:" "$LOCAL_CONFIG" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    [ -n "$_p" ] && APP_DOMAIN_PREFIX="$_p"
fi
# Export so Python subprocesses (process substitutions) can read via os.environ
export NGINX_DOMAIN APP_DOMAIN_PREFIX
# Derive prefix from domain if not explicitly set
if [ "$APP_DOMAIN_PREFIX" = "test" ] && [ -n "$NGINX_DOMAIN" ]; then
    APP_DOMAIN_PREFIX="${NGINX_DOMAIN%%.*}"
fi

OBOX_URL="https://0box.${NGINX_DOMAIN}"
ZAUTH_URL="https://zauth.${NGINX_DOMAIN}"
ZVAULT_URL="https://zvault.${NGINX_DOMAIN}"

# Web app custom domains
VULT_DOMAIN="${APP_DOMAIN_PREFIX}.vult.network"
BOLT_DOMAIN="${APP_DOMAIN_PREFIX}.bolt.holdings"
BLIMP_DOMAIN="${APP_DOMAIN_PREFIX}.blimp.software"
ATLUS_DOMAIN="${APP_DOMAIN_PREFIX}.atlus.cloud"

# Web app local ports (must match deploy_local.sh APP_PORTS)
declare -A APP_PORTS=( [vult]=3003 [bolt]=3002 [blimp]=3006 [explorer]=3001 )

# ── Globals ───────────────────────────────────────────────────────────────────
PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0
FAILED_CHECKS=()

RUN_TESTS=false
TEST_SUITES=()
AUTO_FIX=false
RUN_SECTION=""

# ── Helpers ───────────────────────────────────────────────────────────────────
log_header() {
    echo ""
    echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
}
log_pass()  { echo -e "  ${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT+1)); }
log_fail()  { echo -e "  ${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT+1)); FAILED_CHECKS+=("$1"); }
log_warn()  { echo -e "  ${YELLOW}[WARN]${NC} $1"; WARN_COUNT=$((WARN_COUNT+1)); }
log_info()  { echo -e "  ${CYAN}[INFO]${NC} $1"; }
log_fix()   { echo -e "  ${YELLOW}[FIX ]${NC} $1"; }

curl_json() { curl -s -m "${2:-10}" "$1" 2>/dev/null; }
curl_ok()   { curl -s -o /dev/null -w '%{http_code}' -m "${2:-5}" "$1" 2>/dev/null; }

# Check CORS for a single origin → endpoint pair
# Usage: _check_cors_pair <ep_label> <url> <origin>
_check_cors_pair() {
    local label="$1" url="$2" origin="$3"
    local resp
    resp=$(curl -sk -X OPTIONS "$url" \
        -H "Origin: $origin" \
        -H "Access-Control-Request-Method: GET" \
        -H "Access-Control-Request-Headers: Content-Type,Authorization,X-App-Client-ID,X-APP-TYPE" \
        -D - -o /dev/null -m 8 2>/dev/null)
    local acao; acao=$(echo "$resp" | grep -i 'Access-Control-Allow-Origin:' | tr -d '\r' | awk '{print $2}')
    local acac; acac=$(echo "$resp" | grep -i 'Access-Control-Allow-Credentials:' | tr -d '\r' | awk '{print $2}')
    if [ "$acao" = "$origin" ] && [ "$acac" = "true" ]; then
        log_pass "CORS $label ← $(basename $origin): origin=$acao credentials=true"
    elif echo "$acao" | grep -q "$origin"; then
        log_pass "CORS $label ← $(basename $origin): OK"
    elif [ "$acao" = "*" ]; then
        log_warn "CORS $label ← $(basename $origin): wildcard '*' — credentialed requests will fail"
    elif [ -z "$acao" ]; then
        log_fail "CORS $label ← $(basename $origin): NO header — Fix: bash scripts/deploy_local.sh nginx"
    else
        log_fail "CORS $label ← $(basename $origin): wrong origin '$acao' — Fix: bash scripts/deploy_local.sh nginx"
    fi
}

# Check if an app .env or env-related config has dev/wrong domain references
_check_env_urls() {
    local app="$1"
    local env_file="$2"
    if [ ! -f "$env_file" ]; then
        log_warn "$app: .env not found at $env_file"
        return
    fi
    # Check for stale dev/prod domain URLs
    local bad
    bad=$(grep -E "dev\.0chain\.net|dev\.zus\.network|devtest\.zus\.network|prod\.zus\.network" "$env_file" 2>/dev/null | grep -v "^#" | head -5)
    if [ -z "$bad" ]; then
        log_pass "$app: .env has no stale dev/prod URLs"
    else
        log_fail "$app: .env contains wrong domain URLs:"
        echo "$bad" | while IFS= read -r l; do log_info "    $l"; done
        log_info "Fix: rebuild web apps — bash scripts/deploy_local.sh web-apps"
    fi
    # Check 0box URL in .env points to our server
    local obox_url_in_env
    obox_url_in_env=$(grep -E "NEXT_PUBLIC.*0BOX|NEXT_PUBLIC.*ZUS_API|0BOX_URL|ZUS_API" "$env_file" 2>/dev/null | grep -v "^#" | head -3)
    if [ -n "$obox_url_in_env" ]; then
        if echo "$obox_url_in_env" | grep -q "$NGINX_DOMAIN"; then
            log_pass "$app: .env 0box URL points to $NGINX_DOMAIN"
        else
            log_warn "$app: .env 0box URL may not point to test server: $obox_url_in_env"
        fi
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tests)
                RUN_TESTS=true; shift
                while [[ $# -gt 0 ]] && [[ "$1" != --* ]]; do
                    TEST_SUITES+=("$1"); shift
                done
                ;;
            --fix)     AUTO_FIX=true; shift ;;
            --section) RUN_SECTION="$2"; shift 2 ;;
            --help|-h)
                echo "Usage: $0 [--tests [suites...]] [--fix] [--section <name>]"
                echo "Sections: chain services cors webapps vult blimp bolt atlus blobbers"
                exit 0 ;;
            *) echo "Unknown arg: $1"; exit 1 ;;
        esac
    done
    if [ ${#TEST_SUITES[@]} -eq 0 ] && $RUN_TESTS; then
        TEST_SUITES=(api cli tokenomics sdk zs3 mc rclone)
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 1: Chain Health
# ══════════════════════════════════════════════════════════════════════════════
check_chain_health() {
    log_header "Chain Health"

    for container in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2; do
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            local rc; rc=$(docker inspect "$container" --format '{{.RestartCount}}' 2>/dev/null || echo 0)
            if   [ "$rc" -gt 2 ]; then log_fail "$container CRASH-LOOPING (restarts=$rc) — docker logs $container"
            elif [ "$rc" -gt 0 ]; then log_warn "$container restarted $rc time(s)"
            else                        log_pass "$container running (restarts=0)"
            fi
        else
            log_fail "$container NOT RUNNING"
        fi
    done

    local s1_round=0
    s1_round=$(curl_json "http://127.0.0.1:7171/v1/chain/get/stats" | \
        python3 -c "import sys,json; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo 0)
    local s2_round=0
    s2_round=$(curl_json "http://127.0.0.1:7172/v1/chain/get/stats" | \
        python3 -c "import sys,json; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo 0)

    [ "$s1_round" -gt 0 ] && log_pass "Sharder-1 responding (round $s1_round)" || log_fail "Sharder-1 NOT responding"
    [ "$s2_round" -gt 0 ] && log_pass "Sharder-2 responding (round $s2_round)" || log_fail "Sharder-2 NOT responding"

    sleep 3
    # Check the best (highest-round) sharder to confirm chain is advancing
    local _best_url="http://127.0.0.1:7171"
    local _best_round=$s1_round
    [ "${s2_round:-0}" -gt "${s1_round:-0}" ] && _best_url="http://127.0.0.1:7172" && _best_round=$s2_round
    local s_r2
    s_r2=$(curl_json "${_best_url}/v1/chain/get/stats" | \
        python3 -c "import sys,json; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo 0)
    if [ "$s_r2" -gt "$_best_round" ]; then
        log_pass "Chain advancing (round $_best_round → $s_r2)"
    elif [ "$_best_round" -gt 0 ]; then
        log_fail "Chain STUCK at round $_best_round (neither sharder advancing)"
    fi
    # Warn if one sharder is significantly behind the other (>1000 rounds)
    if [ "${s1_round:-0}" -gt 0 ] && [ "${s2_round:-0}" -gt 0 ]; then
        local _diff=$(( s2_round - s1_round ))
        [ "$_diff" -lt 0 ] && _diff=$(( s1_round - s2_round ))
        [ "$_diff" -gt 1000 ] && log_warn "Sharder round gap: S1=$s1_round vs S2=$s2_round (diff=$_diff — one sharder may be catching up)"
    fi

    local miner_ok=0
    for i in 1 2 3 4; do
        local r; r=$(curl_json "http://198.18.0.7${i}:707${i}/v1/chain/get/stats" 3 | \
            python3 -c "import sys,json; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo 0)
        [ "$r" -gt 0 ] && miner_ok=$((miner_ok+1))
    done
    if [ "$miner_ok" -ge 3 ]; then log_pass "$miner_ok/4 miners responding"
    else log_fail "Only $miner_ok/4 miners responding (need ≥3)"; fi

    if docker ps --format '{{.Names}}' | grep -q "^kafka$" && \
       docker ps --format '{{.Names}}' | grep -q "0box"; then
        local obox_round
        obox_round=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT COALESCE(MAX(round),0) FROM snapshots;" 2>/dev/null | tr -d ' \n' || echo 0)
        if [ "$s1_round" -gt 0 ] && [ "$obox_round" -gt 0 ]; then
            local lag=$(( s1_round - obox_round ))
            if   [ "$lag" -lt 200 ];  then log_pass "Kafka pipeline: lag=$lag (chain=$s1_round, 0box=$obox_round)"
            elif [ "$lag" -lt 1000 ]; then log_warn "Kafka pipeline lagging: lag=$lag — bash scripts/deploy_local.sh fix-kafka"
            else
                log_fail "Kafka pipeline STALLED: lag=$lag — bash scripts/deploy_local.sh fix-kafka"
                $AUTO_FIX && { log_fix "Running fix-kafka..."; bash "${SCRIPT_DIR}/deploy_local.sh" fix-kafka; }
            fi
        else
            log_warn "Cannot determine Kafka lag (chain=$s1_round, 0box=$obox_round)"
        fi
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 2: Backend Service Health
# ══════════════════════════════════════════════════════════════════════════════
check_services() {
    log_header "Backend Service Health"

    _http() {
        local name="$1" url="$2"
        local code; code=$(curl_ok "$url")
        if [[ "$code" =~ ^(200|301|302|304|405)$ ]]; then log_pass "$name reachable (HTTP $code)"
        else log_fail "$name — HTTP $code — $url"; fi
    }

    _http "0dns"          "http://127.0.0.1:9091/network"
    _http "0box API"      "http://127.0.0.1:9081/"
    # zauth/zvault return 404 on root — check via docker ps instead
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "zauth"; then
        log_pass "zauth container running"
    else
        log_fail "zauth container NOT running — bash scripts/deploy_local.sh services"
    fi
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "zvault"; then
        log_pass "zvault container running"
    else
        log_fail "zvault container NOT running — bash scripts/deploy_local.sh services"
    fi
    _http "ZS3 server"    "http://127.0.0.1:9100/minio/health/live"
    _http "Elasticsearch" "http://127.0.0.1:9200/_cluster/health"

    # Render/Gotenberg service (used by Blimp for PDF/image carousel)
    local render_code; render_code=$(curl_ok "http://127.0.0.1:3010/health" 5)
    if [[ "$render_code" =~ ^(200|204)$ ]]; then
        log_pass "Gotenberg render service (port 3010) reachable"
    else
        log_warn "Gotenberg render service (port 3010) NOT reachable (HTTP $render_code) — PDF carousel will not work"
        log_info "Fix: bash scripts/deploy_local.sh services  (starts gotenberg container)"
    fi

    # 0box config checks
    local bw; bw=$(docker exec 0box sh -c \
        "grep 'block_worker' /0box/config/0box.yaml 2>/dev/null || grep 'block_worker' /root/Code/0box/docker.local/config/0box.yaml 2>/dev/null" \
        | awk '{print $2}' | tr -d '"' | xargs)
    if echo "$bw" | grep -qE "198\.18\.0\.100|127\.0\.0\.1|localhost"; then
        log_pass "0box block_worker → local chain ($bw)"
    elif [ -n "$bw" ]; then
        log_fail "0box block_worker → WRONG chain: $bw (should be http://198.18.0.100:9091)"
        log_info "Fix: update 0box.yaml block_worker in deploy_local.sh fix_0box_config()"
    else
        log_warn "Could not read 0box block_worker config"
    fi

    local dm; dm=$(docker inspect --format='{{range .Config.Cmd}}{{.}} {{end}}' 0box 2>/dev/null | \
        grep -oP -- '--deployment_mode \K\d+' || echo "unknown")
    if [ "$dm" = "3" ]; then
        log_pass "0box deployment_mode=3 (auth bypass — Vult/Blimp/Bolt work without Firebase)"
    else
        log_fail "0box deployment_mode=$dm (should be 3) — all web apps get 401 Unauthorized"
        log_info "Fix: set --deployment_mode 3 in 0box CMD in docker-compose.yml"
    fi

    # Crawler
    if docker ps --format '{{.Names}}' | grep -q "^crawler$"; then
        log_pass "Crawler container running"
        # Check allocation from crawler.yaml
        local crawler_yaml="${BASE_DIR}/crawler/docker.local/config/crawler.yaml"
        local alloc_id; alloc_id=$(grep -A1 "^allocations:" "$crawler_yaml" 2>/dev/null | \
            grep "^  - " | head -1 | awk '{print $2}' | tr -d ' ')
        if [ -n "$alloc_id" ]; then
            local expiry; expiry=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/allocation?allocation=${alloc_id}" | \
                python3 -c "import sys,json; print(json.load(sys.stdin).get('expiration_date',0))" 2>/dev/null || echo 0)
            local now_unix; now_unix=$(date +%s)
            if [ "${expiry:-0}" -gt $(( now_unix + 7*86400 )) ] 2>/dev/null; then
                log_pass "Crawler allocation valid (expires $(date -d @${expiry} 2>/dev/null || echo $expiry))"
            elif [ "${expiry:-0}" -gt "$now_unix" ] 2>/dev/null; then
                log_warn "Crawler allocation expires soon — bash scripts/deploy_local.sh crawler"
            else
                log_fail "Crawler allocation EXPIRED or invalid — bash scripts/deploy_local.sh crawler"
                $AUTO_FIX && { log_fix "Refreshing crawler..."; bash "${SCRIPT_DIR}/deploy_local.sh" crawler; }
            fi
        else
            log_warn "No allocation found in crawler.yaml — bash scripts/deploy_local.sh crawler"
        fi
    else
        log_fail "Crawler NOT running — bash scripts/deploy_local.sh crawler"
        $AUTO_FIX && { log_fix "Starting crawler..."; bash "${SCRIPT_DIR}/deploy_local.sh" crawler; }
    fi

    # 0box DB
    local am as ab
    am=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM miners WHERE active=true;" 2>/dev/null | tr -d ' \n' || echo 0)
    as=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM sharders WHERE active=true;" 2>/dev/null | tr -d ' \n' || echo 0)
    ab=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM blobbers WHERE not_available=false;" 2>/dev/null | tr -d ' \n' || echo 0)

    [ "${am:-0}" -gt 0 ] && log_pass "0box DB: $am active miners" || \
        { log_fail "0box DB: 0 active miners — Atlus blank"; $AUTO_FIX && bash "${SCRIPT_DIR}/deploy_local.sh" fix-kafka; }
    [ "${as:-0}" -gt 0 ] && log_pass "0box DB: $as active sharders" || log_fail "0box DB: 0 active sharders"
    [ "${ab:-0}" -gt 0 ] && log_pass "0box DB: $ab available blobbers" || \
        log_warn "0box DB: 0 available blobbers — Blimp allocation may fail"

    # Kafka pipeline health — check snapshots advancing in 0box DB
    local snap_count snap_latest
    snap_count=$(docker exec -e PGPASSWORD=zbox_server postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM snapshots;" 2>/dev/null | tr -d ' \n' || echo 0)
    snap_latest=$(docker exec -e PGPASSWORD=zbox_server postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT COALESCE(MAX(round),0) FROM snapshots;" 2>/dev/null | tr -d ' \n' || echo 0)
    if [ "${snap_count:-0}" -gt 0 ]; then
        log_pass "Kafka pipeline: $snap_count snapshots (latest round: $snap_latest)"
    else
        log_fail "Kafka pipeline: 0 snapshots in 0box DB — events not flowing"
        log_info "Fix: bash scripts/deploy_local.sh fix-kafka"
    fi

    # Crawler allocation — check it includes all regular blobbers (not enterprise)
    if docker ps --format '{{.Names}}' | grep -q "^crawler$"; then
        local crawler_yaml="${BASE_DIR}/crawler/docker.local/config/crawler.yaml"
        local crawl_alloc; crawl_alloc=$(grep -A1 "^allocations:" "$crawler_yaml" 2>/dev/null | \
            grep "^  - " | head -1 | awk '{print $2}' | tr -d ' ')
        if [ -n "$crawl_alloc" ]; then
            local crawl_blobs; crawl_blobs=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/allocation?allocation=${crawl_alloc}" | \
                python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('blobbers',[])))" 2>/dev/null || echo 0)
            local chain_regular; chain_regular=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getblobbers?limit=20" | \
                python3 -c "import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[])); print(sum(1 for b in nodes if not b.get('is_enterprise',False)))" 2>/dev/null || echo 0)
            if [ "${crawl_blobs:-0}" -ge "${chain_regular:-1}" ] 2>/dev/null; then
                log_pass "Crawler allocation has $crawl_blobs blobbers (all $chain_regular regular blobbers)"
            else
                log_warn "Crawler allocation has $crawl_blobs blobbers but $chain_regular regular blobbers on chain — recreate with all"
                log_info "Fix: bash scripts/deploy_local.sh crawler"
            fi
        fi
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 3: CORS — all three apps × all backend endpoints
# ══════════════════════════════════════════════════════════════════════════════
check_cors() {
    log_header "CORS — Vult / Blimp / Bolt"

    local -A ORIGINS=(
        ["vult"]="https://${VULT_DOMAIN}"
        ["blimp"]="https://${BLIMP_DOMAIN}"
        ["bolt"]="https://${BOLT_DOMAIN}"
        ["atlus"]="https://${ATLUS_DOMAIN}"
    )

    # Backend endpoints all apps hit
    local -A ENDPOINTS=(
        ["0box"]="${OBOX_URL}/v2/user/exist"
        ["miner01"]="https://${NGINX_DOMAIN}/miner01/"
        ["sharder01"]="https://${NGINX_DOMAIN}/sharder01/"
        ["zauth"]="${ZAUTH_URL}/"
    )

    for ep_name in "${!ENDPOINTS[@]}"; do
        for app in vult blimp bolt atlus; do
            _check_cors_pair "$ep_name" "${ENDPOINTS[$ep_name]}" "${ORIGINS[$app]}"
        done
    done

    # WebSocket (Vult real-time updates) — test via local 0box port, not Cloudflare
    local ws_code
    ws_code=$(curl -sk \
        -H 'Upgrade: websocket' -H 'Connection: Upgrade' \
        -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' \
        -H 'Sec-WebSocket-Version: 13' \
        -H "Origin: https://${VULT_DOMAIN}" \
        "http://127.0.0.1:9081/v1/ws/subscribe/user/test" \
        -o /dev/null -w '%{http_code}' -m 8 2>/dev/null) || true
    ws_code=${ws_code:-0}
    if [[ "$ws_code" =~ ^(101|200|400|404)$ ]]; then
        log_pass "WebSocket upgrade passes 0box (HTTP $ws_code)"
    else
        log_fail "WebSocket NOT working on 0box port 9081 (HTTP $ws_code) — Fix: bash scripts/deploy_local.sh services"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 4: Web App Serving — all 5 apps
# ══════════════════════════════════════════════════════════════════════════════
check_web_apps() {
    log_header "Web App Serving (Vult / Bolt / Blimp / Explorer)"

    local WEB_APPS_DIR="${BASE_DIR}/web-apps/packages"

    # Main page check for all apps
    for app in vult bolt blimp explorer; do
        local port="${APP_PORTS[$app]}"
        local code; code=$(curl_ok "http://127.0.0.1:${port}/" 5)
        if [[ "$code" =~ ^(200|301|302|304)$ ]]; then
            log_pass "$app running on port $port (HTTP $code)"
        else
            log_fail "$app NOT responding on port $port (HTTP $code)"
            log_info "Fix: bash scripts/deploy_local.sh web-apps"
        fi
    done

    # Sub-page / UI link checks — verify key pages and navigation links load
    # Each entry: "app:port:path:description"
    local ui_checks=(
        "vult:3003:/storage:Storage page (file browser)"
        "vult:3003:/settings:Settings page (manage allocations)"
        "vult:3003:/profile:Profile page (wallet details)"
        "blimp:3006:/files:Files page (file manager)"
        "bolt:3002:/wallet:Wallet page (balance + send)"
        "bolt:3002:/stake:Stake page (provider staking)"
        "explorer:3001:/miners:Miners page (Atlus miner list)"
        "explorer:3001:/sharders:Sharders page (Atlus sharder list)"
        "explorer:3001:/blobbers:Blobbers page (Atlus blobber list)"
        "explorer:3001:/transactions:Transactions page"
        "explorer:3001:/blocks:Blocks page"
    )
    local page_ok=0 page_warn=0
    for entry in "${ui_checks[@]}"; do
        local _app="${entry%%:*}"; local _rest="${entry#*:}"
        local _port="${_rest%%:*}"; _rest="${_rest#*:}"
        local _path="${_rest%%:*}"; local _desc="${_rest#*:}"
        local _code; _code=$(curl -sk -o /dev/null -w '%{http_code}' -m 6 \
            "http://127.0.0.1:${_port}${_path}" 2>/dev/null || echo 0)
        if [[ "$_code" =~ ^(200|301|302|304)$ ]]; then
            page_ok=$((page_ok+1))
        else
            log_warn "UI page ${_app}${_path} (${_desc}): HTTP $_code"
            page_warn=$((page_warn+1))
        fi
    done
    local page_total=$(( page_ok + page_warn ))
    if [ "$page_warn" -eq 0 ]; then
        log_pass "UI sub-pages: all $page_ok/$page_total pages return 200/301"
    else
        log_warn "UI sub-pages: $page_ok/$page_total OK, $page_warn returned non-200 (may need auth redirect)"
    fi

    # 0box API button-power checks — verify endpoints that drive key UI buttons
    # These are the API calls triggered by clicking buttons in the web apps.
    local _ts; _ts=$(date +%s)
    local api_ok=0 api_fail=0
    # Blobber list (powers Vult/Blimp allocation blobber selector)
    local _r; _r=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' \
        "${OBOX_URL}/v2/blobbers?limit=10&offset=0" 2>/dev/null || echo 0)
    [[ "$_r" =~ ^(200|206)$ ]] && api_ok=$((api_ok+1)) || { log_warn "0box /v2/blobbers: HTTP $_r"; api_fail=$((api_fail+1)); }

    # Provider brands (powers allocation brand selector)
    _r=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' \
        "${OBOX_URL}/v2/provider-brands" 2>/dev/null || echo 0)
    [[ "$_r" =~ ^(200|206)$ ]] && api_ok=$((api_ok+1)) || { log_warn "0box /v2/provider-brands: HTTP $_r"; api_fail=$((api_fail+1)); }

    # 0box network graph data (powers Atlus chart buttons)
    _r=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' \
        "${OBOX_URL}/v2/graph-txns-count?from=$((_ts-86400))&to=${_ts}&data-points=10" 2>/dev/null || echo 0)
    [[ "$_r" =~ ^(200|206)$ ]] && api_ok=$((api_ok+1)) || { log_warn "0box /v2/graph-txns-count: HTTP $_r"; api_fail=$((api_fail+1)); }

    # 0box total data (powers Atlus summary cards)
    _r=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' \
        "${OBOX_URL}/v2/total-stored-data" 2>/dev/null || echo 0)
    [[ "$_r" =~ ^(200|206)$ ]] && api_ok=$((api_ok+1)) || { log_warn "0box /v2/total-stored-data: HTTP $_r"; api_fail=$((api_fail+1)); }

    # Sharder chain stats (powers Explorer block counter in Atlus header)
    _r=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' \
        "${SHARDER_URL}/v1/chain/get/stats" 2>/dev/null || echo 0)
    [ "$_r" = "200" ] && api_ok=$((api_ok+1)) || { log_warn "Sharder /v1/chain/get/stats: HTTP $_r"; api_fail=$((api_fail+1)); }

    local api_total=$(( api_ok + api_fail ))
    if [ "$api_fail" -eq 0 ]; then
        log_pass "UI button API endpoints: all $api_ok/$api_total responding"
    else
        log_warn "UI button API endpoints: $api_ok/$api_total OK, $api_fail failing"
    fi

    # Check custom domain URLs serve app content
    for app_entry in "vult:${VULT_DOMAIN}" "bolt:${BOLT_DOMAIN}" "blimp:${BLIMP_DOMAIN}" "explorer:${ATLUS_DOMAIN}"; do
        local app="${app_entry%%:*}"
        local domain="${app_entry##*:}"
        local code; code=$(curl -sk -o /dev/null -w '%{http_code}' -m 8 "https://${domain}/" 2>/dev/null || echo 0)
        if [[ "$code" =~ ^(200|301|302|304)$ ]]; then
            log_pass "$app custom domain https://${domain}/ reachable (HTTP $code)"
        else
            log_warn "$app custom domain https://${domain}/ returned HTTP $code"
            log_info "Check: DNS record + nginx config for ${domain}"
        fi
    done

    # .env hardcoded URL checks
    for app in vult bolt blimp; do
        local env_file="${WEB_APPS_DIR}/${app}/.env"
        _check_env_urls "$app" "$env_file"
    done
    # Also check shared .env
    _check_env_urls "shared" "${WEB_APPS_DIR}/shared/.env"

    # Next.js process check — only warn if apps are also NOT serving HTTP
    local next_procs; next_procs=$(pgrep -c -f "node.*next" 2>/dev/null | head -1 | tr -d '[:space:]' || echo 0); next_procs=${next_procs:-0}
    local serving_count=0
    for p in 3003 3002 3006 3001; do
        curl -sk "http://127.0.0.1:${p}/" -o /dev/null -w '%{http_code}' -m 3 2>/dev/null | grep -qE '^(200|301|302)' && serving_count=$((serving_count+1)) || true
    done
    if [ "$next_procs" -gt 0 ] || [ "$serving_count" -ge 3 ]; then
        log_pass "Web app processes OK ($serving_count/4 ports serving, $next_procs Next.js procs)"
    else
        log_warn "Web app processes unclear ($serving_count/4 ports serving) — check manually"
        log_info "Fix: bash scripts/deploy_local.sh web-apps"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 5: Vult — Storage app (upload, carousel, allocation, replace blobber)
# ══════════════════════════════════════════════════════════════════════════════
check_vult() {
    log_header "Vult — Storage App Checks"

    # Vult custom domain UI/UX navigation page checks (test.vult.network)
    log_info "Checking Vult custom domain navigation pages (https://${VULT_DOMAIN})..."
    local vult_pages=(
        "/:Home (login/signup)"
        "/storage:Storage (file browser)"
        "/settings:Settings (allocations)"
        "/profile:Profile (wallet details)"
    )
    local vult_pg_ok=0 vult_pg_warn=0
    for pg in "${vult_pages[@]}"; do
        local _path="${pg%%:*}" _desc="${pg#*:}"
        local _c; _c=$(curl -sk -o /dev/null -w '%{http_code}' -m 8 \
            "https://${VULT_DOMAIN}${_path}" 2>/dev/null || echo 0)
        if [[ "$_c" =~ ^(200|301|302|304)$ ]]; then
            vult_pg_ok=$((vult_pg_ok+1))
        else
            log_warn "Vult page ${_path} (${_desc}): HTTP $_c"
            vult_pg_warn=$((vult_pg_warn+1))
        fi
    done
    local vult_pg_total=$(( vult_pg_ok + vult_pg_warn ))
    if [ "$vult_pg_warn" -eq 0 ]; then
        log_pass "Vult domain pages: all $vult_pg_ok/$vult_pg_total reachable at https://${VULT_DOMAIN}"
    else
        log_warn "Vult domain pages: $vult_pg_ok/$vult_pg_total OK, $vult_pg_warn failed — check nginx + DNS for ${VULT_DOMAIN}"
    fi

    if [ ! -x "$ZBOX" ]; then
        log_warn "zbox binary not found at $ZBOX — skipping storage operation checks"
        return
    fi
    local W="--configDir $ZCN_CONFIG_DIR --wallet $ZCN_WALLET_FILE --config $ZCN_CONFIG_FILE"

    # a. Faucet — can pour tokens
    log_info "Faucet: pouring tokens to local.json..."
    local faucet_out
    faucet_out=$($ZWALLET faucet --methodName pour --input '{"Once":true}' $W 2>&1 || true)
    if echo "$faucet_out" | grep -qiE "success|Execute faucet|Hash:"; then
        log_pass "Faucet: poured tokens successfully"
    else
        log_warn "Faucet: $(echo "$faucet_out" | tail -1)"
    fi

    # b. Twilio phone verify/signup (POST /v2/twilio/phone/verify/signup)
    # In dev/no-auth mode (deployment_mode=3), 0box skips Firebase and requires user_id.
    # The web-apps fix (user.js) sends user_id from firebaseTokens.uid or a devnet- fallback.
    local twilio_uid
    twilio_uid="verify-check-$(hostname | tr -d '.')"
    local twilio_body twilio_code
    twilio_body=$(curl -sk -m 10 -X POST \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"verify@check.local\",\"firebase_token\":\"test\",\"otp\":\"000000\",\"phone_number\":\"+19990000001\",\"username\":\"${twilio_uid}\",\"user_id\":\"${twilio_uid}\"}" \
        -w '\n__HTTP_CODE__%{http_code}' \
        "${OBOX_URL}/v2/twilio/phone/verify/signup" 2>/dev/null || echo "")
    twilio_code=$(echo "$twilio_body" | grep -oP '__HTTP_CODE__\K\d+' || echo 0)
    twilio_body=$(echo "$twilio_body" | sed 's/__HTTP_CODE__[0-9]*//')
    if [[ "$twilio_code" =~ ^(502|503|504|000)$ ]]; then
        log_fail "0box /v2/twilio/phone/verify/signup UNREACHABLE (HTTP $twilio_code) — 0box down or nginx not routing"
    elif echo "$twilio_body" | grep -qi "OTP verified successfully\|duplicate key"; then
        log_pass "Signup endpoint: working (HTTP $twilio_code — user_id accepted)"
    elif echo "$twilio_body" | grep -qi "user_id is required"; then
        log_fail "Signup: returns 'user_id is required' — web-apps fix not deployed"
        log_info "  Fix: bash scripts/deploy_local.sh web-apps"
    elif [[ "$twilio_code" =~ ^(200|400|401|403)$ ]]; then
        log_info "Signup: HTTP $twilio_code — $(echo "$twilio_body" | head -c 200)"
    else
        log_warn "Signup unexpected HTTP $twilio_code: $(echo "$twilio_body" | head -c 200)"
    fi

    # b2. Login endpoint (POST /v2/twilio/phone/verify/login)
    # In dev/no-auth mode (IsDevelopmentNoAuth), user_id bypasses Firebase — expect "OTP verified".
    local login_ts; login_ts=$(date +%s)
    local login_body login_code
    login_body=$(curl -sk -m 10 -X POST \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"verify@check.local\",\"firebase_token\":\"test\",\"otp\":\"000000\",\"phone_number\":\"+1999${login_ts: -7}\",\"user_id\":\"verify-login-${login_ts}\"}" \
        -w '\n__HTTP_CODE__%{http_code}' \
        "${OBOX_URL}/v2/twilio/phone/verify/login" 2>/dev/null || echo "")
    login_code=$(echo "$login_body" | grep -oP '__HTTP_CODE__\K\d+' || echo 0)
    login_body=$(echo "$login_body" | sed 's/__HTTP_CODE__[0-9]*//')
    if [[ "$login_code" =~ ^(502|503|504|000)$ ]]; then
        log_fail "Login endpoint UNREACHABLE (HTTP $login_code) — 0box down"
    elif echo "$login_body" | grep -qi "OTP verified"; then
        log_pass "Login: OTP verified (dev bypass working)"
    elif echo "$login_body" | grep -qi "firebase\|token\|invalid\|credential"; then
        log_warn "Login: Firebase auth required (dev bypass not active) — HTTP $login_code"
    else
        log_warn "Login: unexpected HTTP $login_code: $(echo "$login_body" | head -c 200)"
    fi

    # c. Free allocation via 0box API, then fall back to regular allocation
    # In dev/no-auth mode the v2private middleware is bypassed — wallet headers suffice.
    local alloc_id=""
    local client_id pub_key alloc_ts test_uid
    client_id=$(python3 -c "import json; d=json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}')); print(d['client_id'])" 2>/dev/null || echo "")
    pub_key=$(python3 -c "import json; d=json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}')); print(d['keys'][0]['public_key'])" 2>/dev/null || echo "")
    alloc_ts=$(date +%s)
    test_uid="verify-$(hostname | tr -d '.')-${alloc_ts}"

    if [ -n "$client_id" ] && [ -n "$pub_key" ]; then
        local free_resp
        free_resp=$(curl -sk -m 15 \
            -H "X-App-Client-ID: $client_id" \
            -H "X-App-Client-Key: $pub_key" \
            -H "X-App-Timestamp: $alloc_ts" \
            -H "X-App-ID-Token: test-token" \
            -H "X-App-User-ID: $test_uid" \
            -H "X-App-Type: vult" \
            "${OBOX_URL}/v2/freestorage" 2>/dev/null || echo "")
        if echo "$free_resp" | python3 -c "
import sys,json
d=json.load(sys.stdin)
assert d.get('assigner') and d.get('signature'), 'no marker fields'
" 2>/dev/null; then
            echo "$free_resp" > /tmp/verify_free_marker_$$.json
            local free_alloc_out
            free_alloc_out=$($ZBOX newallocation --free_storage /tmp/verify_free_marker_$$.json $W 2>&1)
            rm -f /tmp/verify_free_marker_$$.json
            alloc_id=$(echo "$free_alloc_out" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1)
            [ -z "$alloc_id" ] && alloc_id=$(echo "$free_alloc_out" | grep -oE "[a-f0-9]{64}" | tail -1)
            if [ -n "$alloc_id" ]; then
                log_pass "Free allocation via 0box API: ${alloc_id:0:16}..."
            else
                log_warn "Free allocation ticket obtained but create failed: $(echo "$free_alloc_out" | tail -2)"
            fi
        else
            local free_code
            free_code=$(echo "$free_resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error','?'))" 2>/dev/null || echo "$free_resp" | head -c 100)
            log_warn "0box /v2/freestorage: $free_code (will fall back to regular allocation)"
        fi
    fi

    # Fall back to regular allocation if free allocation didn't work
    if [ -z "$alloc_id" ]; then
        local alloc_out
        alloc_out=$($ZBOX newallocation --lock 0.5 --size 20971520 $W 2>&1)
        alloc_id=$(echo "$alloc_out" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1)
        [ -z "$alloc_id" ] && alloc_id=$(echo "$alloc_out" | grep -oE "[a-f0-9]{64}" | tail -1)
        [ -n "$alloc_id" ] && log_pass "Allocation created (regular fallback): ${alloc_id:0:16}..." || \
            log_fail "Could not create allocation: $(echo "$alloc_out" | tail -3)"
    fi

    # Wait for allocation to be committed on chain and for blobbers to process the allocation event
    [ -n "$alloc_id" ] && sleep 30

    # Verify the allocation actually exists on chain (the parsed ID could be a txn hash if creation failed)
    if [ -n "$alloc_id" ]; then
        local alloc_check
        alloc_check=$(curl -s "${SHARDER_URL}/v1/screst/${STORAGE_SC}/allocation?allocation=${alloc_id}" 2>/dev/null)
        if ! echo "$alloc_check" | python3 -c "import json,sys; d=json.load(sys.stdin); assert d.get('id'), 'not found'" 2>/dev/null; then
            log_fail "Allocation ${alloc_id:0:16}... not found on chain (creation likely failed) — skipping upload tests"
            alloc_id=""
        fi
    fi

    if [ -n "$alloc_id" ]; then
        # Prepare test files
        echo "Vult verify test $(date)" > /tmp/verify_vult.txt
        printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' > /tmp/verify_vult.png
        printf '%%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 3 3]>>endobj\nxref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%%%EOF\n' > /tmp/verify_vult.pdf
        # Use timestamp-suffixed remote paths to avoid duplicate_file errors on repeated runs
        local ts_suffix="${alloc_ts}"

        # d. Upload: text, image, PDF
        local up_txt; up_txt=$($ZBOX upload --allocation "$alloc_id" \
            --localpath /tmp/verify_vult.txt --remotepath /verify_vult_${ts_suffix}.txt $W 2>&1)
        echo "$up_txt" | grep -qiE "success|uploaded" && log_pass "Upload text file" || \
            log_fail "Upload text file FAILED: $(echo "$up_txt" | tail -2)"

        local up_img; up_img=$($ZBOX upload --allocation "$alloc_id" \
            --localpath /tmp/verify_vult.png --remotepath /verify_vult_${ts_suffix}.png $W 2>&1)
        echo "$up_img" | grep -qiE "success|uploaded" && log_pass "Upload image (PNG)" || \
            log_warn "Upload PNG: $(echo "$up_img" | tail -1)"

        local up_pdf; up_pdf=$($ZBOX upload --allocation "$alloc_id" \
            --localpath /tmp/verify_vult.pdf --remotepath /verify_vult_${ts_suffix}.pdf $W 2>&1)
        echo "$up_pdf" | grep -qiE "success|uploaded" && log_pass "Upload PDF file" || \
            log_warn "Upload PDF: $(echo "$up_pdf" | tail -1)"

        # e. List files (carousel source data)
        local ls_out; ls_out=$($ZBOX list --allocation "$alloc_id" --remotepath / $W 2>&1)
        local file_count; file_count=$(echo "$ls_out" | grep -cE "\.txt|\.png|\.pdf" | tr -d '[:space:]')
        file_count="${file_count:-0}"
        [ "${file_count:-0}" -gt 0 ] && log_pass "Listed $file_count files in allocation (carousel source)" || \
            log_warn "List returned no files: $(echo "$ls_out" | tail -2)"

        # f. Carousel render: Blimp calls 0box /v2/convert-to-pdf → 0box → Gotenberg (Docker network)
        # NOTE: The real carousel path is Browser→0box→Gotenberg, NOT Browser→Gotenberg directly.
        # Direct port 3010 access is only for health checking the container itself.
        local goten_code; goten_code=$(curl_ok "http://127.0.0.1:3010/health" 5)
        if [[ "$goten_code" =~ ^(200|204)$ ]]; then
            log_pass "Gotenberg container: healthy (port 3010)"
            # Test that Gotenberg is reachable FROM 0box's Docker network (the actual path used)
            # 0box calls gotenberg:3000 internally — must use wget (curl not in 0box container)
            local goten_via_0box
            goten_via_0box=$(docker exec 0box wget -q -S -O /dev/null \
                http://gotenberg:3000/health 2>&1 | grep -oP 'HTTP/\S+ \K\d+' | head -1 || echo "0")
            if [[ "$goten_via_0box" =~ ^(200|204)$ ]]; then
                log_pass "Gotenberg reachable from 0box container (http://gotenberg:3000) — carousel path OK"
            else
                log_warn "Gotenberg NOT reachable from 0box container (HTTP $goten_via_0box) — carousel will fail"
                log_info "Fix: ensure gotenberg and 0box are on same Docker network (testnet0)"
            fi
            # Test that 0box /v2/convert-to-pdf endpoint is registered (returns 400 for missing params, NOT 404)
            local zbox_pdf_code
            zbox_pdf_code=$(curl -sk -o /dev/null -w '%{http_code}' -m 10 \
                "http://127.0.0.1:9081/v2/convert-to-pdf" \
                2>/dev/null || echo "0")
            if [[ "$zbox_pdf_code" =~ ^(400|401|403|422)$ ]]; then
                log_pass "0box /v2/convert-to-pdf endpoint registered (HTTP $zbox_pdf_code — missing params as expected)"
            elif [[ "$zbox_pdf_code" == "404" ]]; then
                log_warn "0box /v2/convert-to-pdf returned 404 — endpoint may not be registered"
            else
                log_warn "0box /v2/convert-to-pdf: HTTP $zbox_pdf_code (expected 400/401)"
            fi
        else
            log_warn "Gotenberg not reachable (HTTP $goten_code) — PDF carousel thumbnails will fail"
            log_info "Fix: bash scripts/deploy_local.sh services"
        fi

        # g. Download files + SHA256 hash verification (data integrity check)
        local sha_pdf_up sha_img_up
        sha_pdf_up=$(sha256sum /tmp/verify_vult.pdf | awk '{print $1}')
        sha_img_up=$(sha256sum /tmp/verify_vult.png | awk '{print $1}')

        local dl_pdf_out
        dl_pdf_out=$($ZBOX download --allocation "$alloc_id" \
            --remotepath /verify_vult_${ts_suffix}.pdf --localpath /tmp/dl_pdf_$$.pdf $W 2>&1)
        if [ -f /tmp/dl_pdf_$$.pdf ] && [ -s /tmp/dl_pdf_$$.pdf ]; then
            local sha_pdf_dl; sha_pdf_dl=$(sha256sum /tmp/dl_pdf_$$.pdf | awk '{print $1}')
            [ "$sha_pdf_up" = "$sha_pdf_dl" ] && \
                log_pass "Download PDF + SHA256 hash verified (content matches upload)" || \
                log_fail "Download PDF: hash MISMATCH — data corruption detected!"
        else
            log_fail "Download PDF FAILED: $(echo "$dl_pdf_out" | tail -2)"
        fi
        rm -f /tmp/dl_pdf_$$.pdf

        local dl_img_out
        dl_img_out=$($ZBOX download --allocation "$alloc_id" \
            --remotepath /verify_vult_${ts_suffix}.png --localpath /tmp/dl_img_$$.png $W 2>&1)
        if [ -f /tmp/dl_img_$$.png ] && [ -s /tmp/dl_img_$$.png ]; then
            local sha_img_dl; sha_img_dl=$(sha256sum /tmp/dl_img_$$.png | awk '{print $1}')
            [ "$sha_img_up" = "$sha_img_dl" ] && \
                log_pass "Download image + SHA256 hash verified (content matches upload)" || \
                log_warn "Download image: hash mismatch"
        else
            log_warn "Download image: $(echo "$dl_img_out" | tail -1)"
        fi
        rm -f /tmp/dl_img_$$.png

        # g2. Gotenberg render: actually convert uploaded PDF through 0box → Gotenberg pipeline
        local render_ticket render_hash
        render_ticket=$($ZBOX share --allocation "$alloc_id" --remotepath /verify_vult_${ts_suffix}.pdf $W 2>&1 \
            | grep -oP 'Auth token :\K\S+')
        # Get lookup_hash via JSON list (separate from the non-JSON ls_out above)
        local ls_json_out
        ls_json_out=$($ZBOX list --allocation "$alloc_id" --remotepath / --json $W 2>&1)
        render_hash=$(echo "$ls_json_out" | python3 -c "
import json,sys
for line in sys.stdin:
    line=line.strip()
    if line.startswith('[') or line.startswith('{'):
        try:
            data=json.loads(line)
            if isinstance(data,list):
                for f in data:
                    n=f.get('name',f.get('Name',''))
                    if 'verify_vult' in n and n.endswith('.pdf'):
                        print(f.get('lookup_hash',f.get('LookupHash',''))); break
        except: pass
" 2>/dev/null)
        if [ -n "$render_ticket" ] && [ -n "$render_hash" ]; then
            local encoded_ticket
            encoded_ticket=$(python3 -c "import urllib.parse; print(urllib.parse.quote('''${render_ticket}'''))" 2>/dev/null)
            local render_code render_size
            render_code=$(curl -sk -o /tmp/render_pdf_$$.pdf -w '%{http_code}' -m 60 \
                "http://127.0.0.1:9081/v2/convert-to-pdf?auth_ticket=${encoded_ticket}&lookup_hash=${render_hash}" \
                -H "X-App-Client-ID: $client_id" \
                -H "X-App-Client-Key: $pub_key" \
                -H "X-App-Timestamp: $(date +%s)" \
                -H "X-App-ID-Token: test" \
                -H "X-App-User-ID: $test_uid" \
                -H "X-App-Type: vult" 2>/dev/null || echo "0")
            if [ "$render_code" = "200" ] && [ -f /tmp/render_pdf_$$.pdf ] && [ -s /tmp/render_pdf_$$.pdf ]; then
                local pdf_header
                pdf_header=$(head -c 5 /tmp/render_pdf_$$.pdf 2>/dev/null)
                if [ "$pdf_header" = "%PDF-" ]; then
                    render_size=$(wc -c < /tmp/render_pdf_$$.pdf | tr -d '[:space:]')
                    log_pass "Gotenberg render: PDF converted successfully (${render_size} bytes, valid %PDF header)"
                else
                    log_fail "Gotenberg render: HTTP 200 but output is not a valid PDF"
                fi
            elif [ "$render_code" = "0" ]; then
                log_fail "Gotenberg render: 0box /v2/convert-to-pdf unreachable"
            else
                local render_err
                render_err=$(cat /tmp/render_pdf_$$.pdf 2>/dev/null | head -c 200)
                log_fail "Gotenberg render: HTTP $render_code — $render_err"
            fi
            rm -f /tmp/render_pdf_$$.pdf
        else
            log_warn "Gotenberg render: could not get auth_ticket or lookup_hash (skipping)"
        fi

        # g3. zvault/zauth split-key vault flow
        # Tests the full key custody pipeline: store key → generate split wallet → setup in zauth → sign message
        local zvault_jwt_secret
        zvault_jwt_secret=$(docker exec zvault-zvault-1 cat /zvault/config/zvault.yaml 2>/dev/null \
            | grep 'jwt_secret:' | awk -F': ' '{print $2}' | tr -d '"' | xargs)
        if [ -n "$zvault_jwt_secret" ]; then
            local vault_jwt
            vault_jwt=$(python3 -c "
import json, hmac, hashlib, base64, time
def b64url(data):
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode()
secret = '${zvault_jwt_secret}'
header = b64url(json.dumps({'alg':'HS256','typ':'JWT'}).encode())
payload = b64url(json.dumps({'sub':'verify-$$','user_id':'verify-$$','iat':int(time.time()),'exp':int(time.time())+3600}).encode())
msg = header + '.' + payload
sig = b64url(hmac.new(secret.encode(), msg.encode(), hashlib.sha256).digest())
print(msg + '.' + sig)
" 2>/dev/null)
            if [ -z "$vault_jwt" ]; then
                log_warn "zvault: could not generate JWT token (python3 hmac issue)"
            else
                local priv_key
                priv_key=$(python3 -c "import json; print(json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}'))['keys'][0]['private_key'])" 2>/dev/null)

                # Step a: Store private key in zvault
                local store_code
                store_code=$(curl -sk -o /dev/null -w '%{http_code}' -m 10 -X POST "${ZVAULT_URL}/store" \
                    -H "X-Jwt-Token: $vault_jwt" \
                    -H "Content-Type: application/json" \
                    -d "{\"mnemonic\":\"verify-test-mnemonic-$$\",\"private_key\":\"$priv_key\"}" 2>/dev/null || echo "0")
                if [[ "$store_code" =~ ^(200|201|409)$ ]]; then
                    log_pass "zvault /store: key stored (HTTP $store_code)"
                else
                    log_fail "zvault /store: HTTP $store_code"
                fi

                # Step b: Generate split wallet
                local wallet_resp wallet_code split_cid
                wallet_resp=$(curl -sk -m 10 -X POST "${ZVAULT_URL}/wallet" \
                    -H "X-Jwt-Token: $vault_jwt" \
                    -w '\n__HTTP__%{http_code}' 2>/dev/null || echo "")
                wallet_code=$(echo "$wallet_resp" | grep -oP '__HTTP__\K\d+' || echo "0")
                wallet_resp=$(echo "$wallet_resp" | sed 's/__HTTP__[0-9]*//')
                split_cid=$(echo "$wallet_resp" | python3 -c "import json,sys; print(json.load(sys.stdin).get('client_id',''))" 2>/dev/null || echo "")
                if [[ "$wallet_code" =~ ^(200|201)$ ]] && [ -n "$split_cid" ]; then
                    log_pass "zvault /wallet: split wallet created (${split_cid:0:16}...)"
                else
                    log_fail "zvault /wallet: HTTP $wallet_code"
                fi

                # Step c: Generate split key (zvault calls zauth /setup internally)
                if [ -n "$split_cid" ]; then
                    local key_code
                    key_code=$(curl -sk -o /dev/null -w '%{http_code}' -m 10 -X POST "${ZVAULT_URL}/key/$split_cid" \
                        -H "X-Jwt-Token: $vault_jwt" 2>/dev/null || echo "0")
                    [[ "$key_code" =~ ^(200|201)$ ]] && \
                        log_pass "zvault /key: split key generated (HTTP $key_code)" || \
                        log_fail "zvault /key: HTTP $key_code"

                    # Step d: Verify keys stored
                    local keys_resp keys_count
                    keys_resp=$(curl -sk -m 10 "${ZVAULT_URL}/keys/$split_cid" \
                        -H "X-Jwt-Token: $vault_jwt" 2>/dev/null || echo "{}")
                    keys_count=$(echo "$keys_resp" | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('keys',[])))" 2>/dev/null || echo "0")
                    [ "${keys_count:-0}" -gt 0 ] && \
                        log_pass "zvault /keys: $keys_count split key(s) retrieved" || \
                        log_fail "zvault /keys: no keys found"

                    # Step e: Sign message via zauth using known-good BLS key pair
                    # (zvault-generated keys have no pre-computed valid signature, so we use test constants)
                    local _za_cid="03df8919e4d76f6ffa9e23c388576f0baa02360d6e903a84d69689e0b2b5a28c"
                    local _za_pk="ff132aaf4eeb517478c7bdd19ba887dbd909c4527b78ac989f723e5b5c349f03dc5a737071040edd5ba7a593e12264d895ad9cace1a50321886653ca8c366e13"
                    local _za_sk="863f94d6deacc75c658db3e22d27c31944fc562aeef6e5b97a31d9e25294111a"
                    local _za_peer="f06fe5513831714965ca33080052b3873bb2e785c5d2abd88a2f958e74872617adf6e8c5fe5cb8ed0e0c218cde796707c0a9ef8f7e77756032eada0d5f3c9d00"
                    local _za_hash="7a8950d472c3a5a7cf0f27019c4507275d24e0bd3a97e1dedc7d77a132d9d6d3"
                    local _za_sig="5d65c31f8bcf64c259c3e155b78bb4c7c5ea2dae1d0d6da1a8b735f2bda2db84"
                    # Setup with known key pair
                    curl -sk -o /dev/null -m 10 -X POST "${ZAUTH_URL}/setup" \
                        -H "X-Jwt-Token: $vault_jwt" \
                        -H "X-Peer-Public-Key: $_za_peer" \
                        -H "Content-Type: application/json" \
                        -d "{\"user_id\":\"verify-$$\",\"client_id\":\"$_za_cid\",\"client_key\":\"$_za_pk\",\"public_key\":\"$_za_pk\",\"private_key\":\"$_za_sk\",\"peer_public_key\":\"$_za_peer\",\"expired_at\":0}" \
                        2>/dev/null
                    local sign_resp sign_code
                    sign_resp=$(curl -sk -m 10 -X POST "${ZAUTH_URL}/sign/msg" \
                        -H "X-Jwt-Token: $vault_jwt" \
                        -H "X-Peer-Public-Key: $_za_peer" \
                        -H "Content-Type: application/json" \
                        -d "{\"hash\":\"$_za_hash\",\"signature\":\"$_za_sig\",\"client_id\":\"$_za_cid\"}" \
                        -w '\n__HTTP__%{http_code}' 2>/dev/null || echo "")
                    sign_code=$(echo "$sign_resp" | grep -oP '__HTTP__\K\d+' || echo "0")
                    sign_resp=$(echo "$sign_resp" | sed 's/__HTTP__[0-9]*//')
                    if [[ "$sign_code" =~ ^(200|201)$ ]] && echo "$sign_resp" | grep -q '"sig"'; then
                        log_pass "zauth /sign/msg: message signed successfully"
                    else
                        log_warn "zauth /sign/msg: HTTP $sign_code — $(echo "$sign_resp" | head -c 100)"
                    fi
                    # Cleanup zauth test key
                    curl -sk -o /dev/null -m 10 -X POST "${ZAUTH_URL}/delete/$_za_cid" \
                        -H "X-Jwt-Token: $vault_jwt" -H "X-Peer-Public-Key: $_za_peer" 2>/dev/null

                    # Step f: Cleanup
                    curl -sk -o /dev/null -m 10 -X POST "${ZVAULT_URL}/delete/$split_cid" \
                        -H "X-Jwt-Token: $vault_jwt" 2>/dev/null
                fi
            fi
        else
            log_warn "zvault: could not read jwt_secret from config (container not running?)"
        fi

        # h. Upgrade allocation size (using tokens from faucet)
        local upd_out; upd_out=$($ZBOX updateallocation --allocation "$alloc_id" \
            --size 41943040 $W 2>&1)
        echo "$upd_out" | grep -qiE "success|updated|Hash:" && log_pass "Upgrade allocation size" || \
            log_warn "Upgrade allocation: $(echo "$upd_out" | tail -1)"

        # j. Replace blobber via Settings (updateallocation --add_blobber + --remove_blobber)
        # This is exactly what Vult Settings → Manage Allocation → Replace Blobber does.
        local blobbers_raw
        blobbers_raw=$($ZBOX ls-blobbers --json --silent $W 2>/dev/null || echo "[]")
        local blobber_count
        blobber_count=$(echo "$blobbers_raw" | \
            python3 -c "import sys,json; d=json.load(sys.stdin); print(len([b for b in d if not b.get('is_enterprise',False)]))" \
            2>/dev/null | tr -d '[:space:]'); blobber_count="${blobber_count:-0}"
        if [ "${blobber_count}" -ge 5 ]; then
            log_pass "Replace blobber: $blobber_count regular blobbers available (≥5 required)"
            local alloc_raw alloc_json old_blobber new_blobber
            alloc_raw=$($ZBOX getallocation --allocation "$alloc_id" --json $W 2>&1)
            alloc_json=$(echo "$alloc_raw" | grep -v 'sdk.go\|wallet_base\|SDK Version\|INFO\|WARN\|ERROR' | grep '{' | tail -1)
            # Extract blobber IDs: one to remove (from alloc) and one to add (not in alloc)
            read -r old_blobber new_blobber < <(echo "$alloc_json $blobbers_raw" | python3 -c "
import sys,json
parts=sys.stdin.read().split(None,1)
try:
    alloc=json.loads(parts[0]); all_b=json.loads(parts[1])
    in_alloc={b.get('id','') for b in alloc.get('blobbers',[])}
    old=list(in_alloc)[0] if in_alloc else ''
    new=next((b['id'] for b in all_b if not b.get('is_enterprise') and b['id'] not in in_alloc),'')
    print(old,new)
except: print('','')
" 2>/dev/null)
            if [ -n "$old_blobber" ] && [ -n "$new_blobber" ]; then
                local replace_out
                replace_out=$($ZBOX updateallocation --allocation "$alloc_id" \
                    --add_blobber "$new_blobber" --remove_blobber "$old_blobber" \
                    --lock 0.5 $W 2>&1)
                echo "$replace_out" | grep -qiE "Allocation updated|Repair file completed|txId" && \
                    log_pass "Replace blobber (Settings): swap ${old_blobber:0:12}→${new_blobber:0:12} succeeded" || \
                    log_warn "Replace blobber: $(echo "$replace_out" | grep -v 'sdk.go\|INFO\|DEBUG' | tail -2)"
            else
                log_warn "Replace blobber: could not identify old/new blobber IDs"
            fi
        else
            log_warn "Replace blobber: only $blobber_count regular blobbers (need ≥5)"
        fi

        # k. Public + Private folder share
        # Public share: any anonymous user can download via authticket (no clientid restriction)
        # Private share: only the specified clientid can redeem the authticket
        local share_content="Public share verify $$"
        echo "$share_content" > /tmp/pub_share_$$.txt
        $ZBOX upload --allocation "$alloc_id" \
            --localpath /tmp/pub_share_$$.txt --remotepath /shared_folder/pub.txt $W >/dev/null 2>&1
        local pub_share_out pub_ticket
        pub_share_out=$($ZBOX share --allocation "$alloc_id" --remotepath /shared_folder/pub.txt $W 2>&1)
        pub_ticket=$(echo "$pub_share_out" | grep -oP 'Auth token :\K\S+')
        if [ -n "$pub_ticket" ]; then
            local pub_dl_out
            pub_dl_out=$($ZBOX download --authticket "$pub_ticket" --localpath /tmp/pub_dl_$$.txt $W 2>&1)
            if echo "$pub_dl_out" | grep -qiE "Status completed|100%"; then
                local dl_content; dl_content=$(tr -d '\n' < /tmp/pub_dl_$$.txt 2>/dev/null)
                [ "$share_content" = "$dl_content" ] && \
                    log_pass "Public folder share: authticket download + content verified" || \
                    log_warn "Public folder share: downloaded but content mismatch"
            else
                log_warn "Public folder share: download via authticket failed"
            fi
            rm -f /tmp/pub_dl_$$.txt
        else
            log_warn "Public folder share: no authticket generated"
        fi
        rm -f /tmp/pub_share_$$.txt

        # Private share: create a temp wallet, share specifically with its client_id
        local priv_wname="pvt_$$"
        local PW="--configDir $ZCN_CONFIG_DIR --wallet $priv_wname --config $ZCN_CONFIG_FILE"
        $ZWALLET create-wallet $W --wallet "$priv_wname" >/dev/null 2>&1 || true
        local priv_client_id
        priv_client_id=$(python3 -c "
import json, os
for p in ['${ZCN_CONFIG_DIR}/${priv_wname}.json', '${ZCN_CONFIG_DIR}/${priv_wname}']:
    if os.path.exists(p):
        try: print(json.load(open(p))['client_id']); break
        except: pass
" 2>/dev/null)
        if [ -z "$priv_client_id" ]; then
            log_warn "Private folder share: could not create temp wallet (skipping)"
        else
            local priv_content="Private share verify $$"
            echo "$priv_content" > /tmp/priv_share_$$.txt
            $ZBOX upload --allocation "$alloc_id" \
                --localpath /tmp/priv_share_$$.txt --remotepath /shared_folder/priv.txt $W >/dev/null 2>&1
            local priv_share_out priv_ticket
            priv_share_out=$($ZBOX share --allocation "$alloc_id" \
                --remotepath /shared_folder/priv.txt --clientid "$priv_client_id" $W 2>&1)
            priv_ticket=$(echo "$priv_share_out" | grep -oP 'Auth token :\K\S+')
            if [ -n "$priv_ticket" ]; then
                local priv_dl_out
                priv_dl_out=$($ZBOX download --authticket "$priv_ticket" \
                    --localpath /tmp/priv_dl_$$.txt $PW 2>&1)
                echo "$priv_dl_out" | grep -qiE "Status completed|100%" && \
                    log_pass "Private folder share: authorized wallet downloaded successfully" || \
                    log_warn "Private folder share: authorized download failed: $(echo "$priv_dl_out" | tail -1)"
                rm -f /tmp/priv_dl_$$.txt
            else
                log_warn "Private folder share: no authticket generated"
            fi
            rm -f /tmp/priv_share_$$.txt
        fi
        rm -f "${ZCN_CONFIG_DIR}/${priv_wname}.json"

        # Cleanup primary allocation
        $ZBOX cancel-allocation --allocation "$alloc_id" $W >/dev/null 2>&1 || true
        rm -f /tmp/verify_vult.txt /tmp/verify_vult.png /tmp/verify_vult.pdf
    else
        log_fail "Could not create allocation — skipping upload/download/carousel/replace checks"
    fi

    # h+i. Second wallet: create wallet, allocation, upload, switch wallets, verify isolation
    log_info "Second wallet flow: create wallet → fund → allocate → upload → switch..."
    local sw_name="vw2_$$"  # short name to avoid path-length issues
    local sw_file="${ZCN_CONFIG_DIR}/${sw_name}.json"
    local W2="--configDir $ZCN_CONFIG_DIR --wallet $sw_name --config $ZCN_CONFIG_FILE"
    local sw_create; sw_create=$($ZWALLET create-wallet $W --wallet "$sw_name" 2>&1 || true)
    if [ -f "$sw_file" ] || echo "$sw_create" | grep -qiE "created|success|wallet"; then
        log_pass "Second wallet created (${sw_name})"

        # Fund second wallet via faucet
        $ZWALLET faucet --methodName pour --input '{"Once":true}' $W2 >/dev/null 2>&1 || true

        # Create allocation for second wallet
        local sw_alloc_out sw_alloc_id
        sw_alloc_out=$($ZBOX newallocation --lock 0.5 --size 10485760 $W2 2>&1)
        sw_alloc_id=$(echo "$sw_alloc_out" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1)
        [ -z "$sw_alloc_id" ] && sw_alloc_id=$(echo "$sw_alloc_out" | grep -oE "[a-f0-9]{64}" | tail -1)

        if [ -n "$sw_alloc_id" ]; then
            log_pass "Second wallet allocation: ${sw_alloc_id:0:16}..."
            sleep 20  # wait for allocation to be indexed before upload

            # Upload a file using second wallet
            echo "Second wallet verify $(date)" > /tmp/vw2_$$.txt
            local sw_up; sw_up=$($ZBOX upload --allocation "$sw_alloc_id" \
                --localpath /tmp/vw2_$$.txt --remotepath /vw2.txt $W2 2>&1)
            echo "$sw_up" | grep -qiE "success|uploaded" && log_pass "Second wallet upload" || \
                log_warn "Second wallet upload: $(echo "$sw_up" | tail -1)"

            # List second wallet's files
            local sw_ls; sw_ls=$($ZBOX list --allocation "$sw_alloc_id" --remotepath / $W2 2>&1)
            echo "$sw_ls" | grep -qi "vw2" && \
                log_pass "Switch wallet: second wallet files visible" || \
                log_warn "Switch wallet: second wallet files not visible: $(echo "$sw_ls" | tail -1)"

            # Verify wallet isolation — first wallet's allocation is still accessible
            if [ -n "$alloc_id" ]; then
                local sw_ls1; sw_ls1=$($ZBOX list --allocation "$alloc_id" --remotepath / $W 2>&1)
                echo "$sw_ls1" | grep -qiE "\.txt|\.png|\.pdf" && \
                    log_pass "Wallet isolation: first wallet's files still accessible after switch" || \
                    log_warn "Wallet isolation: first wallet files not visible: $(echo "$sw_ls1" | tail -1)"
            fi

            $ZBOX cancel-allocation --allocation "$sw_alloc_id" $W2 >/dev/null 2>&1 || true
        else
            log_warn "Second wallet allocation failed: $(echo "$sw_alloc_out" | tail -2)"
        fi

        rm -f "$sw_file" /tmp/vw2_$$.txt
    else
        log_warn "Second wallet creation failed: $(echo "$sw_create" | tail -1)"
    fi

    # k. Hardcoded URL check in live service configs
    local bad_urls
    # Only check active YAML/env config files — not nginx.conf (legacy), logs, or backups
    bad_urls=$(grep -rn "dev\.0chain\.net\|dev\.zus\.network\|devtest\.zus\.network" \
        /root/Code/0box/docker.local/config/ \
        /root/Code/blobber/docker.local/ 2>/dev/null \
        | grep -v ".bak" | grep -v "nginx.conf" | grep -v ".log:" | grep -v "^Binary" \
        | grep -E "\.(yaml|yml|env|json):" | head -10)
    if [ -z "$bad_urls" ]; then
        log_pass "No stale dev/devtest URLs in service configs"
    else
        log_fail "Stale dev URLs found in service configs:"
        echo "$bad_urls" | while IFS= read -r l; do log_info "  $l"; done
        log_info "Fix: bash scripts/deploy_local.sh web-apps  (rebuilds with correct .env)"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 6: Blimp — File Manager (allocation, upload, add-blobber, repair,
#            upload-only mode, freeze mode, carousel, download+hash)
# ══════════════════════════════════════════════════════════════════════════════
check_blimp() {
    log_header "Blimp — File Manager Checks"
    local BW="--configDir $ZCN_CONFIG_DIR --wallet $ZCN_WALLET_FILE --config $ZCN_CONFIG_FILE"

    # a. Blimp app serving
    local code; code=$(curl_ok "http://127.0.0.1:${APP_PORTS[blimp]}/" 5)
    [[ "$code" =~ ^(200|301|302|304)$ ]] && log_pass "Blimp app serving on port ${APP_PORTS[blimp]}" || \
        log_fail "Blimp app NOT serving (HTTP $code) — bash scripts/deploy_local.sh web-apps"

    # b. Auth: 0box deployment_mode=3 means no Firebase required
    local dm; dm=$(docker inspect --format='{{range .Config.Cmd}}{{.}} {{end}}' 0box 2>/dev/null | \
        grep -oP -- '--deployment_mode \K\d+' || echo "unknown")
    [ "$dm" = "3" ] && log_pass "Blimp auth: 0box deployment_mode=3 (no Firebase required)" || \
        log_fail "Blimp auth: 0box deployment_mode=$dm — Blimp users will get 'Unauthorized'"

    # c. 0box blobber availability (required for free allocation button)
    local avail_blobbers
    avail_blobbers=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM blobbers b JOIN provider_brand pb ON b.brand_id=pb.id
         WHERE b.not_available=false AND b.is_enterprise=false;" \
        2>/dev/null | tr -d ' \n' || echo 0)
    if [ "${avail_blobbers:-0}" -ge 3 ]; then
        log_pass "0box: $avail_blobbers regular blobbers available for Blimp allocation"
    else
        log_fail "0box: only $avail_blobbers regular blobbers — Blimp 'no provider match'"
        log_info "Fix: bash scripts/deploy_local.sh fix-kafka"
        $AUTO_FIX && bash "${SCRIPT_DIR}/deploy_local.sh" fix-kafka
    fi

    # d. Gotenberg carousel render service
    # Real carousel path: Browser → 0box /v2/convert-to-pdf → 0box → gotenberg:3000 (Docker network)
    local goten_code; goten_code=$(curl_ok "http://127.0.0.1:3010/health" 5)
    if [[ "$goten_code" =~ ^(200|204)$ ]]; then
        log_pass "Blimp carousel: Gotenberg container healthy (port 3010)"
        # Test Gotenberg is reachable FROM 0box's Docker network (the actual carousel path)
        local goten_via_0box
        goten_via_0box=$(docker exec 0box wget -q -S -O /dev/null \
            http://gotenberg:3000/health 2>&1 | grep -oP 'HTTP/\S+ \K\d+' | head -1 || echo "0")
        if [[ "$goten_via_0box" =~ ^(200|204)$ ]]; then
            log_pass "Blimp carousel: Gotenberg reachable from 0box container — carousel path OK"
        else
            log_warn "Blimp carousel: Gotenberg NOT reachable from 0box container (HTTP $goten_via_0box) — carousel will fail"
            log_info "Fix: ensure gotenberg and 0box are on same Docker network (testnet0)"
        fi
        # Test 0box /v2/convert-to-pdf endpoint is registered (400=exists, 404=missing)
        local zbox_pdf_code
        zbox_pdf_code=$(curl -sk -o /dev/null -w '%{http_code}' -m 10 \
            "http://127.0.0.1:9081/v2/convert-to-pdf" 2>/dev/null || echo "0")
        if [[ "$zbox_pdf_code" =~ ^(400|401|403|422)$ ]]; then
            log_pass "Blimp carousel: 0box /v2/convert-to-pdf endpoint registered (HTTP $zbox_pdf_code)"
        elif [[ "$zbox_pdf_code" == "404" ]]; then
            log_warn "Blimp carousel: 0box /v2/convert-to-pdf returned 404 — endpoint not registered"
        else
            log_warn "Blimp carousel: 0box /v2/convert-to-pdf HTTP $zbox_pdf_code (expected 400/401)"
        fi
    else
        log_warn "Blimp carousel: Gotenberg NOT running (HTTP $goten_code)"
        log_info "Fix: bash scripts/deploy_local.sh services"
    fi

    # e. Faucet — fund wallet for standard allocation
    local faucet_out
    faucet_out=$($ZWALLET faucet --methodName pour --input '{"Once":true}' $BW 2>&1 || true)
    echo "$faucet_out" | grep -qiE "success|Execute faucet|Hash:" && \
        log_pass "Faucet: tokens poured for Blimp allocation" || \
        log_warn "Faucet: $(echo "$faucet_out" | tail -1)"

    # f. Create standard allocation (Blimp: New Allocation button with faucet tokens)
    local bl_alloc_out bl_alloc_id
    bl_alloc_out=$($ZBOX newallocation --lock 1 --size 20971520 $BW 2>&1)
    bl_alloc_id=$(echo "$bl_alloc_out" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1)
    [ -z "$bl_alloc_id" ] && bl_alloc_id=$(echo "$bl_alloc_out" | grep -oE '[a-f0-9]{64}' | tail -1)
    if [ -n "$bl_alloc_id" ]; then
        log_pass "Blimp allocation created: ${bl_alloc_id:0:16}..."
        sleep 12  # wait for chain indexing
    else
        log_fail "Blimp allocation failed: $(echo "$bl_alloc_out" | tail -2)"
    fi

    if [ -n "$bl_alloc_id" ]; then
        # Prepare test files
        echo "Blimp verify $(date)" > /tmp/bl_verify_$$.txt
        printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' > /tmp/bl_verify_$$.png
        printf '%%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 3 3]>>endobj\nxref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%%%EOF\n' > /tmp/bl_verify_$$.pdf

        # g. Upload files (Blimp: drag-drop upload)
        local up_txt up_img up_pdf
        up_txt=$($ZBOX upload --allocation "$bl_alloc_id" \
            --localpath /tmp/bl_verify_$$.txt --remotepath /docs/verify.txt $BW 2>&1)
        echo "$up_txt" | grep -qiE "Success|uploaded" && log_pass "Blimp: upload text file" || \
            log_warn "Blimp upload text: $(echo "$up_txt" | tail -1)"

        local sha_pdf_up; sha_pdf_up=$(sha256sum /tmp/bl_verify_$$.pdf | awk '{print $1}')
        up_pdf=$($ZBOX upload --allocation "$bl_alloc_id" \
            --localpath /tmp/bl_verify_$$.pdf --remotepath /docs/verify.pdf $BW 2>&1)
        echo "$up_pdf" | grep -qiE "Success|uploaded" && log_pass "Blimp: upload PDF" || \
            log_warn "Blimp upload PDF: $(echo "$up_pdf" | tail -1)"

        up_img=$($ZBOX upload --allocation "$bl_alloc_id" \
            --localpath /tmp/bl_verify_$$.png --remotepath /docs/verify.png $BW 2>&1)
        echo "$up_img" | grep -qiE "Success|uploaded" && log_pass "Blimp: upload image (PNG)" || \
            log_warn "Blimp upload PNG: $(echo "$up_img" | tail -1)"

        # h. Download + SHA256 hash (carousel downloads via Blimp file browser)
        local dl_pdf_out sha_pdf_dl
        dl_pdf_out=$($ZBOX download --allocation "$bl_alloc_id" \
            --remotepath /docs/verify.pdf --localpath /tmp/bl_dl_$$.pdf $BW 2>&1)
        if echo "$dl_pdf_out" | grep -qiE "Status completed|100%"; then
            sha_pdf_dl=$(sha256sum /tmp/bl_dl_$$.pdf | awk '{print $1}')
            [ "$sha_pdf_up" = "$sha_pdf_dl" ] && \
                log_pass "Blimp: download PDF + SHA256 hash verified" || \
                log_warn "Blimp: PDF hash mismatch (upload/download content differs)"
        else
            log_warn "Blimp: PDF download failed: $(echo "$dl_pdf_out" | tail -1)"
        fi
        rm -f /tmp/bl_dl_$$.pdf

        # Carousel connectivity already verified above (section d) via 0box Docker path

        # i. Add blobber (Blimp: Settings → Manage Allocation → Add Blobber)
        # Get current blobber count, then add one that's not yet in allocation
        local bl_alloc_raw bl_alloc_json
        bl_alloc_raw=$($ZBOX getallocation --allocation "$bl_alloc_id" --json $BW 2>&1)
        bl_alloc_json=$(echo "$bl_alloc_raw" | grep -v 'sdk.go\|wallet_base\|SDK Version\|INFO\|WARN\|ERROR' | grep '{' | tail -1)
        local bl_blobbers_before add_blobber_id
        bl_blobbers_before=$(echo "$bl_alloc_json" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(len(d.get('blobbers',[])))" 2>/dev/null | tr -d '[:space:]')

        local all_blobbers_raw
        all_blobbers_raw=$($ZBOX ls-blobbers --json --silent $BW 2>/dev/null || echo "[]")
        add_blobber_id=$(echo "$bl_alloc_json $all_blobbers_raw" | python3 -c "
import sys,json
parts=sys.stdin.read().split(None,1)
try:
    alloc=json.loads(parts[0]); all_b=json.loads(parts[1])
    in_alloc={b.get('id','') for b in alloc.get('blobbers',[])}
    result=next((b['id'] for b in all_b if not b.get('is_enterprise') and b['id'] not in in_alloc),'')
    print(result)
except: print('')
" 2>/dev/null | tr -d '[:space:]')

        if [ -n "$add_blobber_id" ]; then
            local add_out
            add_out=$($ZBOX updateallocation --allocation "$bl_alloc_id" \
                --add_blobber "$add_blobber_id" --lock 0.5 $BW 2>&1)
            if echo "$add_out" | grep -qiE "Allocation updated|Repair file completed|txId"; then
                log_pass "Blimp Settings: add blobber ${add_blobber_id:0:16}... succeeded"
                # Verify new blobber appears in allocation info
                sleep 5
                local bl_alloc_raw2 bl_alloc_json2 bl_blobbers_after
                bl_alloc_raw2=$($ZBOX getallocation --allocation "$bl_alloc_id" --json $BW 2>&1)
                bl_alloc_json2=$(echo "$bl_alloc_raw2" | grep -v 'sdk.go\|wallet_base\|SDK Version\|INFO\|WARN\|ERROR' | grep '{' | tail -1)
                bl_blobbers_after=$(echo "$bl_alloc_json2" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(len(d.get('blobbers',[])))" 2>/dev/null | tr -d '[:space:]')
                if [ "${bl_blobbers_after:-0}" -gt "${bl_blobbers_before:-0}" ]; then
                    log_pass "Blimp Settings: new blobber visible in allocation info ($bl_blobbers_before→$bl_blobbers_after blobbers)"
                else
                    log_warn "Blimp Settings: blobber count unchanged after add ($bl_blobbers_before blobbers)"
                fi
                # Upload after add blobber to verify it works with new blobber
                local up_after
                up_after=$($ZBOX upload --allocation "$bl_alloc_id" \
                    --localpath /tmp/bl_verify_$$.txt --remotepath /docs/after_add.txt $BW 2>&1)
                echo "$up_after" | grep -qiE "Success|uploaded" && \
                    log_pass "Blimp: upload works after adding blobber" || \
                    log_warn "Blimp: upload after add blobber: $(echo "$up_after" | tail -1)"
            else
                log_warn "Blimp Settings add blobber: $(echo "$add_out" | grep -v 'sdk.go\|INFO\|DEBUG' | tail -2)"
            fi
        else
            log_warn "Blimp Settings add blobber: no free blobber to add (all in use)"
        fi

        # j. Repair (Blimp: Settings → Repair Allocation)
        # start-repair re-uploads any missing shards across blobbers
        local repair_dir="/tmp/bl_repair_$$"
        mkdir -p "$repair_dir"
        cp /tmp/bl_verify_$$.txt "$repair_dir/verify.txt" 2>/dev/null || true
        cp /tmp/bl_verify_$$.pdf "$repair_dir/verify.pdf" 2>/dev/null || true
        cp /tmp/bl_verify_$$.png "$repair_dir/verify.png" 2>/dev/null || true
        local repair_out
        repair_out=$($ZBOX start-repair --allocation "$bl_alloc_id" \
            --repairpath /docs/ --rootpath "$repair_dir" $BW 2>&1)
        echo "$repair_out" | grep -qi "Repair file completed" && \
            log_pass "Blimp Settings repair: $(echo "$repair_out" | grep -oP 'Repair file completed.*')" || \
            log_warn "Blimp Settings repair: $(echo "$repair_out" | grep -v 'sdk.go\|INFO\|DEBUG' | tail -2)"
        rm -rf "$repair_dir"

        # k. Upload-only mode (Blimp: Settings → Permissions → Upload Only)
        # Forbid delete, update, move, rename — uploads still work, deletions blocked
        local mode_out
        mode_out=$($ZBOX updateallocation --allocation "$bl_alloc_id" \
            --forbid_delete --forbid_update --forbid_move --forbid_rename \
            --lock 0.01 $BW 2>&1)
        if echo "$mode_out" | grep -qiE "Allocation updated|txId"; then
            log_pass "Blimp: upload-only mode set (forbid delete/update/move/rename)"
            # Upload should still work
            local ul_mode_out
            ul_mode_out=$($ZBOX upload --allocation "$bl_alloc_id" \
                --localpath /tmp/bl_verify_$$.txt --remotepath /docs/upload_only_test.txt $BW 2>&1)
            echo "$ul_mode_out" | grep -qiE "Success|uploaded" && \
                log_pass "Blimp upload-only mode: upload works ✓" || \
                log_warn "Blimp upload-only mode: upload failed: $(echo "$ul_mode_out" | tail -1)"
            # Delete should be blocked
            local del_mode_out
            del_mode_out=$($ZBOX delete --allocation "$bl_alloc_id" \
                --remotepath /docs/verify.txt $BW 2>&1)
            echo "$del_mode_out" | grep -qi "not permitted" && \
                log_pass "Blimp upload-only mode: delete blocked ✓" || \
                log_warn "Blimp upload-only mode: delete not blocked: $(echo "$del_mode_out" | tail -1)"
        else
            log_warn "Blimp upload-only mode: $(echo "$mode_out" | grep -v 'sdk.go\|INFO\|DEBUG' | tail -2)"
        fi

        # l. Freeze mode (Blimp: Settings → Freeze Allocation — no new uploads)
        local freeze_out
        freeze_out=$($ZBOX updateallocation --allocation "$bl_alloc_id" \
            --forbid_upload --lock 0.01 $BW 2>&1)
        if echo "$freeze_out" | grep -qiE "Allocation updated|txId"; then
            log_pass "Blimp: freeze mode set (forbid_upload)"
            sleep 5
            # Upload should now be blocked
            local ul_freeze_out
            ul_freeze_out=$($ZBOX upload --allocation "$bl_alloc_id" \
                --localpath /tmp/bl_verify_$$.txt --remotepath /docs/frozen_test.txt $BW 2>&1)
            echo "$ul_freeze_out" | grep -qi "not permitted" && \
                log_pass "Blimp freeze mode: upload blocked ✓" || \
                log_warn "Blimp freeze mode: upload not blocked: $(echo "$ul_freeze_out" | tail -1)"
            # Existing files should still download
            local dl_freeze_out sha_freeze
            dl_freeze_out=$($ZBOX download --allocation "$bl_alloc_id" \
                --remotepath /docs/verify.pdf --localpath /tmp/bl_freeze_dl_$$.pdf $BW 2>&1)
            if echo "$dl_freeze_out" | grep -qiE "Status completed|100%"; then
                sha_freeze=$(sha256sum /tmp/bl_freeze_dl_$$.pdf | awk '{print $1}')
                [ "$sha_pdf_up" = "$sha_freeze" ] && \
                    log_pass "Blimp freeze mode: download still works + hash verified ✓" || \
                    log_warn "Blimp freeze mode: download hash mismatch"
            else
                log_warn "Blimp freeze mode: download failed: $(echo "$dl_freeze_out" | tail -1)"
            fi
            rm -f /tmp/bl_freeze_dl_$$.pdf
        else
            log_warn "Blimp freeze mode: $(echo "$freeze_out" | grep -v 'sdk.go\|INFO\|DEBUG' | tail -2)"
        fi

        # Cleanup
        $ZBOX cancel-allocation --allocation "$bl_alloc_id" $BW >/dev/null 2>&1 || true
        rm -f /tmp/bl_verify_$$.txt /tmp/bl_verify_$$.pdf /tmp/bl_verify_$$.png
    fi

    # m. Blimp .env URL check
    _check_env_urls "Blimp" "${BASE_DIR}/web-apps/packages/blimp/.env"
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 7: Bolt — Wallet App (faucet, send, balance, staking)
# ══════════════════════════════════════════════════════════════════════════════
check_bolt() {
    log_header "Bolt — Wallet App Checks"

    # a. Bolt app serving
    local code; code=$(curl_ok "http://127.0.0.1:${APP_PORTS[bolt]}/" 5)
    [[ "$code" =~ ^(200|301|302|304)$ ]] && log_pass "Bolt app serving on port ${APP_PORTS[bolt]}" || \
        log_fail "Bolt app NOT serving (HTTP $code) — bash scripts/deploy_local.sh web-apps"

    # b. Faucet functional (Bolt lets users pour tokens)
    local faucet_resp
    faucet_resp=$(curl_json "${SHARDER_URL}/v1/screst/${FAUCET_SC}/get")
    if echo "$faucet_resp" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_pass "Faucet SC responding (Bolt can pour tokens)"
    else
        # Faucet SC /get might not exist; try pour via zwallet
        if [ -x "$ZWALLET" ]; then
            local faucet_out
            faucet_out=$($ZWALLET faucet --methodName pour --input '{"Once":true}' \
                --configDir "$ZCN_CONFIG_DIR" --wallet "$ZCN_WALLET_FILE" --config "$ZCN_CONFIG_FILE" 2>&1 || true)
            if echo "$faucet_out" | grep -qiE "success|Execute faucet|Hash:"; then
                log_pass "Faucet SC: pour transaction works"
            else
                log_warn "Faucet SC: $(echo "$faucet_out" | tail -1)"
            fi
        else
            log_warn "Faucet SC /get returned non-JSON; zwallet not found to test further"
        fi
    fi

    # c. Wallet balance check
    if [ -x "$ZWALLET" ]; then
        local bal_out
        bal_out=$($ZWALLET getbalance --configDir "$ZCN_CONFIG_DIR" \
            --wallet "$ZCN_WALLET_FILE" --config "$ZCN_CONFIG_FILE" 2>&1 || true)
        if echo "$bal_out" | grep -qiE "Balance:|ZCN"; then
            log_pass "Bolt wallet balance: $(echo "$bal_out" | grep -oP 'Balance:.*' | head -1)"
        else
            log_warn "Bolt wallet balance check: $(echo "$bal_out" | tail -1)"
        fi
    fi

    # d. Miner staking available (Bolt UI shows stake/unstake)
    local miner_data
    miner_data=$(curl_json "${SHARDER_URL}/v1/screst/${MINER_SC}/getMinerList")
    local staked_miners
    staked_miners=$(echo "$miner_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(sum(1 for n in d.get('Nodes',[]) if n.get('simple_miner',{}).get('total_stake',0)>0))
" 2>/dev/null || echo 0)
    if [ "${staked_miners:-0}" -gt 0 ]; then
        log_pass "Bolt staking: $staked_miners miners with non-zero stake (visible in Bolt UI)"
    else
        log_warn "Bolt staking: no miners have stake — Bolt stake UI shows empty"
        log_info "Fix: bash scripts/deploy_local.sh fund"
    fi

    # e. Auth: 0box deployment_mode=3 (Bolt uses 0box for wallet operations)
    local dm; dm=$(docker inspect --format='{{range .Config.Cmd}}{{.}} {{end}}' 0box 2>/dev/null | \
        grep -oP -- '--deployment_mode \K\d+' || echo "unknown")
    [ "$dm" = "3" ] && log_pass "Bolt auth: 0box deployment_mode=3" || \
        log_fail "Bolt auth: 0box deployment_mode=$dm — Bolt gets 401 on wallet operations"

    # f. Send ZCN transaction test (basic chain functionality for Bolt send)
    if [ -x "$ZWALLET" ]; then
        # Just check the chain responds to a balance query (send requires a recipient)
        local bal
        bal=$(curl_json "${SHARDER_URL}/v1/client/get/balance?client_id=$(python3 -c "import json; print(json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}'))['client_id'])" 2>/dev/null)" | \
            python3 -c "import sys,json; print(json.load(sys.stdin).get('balance',0))" 2>/dev/null || echo "?")
        if [ "$bal" != "?" ] && [ "$bal" != "0" ]; then
            log_pass "Bolt send: wallet has balance=$bal (send transaction possible)"
        else
            log_warn "Bolt send: wallet balance=$bal (may need faucet pour)"
        fi
    fi

    # g. Bolt .env URL check
    _check_env_urls "Bolt" "${BASE_DIR}/web-apps/packages/bolt/.env"
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 8: Atlus / Explorer — dashboard data
# ══════════════════════════════════════════════════════════════════════════════
check_atlus() {
    log_header "Atlus — Explorer Dashboard Data"

    # Miners
    local miner_data
    miner_data=$(curl_json "${SHARDER_URL}/v1/screst/${MINER_SC}/getMinerList")
    local total_miners staked_miners
    total_miners=$(echo "$miner_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); print(len(d.get('Nodes',[])))" 2>/dev/null || echo 0)
    staked_miners=$(echo "$miner_data" | python3 -c "
import sys,json; d=json.load(sys.stdin)
print(sum(1 for n in d.get('Nodes',[]) if n.get('simple_miner',{}).get('total_stake',0)>0))" 2>/dev/null || echo 0)

    [ "$total_miners" -gt 0 ] && log_pass "Miners on chain: $total_miners" || log_fail "No miners from getMinerList"
    [ "${staked_miners:-0}" -gt 0 ] && log_pass "Miners with stake: $staked_miners/$total_miners" || \
        { log_fail "ALL miners have zero stake — Atlus stake column empty — bash scripts/deploy_local.sh fund"; }

    # Sharders
    local sharder_data
    sharder_data=$(curl_json "${SHARDER_URL}/v1/screst/${MINER_SC}/getSharderList")
    local staked_sharders total_sharders
    total_sharders=$(echo "$sharder_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); print(len(d.get('Nodes',[])))" 2>/dev/null || echo 0)
    staked_sharders=$(echo "$sharder_data" | python3 -c "
import sys,json; d=json.load(sys.stdin)
print(sum(1 for n in d.get('Nodes',[]) if n.get('simple_miner',{}).get('total_stake',0)>0))" 2>/dev/null || echo 0)
    [ "${staked_sharders:-0}" -gt 0 ] && log_pass "Sharders with stake: $staked_sharders/$total_sharders" || \
        log_fail "Sharders have zero stake — bash scripts/deploy_local.sh fund"

    # Validators
    local validator_data
    validator_data=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/validators?limit=20&offset=0")
    local total_validators staked_validators active_validators
    total_validators=$(echo "$validator_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[])); print(len(nodes))" 2>/dev/null || echo 0)
    staked_validators=$(echo "$validator_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
print(sum(1 for v in nodes if v.get('total_stake',0)>0))" 2>/dev/null || echo 0)
    active_validators=$(echo "$validator_data" | python3 -c "
import sys,json,time; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[])); now=int(time.time())
print(sum(1 for v in nodes if v.get('last_health_check',0)>now-3600))" 2>/dev/null || echo 0)
    [ "$total_validators" -gt 0 ] && log_pass "Validators on chain: $total_validators (staked: $staked_validators, active: $active_validators)" || \
        log_fail "No validators from SC — bash scripts/deploy_local.sh blobbers"
    if [ "${staked_validators:-0}" -lt 2 ]; then
        log_fail "Less than 2 validators staked — challenges will not generate"
        log_info "Fix: bash scripts/deploy_local.sh fund"
    fi

    # Blobbers
    local blobber_data
    blobber_data=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getblobbers?limit=20&offset=0")
    local total_blobbers challenged_blobbers enterprise_blobbers
    total_blobbers=$(echo "$blobber_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[])); print(len(nodes))" 2>/dev/null || echo 0)
    challenged_blobbers=$(echo "$blobber_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
print(sum(1 for b in nodes if b.get('challenge_passed',0)+b.get('challenge_failed',0)>0))" 2>/dev/null || echo 0)
    enterprise_blobbers=$(echo "$blobber_data" | python3 -c "
import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
print(sum(1 for b in nodes if b.get('is_enterprise',False)))" 2>/dev/null || echo 0)

    [ "$total_blobbers" -gt 0 ] && log_pass "Blobbers on chain: $total_blobbers (enterprise: $enterprise_blobbers)" || \
        log_fail "No blobbers from getblobbers"
    [ "${challenged_blobbers:-0}" -gt 0 ] && log_pass "Blobbers with challenge history: $challenged_blobbers/$total_blobbers" || \
        log_warn "No challenge history — crawler may need time to generate activity"

    # Blobber health check — newer blobbers (2024+) require auth for /_stats.
    # Use /healthcheck endpoint (no auth needed) to verify blobber reachability.
    # Fall back to /_stats HTML check for older blobbers.
    log_info "Checking blobber health endpoints..."
    local stats_ok=0 stats_fail=0
    while IFS= read -r burl; do
        [ -z "$burl" ] && continue
        # Try /healthcheck first (unauthenticated, returns 200 on healthy blobbers)
        local hc_code; hc_code=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' "${burl}/healthcheck" 2>/dev/null) || true
        if [[ "$hc_code" =~ ^(200|204)$ ]]; then
            stats_ok=$((stats_ok+1))
            continue
        fi
        # Fall back: /_stats — newer blobbers return 401 (auth required), older return 200 + HTML
        local sr_code; sr_code=$(curl -sk -m 8 -o /tmp/_stats_resp.html -w '%{http_code}' "${burl}/_stats" 2>/dev/null) || true
        sr_code=${sr_code:-0}
        if [[ "$sr_code" =~ ^(200|301|302)$ ]] && grep -q "Blobber" /tmp/_stats_resp.html 2>/dev/null; then
            stats_ok=$((stats_ok+1))
        elif [[ "$sr_code" =~ ^(401|403)$ ]]; then
            # Auth required — blobber is running, just locked down
            stats_ok=$((stats_ok+1))
        else
            # Final fallback: enterprise blobbers serve info at root but have no /_stats or /healthcheck
            local root_code; root_code=$(curl -sk -m 8 -o /dev/null -w '%{http_code}' "${burl}/" 2>/dev/null) || true
            if [[ "$root_code" =~ ^(200|204)$ ]]; then
                stats_ok=$((stats_ok+1))
            else
                log_fail "Blobber FAIL: ${burl} (healthcheck=$hc_code, /_stats=$sr_code)"
                stats_fail=$((stats_fail+1))
            fi
        fi
    done < <(echo "$blobber_data" | python3 -c "
import sys,json,re
d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
# Inline NGINX_DOMAIN value (not exported to subprocess env)
nginx_domain='${NGINX_DOMAIN}'
for b in nodes:
    url=b.get('url','').rstrip('/')
    if not url: continue
    # Keep: our nginx domain, local IPs (198.18.x.x:port), localhost
    if nginx_domain and nginx_domain in url: print(url)
    elif re.match(r'https?://(127\.|198\.|10\.|172\.)', url): print(url)
" 2>/dev/null)

    if   [ "$stats_ok" -gt 0 ] && [ "$stats_fail" -eq 0 ]; then log_pass "All $stats_ok blobbers healthy"
    elif [ "$stats_ok" -gt 0 ]; then log_warn "$stats_ok blobbers healthy, $stats_fail unreachable"
    else log_fail "All blobber health checks failed"; fi

    # Provider rewards — check that at least some providers have earned rewards
    local reward_count; reward_count=$(docker exec -e PGPASSWORD=zbox_server postgres-0box psql -U zbox_user -d zbox -t -c \
        "SELECT count(*) FROM provider_rewards WHERE total_rewards > 0;" 2>/dev/null | tr -d ' \n' || echo 0)
    if [ "${reward_count:-0}" -gt 0 ]; then
        local reward_top; reward_top=$(docker exec -e PGPASSWORD=zbox_server postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT provider_type, count(*), round(sum(total_rewards)/1e10,2) AS total_zcn FROM provider_rewards WHERE total_rewards>0 GROUP BY provider_type ORDER BY total_zcn DESC;" \
            2>/dev/null | head -5 | tr -s ' ' || echo "")
        log_pass "Provider rewards: $reward_count providers earning rewards"
        [ -n "$reward_top" ] && log_info "Rewards breakdown:$reward_top"
    else
        log_warn "Provider rewards: no providers have earned rewards yet"
        log_info "Fix: check Kafka pipeline flowing — bash scripts/deploy_local.sh fix-kafka"
    fi

    # Transactions — latest block
    local tx_resp
    tx_resp=$(curl_json "${SHARDER_URL}/v1/block/get?content=full&round=latest")
    local tx_count
    tx_count=$(echo "$tx_resp" | python3 -c "
import sys,json; d=json.load(sys.stdin); print(len(d.get('block',{}).get('transactions',[])))" 2>/dev/null || echo 0)
    if [ "$tx_count" -gt 0 ]; then
        log_pass "Latest block has $tx_count transactions"
        # Transaction link (lookup by hash)
        local tx_hash
        tx_hash=$(echo "$tx_resp" | python3 -c "
import sys,json; txns=json.load(sys.stdin).get('block',{}).get('transactions',[])
if txns: print(txns[0].get('hash',''))" 2>/dev/null || echo "")
        if [ -n "$tx_hash" ]; then
            local tx_code; tx_code=$(curl_ok "${SHARDER_URL}/v1/transaction/get/confirmation?hash=${tx_hash}")
            [ "$tx_code" = "200" ] && log_pass "Transaction link works (/v1/transaction/get/confirmation)" || \
                log_fail "Transaction link broken — HTTP $tx_code"
        fi
    else
        log_warn "Latest block has 0 transactions (chain idle)"
    fi

    # 0box graph APIs (Atlus charts) — require from/to range params
    local now_unix; now_unix=$(date +%s)
    local from_unix=$(( now_unix - 86400 * 7 ))  # 7 days ago
    local to_unix="$now_unix"

    for ep_name in "write-price" "challenges" "token-supply" "txns-count" "used-storage" "total-allocations"; do
        local resp
        resp=$(curl -sk -m 10 "${OBOX_URL}/v2/graph-${ep_name}?from=${from_unix}&to=${to_unix}&data-points=10" 2>/dev/null)
        local count
        count=$(echo "$resp" | python3 -c "
import sys,json
d=json.load(sys.stdin)
if isinstance(d,list): print(len(d))
elif isinstance(d,dict):
    for k in ('data','values','successful','total'):
        if k in d and isinstance(d[k],list):
            print(len(d[k])); break
    else: print(0)
else: print(0)
" 2>/dev/null || echo 0)
        if [ "${count:-0}" -gt 0 ]; then
            log_pass "0box graph $ep_name: $count data points"
        else
            log_warn "0box graph $ep_name: no data points (resp: ${resp:0:80})"
            log_info "Fix: bash scripts/deploy_local.sh fix-kafka  (populates graph data)"
        fi
    done

    # Elasticsearch search — verify search returns results (not just health)
    local es_health; es_health=$(curl -s -m 5 "http://127.0.0.1:9200/_cluster/health" 2>/dev/null)
    local es_status; es_status=$(echo "$es_health" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
    if [ -n "$es_status" ]; then
        log_pass "Elasticsearch cluster: $es_status"
        # Try a real search — look for any provider by querying 0box search endpoint
        local es_indices; es_indices=$(curl -s -m 5 "http://127.0.0.1:9200/_cat/indices?format=json" 2>/dev/null | \
            python3 -c "import json,sys; data=json.load(sys.stdin); print(sum(int(i.get('docs.count','0')) for i in data))" 2>/dev/null || echo 0)
        if [ "${es_indices:-0}" -gt 0 ]; then
            log_pass "Elasticsearch has $es_indices indexed documents (search functional)"
        else
            log_warn "Elasticsearch has 0 indexed documents — search will return nothing"
            log_info "Fix: ensure crawler is running — bash scripts/deploy_local.sh crawler"
        fi
        # Monitor ES resource usage
        local es_mem; es_mem=$(docker stats --no-stream --format '{{.MemUsage}}' elasticsearch 2>/dev/null | awk '{print $1}' || echo "unknown")
        local es_cpu; es_cpu=$(docker stats --no-stream --format '{{.CPUPerc}}' elasticsearch 2>/dev/null || echo "unknown")
        log_info "Elasticsearch resources: CPU=$es_cpu  MEM=$es_mem"
    else
        log_fail "Elasticsearch not reachable — Atlus search broken"
        log_info "Fix: check elasticsearch container — docker logs elasticsearch --tail 20"
    fi

    # Atlus app serving
    local atlus_code; atlus_code=$(curl_ok "http://127.0.0.1:${APP_PORTS[explorer]}/" 5)
    [[ "$atlus_code" =~ ^(200|301|302|304)$ ]] && log_pass "Atlus/Explorer app serving on port ${APP_PORTS[explorer]}" || \
        log_fail "Atlus/Explorer app NOT serving (HTTP $atlus_code)"

    # Atlus custom domain UI/UX navigation page checks (test.atlus.cloud)
    log_info "Checking Atlus custom domain navigation pages (https://${ATLUS_DOMAIN})..."
    local atlus_pages=(
        "/:Home (dashboard)"
        "/miners:Miners list"
        "/sharders:Sharders list"
        "/blobbers:Blobbers list"
        "/validators:Validators list"
        "/transactions:Transactions list"
        "/blocks:Blocks list"
    )
    local atlus_pg_ok=0 atlus_pg_warn=0
    for pg in "${atlus_pages[@]}"; do
        local _path="${pg%%:*}" _desc="${pg#*:}"
        local _c; _c=$(curl -sk -o /dev/null -w '%{http_code}' -m 8 \
            "https://${ATLUS_DOMAIN}${_path}" 2>/dev/null || echo 0)
        if [[ "$_c" =~ ^(200|301|302|304)$ ]]; then
            atlus_pg_ok=$((atlus_pg_ok+1))
        else
            log_warn "Atlus page ${_path} (${_desc}): HTTP $_c"
            atlus_pg_warn=$((atlus_pg_warn+1))
        fi
    done
    local atlus_pg_total=$(( atlus_pg_ok + atlus_pg_warn ))
    if [ "$atlus_pg_warn" -eq 0 ]; then
        log_pass "Atlus domain pages: all $atlus_pg_ok/$atlus_pg_total reachable at https://${ATLUS_DOMAIN}"
    else
        log_warn "Atlus domain pages: $atlus_pg_ok/$atlus_pg_total OK, $atlus_pg_warn failed — check nginx + DNS for ${ATLUS_DOMAIN}"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 9: Blobber Health
# ══════════════════════════════════════════════════════════════════════════════
check_blobbers() {
    log_header "Blobber Health"

    local reg_ok=0 reg_fail=0
    for i in $(seq 1 9); do
        local code; code=$(curl_ok "http://127.0.0.1:505${i}/" 3)
        [[ "$code" =~ ^[0-9]+$ ]] && [ "$code" -gt 0 ] && reg_ok=$((reg_ok+1)) || reg_fail=$((reg_fail+1))
    done
    [ "$reg_ok" -gt 0 ] && log_pass "$reg_ok/9 regular blobbers responding" || true
    [ "$reg_fail" -gt 0 ] && log_fail "$reg_fail regular blobbers NOT responding — bash scripts/deploy_local.sh blobbers"

    local ent_ok=0 ent_fail=0
    for i in 1 2 3; do
        local code; code=$(curl_ok "http://127.0.0.1:507${i}/" 3)
        [[ "$code" =~ ^[0-9]+$ ]] && [ "$code" -gt 0 ] && ent_ok=$((ent_ok+1)) || ent_fail=$((ent_fail+1))
    done
    [ "$ent_ok" -gt 0 ] && log_pass "$ent_ok/3 enterprise blobbers responding"
    [ "$ent_fail" -gt 0 ] && log_warn "$ent_fail enterprise blobbers not responding"

    # On-chain health check freshness
    local blobber_data; blobber_data=$(curl_json "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getblobbers?limit=20")
    local stale_count; stale_count=$(echo "$blobber_data" | python3 -c "
import sys,json,time,re,os
d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[])); now=int(time.time())
nginx_domain=os.environ.get('NGINX_DOMAIN','')
# Only check blobbers with real URLs (our nginx domain or known IPs), skip test artifacts
def is_real(b):
    url=b.get('url','')
    if nginx_domain and nginx_domain in url: return True
    return bool(re.match(r'https?://(127\.|198\.|10\.|172\.)',url))
print(sum(1 for b in nodes if is_real(b) and b.get('last_health_check',0)<now-3600))" 2>/dev/null || echo 0)
    [ "${stale_count:-0}" -eq 0 ] && log_pass "All infrastructure blobbers health-checked within last hour" || \
        log_warn "$stale_count blobber(s) stale health check — bash scripts/deploy_local.sh blobbers"

    # Delegate wallet mismatch
    local expected_delegate
    expected_delegate=$(python3 -c "
import json; print(json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}'))['client_id'])" 2>/dev/null || echo "")
    if [ -n "$expected_delegate" ]; then
        local mismatch; mismatch=$(echo "$blobber_data" | python3 -c "
import sys,json,re,os; d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
exp='${expected_delegate}'
nginx_domain=os.environ.get('NGINX_DOMAIN','')
def is_real(b):
    url=b.get('url','')
    if nginx_domain and nginx_domain in url: return True
    return bool(re.match(r'https?://(127\.|198\.|10\.|172\.)',url))
print(sum(1 for b in nodes if is_real(b) and not b.get('is_enterprise',False) and b.get('stake_pool_settings',{}).get('delegate_wallet','') and b.get('stake_pool_settings',{}).get('delegate_wallet','')!=exp))" 2>/dev/null || echo 0)
        [ "${mismatch:-0}" -eq 0 ] && log_pass "Blobber delegate wallets match local.json" || \
            log_fail "$mismatch blobber(s) have wrong delegate_wallet — bl-update tests FAIL — bash scripts/deploy_local.sh blobbers"
    fi
}

# ══════════════════════════════════════════════════════════════════════════════
# SECTION 10: Run Test Suites
# ══════════════════════════════════════════════════════════════════════════════
run_test_suites() {
    log_header "Running Test Suites: ${TEST_SUITES[*]}"
    log_info "Live HTML: /var/log/0chain/test_results.html"

    bash "${SCRIPT_DIR}/run_tests.sh" "${TEST_SUITES[@]}"
    local rc=$?
    [ $rc -eq 0 ] && log_pass "All test suites PASSED" || \
        log_fail "One or more test suites FAILED — see ${SYSTEM_TEST_DIR}/test_results/"
    return $rc
}

# ══════════════════════════════════════════════════════════════════════════════
# SUMMARY
# ══════════════════════════════════════════════════════════════════════════════
print_summary() {
    echo ""
    echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  VERIFICATION SUMMARY${NC}"
    echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
    echo -e "  Domain:   ${NGINX_DOMAIN}  (prefix: ${APP_DOMAIN_PREFIX})"
    echo -e "  Vult:     https://${VULT_DOMAIN}/"
    echo -e "  Blimp:    https://${BLIMP_DOMAIN}/"
    echo -e "  Bolt:     https://${BOLT_DOMAIN}/"
    echo -e "  Atlus:    https://${ATLUS_DOMAIN}/"
    echo ""
    echo -e "  ${GREEN}PASS${NC}: $PASS_COUNT   ${YELLOW}WARN${NC}: $WARN_COUNT   ${RED}FAIL${NC}: $FAIL_COUNT"
    if [ ${#FAILED_CHECKS[@]} -gt 0 ]; then
        echo ""
        echo -e "  ${RED}${BOLD}Failed checks:${NC}"
        for fc in "${FAILED_CHECKS[@]}"; do
            echo -e "    ${RED}✗${NC} $fc"
        done
    fi
    echo ""
    if [ "$FAIL_COUNT" -eq 0 ]; then
        echo -e "  ${GREEN}${BOLD}ALL CHECKS PASSED${NC}"
    else
        echo -e "  ${RED}${BOLD}$FAIL_COUNT FAILED — see above${NC}"
        echo ""
        echo -e "  ${YELLOW}ALL fixes must go into scripts/deploy_local.sh or test code.${NC}"
        echo -e "  ${YELLOW}Never make manual one-off changes on the server.${NC}"
    fi
    echo ""
}

# ══════════════════════════════════════════════════════════════════════════════
# MAIN
# ══════════════════════════════════════════════════════════════════════════════
main() {
    parse_args "$@"

    echo -e "${BOLD}0Chain Comprehensive Verification${NC}"
    echo "Domain: ${NGINX_DOMAIN}  Apps: ${VULT_DOMAIN} / ${BLIMP_DOMAIN} / ${BOLT_DOMAIN}"
    echo "0box: ${OBOX_URL}  |  Time: $(date)"

    case "${RUN_SECTION:-all}" in
        chain)    check_chain_health ;;
        services) check_services ;;
        cors)     check_cors ;;
        webapps)  check_web_apps ;;
        vult)     check_vult ;;
        blimp)    check_blimp ;;
        bolt)     check_bolt ;;
        atlus)    check_atlus ;;
        blobbers) check_blobbers ;;
        all)
            check_chain_health
            check_services
            check_cors
            check_web_apps
            check_vult
            check_blimp
            check_bolt
            check_atlus
            check_blobbers
            ;;
        *) echo "Unknown section: $RUN_SECTION"; exit 1 ;;
    esac

    $RUN_TESTS && run_test_suites

    print_summary
    [ "$FAIL_COUNT" -eq 0 ]
}

main "$@"
