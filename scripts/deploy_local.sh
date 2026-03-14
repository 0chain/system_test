#!/bin/bash
# 0Chain Complete Local Deployment Script
# Deploys all services based on deploy_config.yaml

# NOTE: No 'set -e' — the deploy script must be robust and continue
# through transient failures. Each phase handles its own errors.

# Ensure common binary paths are in PATH (needed for tmux/non-login shells)
export PATH="$PATH:/usr/local/go/bin:/root/go/bin:/usr/local/bin"

# Set up automatic logging: all stdout/stderr goes to both terminal AND log file.
# Uses line-buffered output so logs appear immediately (not block-buffered like pipe to tee).
DEPLOY_LOG="${DEPLOY_LOG:-/tmp/deploy_local.log}"
if [ -z "$_DEPLOY_LOGGING_SET" ]; then
    export _DEPLOY_LOGGING_SET=1
    # Only truncate log for full deploy/redeploy (not individual commands like nginx, start-chaos)
    if [ "${1:-all}" = "all" ] || [ "${1:-all}" = "redeploy" ]; then
        echo "" > "$DEPLOY_LOG"
    fi
    # stdbuf is GNU coreutils (not available on macOS) — fall back to plain tee
    if command -v stdbuf &>/dev/null; then
        exec > >(stdbuf -oL tee -a "$DEPLOY_LOG") 2>&1
    else
        exec > >(tee -a "$DEPLOY_LOG") 2>&1
    fi
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/deploy_config.yaml"
LOCAL_CONFIG_FILE="${SCRIPT_DIR}/deploy_config.local.yaml"  # Per-host overrides (excluded from rsync)
BASE_DIR="${HOME}/Code"

# CLI tool paths
ZWALLET="${BASE_DIR}/zwalletcli/zwallet"
ZBOX="${BASE_DIR}/zboxcli/zbox"

# Configuration
# IMPORTANT: Use local.yaml (localhost addresses) for all SC owner operations.
# The 198.18.0.x addresses in zbox_config.yaml cause authorization failures
# for sc-update-config, mn-update-config, and bl-update when run from the host.
ZCN_CONFIG_DIR="${HOME}/.zcn"
ZCN_CONFIG_FILE="local.yaml"
ZCN_WALLET_FILE="owner.json"
# SC owner wallet — used for sc-update-config (must match the owner_id in genesis block)
# Same as ZCN_WALLET_FILE: single wallet for all operations avoids local.json vs owner.json confusion
ZCN_SC_OWNER_WALLET="${ZCN_SC_OWNER_WALLET:-owner.json}"

# Blobber key config directory
BLOBBER_KEYS_DIR="${BASE_DIR}/blobber/docker.local/keys_config"

# Load per-host local config overrides (deploy_config.local.yaml — NOT rsynced, machine-specific)
# This allows setting domain=test1.zus.network on 65.x without changing the shared config file.
if [ -f "$LOCAL_CONFIG_FILE" ]; then
    _local_domain=$(grep "^  domain:" "$LOCAL_CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    _local_prefix=$(grep "^  domain_prefix:" "$LOCAL_CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    if [ -n "$_local_domain" ] && [ -z "${NGINX_DOMAIN:-}" ]; then
        export NGINX_DOMAIN="$_local_domain"
        print_status "Local config override: NGINX_DOMAIN=${NGINX_DOMAIN}" 2>/dev/null || true
    fi
    if [ -n "$_local_prefix" ] && [ -z "${APP_DOMAIN_PREFIX:-}" ]; then
        export APP_DOMAIN_PREFIX="$_local_prefix"
    fi
fi

# Load secrets from .secrets.env (gitignored) if it exists.
# Secrets include: Firebase keys, Auth0 credentials, Alchemy API key,
# Tenderly RPC URLs, Zendesk token, Kafka/MinIO passwords, etc.
# See .secrets.env.template for the full list.
SECRETS_FILE="${SCRIPT_DIR}/.secrets.env"
if [ -f "$SECRETS_FILE" ]; then
    # Source the secrets file (export all variables)
    set -a
    # shellcheck disable=SC1090
    source "$SECRETS_FILE"
    set +a
    # Don't print the file contents — just confirm it loaded
    print_status "Loaded secrets from ${SECRETS_FILE}" 2>/dev/null || true
else
    print_warning "No .secrets.env found at ${SECRETS_FILE}" 2>/dev/null || true
    print_warning "Copy .secrets.env.template to .secrets.env and fill in values" 2>/dev/null || true
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Run a zwallet/zbox command with retry and exponential backoff.
# Adapted from 0chain's deploy_local.sh — prevents silent failures.
# Usage: run_cmd "description" "full command string"
run_cmd() {
    local desc=$1
    shift
    local cmd="$*"
    local max_retries=5
    local retry=0
    local backoff=3

    print_status "$desc"
    while [ $retry -lt $max_retries ]; do
        local output exit_code
        output=$(eval "$cmd" 2>&1) && exit_code=0 || exit_code=$?

        # Success indicators
        if echo "$output" | grep -qiE "success|confirmed|Execute faucet|already|Hash:|settings updated"; then
            print_status "  OK"
            return 0
        fi

        # Authorization error — don't retry
        if echo "$output" | grep -qiE "unauthorized|access denied"; then
            print_error "  Authorization error — wrong wallet?"
            print_error "  Output: $(echo "$output" | tail -1)"
            return 1
        fi

        # Network/nonce error — retry with backoff
        if echo "$output" | grep -qiE "connection refused|timeout|too less sharders|unexpected end|invalid transaction nonce"; then
            retry=$((retry + 1))
            print_warning "  Network error, retrying in ${backoff}s ($retry/$max_retries): $(echo "$output" | tail -1 | head -c 120)"
            sleep $backoff
            backoff=$((backoff * 2))
            [ $backoff -gt 20 ] && backoff=20
            continue
        fi

        # zwallet returned 0 — treat as success
        if [ $exit_code -eq 0 ]; then
            print_status "  OK"
            return 0
        fi

        retry=$((retry + 1))
        print_warning "  Failed ($retry/$max_retries): $(echo "$output" | tail -1 | head -c 120)"
        sleep $backoff
        backoff=$((backoff * 2))
        [ $backoff -gt 20 ] && backoff=20
    done

    print_error "  FAILED after $max_retries retries: $desc"
    return 1
}

# Parse YAML configuration (basic parsing)
get_config() {
    local key=$1
    grep "^  $key:" "$CONFIG_FILE" 2>/dev/null | head -1 | awk -F': ' '{print $2}' | sed 's/#.*//' | tr -d '"' | xargs
}

# Get branch for a repository from config YAML
# Returns the branch specified in deploy_config.yaml under repositories.<repo>.branch
# Can be overridden via --branch flags (stored in BRANCH_OVERRIDES associative array)
get_repo_branch() {
    local repo="$1"
    local default="${2:-master}"

    # Check CLI overrides first
    if [ -n "${BRANCH_OVERRIDES[$repo]+x}" ]; then
        echo "${BRANCH_OVERRIDES[$repo]}"
        return
    fi

    # Parse from YAML: look for the repo section and its branch field
    local in_repo=false
    local branch=""
    while IFS= read -r line; do
        if echo "$line" | grep -qP "^\s+${repo}:"; then
            in_repo=true
            continue
        fi
        if $in_repo; then
            if echo "$line" | grep -qP '^\s+branch:'; then
                branch=$(echo "$line" | awk -F': ' '{print $2}' | sed 's/#.*//' | tr -d ' "'"'"'')
                break
            fi
            # If we hit another repo key (non-indented property), stop
            if echo "$line" | grep -qP '^\s+\w+:$'; then
                break
            fi
        fi
    done < "$CONFIG_FILE"

    echo "${branch:-$default}"
}

# Checkout repositories to branches specified in deploy_config.yaml
# Reads each repository's branch and checks it out, pulling latest changes.
checkout_branches() {
    print_header "Checking Out Repository Branches"

    local repos=("0chain" "0dns" "blobber" "gosdk" "eblobber" "0box" "zauth-server" "zvault" "zs3server" "crawler" "web-apps" "zboxcli" "zwalletcli")

    for repo in "${repos[@]}"; do
        local branch
        branch=$(get_repo_branch "$repo")
        local repo_path="${BASE_DIR}/${repo}"

        # Handle path override from config
        local config_path
        config_path=$(get_repo_branch_path "$repo")
        if [ -n "$config_path" ]; then
            repo_path="${BASE_DIR}/${config_path}"
        fi

        if [ ! -d "$repo_path" ]; then
            print_warning "Repository not found: $repo_path (skipping)"
            continue
        fi

        print_status "Checking out ${repo} → ${branch}"

        cd "$repo_path"

        # Stash any local changes
        local stash_output
        stash_output=$(git stash 2>&1) || true

        # Fetch latest
        git fetch origin 2>/dev/null || {
            print_warning "Failed to fetch ${repo}, using local branch"
        }

        # Checkout branch
        if git rev-parse --verify "origin/${branch}" >/dev/null 2>&1; then
            git checkout "$branch" 2>/dev/null || git checkout -b "$branch" "origin/${branch}" 2>/dev/null || true
            git fetch origin "$branch" 2>/dev/null || true
            git reset --hard "origin/${branch}" 2>/dev/null || git pull origin "$branch" 2>/dev/null || true
        elif git rev-parse --verify "$branch" >/dev/null 2>&1; then
            git checkout "$branch" 2>/dev/null || true
        else
            print_warning "Branch '${branch}' not found for ${repo}, staying on current branch"
        fi

        local current=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
        local short_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
        print_status "  ${repo}: ${current} (${short_hash})"
    done

    cd "$SCRIPT_DIR"
    print_status "Branch checkout complete!"

    # Print branch summary table and generate deployment info page
    generate_deployment_info
}

# Print deployment branch table and generate HTML info page at /var/www/html/deploy-info.html
generate_deployment_info() {
    print_header "Deployment Branch Summary"

    local repos=("0chain" "0dns" "blobber" "gosdk" "eblobber" "0box" "zauth-server" "zvault" "zs3server" "crawler" "zboxcli" "zwalletcli")
    local deploy_time=$(date '+%Y-%m-%d %H:%M:%S %Z')

    # Print table to console
    printf "\n  %-18s %-30s %-10s\n" "REPOSITORY" "BRANCH" "COMMIT"
    printf "  %-18s %-30s %-10s\n" "──────────────────" "──────────────────────────────" "──────────"

    local html_rows=""
    for repo in "${repos[@]}"; do
        local repo_path="${BASE_DIR}/${repo}"
        local config_path
        config_path=$(get_repo_branch_path "$repo")
        if [ -n "$config_path" ]; then
            repo_path="${BASE_DIR}/${config_path}"
        fi

        if [ -d "$repo_path" ]; then
            local current=$(cd "$repo_path" && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
            local short_hash=$(cd "$repo_path" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
            local commit_date=$(cd "$repo_path" && git log -1 --format='%ci' 2>/dev/null | cut -d' ' -f1 || echo "")
            printf "  %-18s %-30s %-10s\n" "$repo" "$current" "$short_hash"
            html_rows="${html_rows}<tr><td>${repo}</td><td>${current}</td><td>${short_hash}</td><td>${commit_date}</td></tr>\n"
        else
            printf "  %-18s %-30s %-10s\n" "$repo" "(not found)" "-"
        fi
    done
    printf "\n"

    # Generate HTML deployment info page
    local DOMAIN="${NGINX_DOMAIN:-}"
    if [ -z "$DOMAIN" ] && [ -f "$CONFIG_FILE" ]; then
        DOMAIN=$(grep "^  domain:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    fi

    local html_file="/var/www/html/deploy-info.html"
    mkdir -p /var/www/html 2>/dev/null || true

    cat > "$html_file" 2>/dev/null << HTMLEOF || true
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>0Chain Local Deployment - ${DOMAIN:-localhost}</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; color: #333; }
  h1 { color: #1a73e8; border-bottom: 2px solid #1a73e8; padding-bottom: 10px; }
  h2 { color: #555; margin-top: 30px; }
  table { border-collapse: collapse; width: 100%; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); border-radius: 4px; overflow: hidden; }
  th { background: #1a73e8; color: white; padding: 12px 15px; text-align: left; }
  td { padding: 10px 15px; border-bottom: 1px solid #eee; }
  tr:hover { background: #f8f9fa; }
  .meta { color: #888; font-size: 0.9em; }
  .service-link { display: inline-block; margin: 5px 10px 5px 0; padding: 6px 12px; background: #e8f0fe; border-radius: 4px; text-decoration: none; color: #1a73e8; }
  .service-link:hover { background: #d2e3fc; }
  .status-ok { color: #34a853; font-weight: bold; }
  .status-warn { color: #ea8600; font-weight: bold; }
</style>
</head>
<body>
<h1>0Chain Local Deployment</h1>
<p class="meta">Deployed: ${deploy_time} | Domain: ${DOMAIN:-N/A}</p>

<h2>Repository Branches</h2>
<table>
<tr><th>Repository</th><th>Branch</th><th>Commit</th><th>Date</th></tr>
$(echo -e "$html_rows")
</table>

<p class="meta">Generated by deploy_local.sh</p>
</body>
</html>
HTMLEOF

    if [ -f "$html_file" ]; then
        print_status "Deployment info page: http://${DOMAIN:-localhost}/deploy-info.html"
    fi
}

# Generate the landing page dashboard at /var/www/html/dashboard.html
# Links web apps to their custom domains (test.vult.network, etc.) with fallback to path-based
generate_dashboard() {
    local DOMAIN="${1:-test.zus.network}"
    mkdir -p /var/www/html 2>/dev/null || true

    cat > /var/www/html/dashboard.html << 'DASHEOF'
<html><head><title>0Chain Test Network</title>
<style>
body { font-family: monospace; margin: 20px; font-size: 13px; }
a { color: blue; }
h3 { margin: 12px 0 4px; }
table.db { border-collapse: collapse; font-size: 12px; margin: 4px 0; width: 100%; }
table.db td { padding: 3px 8px; vertical-align: top; border: 1px solid #ddd; }
table.db tr.hdr td { background: #e0e0e0; font-weight: bold; }
table.db tr.alt td { background: #f8f8f8; }
.tbl a { color: #1565c0; text-decoration: none; margin-right: 4px; font-size: 11px; }
.tbl a:hover { text-decoration: underline; }
code { background: #eee; padding: 1px 4px; font-size: 11px; }
</style></head><body>
<b>0Chain Test Network</b>
<h3>Chain</h3>
<a href="/miner01/_diagnostics">M1</a> <a href="/miner02/_diagnostics">M2</a> <a href="/miner03/_diagnostics">M3</a> <a href="/miner04/_diagnostics">M4</a> |
<a href="/sharder01/_diagnostics">S1</a> <a href="/sharder02/_diagnostics">S2</a> |
<a href="/0dns/network">0dns</a>
<h3>Blobbers</h3>
<a href="/blobber01/_stats">B1</a> <a href="/blobber02/_stats">B2</a> <a href="/blobber03/_stats">B3</a> <a href="/blobber04/_stats">B4</a> <a href="/blobber05/_stats">B5</a> <a href="/blobber06/_stats">B6</a>
<a href="/blobber07/_stats">B7</a> <a href="/blobber08/_stats">B8</a> <a href="/blobber09/_stats">B9</a> <a href="/blobber10/_stats">B10</a> <a href="/blobber11/_stats">B11</a> <a href="/blobber12/_stats">B12</a>
|
<a href="/eblobber01/_stats">E1</a> <a href="/eblobber02/_stats">E2</a> <a href="/eblobber03/_stats">E3</a> <a href="/eblobber04/_stats">E4</a> <a href="/eblobber05/_stats">E5</a>
<h3>Apps</h3>
DASHEOF

    # Add app links using custom domains (APP_DOMAIN_PREFIX controls test vs test1 etc.)
    local _prefix="${APP_DOMAIN_PREFIX:-test}"
    echo "<a href=\"https://${_prefix}.vult.network\">Vult</a> | <a href=\"https://${_prefix}.bolt.holdings\">Bolt</a> | <a href=\"https://${_prefix}.blimp.software\">Blimp</a> | <a href=\"https://${_prefix}.atlus.cloud\">Explorer</a> | <a href=\"https://${_prefix}.chimney.software\">Chimney</a>" >> /var/www/html/dashboard.html

    cat >> /var/www/html/dashboard.html << 'DASHEOF2'
<h3>Services</h3>
<a href="/logs/0box.log">0box</a> | <a href="/logs/zauth.log">zauth</a> | <a href="/logs/zvault.log">zvault</a> | <a href="/logs/elasticsearch.log">Elastic</a> | <a href="/logs/kafka.log">Kafka</a> | <a href="/logs/render.log">Render</a> | <a href="/logs/crawler.log">Crawler</a>
<h3>Monitoring</h3>
<a href="/vc/html">VC</a> | <a href="/chaos/html">Chaos</a> | <a href="/funding/html">Funding</a> | <a href="/dkg/html">DKG</a> | <a href="/deploy/html">Deploy</a> | <a href="/smoke/html">Smoke Test</a>
<h3>Tests</h3>
<a href="/test/results">Results</a>
<h3>Databases</h3>
<a href="/pgadmin/">pgAdmin (blobbers)</a> | <a href="/pgadmin-sharder/">pgAdmin (sharder)</a> | <a href="/pgadmin-0box/">pgAdmin (0box)</a> | <a href="/pgadmin-zauth/">pgAdmin (zauth)</a> | <a href="/pgadmin-zvault/">pgAdmin (zvault)</a>
<h3>Info</h3>
<a href="/info">Deploy Info</a> | <a href="/containers">Containers</a> | <a href="/logs/">All Logs</a>
</body></html>
DASHEOF2

    print_status "Dashboard generated at /var/www/html/dashboard.html"
}

# Get repo path from config (defaults to repo name)
get_repo_branch_path() {
    local repo="$1"
    local in_repo=false
    local path=""
    while IFS= read -r line; do
        if echo "$line" | grep -qP "^\s+${repo}:"; then
            in_repo=true
            continue
        fi
        if $in_repo; then
            if echo "$line" | grep -qP '^\s+path:'; then
                path=$(echo "$line" | awk -F': ' '{print $2}' | sed 's/#.*//' | tr -d ' "'"'"'')
                break
            fi
            if echo "$line" | grep -qP '^\s+\w+:$'; then
                break
            fi
        fi
    done < "$CONFIG_FILE"
    echo "$path"
}

# Parse --branch overrides from command line
# Usage: --branch 0chain=fix/my-branch --branch blobber=staging
declare -A BRANCH_OVERRIDES
parse_branch_overrides() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --branch)
                if [[ "$2" == *"="* ]]; then
                    local repo="${2%%=*}"
                    local branch="${2#*=}"
                    BRANCH_OVERRIDES["$repo"]="$branch"
                    print_status "Branch override: ${repo} → ${branch}"
                fi
                shift 2
                ;;
            --domain)
                if [ -n "$2" ]; then
                    export NGINX_DOMAIN="$2"
                    # Derive APP_DOMAIN_PREFIX from domain: test.zus.network → test, test1.zus.network → test1
                    local _prefix="${2%%.*}"
                    export APP_DOMAIN_PREFIX="$_prefix"
                    print_status "Domain override: ${2} (prefix: ${_prefix})"
                fi
                shift 2
                ;;
            --server)
                if [ -n "$2" ]; then
                    export REMOTE_SERVER_IP="$2"
                fi
                shift 2
                ;;
            --pass)
                if [ -n "$2" ]; then
                    export REMOTE_SERVER_PASS="$2"
                fi
                shift 2
                ;;
            --user)
                if [ -n "$2" ]; then
                    export REMOTE_SERVER_USER="$2"
                fi
                shift 2
                ;;
            --chain-image)
                if [ -n "$2" ]; then
                    export CHAIN_IMAGE_TAG="$2"
                    print_status "Chain image tag: ${2} (will pull from Docker Hub instead of building)"
                fi
                shift 2
                ;;
            --secrets)
                if [ -n "$2" ]; then
                    export REMOTE_SECRETS_PATH="$2"
                fi
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"

    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    print_status "Docker: OK"

    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        print_error "Docker Compose v2 is not installed"
        exit 1
    fi
    print_status "Docker Compose: OK"

    # Ensure docker-compose (v1 syntax) shim exists for 0chain build scripts that use it
    if ! command -v docker-compose &> /dev/null; then
        cat > /usr/local/bin/docker-compose << 'DCSHIM'
#!/bin/sh
exec docker compose "$@"
DCSHIM
        chmod +x /usr/local/bin/docker-compose
        print_status "Installed docker-compose → docker compose shim"
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_status "Go: $(go version | awk '{print $3}')"

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    print_status "Docker daemon: Running"

    # Check CLI tools — warn only on fresh server or redeploy (checkout_repos will build them)
    # $1 == "redeploy" skips the hard error so a fresh/broken deploy can rebuild from scratch.
    local cli_missing=false
    if [ ! -f "$ZWALLET" ]; then
        if [ -d "${BASE_DIR}/zwalletcli" ] && [ "${1:-}" != "redeploy" ]; then
            print_error "zwallet not found at $ZWALLET (repo exists but binary missing — build failed?)"
            exit 1
        else
            print_warning "zwallet not found (will be built during checkout)"
            cli_missing=true
        fi
    fi
    if [ ! -f "$ZBOX" ]; then
        if [ -d "${BASE_DIR}/zboxcli" ] && [ "${1:-}" != "redeploy" ]; then
            print_error "zbox not found at $ZBOX (repo exists but binary missing — build failed?)"
            exit 1
        else
            print_warning "zbox not found (will be built during checkout)"
            cli_missing=true
        fi
    fi
    if [ "$cli_missing" = "false" ]; then
        print_status "CLI tools: OK"
    fi

    # Check jq
    if ! command -v jq &> /dev/null; then
        print_error "jq is not installed (required for wallet nonce reset)"
        exit 1
    fi
    print_status "jq: OK"
}

# Deploy to a remote server from the LOCAL machine.
# Handles: rsync, bootstrap, secrets copy, deploy_config.local.yaml, and running the deploy.
# Usage (from local): bash scripts/deploy_local.sh remote-deploy --server IP --pass PASS --domain DOMAIN [redeploy|all|...]
remote_deploy() {
    local deploy_cmd="${EXTRA_ARGS[0]:-redeploy}"
    local server_ip="${REMOTE_SERVER_IP:-}"
    local server_user="${REMOTE_SERVER_USER:-root}"
    local server_pass="${REMOTE_SERVER_PASS:-}"
    local domain="${NGINX_DOMAIN:-}"

    if [ -z "$server_ip" ]; then
        print_error "Server IP required. Use: --server <IP>"
        print_error "Example: bash scripts/deploy_local.sh remote-deploy --server <server-ip> --pass 'mypass' --domain test2.zus.network redeploy"
        exit 1
    fi

    # Build SSH and rsync command arrays (handles special chars in passwords safely)
    local -a SSH_CMD RSYNC_CMD
    if [ -n "$server_pass" ]; then
        SSH_CMD=(sshpass -p "$server_pass" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -o ServerAliveInterval=30 -o ServerAliveCountMax=120)
        RSYNC_CMD=(sshpass -p "$server_pass" rsync -e "ssh -o StrictHostKeyChecking=no")
    else
        SSH_CMD=(ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -o ServerAliveInterval=30 -o ServerAliveCountMax=120)
        RSYNC_CMD=(rsync)
    fi
    local REMOTE="${server_user}@${server_ip}"
    local REMOTE_BASE="/root/Code"
    local REMOTE_SYSTEM_TEST="${REMOTE_BASE}/system_test"

    print_header "Remote Deploy: ${server_ip} (command: ${deploy_cmd})"

    # --- Step 1: Create remote directory ---
    print_status "Creating remote directories..."
    "${SSH_CMD[@]}" "$REMOTE" "mkdir -p ${REMOTE_SYSTEM_TEST}"

    # --- Step 2: Rsync system_test code (exclude Mac-only binaries) ---
    print_status "Syncing system_test code to ${REMOTE}..."
    "${RSYNC_CMD[@]}" -avz --delete \
        --exclude 'zbox' --exclude 'zwallet' \
        --exclude 'mc' --exclude 'warp' --exclude 'rclone-zus' \
        --exclude '.git' --exclude '__pycache__' \
        "${SCRIPT_DIR}/../" "${REMOTE}:${REMOTE_SYSTEM_TEST}/"

    # --- Step 3: Copy secrets ---
    print_status "Copying secrets..."
    # Use --secrets path if provided, otherwise default to scripts/.secrets.env next to this script
    local secrets_src="${REMOTE_SECRETS_PATH:-${SCRIPT_DIR}/.secrets.env}"
    if [ -f "$secrets_src" ]; then
        "${RSYNC_CMD[@]}" -avz "$secrets_src" "${REMOTE}:${REMOTE_SYSTEM_TEST}/scripts/.secrets.env"
        print_status "  Copied .secrets.env from ${secrets_src}"
    else
        print_warning "  .secrets.env not found at ${secrets_src} — skipping (use --secrets /path/to/.secrets.env)"
    fi
    if [ -f "${SCRIPT_DIR}/0box_firebase_key.json" ]; then
        "${RSYNC_CMD[@]}" -avz "${SCRIPT_DIR}/0box_firebase_key.json" "${REMOTE}:${REMOTE_SYSTEM_TEST}/scripts/0box_firebase_key.json"
        print_status "  Copied 0box_firebase_key.json"
    fi

    # --- Step 4: Copy .zcn wallet files ---
    local local_zcn="${HOME}/.zcn"
    if [ -f "${local_zcn}/owner.json" ]; then
        "${SSH_CMD[@]}" "$REMOTE" "mkdir -p /root/.zcn"
        "${RSYNC_CMD[@]}" -avz "${local_zcn}/owner.json" "${REMOTE}:/root/.zcn/owner.json"
        print_status "  Copied ~/.zcn/owner.json"
    fi
    # Copy miner_sc_owner.json if present
    if [ -f "${local_zcn}/miner_sc_owner.json" ]; then
        "${RSYNC_CMD[@]}" -avz "${local_zcn}/miner_sc_owner.json" "${REMOTE}:/root/.zcn/miner_sc_owner.json"
        print_status "  Copied ~/.zcn/miner_sc_owner.json"
    fi

    # --- Step 5: Write deploy_config.local.yaml on remote (domain override) ---
    if [ -n "$domain" ]; then
        local prefix="${domain%%.*}"
        print_status "Setting domain ${domain} (prefix: ${prefix}) on remote..."
        "${SSH_CMD[@]}" "$REMOTE" "cat > ${REMOTE_SYSTEM_TEST}/scripts/deploy_config.local.yaml << 'LOCALEOF'
settings:
  domain: ${domain}
  domain_prefix: ${prefix}
LOCALEOF"
    fi

    # --- Step 6: Bootstrap if Docker is not installed ---
    print_status "Checking if server needs bootstrap..."
    if ! "${SSH_CMD[@]}" "$REMOTE" 'docker --version > /dev/null 2>&1'; then
        print_status "Running bootstrap (installs Docker, Go, tools, clones repos)..."
        local domain_flag=""
        [ -n "$domain" ] && domain_flag="--domain $domain"
        "${SSH_CMD[@]}" "$REMOTE" \
            "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin && cd ${REMOTE_SYSTEM_TEST} && bash scripts/deploy_local.sh bootstrap ${domain_flag}"
    else
        print_status "Docker found — server already bootstrapped"
        # Check ALL required repos for a complete .git directory (partial/failed clones count as missing)
        local required_repos="0chain blobber gosdk 0box zauth-server zvault zs3server crawler web-apps zboxcli zwalletcli"
        local needs_clone=false
        for _repo in $required_repos; do
            if ! "${SSH_CMD[@]}" "$REMOTE" "test -d ${REMOTE_BASE}/${_repo}/.git" 2>/dev/null; then
                print_status "  Repo missing or incomplete: ${_repo}"
                needs_clone=true
            fi
        done
        if $needs_clone; then
            print_status "Running clone-repos to fetch missing repos..."
            "${SSH_CMD[@]}" "$REMOTE" \
                "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin && cd ${REMOTE_SYSTEM_TEST} && bash scripts/deploy_local.sh clone-repos"
        else
            print_status "All required repos present"
        fi
    fi

    # --- Step 7: Run the deploy command on the remote server ---
    local domain_flag=""
    [ -n "$domain" ] && domain_flag="--domain $domain"
    print_header "Running '${deploy_cmd}' on ${server_ip}..."
    print_status "Output streaming live. Log also at ${server_ip}:/tmp/deploy_local.log"
    "${SSH_CMD[@]}" "$REMOTE" \
        "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin && cd ${REMOTE_SYSTEM_TEST} && bash scripts/deploy_local.sh ${domain_flag} ${deploy_cmd}"

    print_header "Remote Deploy Complete: ${server_ip}"
}

# Bootstrap a fresh Ubuntu server: install Docker, Go, system tools, clone all repos
# Run once on a brand-new server before 'redeploy'
bootstrap_server() {
    print_header "Bootstrapping Fresh Server"

    if [ "$(uname)" = "Darwin" ]; then
        print_error "bootstrap_server is for Linux servers only"
        exit 1
    fi

    # --- Docker ---
    if ! command -v docker &> /dev/null; then
        print_status "Installing Docker..."
        apt-get update -qq
        apt-get install -y ca-certificates curl gnupg lsb-release
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
            | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update -qq
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
        systemctl enable docker
        systemctl start docker
        print_status "Docker $(docker --version) installed"
    else
        print_status "Docker already installed: $(docker --version)"
    fi

    # --- Go ---
    local GO_VERSION="1.22.9"
    if ! command -v go &> /dev/null || ! go version | grep -q "go${GO_VERSION}"; then
        print_status "Installing Go ${GO_VERSION}..."
        local GO_ARCH="amd64"
        curl -fsSL "https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz
        rm -rf /usr/local/go
        tar -C /usr/local -xzf /tmp/go.tar.gz
        rm /tmp/go.tar.gz
        # Ensure Go is in PATH for this session
        export PATH="$PATH:/usr/local/go/bin"
        print_status "Go $(go version) installed"
    else
        print_status "Go already installed: $(go version)"
    fi

    # Persist Go in PATH system-wide
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin:/root/go/bin' > /etc/profile.d/go.sh
    fi

    # --- System tools ---
    print_status "Installing system tools..."
    apt-get update -qq
    apt-get install -y git jq python3 python3-pip nginx curl wget build-essential gcc g++ make \
        apt-transport-https software-properties-common pigz sshpass xvfb 2>/dev/null || true

    # Node.js + npm + pm2
    if ! command -v node &> /dev/null; then
        print_status "Installing Node.js..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi
    if ! command -v pm2 &> /dev/null; then
        print_status "Installing pm2..."
        npm install -g pm2
        pm2 install pm2-logrotate
    fi

    # golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        print_status "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
            sh -s -- -b /usr/local/bin v1.57.2 2>/dev/null || true
    fi

    print_status "System tools installed"

    # --- Clone repos ---
    clone_repos

    print_header "Bootstrap Complete!"
    echo ""
    print_status "Checklist before running redeploy:"
    print_status "  1. Copy ~/.zcn/local.json (SC owner wallet) to this server"
    print_status "  2. Copy scripts/.secrets.env (Firebase key, Kafka password, etc.)"
    print_status "  3. Copy scripts/0box_firebase_key.json (Firebase service account)"
    print_status "  4. Update scripts/deploy_config.yaml if needed (branches, domain)"
    echo ""
    print_status "Then run:"
    local _domain="${NGINX_DOMAIN:-test.zus.network}"
    print_status "  bash scripts/deploy_local.sh --domain ${_domain} redeploy"
}

# Clone all required repositories into BASE_DIR
clone_repos() {
    print_header "Cloning Repositories"

    mkdir -p "$BASE_DIR"

    local -A REPO_URLS=(
        ["0chain"]="https://github.com/0chain/0chain.git"
        ["0dns"]="https://github.com/0chain/0dns.git"
        ["blobber"]="https://github.com/0chain/blobber.git"
        ["gosdk"]="https://github.com/0chain/gosdk.git"
        ["eblobber"]="https://github.com/0chain/blobber.git"
        ["0box"]="https://github.com/0chain/0box.git"
        ["zauth-server"]="https://github.com/0chain/zauth-server.git"
        ["zvault"]="https://github.com/0chain/zvault.git"
        ["zs3server"]="https://github.com/0chain/zs3server.git"
        ["crawler"]="https://github.com/0chain/crawler.git"
        ["web-apps"]="https://github.com/0chain/web-apps.git"
        ["zboxcli"]="https://github.com/0chain/zboxcli.git"
        ["zwalletcli"]="https://github.com/0chain/zwalletcli.git"
        ["rclone_zus"]="https://github.com/0chain/rclone.git"
    )

    for repo in "${!REPO_URLS[@]}"; do
        local dest="${BASE_DIR}/${repo}"
        local url="${REPO_URLS[$repo]}"

        if [ -d "$dest/.git" ]; then
            print_status "${repo}: already cloned, fetching latest..."
            git -C "$dest" fetch origin 2>/dev/null || true
        else
            # Directory exists but has no .git (failed/partial clone) — remove and reclone
            if [ -d "$dest" ]; then
                print_status "Removing incomplete ${repo} directory (no .git found)..."
                rm -rf "$dest"
            fi
            print_status "Cloning ${repo}..."
            # Embed GITHUB_TOKEN for private repos; mask token in logs
            local clone_url="$url"
            if [ -n "${GITHUB_TOKEN:-}" ]; then
                clone_url="${url/https:\/\/github.com/https:\/\/${GITHUB_TOKEN}@github.com}"
            fi
            local clone_out clone_exit
            clone_out=$(GIT_TERMINAL_PROMPT=0 git clone --depth=1 "$clone_url" "$dest" 2>&1)
            clone_exit=$?
            if [ $clone_exit -ne 0 ]; then
                print_warning "Failed to clone ${repo}: $(echo "$clone_out" | grep -i 'fatal\|error' | head -1)"
                if [ -z "${GITHUB_TOKEN:-}" ]; then
                    print_warning "  → Set GITHUB_TOKEN in scripts/.secrets.env for private repos"
                fi
            else
                echo "$clone_out" | tail -2
            fi
        fi
    done

    # Build CLI tools (zbox, zwallet) — inject local gosdk (LFB-aware sharder selection)
    checkout_gosdk_for_dependent "zboxcli"
    inject_local_gosdk "${BASE_DIR}/zboxcli"
    print_status "Building zboxcli..."
    cd "${BASE_DIR}/zboxcli" && make install 2>/dev/null || \
        (make build 2>/dev/null && cp zbox "${BASE_DIR}/zboxcli/") || \
        print_warning "zboxcli build failed"
    cleanup_injected_gosdk "${BASE_DIR}/zboxcli"

    checkout_gosdk_for_dependent "zwalletcli"
    inject_local_gosdk "${BASE_DIR}/zwalletcli"
    print_status "Building zwalletcli..."
    cd "${BASE_DIR}/zwalletcli" && make install 2>/dev/null || \
        (make build 2>/dev/null && cp zwallet "${BASE_DIR}/zwalletcli/") || \
        print_warning "zwalletcli build failed"
    cleanup_injected_gosdk "${BASE_DIR}/zwalletcli"

    cd "$SCRIPT_DIR"
    print_status "Repository setup complete"
}

# Setup loopback aliases (macOS only - Linux uses Docker bridge networking)
setup_loopback() {
    print_header "Setting Up Loopback Aliases"

    # On Linux, Docker's testnet0 bridge network handles routing to 198.18.x.x
    # Loopback aliases are only needed on macOS where Docker runs in a VM
    if [ "$(uname)" != "Darwin" ]; then
        print_status "Linux detected - skipping loopback aliases (Docker bridge handles routing)"
        return 0
    fi

    if [ -f "${SCRIPT_DIR}/setup_loopback.sh" ]; then
        print_status "Running setup_loopback.sh (requires sudo)..."
        sudo bash "${SCRIPT_DIR}/setup_loopback.sh"
    else
        print_warning "setup_loopback.sh not found, configuring manually..."

        # Miners
        sudo ifconfig lo0 alias 198.18.0.71 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.72 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.73 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.74 2>/dev/null || true

        # Sharders
        sudo ifconfig lo0 alias 198.18.0.81 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.82 2>/dev/null || true

        # 0dns
        sudo ifconfig lo0 alias 198.18.0.100 2>/dev/null || true

        # Blobbers 1-6
        sudo ifconfig lo0 alias 198.18.0.91 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.92 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.93 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.94 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.95 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.96 2>/dev/null || true

        # Blobbers 7-12
        sudo ifconfig lo0 alias 198.18.0.97 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.98 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.99 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.110 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.111 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.112 2>/dev/null || true

        # Validators 1-6
        sudo ifconfig lo0 alias 198.18.0.61 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.62 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.63 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.64 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.65 2>/dev/null || true
        sudo ifconfig lo0 alias 198.18.0.66 2>/dev/null || true

        print_status "Loopback aliases configured"
    fi
}

# Create ZCN config (local.yaml with localhost addresses)
setup_zcn_config() {
    print_header "Setting Up ZCN Configuration"

    mkdir -p "${ZCN_CONFIG_DIR}"

    # Create local.yaml with localhost addresses
    # IMPORTANT: localhost addresses are required for SC owner operations to work correctly.
    # Using 198.18.0.x addresses causes authorization failures for sc-update-config, etc.
    cat > "${ZCN_CONFIG_DIR}/${ZCN_CONFIG_FILE}" << 'EOF'
---
block_worker: http://localhost:9091
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 10
confirmation_chain_length: 3
max_txn_query: 10
query_sleep_time: 5

miners:
  - http://localhost:7071
  - http://localhost:7072
  - http://localhost:7073
  - http://localhost:7074

sharders:
  - http://localhost:7171
  - http://localhost:7172
EOF

    print_status "ZCN config created at ${ZCN_CONFIG_DIR}/${ZCN_CONFIG_FILE}"

    # Always copy the SC owner wallet to owner.json (overwrite stale wallets from previous deploys).
    # Validate JSON content (not just file existence) — a 0-byte file from interrupted rsync passes -f.
    local sc_wallet_src="${BASE_DIR}/system_test/tests/cli_tests/config/wallets/sc_owner_wallet.json"
    if [ -f "$sc_wallet_src" ] && [ -s "$sc_wallet_src" ] && jq -e '.client_id' "$sc_wallet_src" >/dev/null 2>&1; then
        cp "$sc_wallet_src" "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
        local wallet_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)
        print_status "Copied SC owner wallet to ${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE} (client_id: ${wallet_id:0:16}...)"
    elif [ -f "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" ] && [ -s "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" ] && jq -e '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" >/dev/null 2>&1; then
        print_status "SC owner wallet source empty/invalid, but existing ${ZCN_WALLET_FILE} is valid — keeping it"
    else
        print_warning "SC owner wallet not found or invalid, creating new wallet..."
        $ZWALLET create-wallet --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
    fi
}

# Reset all wallet nonces to 0 (required after fresh chain start)
reset_wallet_nonces() {
    print_header "Resetting Wallet Nonces"

    # Reset blobber and validator wallet nonces (all blobbers 1-12)
    for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
        for prefix in b0bnode b0vnode; do
            local wallet_file="${BLOBBER_KEYS_DIR}/${prefix}${i}_keys.txt.json"
            if [ -f "$wallet_file" ]; then
                local current_nonce=$(jq '.nonce' "$wallet_file")
                if [ "$current_nonce" != "0" ]; then
                    jq '.nonce = 0' "$wallet_file" > "${wallet_file}.tmp" && mv "${wallet_file}.tmp" "$wallet_file"
                    print_status "Reset nonce for ${prefix}${i} (was $current_nonce)"
                fi
            fi
        done
    done

    # Reset owner wallet nonce
    local owner_wallet="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    if [ -f "$owner_wallet" ]; then
        local current_nonce=$(jq '.nonce' "$owner_wallet")
        if [ "$current_nonce" != "0" ]; then
            jq '.nonce = 0' "$owner_wallet" > "${owner_wallet}.tmp" && mv "${owner_wallet}.tmp" "$owner_wallet"
            print_status "Reset owner wallet nonce (was $current_nonce)"
        fi
    fi

    # Reset sc_owner_wallet nonce in test config
    local sc_owner="${BASE_DIR}/system_test/tests/cli_tests/config/wallets/sc_owner_wallet.json"
    if [ -f "$sc_owner" ]; then
        local current_nonce=$(jq '.nonce' "$sc_owner")
        if [ "$current_nonce" != "0" ]; then
            jq '.nonce = 0' "$sc_owner" > "${sc_owner}.tmp" && mv "${sc_owner}.tmp" "$sc_owner"
            print_status "Reset sc_owner_wallet nonce (was $current_nonce)"
        fi
    fi

    print_status "All wallet nonces reset to 0"
}

# Wait for chain to be ready
# Validate 0dns lists all expected miners and sharders
# Called after 0dns starts to verify the network configuration is complete.
# If nodes are missing, tests will fail with cryptic "not enough blobbers" errors.
validate_0dns() {
    local expected_miners="${EXPECTED_MINERS:-4}"
    local expected_sharders="${EXPECTED_SHARDERS:-2}"
    local max_retries=6

    for attempt in $(seq 1 $max_retries); do
        local dns_response
        dns_response=$(curl -s http://127.0.0.1:9091/dns/network 2>/dev/null || curl -s http://127.0.0.1:9091/network 2>/dev/null || echo "{}")

        local num_miners num_sharders
        num_miners=$(echo "$dns_response" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('miners',[])))" 2>/dev/null || echo "0")
        num_sharders=$(echo "$dns_response" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('sharders',[])))" 2>/dev/null || echo "0")

        if [ "${num_miners:-0}" -ge "$expected_miners" ] && [ "${num_sharders:-0}" -ge "$expected_sharders" ]; then
            print_status "0dns validation passed: $num_miners miners, $num_sharders sharders"
            return 0
        fi

        if [ "$attempt" -lt "$max_retries" ]; then
            print_status "0dns has $num_miners miners, $num_sharders sharders (attempt $attempt/$max_retries), waiting 5s..."
            sleep 5
        fi
    done

    # Final state after all retries
    print_warning "0dns is missing nodes after ${max_retries} retries: $num_miners/$expected_miners miners, $num_sharders/$expected_sharders sharders"
}

wait_for_chain() {
    print_header "Waiting for Chain to Start"

    local max_attempts=60
    local attempt=0

    # Phase 1: Wait for 0dns to respond
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://127.0.0.1:9091/network > /dev/null 2>&1; then
            print_status "Chain API is responding!"
            validate_0dns
            break
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    if [ $attempt -ge $max_attempts ]; then
        print_error "Chain API did not respond within timeout"
        return 1
    fi

    # Phase 2: Wait for blocks to finalize (DKG must complete first)
    print_status "Waiting for block finalization (DKG completion)..."
    local finalize_attempts=0
    local finalize_max=90  # up to 3 minutes

    # Get sharder list from 0dns
    local sharders
    sharders=$(curl -s http://127.0.0.1:9091/network 2>/dev/null \
        | python3 -c "import sys,json; d=json.load(sys.stdin); [print(s) for s in d.get('sharders',[])]" 2>/dev/null)

    while [ $finalize_attempts -lt $finalize_max ]; do
        for sharder_url in $sharders; do
            local lfr
            lfr=$(curl -s "${sharder_url}/v1/chain/get/stats" -m 5 2>/dev/null \
                | python3 -c "import sys,json; print(json.load(sys.stdin).get('latest_finalized_round',0))" 2>/dev/null || echo "0")
            if [ "${lfr:-0}" -gt 0 ]; then
                print_status "Chain is finalizing blocks! (LFR: $lfr)"
                return 0
            fi
        done

        finalize_attempts=$((finalize_attempts + 1))
        echo -n "."
        sleep 2
    done

    print_error "Chain did not finalize blocks within timeout"
    return 1
}

# Initialize chain configuration
init_chain_config() {
    print_header "Initializing Chain Configuration"

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
    # SC owner wallet for sc-update-config (must match the SC owner_id in genesis)
    local WO="--wallet $ZCN_SC_OWNER_WALLET --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
    # MinerSC may have a different owner on chains with historical state (miner_sc_owner.json).
    # Verify client_id matches on-chain MinerSC owner_id — stale files cause auth failures on fresh chains.
    local WOM="$WO"
    local _miner_sc="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
    local _sharder_wom="${SHARDER_BASE_URL:-http://198.18.0.81:7171}"
    local _onchain_miner_owner
    _onchain_miner_owner=$(curl -sf "${_sharder_wom}/v1/screst/${_miner_sc}/configs" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); f=d.get('fields',d); print(f.get('owner_id',''))" 2>/dev/null || echo "")
    if [ -f "${ZCN_CONFIG_DIR}/miner_sc_owner.json" ]; then
        local _wom_id
        _wom_id=$(python3 -c "import json; print(json.load(open('${ZCN_CONFIG_DIR}/miner_sc_owner.json')).get('client_id',''))" 2>/dev/null || echo "")
        if [ -n "$_onchain_miner_owner" ] && [ "$_wom_id" = "$_onchain_miner_owner" ]; then
            WOM="--wallet miner_sc_owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
            print_status "Using miner_sc_owner.json for mn-update-config (verified MinerSC owner match)"
        else
            print_status "miner_sc_owner.json (${_wom_id:0:16}...) != on-chain MinerSC owner (${_onchain_miner_owner:0:16}...) — using owner.json"
        fi
    fi

    # ========== Restore StorageSC owner if a test changed it ==========
    local storage_sc="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local sharder_url="${SHARDER_BASE_URL:-http://198.18.0.81:7171}"
    local expected_sc_owner
    expected_sc_owner=$(jq -r '.client_id' "${BASE_DIR}/system_test/tests/cli_tests/config/wallets/sc_owner_wallet.json" 2>/dev/null)
    local current_sc_owner
    current_sc_owner=$(curl -sf "${sharder_url}/v1/screst/${storage_sc}/storage-config" 2>/dev/null | jq -r '.fields.owner_id // .owner_id // empty')
    if [ -n "$expected_sc_owner" ] && [ -n "$current_sc_owner" ] && [ "$current_sc_owner" != "$expected_sc_owner" ]; then
        print_warning "StorageSC owner mismatch: on-chain=${current_sc_owner:0:16}... expected=${expected_sc_owner:0:16}... — attempting restore..."
        local restore_wallet_file=""
        # Check ~/.zcn wallets first (miner_sc_owner.json is often the culprit since wallets[500] = miner_sc_owner)
        for candidate in "${ZCN_CONFIG_DIR}/miner_sc_owner.json" "${ZCN_CONFIG_DIR}/miner_sc_owner_wallet.json"; do
            if [ -f "$candidate" ]; then
                local cid
                cid=$(jq -r '.client_id // empty' "$candidate" 2>/dev/null)
                if [ "$cid" = "$current_sc_owner" ]; then
                    restore_wallet_file="$candidate"
                    break
                fi
            fi
        done
        # Fall back: search test config dir (skip wallets.json array file)
        if [ -z "$restore_wallet_file" ]; then
            while IFS= read -r f; do
                if [ "$(basename "$f")" = "wallets.json" ]; then continue; fi
                local cid
                cid=$(jq -r '.client_id // empty' "$f" 2>/dev/null)
                if [ "$cid" = "$current_sc_owner" ]; then
                    restore_wallet_file="$f"
                    break
                fi
            done < <(find "${BASE_DIR}/system_test/tests/cli_tests/config/" -name '*.json' 2>/dev/null)
        fi
        if [ -n "$restore_wallet_file" ]; then
            print_status "Using wallet: $restore_wallet_file"
            jq '.nonce = 0' "$restore_wallet_file" > "${ZCN_CONFIG_DIR}/restore_sc_owner.json"
            local WO_RESTORE="--wallet restore_sc_owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
            if $ZWALLET sc-update-config --keys owner_id --values "$expected_sc_owner" $WO_RESTORE 2>&1; then
                print_status "StorageSC owner restored to ${expected_sc_owner:0:16}..."
                sleep 5
            else
                print_error "Failed to restore StorageSC owner — SC config updates may fail!"
            fi
        else
            print_error "No wallet file found with client_id=${current_sc_owner:0:16}... — SC config updates may fail!"
        fi
    fi

    # Enforce faucet limits on the running chain.
    # patch_sc_yaml() sets genesis values but CANNOT fix a chain that started with old sc.yaml.
    # updateSettings enforces the correct limits regardless of what genesis had.
    # NOTE: faucet owner_id is set to owner.json by patch_sc_yaml(), so $WO is correct.
    run_cmd "Enforcing faucet limits (pour=100, max_pour=10000, periodic=10000, global=10M)..." \
        "$ZWALLET faucet --methodName updateSettings --input '{\"pour_limit\":100,\"max_pour_amount\":10000,\"periodic_limit\":10000,\"global_limit\":10000000,\"individual_reset\":\"1h\"}' $WO"
    sleep 3

    # Get faucet tokens for the owner wallet
    run_cmd "Funding owner wallet (100 ZCN)..." \
        "$ZWALLET faucet --methodName pour --input '{}' --tokens 100 $W"
    sleep 3

    # Configure hardforks (includes Nyx) — CRITICAL, must succeed
    # Hardforks must be set BEFORE cost.vc_add (hermes hardfork enables vc_add)
    # add-hardfork requires SC owner wallet
    #
    # IMPORTANT: On a fresh chain, cost.add_hardfork is not set → defaults to MaxInt64.
    # The gosdk fee estimation overflows (MaxInt64 * coeff = 0) → fee=0, but minFee=10 ZCN.
    # Fix: set cost.add_hardfork=1001 first (fee = 1001 * 10000000 / 10^9 = 10.01 ZCN > 10 ZCN minFee).
    # This uses mn-update-config (cost.update_settings=136 → fee=1.36 ZCN, always works).
    run_cmd "Setting cost.add_hardfork=1001 (enables fee computation for add-hardfork)..." \
        "$ZWALLET mn-update-config --keys 'cost.add_hardfork' --values '1001' $WOM"
    sleep 3
    run_cmd "Adding hardforks at round 0 (apollo through Nyx)..." \
        "$ZWALLET add-hardfork --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,jason,odysseus,Nyx' --rounds '0,0,0,0,0,0,0,0,0,0,0,0' $WO"
    sleep 5  # Wait for hardfork to propagate

    # Configure miner VC settings (mn-update-config requires MinerSC owner wallet)
    run_cmd "Setting VC round durations (10,20,10,10,20)..." \
        "$ZWALLET mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' --values '10,20,10,10,20' $WOM"

    run_cmd "Setting k_percent=0.6, x_percent=0.6..." \
        "$ZWALLET mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' $WOM"

    run_cmd "Setting min_n=3, min_s=1..." \
        "$ZWALLET mn-update-config --keys 'min_n,min_s' --values '3,1' $WOM"

    run_cmd "Setting max_delegates=200..." \
        "$ZWALLET mn-update-config --keys 'max_delegates' --values '200' $WOM"

    run_cmd "Setting num_sharders_rewarded=2..." \
        "$ZWALLET mn-update-config --keys 'num_sharders_rewarded' --values '2' $WOM"

    # share_ratio must be < 1 (fraction of block reward for miners).
    # If set > 1 (e.g. via manual mn-update-config), payfees transaction fails on every block:
    # "error splitting rewards by ratio: uint64 minus overflow"
    run_cmd "Setting share_ratio=0.16 (must be < 1, prevents payfees overflow)..." \
        "$ZWALLET mn-update-config --keys 'share_ratio' --values '0.16' $WOM"

    # health_check_period=60m gives a 30-min buffer vs the Atlus isGoodHealth threshold of 90m.
    # Default 90m == threshold → any missed cycle makes miner appear unhealthy in Atlus.
    run_cmd "Setting health_check_period=60m (gives buffer vs 90m Atlus threshold)..." \
        "$ZWALLET mn-update-config --keys 'health_check_period' --values '60m' $WOM"

    # Configure global settings (global-update-config requires SC owner wallet)
    run_cmd "Enabling view change..." \
        "$ZWALLET global-update-config --keys 'server_chain.view_change' --values 'true' $WO"

    run_cmd "Setting block proposal max_wait_time=500ms..." \
        "$ZWALLET global-update-config --keys 'server_chain.block.proposal.max_wait_time' --values '500ms' $WO"

    # CRITICAL: Enable EventDB debug mode for tokenomics tests (alloc-challenge-rewards endpoint).
    # The on-chain global settings default to debug=false, overriding the local 0chain.yaml debug=true.
    # Without this, reward_providers and reward_delegates tables are never populated.
    run_cmd "Enabling EventDB debug mode for challenge reward tracking..." \
        "$ZWALLET global-update-config --keys 'server_chain.dbs.settings.debug' --values 'true' $WO"

    # cost.vc_add must be set AFTER hardforks (hermes enables vc_add function)
    # Note: stored internally in Cost map but not shown by REST /configs API
    run_cmd "Setting cost.vc_add=361 (requires hermes hardfork)..." \
        "$ZWALLET mn-update-config --keys 'cost.vc_add' --values '361' $WOM"

    # CRITICAL: Reduce transaction fees.
    # Actual formula: fee_SAS = cost_unit × 10^12 / cost_fee_coeff — HIGHER coeff = CHEAPER fees.
    # At cost_fee_coeff=100000: challenge_response=0.728 ZCN, new_allocation=1 ZCN (max_fee cap).
    # At cost_fee_coeff=10000000: challenge_response=0.00728 ZCN, new_allocation=0.019 ZCN.
    # Enterprise tests lock 5 ZCN, expect balance within 15% after cancel → need fee <0.75 ZCN.
    # NOTE: Miners read cost_fee_coeff from their LOCAL 0chain.yaml at startup, NOT from
    # globalSettings. global-update-config updates the SC state but the miner pre-flight
    # fee check (chain/handler.go) uses the in-memory config. MUST patch the yaml file too.
    local MINER_CHAIN_CONFIG="${BASE_DIR}/0chain/docker.local/config/0chain.yaml"
    if [ -f "$MINER_CHAIN_CONFIG" ]; then
        sed -i "s/cost_fee_coeff: [0-9]*/cost_fee_coeff: 10000000/" "$MINER_CHAIN_CONFIG"
        # Also set max_fee low in yaml for future builds (in-memory updated via global-update-config)
        sed -i "s/max_fee: [0-9.]*/max_fee: 0.00001/" "$MINER_CHAIN_CONFIG"
        print_status "Patched 0chain.yaml: cost_fee_coeff=10000000, max_fee=0.00001"
    fi
    run_cmd "Setting cost_fee_coeff=10000000 (fees 10000x cheaper, fee=cost×10^12/coeff)..." \
        "$ZWALLET global-update-config --keys 'server_chain.transaction.cost_fee_coeff' --values '10000000' $WO"
    # CRITICAL: Set max_fee low so sharder health check transactions can go through.
    # The sharder binary computes fee=cost/10000000 ZCN (coeff=10000000) but miners validate
    # using coeff=1000, giving minFee=cost/1000 ZCN (10000x higher). Result: sharder health
    # check transactions are rejected as "insufficient transaction fee".
    # Fix: cap max_fee at 0.00001 ZCN (100,000 SAS) so minFee = min(1,450,000 SAS, 100,000) = 100,000.
    # Sharder submits 145,000 SAS > 100,000 → accepted. All transactions still work.
    run_cmd "Setting max_fee=0.00001 ZCN (100,000 SAS) to allow sharder health check transactions..." \
        "$ZWALLET global-update-config --keys 'server_chain.transaction.max_fee' --values '0.00001' $WO"

    # Storage SC config for free allocations (requires SC owner wallet)
    # read_price is always 0, so both min and max of the range are 0
    run_cmd "Setting free_allocation read_price_range=[0,0] (read_price always 0)..." \
        "$ZWALLET sc-update-config --keys 'free_allocation_settings.read_price_range.min,free_allocation_settings.read_price_range.max' --values '0,0' $WO"

    # CRITICAL: Storage SC settings required for tests (requires SC owner wallet)
    run_cmd "Setting min_alloc_size=1024 (tests use 2048 bytes)..." \
        "$ZWALLET sc-update-config --keys 'min_alloc_size' --values '1024' $WO"

    run_cmd "Setting min_write_price=0.001 (chain-enforced minimum; 0chain code floor is 0.001 ZCN)..." \
        "$ZWALLET sc-update-config --keys 'min_write_price' --values '0.001' $WO"

    run_cmd "Resetting free_allocation write_price_range.min=0..." \
        "$ZWALLET sc-update-config --keys 'free_allocation_settings.write_price_range.min' --values '0' $WO"

    run_cmd "Enabling challenge generation..." \
        "$ZWALLET sc-update-config --keys 'challenge_enabled' --values 'true' $WO"

    # Generate challenges every round (gap=1) and challenge all blobbers (select=20).
    # Default gap=3 / select=5 is too slow for test allocations with small data (2MB).
    # New allocations compete by weight with existing large allocations; increasing these
    # settings ensures challenges are generated within seconds of a write marker commit.
    run_cmd "Setting challenge_generation_gap=1, max_blobber_select_for_challenge=20..." \
        "$ZWALLET sc-update-config --keys 'challenge_generation_gap,max_blobber_select_for_challenge' --values '1,20' $WO"

    # Set time_unit for allocation pricing (controls per-period cost calculation)
    # Default may be too short for local testing. 1d = realistic pricing intervals.
    local TIME_UNIT="${TIME_UNIT:-720h}"
    run_cmd "Setting time_unit=${TIME_UNIT} (allocation pricing interval)..." \
        "$ZWALLET sc-update-config --keys 'time_unit' --values '$TIME_UNIT' $WO"

    # Set health_check_period for blobber/validator health monitoring.
    # Must be >= validator healthcheck.frequency (50m in 0chain_validator.yaml).
    # 60m gives headroom; 30m would cause all validators to fail challenge eligibility.
    local HEALTH_CHECK_PERIOD="${HEALTH_CHECK_PERIOD:-60m}"
    run_cmd "Setting health_check_period=${HEALTH_CHECK_PERIOD}..." \
        "$ZWALLET sc-update-config --keys 'health_check_period' --values '$HEALTH_CHECK_PERIOD' $WO"

    # Fix SC function costs to match gosdk fee estimation (gosdk estimates cost=N-1, chain uses N).
    # Without these fixes blobbers get "insufficient transaction fee" errors.
    run_cmd "Setting cost.blobber_health_check=97 (gosdk estimates 97, chain default 98)..." \
        "$ZWALLET sc-update-config --keys 'cost.blobber_health_check' --values '97' $WO"
    run_cmd "Setting cost.commit_connection=743 (gosdk estimates 743, chain default 744)..." \
        "$ZWALLET sc-update-config --keys 'cost.commit_connection' --values '743' $WO"

    print_status "Chain configuration complete!"

    # Update miner and sharder num_delegates to 10 and set delegate_wallet.
    # Tests that verify max_delegates behavior need a low value (10) so they can
    # exhaust the pool limit within a test. Stale pools from previous test runs
    # are cleaned by reset_test_state() before each test cycle.
    # - delegate_wallet: set to SC owner wallet for consistent control
    local sc_owner_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)
    print_status "Updating miner/sharder num_delegates=10, delegate_wallet=${sc_owner_id:0:16}..."

    # Note: On a fresh deploy, fix_miner_sharder_config() already sets delegate_wallet
    # and num_delegates in 0chain.yaml BEFORE miners start. This on-chain update is a
    # backup to ensure values are correct. If delegate_wallet was already set in config,
    # the mn-update-settings should succeed. If it fails with "access denied", it means
    # miners registered with an empty delegate_wallet (config wasn't fixed before start).
    local miner_ids=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)
    local miner_update_failed=0
    for miner_id in $miner_ids; do
        print_status "  Setting miner ${miner_id:0:16}... num_delegates=200, delegate_wallet"
        local output
        output=$($ZWALLET mn-update-settings \
            --id "$miner_id" \
            --num_delegates 200 \
            --delegate_wallet "$sc_owner_id" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1) || {
            if echo "$output" | grep -q "access denied"; then
                print_warning "  Miner ${miner_id:0:16} mn-update-settings: access denied (delegate_wallet was empty at registration)"
                miner_update_failed=1
            else
                print_warning "  Miner ${miner_id:0:16} mn-update-settings failed: $output"
            fi
        }
        sleep 2
    done

    local sharder_ids=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)
    for sharder_id in $sharder_ids; do
        print_status "  Setting sharder ${sharder_id:0:16}... num_delegates=200, delegate_wallet"
        local output
        output=$($ZWALLET mn-update-settings \
            --id "$sharder_id" \
            --sharder \
            --num_delegates 200 \
            --delegate_wallet "$sc_owner_id" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1) || {
            if echo "$output" | grep -q "access denied"; then
                print_warning "  Sharder ${sharder_id:0:16} mn-update-settings: access denied (delegate_wallet was empty at registration)"
                miner_update_failed=1
            else
                print_warning "  Sharder ${sharder_id:0:16} mn-update-settings failed: $output"
            fi
        }
        sleep 2
    done

    if [ "$miner_update_failed" = "1" ]; then
        print_warning "Some miner/sharder settings could not be updated (empty delegate_wallet at registration)."
        print_warning "Fix: Ensure fix_miner_sharder_config() runs BEFORE start_chain() on fresh deploy."
    fi
    print_status "Miner/sharder settings update complete!"
}

# Fix validator config (block_worker, delegate_wallet, service_charge)
# Common issue: validator config ships with wrong defaults that prevent registration.
fix_validator_config() {
    print_header "Fixing Validator Configuration"

    local VALIDATOR_CONFIG="${BASE_DIR}/blobber/config/0chain_validator.yaml"
    if [ ! -f "$VALIDATOR_CONFIG" ]; then
        print_warning "Validator config not found at $VALIDATOR_CONFIG"
        return 0
    fi

    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local delegate_wallet=$(jq -r '.client_id' "$SC_OWNER_WALLET" 2>/dev/null)

    if [ -z "$delegate_wallet" ]; then
        print_warning "Could not read delegate wallet ID from $SC_OWNER_WALLET"
        return 0
    fi

    # ALWAYS set block_worker to local 0dns. The validator config often ships with
    # public devnet URLs (dev.0chain.net, etc.) which causes validators to discover
    # miners/sharders from the wrong network, preventing registration on the local chain.
    print_status "Setting validator block_worker to local 0dns..."
    sed -i.bak "s|block_worker:.*|block_worker: http://198.18.0.100:9091|" "$VALIDATOR_CONFIG"

    # Fix delegate_wallet: must match the SC owner / blobber delegate wallet
    print_status "Setting validator delegate_wallet to $delegate_wallet..."
    sed -i.bak "s|delegate_wallet:.*|delegate_wallet: '$delegate_wallet'|" "$VALIDATOR_CONFIG"

    # Fix service_charge: must not exceed SC max (0.5). Default is often 1 (100%).
    local current_charge=$(grep "service_charge:" "$VALIDATOR_CONFIG" | head -1 | awk '{print $2}' | tr -d "'\"")
    if [ -n "$current_charge" ]; then
        local too_high=$(python3 -c "print(1 if float('$current_charge') > 0.5 else 0)" 2>/dev/null || echo "0")
        if [ "$too_high" = "1" ]; then
            print_status "Fixing validator service_charge from $current_charge to 0.3..."
            sed -i.bak "s|service_charge:.*|service_charge: 0.3|" "$VALIDATOR_CONFIG"
        fi
    fi

    # Fix num_delegates: default of 1 blocks staking tests
    local current_nd=$(grep "num_delegates:" "$VALIDATOR_CONFIG" | head -1 | awk '{print $2}' | tr -d "'\"")
    if [ -n "$current_nd" ] && [ "$current_nd" -lt 100 ] 2>/dev/null; then
        print_status "Fixing validator num_delegates from $current_nd to 100..."
        sed -i.bak "s|num_delegates:.*|num_delegates: 100|" "$VALIDATOR_CONFIG"
    fi

    # Clean up backup files
    rm -f "${VALIDATOR_CONFIG}.bak"

    # Verify the config was updated correctly
    local actual_bw=$(grep 'block_worker:' "$VALIDATOR_CONFIG" | head -1 | awk '{print $2}')
    if echo "$actual_bw" | grep -q "198.18.0.100"; then
        print_status "Validator config fixed! (block_worker=local, delegate_wallet=${delegate_wallet:0:16}...)"
    else
        print_error "Validator config may not have been updated correctly: block_worker=$actual_bw"
    fi
}

# Fix miner/sharder 0chain.yaml config (delegate_wallet, number_of_delegates)
# MUST run BEFORE first miner/sharder start. Once miners register with empty
# delegate_wallet, it cannot be updated on-chain (mn-update-settings returns
# "access denied" because no wallet matches an empty delegate_wallet).
fix_miner_sharder_config() {
    print_header "Fixing Miner/Sharder Configuration (delegate_wallet + num_delegates)"

    local CHAIN_YAML="${BASE_DIR}/0chain/docker.local/config/0chain.yaml"
    if [ ! -f "$CHAIN_YAML" ]; then
        print_warning "0chain.yaml not found at $CHAIN_YAML"
        return 0
    fi

    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local delegate_wallet=$(jq -r '.client_id' "$SC_OWNER_WALLET" 2>/dev/null)

    if [ -z "$delegate_wallet" ]; then
        print_warning "Could not read delegate wallet ID from $SC_OWNER_WALLET"
        return 0
    fi

    # Set delegate_wallet to SC owner wallet (chain owner)
    print_status "Setting miner/sharder delegate_wallet to ${delegate_wallet:0:16}..."
    sed -i.bak "s|^delegate_wallet:.*|delegate_wallet: \"$delegate_wallet\"|" "$CHAIN_YAML"

    # Set number_of_delegates to 200 (enough for test wallets + infrastructure stakes without unstaking)
    print_status "Setting miner/sharder number_of_delegates to 200..."
    sed -i.bak 's/^number_of_delegates:.*/number_of_delegates: 200/' "$CHAIN_YAML"

    rm -f "${CHAIN_YAML}.bak"

    # Verify
    local actual_dw=$(grep '^delegate_wallet:' "$CHAIN_YAML" | head -1 | sed 's/delegate_wallet: *"\{0,1\}//' | sed 's/"\{0,1\}$//')
    local actual_nd=$(grep '^number_of_delegates:' "$CHAIN_YAML" | head -1 | awk '{print $2}')
    if [ "$actual_dw" = "$delegate_wallet" ]; then
        print_status "Miner/sharder config fixed! delegate_wallet=${actual_dw:0:16}..., num_delegates=$actual_nd"
    else
        print_error "CRITICAL: delegate_wallet in config ($actual_dw) != expected ($delegate_wallet)"
        return 1
    fi
}

# Create Docker network and start chain (miners, sharders, 0dns)
patch_sc_yaml() {
    # Patch sc.yaml BEFORE chain start. These settings are baked into genesis and
    # CANNOT be changed on-chain later (unlike storage SC settings).
    local SC_YAML="${BASE_DIR}/0chain/docker.local/config/sc.yaml"
    if [ ! -f "$SC_YAML" ]; then
        print_warning "sc.yaml not found at $SC_YAML, skipping genesis patching"
        return 0
    fi

    # Set time_unit to 720h in genesis. If omitted, default is often much shorter (e.g. 10m)
    # which causes allocations to expire extremely fast and fills blobbers from test runs.
    sed -i.bak 's/^\(\s*\)time_unit:.*/\1time_unit: "720h"/' "$SC_YAML"
    rm -f "${SC_YAML}.bak"
    print_status "sc.yaml time_unit → 720h"

    print_status "Patching sc.yaml faucet limits for test suites..."
    # Increase faucet limits to prevent exhaustion during full test suite runs.
    # Default periodic_limit=1000 ZCN exhausted after ~100 wallet creates.
    # Default global_limit=100000 ZCN exhausted after running full suites a few times.
    # NOTE: ensure_chain_config() also calls 'faucet updateSettings' so these limits are
    # enforced even on chains that started before this patch was applied.
    sed -i.bak \
        -e 's/^\(\s*\)pour_limit:.*/\1pour_limit: 100/' \
        -e 's/^\(\s*\)max_pour_amount:.*/\1max_pour_amount: 10000/' \
        -e 's/^\(\s*\)periodic_limit:.*/\1periodic_limit: 10000/' \
        -e 's/^\(\s*\)global_limit:.*/\1global_limit: 10000000/' \
        -e 's/^\(\s*\)individual_reset:.*/\1individual_reset: 1h/' \
        "$SC_YAML"
    rm -f "${SC_YAML}.bak"
    print_status "sc.yaml faucet config: pour_limit=100, max_pour=10000, periodic=10000, global=10M, reset=1h"

    # Note: min_write_price=0.001 is the chain-enforced floor in the 0chain code.
    # Attempts to set it lower (e.g. 0.0001) via sc-update-config are silently ignored.
    # The sc.yaml default of 0.001 is already correct; no patch needed.

    # Set SC owner_id to owner.json so configure_chain() can use owner.json for SC ops.
    # Without this, the genesis owner_id defaults to local.json's client_id,
    # causing sc-update-config / mn-update-config to fail with "unauthorized access".
    local owner_id
    owner_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_SC_OWNER_WALLET}" 2>/dev/null)
    if [ -n "$owner_id" ] && [ "$owner_id" != "null" ]; then
        sed -i "s/^\(\s*\)owner_id:.*/\1owner_id: ${owner_id}/" "$SC_YAML"
        print_status "sc.yaml owner_id → ${owner_id:0:16}... (matches ${ZCN_SC_OWNER_WALLET})"
    else
        print_warning "Could not read owner_id from ${ZCN_SC_OWNER_WALLET}, sc.yaml owner_id unchanged"
    fi
}

# Kill all stale deploy/test processes from previous sessions before a fresh redeploy.
# Uses a blocking flock so if two deploys start simultaneously, the first one acquires
# the lock and kills the other (including while it's waiting for the lock). The second
# process is killed by SIGKILL and never gets to run. Only one deploy proceeds.
kill_stale_processes() {
    local my_pid=$$
    print_status "Stopping previous deploy/test processes (current PID: ${my_pid})..."

    {
        flock -w 60 201  # Block up to 60s; first caller wins, others wait (and get killed)

        local stale
        stale=$(pgrep -f 'scripts/deploy_local\.sh' 2>/dev/null | grep -v "^${my_pid}$" | tr '\n' ' ')
        if [ -n "$stale" ]; then
            print_status "Killing stale deploy PIDs: ${stale}"
            # shellcheck disable=SC2086
            kill -9 $stale 2>/dev/null || true
        fi

        # Kill run_tests.sh workers — their tee subprocesses die naturally when parent is killed.
        pkill -9 -f 'scripts/run_tests\.sh' 2>/dev/null || true
        # NOTE: do NOT pkill "tee -a /tmp/deploy_local.log" — that would kill OUR OWN tee
        # subprocess (set up at script start via `exec > >(tee ...)`) and break stdout.

        sleep 1
    } 201>/tmp/deploy_kill.lock

    print_status "Stale processes stopped. Proceeding with clean redeploy."
}

clean_chain_data() {
    # Clean chain data directories (equivalent of 0chain's clean.sh + init.setup.sh).
    # This removes stale RocksDB LFB, sharder postgres data, redis, etc.
    # WITHOUT this, miners load old LFB from rocksdb/config/ and deadlock against
    # freshly-started sharders. This was the root cause of the chain stuck issue.
    print_header "Cleaning Chain Data"

    local DOCKER_LOCAL="${BASE_DIR}/0chain/docker.local"

    # Stop chain containers AND enterprise blobbers first.
    # Enterprise blobbers must be stopped to prevent them from re-registering
    # with the old delegate_wallet on the fresh chain (delegate_wallet is immutable
    # after first registration).
    print_status "Stopping chain containers and enterprise blobbers..."
    for c in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2 eblobber-1 eblobber-2 eblobber-3; do
        docker stop "$c" 2>/dev/null || true
    done
    sleep 2

    # Kill background scripts that depend on the chain
    for script in vc.sh chaos.sh chaos_light.sh monitor.sh; do
        pkill -f "$script" 2>/dev/null || true
    done

    # Clean miner data (matches 0chain clean.sh: rocksdb config*, mb*, state*, dkg* + redis)
    for i in 1 2 3 4 5 6 7 8; do
        local mdir="$DOCKER_LOCAL/miner${i}"
        if [ -d "$mdir" ]; then
            print_status "Cleaning miner-$i data..."
            rm -rf "$mdir"/log/*
            rm -rf "$mdir"/data/redis/state/*
            rm -rf "$mdir"/data/redis/transactions/*
            rm -rf "$mdir"/data/rocksdb/config*
            rm -rf "$mdir"/data/rocksdb/mb*
            rm -rf "$mdir"/data/rocksdb/state*
            rm -rf "$mdir"/data/rocksdb/dkg*
        fi
    done

    # Clean sharder data (rocksdb, postgresql, blocks)
    for i in 1 2 3 4; do
        local sdir="$DOCKER_LOCAL/sharder${i}"
        if [ -d "$sdir" ]; then
            print_status "Cleaning sharder-$i data..."
            rm -rf "$sdir"/log/*
            rm -rf "$sdir"/data/rocksdb/*
            rm -rf "$sdir"/data/postgresql/*
            rm -rf "$sdir"/data/postgresql2/*
            rm -rf "$sdir"/data/blocks/*
        fi
    done

    # Clean kafka data
    rm -rf "$DOCKER_LOCAL/kafka/config/*" 2>/dev/null || true
    rm -rf "$DOCKER_LOCAL/kafka/data/*" 2>/dev/null || true

    # Re-initialize directories (equivalent of init.setup.sh)
    print_status "Re-initializing data directories..."
    for i in 1 2 3 4; do
        mkdir -p "$DOCKER_LOCAL/miner${i}/data/redis/state"
        mkdir -p "$DOCKER_LOCAL/miner${i}/data/redis/transactions"
        mkdir -p "$DOCKER_LOCAL/miner${i}/data/rocksdb"
        mkdir -p "$DOCKER_LOCAL/miner${i}/log"
    done
    for i in 1 2; do
        mkdir -p "$DOCKER_LOCAL/sharder${i}/data/blocks"
        mkdir -p "$DOCKER_LOCAL/sharder${i}/data/rocksdb"
        mkdir -p "$DOCKER_LOCAL/sharder${i}/data/postgresql"
        mkdir -p "$DOCKER_LOCAL/sharder${i}/data/postgresql2"
        mkdir -p "$DOCKER_LOCAL/sharder${i}/log"
    done

    print_status "Chain data cleaned and directories re-initialized"
}

ensure_chain_mocks() {
    # go mod vendor -v includes test dependencies. Some test files import internal mock
    # packages (e.g. 0chain.net/core/mocks) that are generated files not tracked in git.
    # If missing, the Docker build fails at the "go mod vendor -v" step.
    # This function creates minimal stub files to satisfy the vendor step.
    local GOROOT="${BASE_DIR}/0chain/code/go/0chain.net"
    local -A MOCK_PATHS=(
        ["${GOROOT}/core/mocks/mocks.go"]="package mocks"
        ["${GOROOT}/core/config/mocks/mocks.go"]="package mocks"
        ["${GOROOT}/chaincore/chain/state/mocks/mocks.go"]="package mocks"
        ["${GOROOT}/smartcontract/benchmark/mocks/mocks.go"]="package mocks"
    )
    local created=0
    for path in "${!MOCK_PATHS[@]}"; do
        if [ ! -f "$path" ]; then
            mkdir -p "$(dirname "$path")"
            echo "${MOCK_PATHS[$path]}" > "$path"
            created=$((created + 1))
        fi
    done
    if [ "$created" -gt 0 ]; then
        print_status "Created $created stub mock file(s) for go mod vendor"
    fi
}

start_chain() {
    print_header "Starting Chain Infrastructure"

    # CRITICAL: Replace sync_clock.sh with no-op on VPS.
    # sync_clock.sh runs `docker run --privileged alpine hwclock -s` which hangs
    # indefinitely on Hetzner VPS (no real hardware clock). Blocks deploy 20+ min per call.
    local sync_clock="${BASE_DIR}/0chain/docker.local/bin/sync_clock.sh"
    if [ -f "$sync_clock" ] && grep -q 'hwclock' "$sync_clock" 2>/dev/null; then
        printf '#!/bin/bash\necho "sync_clock skipped (VPS — no hardware clock)"\n' > "$sync_clock"
        print_status "Replaced sync_clock.sh with no-op (VPS fix)"
    fi

    # Fix git safe.directory for all repos (prevents 'dubious ownership' errors in Docker builds)
    for repo_dir in "${BASE_DIR}/0chain" "${BASE_DIR}/blobber" "${BASE_DIR}/0box" "${BASE_DIR}/zauth-server" "${BASE_DIR}/zvault" "${BASE_DIR}/gosdk" "${BASE_DIR}/eblobber"; do
        [ -d "$repo_dir/.git" ] && git config --global --add safe.directory "$repo_dir" 2>/dev/null || true
    done

    # Ensure generated mock stubs exist before building Docker images.
    # These are needed by go mod vendor -v (test imports) but not tracked in git.
    ensure_chain_mocks

    # Patch sc.yaml before first chain start (genesis settings)
    patch_sc_yaml

    # CRITICAL: Fix delegate_wallet + num_delegates in 0chain.yaml BEFORE first start.
    # Once miners register with empty delegate_wallet, it CANNOT be updated on-chain
    # (mn-update-settings returns "access denied" because no wallet matches empty string).
    fix_miner_sharder_config

    # CRITICAL: Configure Kafka in sharder 0chain.yaml BEFORE sharders start.
    # Sharders read kafka config at startup from 0chain.yaml (both server_chain.kafka
    # and root-level kafka: blocks). If not configured before start, sharders won't
    # push events to Kafka, and 0box graph/aggregate endpoints won't work.
    # This is idempotent — the Phase 7 call in main() also runs it for 0box config.
    configure_kafka_in_configs

    # Clean chain data to prevent stale LFB deadlock.
    # Miners load LFB from rocksdb/config/ on startup. If this is stale
    # from a previous deployment, miners reject sharder blocks as "round too old"
    # and the chain deadlocks immediately. clean.sh prevents this.
    clean_chain_data

    # Create Docker network
    if ! docker network ls --format '{{.Name}}' | grep -q "testnet0"; then
        print_status "Creating Docker network testnet0..."
        docker network create --driver bridge --subnet 198.18.0.0/15 testnet0
    else
        print_status "Docker network testnet0 already exists"
    fi

    # Copy magic block file
    local CHAIN_CONFIG="${BASE_DIR}/0chain/docker.local/config"
    if [ -f "${CHAIN_CONFIG}/b0magicBlock_4_miners_2_sharders.json" ]; then
        cp "${CHAIN_CONFIG}/b0magicBlock_4_miners_2_sharders.json" "${CHAIN_CONFIG}/b0magicBlock.json"
        print_status "Copied magic block file (4 miners + 2 sharders)"
    fi

    # Get chain images: pull from Docker Hub if CHAIN_IMAGE_TAG is set, otherwise build locally.
    # Usage: CHAIN_IMAGE_TAG=pr-3469-4d2f1c9a to pull 0chaindev/miner:TAG and 0chaindev/sharder:TAG
    cd "${BASE_DIR}/0chain"
    if [ -n "${CHAIN_IMAGE_TAG:-}" ]; then
        print_status "Pulling chain images from Docker Hub (tag: ${CHAIN_IMAGE_TAG})..."
        if ! docker image inspect sharder > /dev/null 2>&1; then
            docker pull "0chaindev/sharder:${CHAIN_IMAGE_TAG}" 2>&1 && \
                docker tag "0chaindev/sharder:${CHAIN_IMAGE_TAG}" sharder:latest
            print_status "Pulled and tagged sharder:latest"
        fi
        if ! docker image inspect miner > /dev/null 2>&1; then
            docker pull "0chaindev/miner:${CHAIN_IMAGE_TAG}" 2>&1 && \
                docker tag "0chaindev/miner:${CHAIN_IMAGE_TAG}" miner:latest
            print_status "Pulled and tagged miner:latest"
        fi
    else
        if ! docker image inspect zchain_build_base > /dev/null 2>&1 || ! docker image inspect zchain_run_base > /dev/null 2>&1; then
            print_status "Building base images (zchain_build_base + zchain_run_base)..."
            local _base_ok=false
            for _base_try in 1 2 3; do
                if docker.local/bin/build.base.sh 2>&1; then _base_ok=true; break; fi
                print_warning "Base image build failed (attempt $_base_try/3), retrying in 15s..."
                sleep 15
            done
            $_base_ok || { print_error "Base image build failed after 3 attempts"; return 1; }
        fi
        if ! docker image inspect sharder > /dev/null 2>&1; then
            print_status "Building sharder image..."
            local _si_ok=false
            for _si_try in 1 2 3; do
                if docker.local/bin/build.sharders.sh 2>&1; then _si_ok=true; break; fi
                print_warning "sharder image build failed (attempt $_si_try/3), retrying in 15s..."
                sleep 15
            done
            $_si_ok || { print_error "Sharder image build failed after 3 attempts"; return 1; }
        fi
        if ! docker image inspect miner > /dev/null 2>&1; then
            print_status "Building miner image..."
            local _mi_ok=false
            for _mi_try in 1 2 3; do
                if docker.local/bin/build.miners.sh 2>&1; then _mi_ok=true; break; fi
                print_warning "miner image build failed (attempt $_mi_try/3), retrying in 15s..."
                sleep 15
            done
            $_mi_ok || { print_error "Miner image build failed after 3 attempts"; return 1; }
        fi
    fi

    # Start sharders first (--force-recreate ensures fresh container state; retry on transient Docker Hub failures)
    print_status "Starting sharders..."
    for i in 1 2; do
        cd "${BASE_DIR}/0chain/docker.local/build.sharder"
        local _sharder_ok=false
        for _st in 1 2 3; do
            if SHARDER=$i docker compose -p sharder$i -f b0docker-compose.yml up -d --force-recreate 2>&1; then
                _sharder_ok=true; break
            fi
            print_warning "sharder-$i start failed (attempt $_st/3), retrying in 10s..."
            sleep 10
        done
        if $_sharder_ok; then
            print_status "Started sharder-$i"
        else
            print_error "Failed to start sharder-$i after 3 attempts"
        fi
    done
    sleep 5

    # Start miners (--force-recreate ensures fresh container state; retry on transient Docker Hub failures)
    print_status "Starting miners..."
    for i in 1 2 3 4; do
        cd "${BASE_DIR}/0chain/docker.local/build.miner"
        local _miner_ok=false
        for _mt in 1 2 3; do
            if MINER=$i docker compose -p miner$i -f b0docker-compose.yml up -d --force-recreate 2>&1; then
                _miner_ok=true; break
            fi
            print_warning "miner-$i start failed (attempt $_mt/3), retrying in 10s..."
            sleep 10
        done
        if $_miner_ok; then
            print_status "Started miner-$i"
        else
            print_error "Failed to start miner-$i after 3 attempts"
        fi
    done
    sleep 5

    # Start 0dns (if not already running — may have been started in Phase 2 before the chain)
    if ! docker ps --format '{{.Names}}' | grep -q '^0dns$'; then
        start_0dns
    else
        print_status "0dns already running — skipping restart"
    fi

    print_status "Chain infrastructure started!"
}

# Start 0dns service (can be called independently of the chain).
# 0dns must be up before 0box starts because 0box panics if it can't reach block_worker.
# Must be called BEFORE start_0box() and BEFORE start_chain() in the deployment sequence.
start_0dns() {
    # Copy genesis magic block to 0dns config (0dns serves it to clients for network discovery)
    local DNS_MB="${BASE_DIR}/0dns/docker.local/config/b0magicBlock.json"
    local CHAIN_MB_FULL="${BASE_DIR}/0chain/docker.local/config/b0magicBlock_4_miners_2_sharders.json"
    local CHAIN_MB="${BASE_DIR}/0chain/docker.local/config/b0magicBlock.json"
    if [ -f "$CHAIN_MB_FULL" ]; then
        cp "$CHAIN_MB_FULL" "$DNS_MB"
        print_status "Copied genesis magic block (4M+2S) to 0dns config"
    elif [ -f "$CHAIN_MB" ]; then
        cp "$CHAIN_MB" "$DNS_MB"
        print_status "Copied genesis magic block to 0dns config"
    fi

    # Fix 0dns config before starting
    local DNS_CONFIG="${BASE_DIR}/0dns/docker.local/config/0dns.yaml"
    if [ -f "$DNS_CONFIG" ]; then
        # Ensure use_localhost is false (Docker containers need actual IPs, not localhost)
        sed -i.bak 's/use_localhost: true/use_localhost: false/' "$DNS_CONFIG" 2>/dev/null || true
        rm -f "${DNS_CONFIG}.bak"
    fi

    # Fix 0dns docker-compose IP to 198.18.0.100 (must match block_worker in test configs)
    # CRITICAL: 0dns must NOT use 198.18.0.98 (conflicts with blobber-8 on testnet0)
    local DNS_COMPOSE="${BASE_DIR}/0dns/docker.local/docker-compose.yml"
    if [ -f "$DNS_COMPOSE" ] && ! grep -q "198.18.0.100" "$DNS_COMPOSE"; then
        print_status "Fixing 0dns IP to 198.18.0.100 in docker-compose.yml..."
        sed -i.bak 's/198\.18\.0\.[0-9]*/198.18.0.100/g' "$DNS_COMPOSE"
        rm -f "${DNS_COMPOSE}.bak"
    fi

    # Ensure Docker network exists before starting 0dns (it connects to testnet0)
    if ! docker network ls --format '{{.Name}}' | grep -q "testnet0"; then
        print_status "Creating Docker network testnet0 for 0dns..."
        docker network create --driver bridge --subnet 198.18.0.0/15 testnet0 2>/dev/null || true
    fi

    print_status "Starting 0dns..."
    cd "${BASE_DIR}/0dns/docker.local"
    local _dns_ok=false
    for _dns_try in 1 2 3; do
        if docker compose -p 0dns up -d --build --force-recreate 2>&1; then
            _dns_ok=true
            break
        fi
        print_warning "0dns build/start failed (attempt ${_dns_try}/3), retrying in 10s..."
        sleep 10
    done
    if $_dns_ok; then
        print_status "0dns started (198.18.0.100:9091)"
    else
        print_error "Failed to start 0dns after 3 attempts"
    fi
}

# Deploy enterprise blobbers using the eblobber repo
# Enterprise blobbers use:
#   - eblobber repo (${BASE_DIR}/eblobber) - separate from regular blobber repo
#   - eblobber repo's config/ directory (configured with is_enterprise: true)
#   - eblobber repo's docker.local/keys_config/ for keys
#   - eblobber Docker image (built from eblobber repo)
#   - Custom eb0docker-compose.yml generated in eblobber/docker.local/
#   - Container names: eblobber-{1,2,3}, postgres-eblob-{1,2,3}
#   - IPs: 198.18.0.201-203, ports: 5071-5073
#   - NO validators needed for enterprise blobbers
#
# IMPORTANT: The eblobber binary computes client_id (wallet ID) differently from
# regular blobber. Same public key produces a different client_id hash.
# The ID in keys_config/*.txt line 2 does NOT match the actual wallet ID.
# Must get real ID from blobber logs after startup.
#
# Prerequisites:
#   - eblobber repo at ${BASE_DIR}/eblobber
#   - eblobber Docker image must be pre-built
#   - keys in eblobber/docker.local/keys_config/b0bnode{1..5}_keys.txt
build_and_deploy_enterprise_blobbers() {
    print_header "Building and Deploying Enterprise Blobbers (eblobber repo)"

    local EBLOBBER_DIR="${BASE_DIR}/eblobber"
    local EBLOBBER_DOCKER_DIR="${EBLOBBER_DIR}/docker.local"
    local EBLOBBER_CONFIG="${EBLOBBER_DIR}/config"
    local ENTERPRISE_COUNT=5  # Deploy 5 enterprise blobbers (BLOBBER=1..5)

    if [ ! -d "$EBLOBBER_DIR" ]; then
        print_error "eblobber repo not found at $EBLOBBER_DIR"
        print_error "Clone it: git clone https://github.com/0chain/eblobber.git ${EBLOBBER_DIR}"
        return 1
    fi

    # Step 0: Checkout correct gosdk branch for eblobber (enterprise-blobber branch)
    # eblobber depends on gosdk enterprise-blobber branch, not the default lfb branch.
    # Without this, go mod tidy during build picks up the wrong gosdk code.
    checkout_gosdk_for_dependent "eblobber"

    # Step 1: Build eblobber Docker image if not present
    # The eblobber repo uses the same build pattern as the regular blobber repo:
    #   1. build.base.sh  → builds eblobber_base (with herumi MCL/BLS crypto libs)
    #   2. build.blobber.sh → builds eblobber from eblobber_base (uses DOCKER_IMAGE_BASE ARG)
    # IMPORTANT: We use DOCKER_IMAGE_BASE=eblobber_base to avoid overwriting the regular
    # blobber's blobber_base image. Without this, building eblobber after blobber would
    # corrupt the regular blobber's base image, causing all regular blobbers to run
    # enterprise code (the "Invalid version marker passed" bug).
    if ! docker image inspect eblobber >/dev/null 2>&1; then
        print_status "Building eblobber Docker image..."
        inject_local_gosdk "$EBLOBBER_DIR" \
            "${EBLOBBER_DIR}/docker.local/blobber.Dockerfile" \
            "${EBLOBBER_DIR}/docker.local/validator.Dockerfile"
        cd "$EBLOBBER_DIR"

        # Detect required Go version from go.mod AND injected gosdk's go.mod; patch base.Dockerfile
        # if it uses an older version. The eblobber's own go.mod may say go 1.21, but the injected
        # gosdk (feat/enterprise-blobber) requires go 1.22.5. Go 1.21 inside Docker refuses to
        # process modules requiring go 1.22+, so we must bump the base image.
        local eblobber_required_go gosdk_required_go
        eblobber_required_go=$(grep '^go ' "${EBLOBBER_DIR}/go.mod" 2>/dev/null | awk '{print $2}' | head -1)
        gosdk_required_go=$(grep '^go ' "${EBLOBBER_DIR}/gosdk/go.mod" 2>/dev/null | awk '{print $2}' | head -1)
        # Use whichever requires the higher Go version
        if [ -n "$gosdk_required_go" ]; then
            local _eb_minor _gs_minor
            _eb_minor=$(echo "${eblobber_required_go:-0}" | cut -d. -f2)
            _gs_minor=$(echo "$gosdk_required_go" | cut -d. -f2)
            if [ "$_gs_minor" -gt "${_eb_minor:-0}" ] 2>/dev/null; then
                eblobber_required_go="$gosdk_required_go"
            elif [ "$_gs_minor" -eq "${_eb_minor:-0}" ] 2>/dev/null; then
                # Same minor — compare patch version
                local _eb_patch _gs_patch
                _eb_patch=$(echo "${eblobber_required_go:-0}" | cut -d. -f3)
                _gs_patch=$(echo "$gosdk_required_go" | cut -d. -f3)
                if [ "${_gs_patch:-0}" -gt "${_eb_patch:-0}" ] 2>/dev/null; then
                    eblobber_required_go="$gosdk_required_go"
                fi
            fi
        fi
        local base_dockerfile="${EBLOBBER_DIR}/docker.local/base.Dockerfile"
        if [ -n "$eblobber_required_go" ] && [ -f "$base_dockerfile" ]; then
            local base_go
            base_go=$(grep -oP '(?<=FROM golang:)[0-9]+\.[0-9]+' "$base_dockerfile" | head -1)
            local req_minor cur_minor
            req_minor=$(echo "$eblobber_required_go" | cut -d. -f2)
            cur_minor=$(echo "$base_go" | cut -d. -f2)
            if [ -n "$req_minor" ] && [ -n "$cur_minor" ] && [ "$req_minor" -gt "$cur_minor" ] 2>/dev/null; then
                local target_alpine
                if [ "$req_minor" -le 21 ]; then target_alpine="alpine3.18"
                elif [ "$req_minor" -eq 22 ]; then target_alpine="alpine3.19"
                else target_alpine="alpine3.20"
                fi
                print_status "go.mod requires go ${eblobber_required_go} — updating base.Dockerfile to golang:${eblobber_required_go}-${target_alpine}..."
                sed -i "s|FROM golang:[0-9][0-9.]*-alpine[0-9.]*|FROM golang:${eblobber_required_go}-${target_alpine}|" "$base_dockerfile"
                if ! grep -q 'CGO_CFLAGS' "$base_dockerfile"; then
                    sed -i '/^FROM golang:/a ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE=1"' "$base_dockerfile"
                fi
            fi
        fi

        # Build eblobber_base with herumi MCL/BLS crypto libraries
        # Uses separate tag (eblobber_base) to avoid overwriting regular blobber's blobber_base
        print_status "Building eblobber_base image..."
        DOCKER_IMAGE_BASE=eblobber_base docker.local/bin/build.base.sh 2>&1 || {
            cleanup_injected_gosdk "$EBLOBBER_DIR"
            print_error "Failed to build eblobber_base image"
            return 1
        }

        # Build the enterprise blobber image, tagged as 'eblobber', from eblobber_base
        DOCKER_IMAGE_BASE=eblobber_base DOCKER_IMAGE_BLOBBER="-t eblobber" docker.local/bin/build.blobber.sh 2>&1 || {
            cleanup_injected_gosdk "$EBLOBBER_DIR"
            print_error "Failed to build eblobber Docker image"
            return 1
        }
        cleanup_injected_gosdk "$EBLOBBER_DIR"
    else
        print_status "eblobber Docker image already exists"
    fi

    # Use the SAME delegate_wallet as regular blobbers (local.json) — NOT sc_owner.json
    # This ensures bl-update works for ALL blobbers with the same wallet.
    # delegate_wallet is immutable after first registration, so it MUST match
    # the wallet used in fix_blobber_config() and stake_and_configure_blobbers().
    local SC_OWNER_ID
    SC_OWNER_ID=$(cat "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null | jq -r '.client_id' 2>/dev/null || echo "")

    # Step 2: Configure eblobber config for local chain
    print_status "Configuring eblobber config for local chain..."
    local ECONFIG="${EBLOBBER_CONFIG}/0chain_blobber.yaml"
    if [ -f "$ECONFIG" ]; then
        cp "$ECONFIG" "${ECONFIG}.bak"
        sed -i.tmp 's/is_enterprise:.*/is_enterprise: true/' "$ECONFIG"
        sed -i.tmp "s|block_worker:.*|block_worker: http://198.18.0.100:9091|" "$ECONFIG"
        # Add storage_version if not present, or update if present
        if grep -q 'storage_version:' "$ECONFIG"; then
            sed -i.tmp 's/storage_version:.*/storage_version: 1/' "$ECONFIG"
        else
            sed -i.tmp '/^capacity:/a\storage_version: 1' "$ECONFIG"
        fi
        sed -i.tmp 's/service_charge:.*/service_charge: 0.3/' "$ECONFIG"
        sed -i.tmp 's/read_price:.*/read_price: 0.00/' "$ECONFIG"
        sed -i.tmp 's/write_price:.*/write_price: 0.001/' "$ECONFIG"
        sed -i.tmp -E 's/frequency: (60m|1m)/frequency: 5m/' "$ECONFIG"
        if [ -n "$SC_OWNER_ID" ]; then
            sed -i.tmp "s/delegate_wallet:.*/delegate_wallet: '${SC_OWNER_ID}'/" "$ECONFIG"
        fi
        rm -f "${ECONFIG}.tmp"
        print_status "Enterprise config: is_enterprise=true, storage_version=1, delegate_wallet=${SC_OWNER_ID:0:16}..."
    else
        print_error "eblobber config not found at $ECONFIG"
        return 1
    fi

    # Step 3: Generate eb0docker-compose.yml with enterprise-specific naming
    # Uses distinct container names, IPs, and ports to avoid conflicts with regular blobbers
    print_status "Generating enterprise compose template (eb0docker-compose.yml)..."
    cd "$EBLOBBER_DOCKER_DIR"

    cat > "eb0docker-compose.yml" << 'COMPEOF'
version: "3"
services:
  postgres:
    container_name: postgres-eblob-${BLOBBER}
    image: postgres:14
    environment:
      POSTGRES_DB: blobber_meta
      POSTGRES_PORT: 5432
      POSTGRES_HOST: postgres-eblob-${BLOBBER}
      POSTGRES_USER: blobber_user
      POSTGRES_PASSWORD: blobber
      POSTGRES_HOST_AUTH_METHOD: trust
      SLOW_TABLESPACE_PATH: /var/lib/postgresql/hdd
      SLOW_TABLESPACE: hdd_tablespace
    volumes:
      - ../config/postgresql.conf:/etc/postgresql/postgresql.conf
      - ./eblobber${BLOBBER}/data/postgresql:/var/lib/postgresql/data
      - ./sql_init:/docker-entrypoint-initdb.d
      - ./eblobber${BLOBBER}/data/postgresql2:/var/lib/postgresql/hdd
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
    restart: unless-stopped
    networks:
      default:

  eblobber:
    container_name: eblobber-${BLOBBER}
    image: eblobber
    environment:
      - DOCKER=true
      - DB_HOST=postgres-eblob-${BLOBBER}
      - DB_NAME=blobber_meta
      - DB_USER=blobber_user
      - DB_PASSWORD=blobber
      - DB_PORT=5432
      - AWS_ACCESS_KEY_ID=key_id
      - AWS_SECRET_ACCESS_KEY=secret_key
      - BLOBBER_SECRET_NAME=blobber_secret_name
      - AWS_REGION=aws_region
    depends_on:
      - postgres
    links:
      - postgres:postgres
    volumes:
      - ../config:/blobber/config
      - ./eblobber${BLOBBER}/files:/blobber/files
      - ./eblobber${BLOBBER}/data:/blobber/data
      - ./eblobber${BLOBBER}/log:/blobber/log
      - ./ekeys_config:/blobber/keysconfig
      - ./eblobber${BLOBBER}/data/tmp:/tmp
    ports:
      - "507${BLOBBER}:507${BLOBBER}"
      - "3160${BLOBBER}:3160${BLOBBER}"
    command: ./bin/blobber --port 507${BLOBBER} --grpc_port 3160${BLOBBER} --hostname 198.18.0.20${BLOBBER} --deployment_mode 0 --keys_file keysconfig/b0bnode${BLOBBER}_keys.txt --files_dir /blobber/files --log_dir /blobber/log --db_dir /blobber/data
    restart: unless-stopped
    networks:
      default:
      testnet0:
        ipv4_address: 198.18.0.20${BLOBBER}

networks:
  default:
    driver: bridge
  testnet0:
    external: true
COMPEOF

    print_status "Generated eb0docker-compose.yml"

    # Patch compose with --hosturl BEFORE starting containers.
    # Without --hosturl, blobber's node.Self.GetURLBase() returns the Docker IP
    # (e.g. http://198.18.0.204:5074), but the SDK signs V2 signatures using the
    # external URL from chain (e.g. https://test.zus.network/eblobber04/).
    # This mismatch causes "invalid_signature: Invalid signature" on every upload.
    local _eb_domain="${NGINX_DOMAIN:-test.zus.network}"
    if ! grep -q -- '--hosturl' "eb0docker-compose.yml"; then
        sed -i "s|--hostname 198\.18\.0\.20\${BLOBBER}|--hosturl https://${_eb_domain}/eblobber0\${BLOBBER}/ --hostname 198.18.0.20\${BLOBBER}|" "eb0docker-compose.yml" \
            && print_status "Patched eb0docker-compose.yml with --hosturl https://${_eb_domain}/eblobber0N/" \
            || print_warning "Could not patch eb0docker-compose.yml with --hosturl"
    fi

    # Step 3b: Ensure key files exist for all enterprise blobbers
    # Keys for blobbers 1-3 should already exist. For 4+, generate BLS keys using
    # the 0chain keygen tool (from core/encryption/keys/).
    local EKEYS_DIR="${EBLOBBER_DOCKER_DIR}/ekeys_config"
    mkdir -p "$EKEYS_DIR"
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        if [ ! -f "${EKEYS_DIR}/b0bnode${i}_keys.txt" ]; then
            print_status "Generating BLS keys for enterprise blobber $i..."

            # Build keygen tool if not present
            local KEYGEN_BIN="/tmp/0chain_keygen"
            if [ ! -f "$KEYGEN_BIN" ]; then
                local KEYGEN_SRC="${BASE_DIR}/0chain/code/go/0chain.net/core/encryption/keys"
                if [ -d "$KEYGEN_SRC" ]; then
                    cd "$KEYGEN_SRC"
                    go build -o "$KEYGEN_BIN" . 2>/dev/null || {
                        print_warning "Failed to build keygen tool for eblobber-$i keys"
                        continue
                    }
                    cd "$EBLOBBER_DOCKER_DIR"
                else
                    print_warning "Keygen source not found at $KEYGEN_SRC"
                    continue
                fi
            fi

            # Generate BLS keys (bls0chain scheme - same as miners/sharders/blobbers)
            "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                -keys_file_name "b0bnode${i}_keys.txt" \
                -keys_file_path "$EKEYS_DIR" 2>/dev/null || {
                print_warning "Failed to generate keys for eblobber-$i"
                continue
            }
            print_status "Generated BLS keys for eblobber-$i"
        fi
    done

    # Step 4: Create data directories for each enterprise blobber
    print_status "Creating data directories for enterprise blobbers..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        mkdir -p "eblobber${i}/data/postgresql"
        mkdir -p "eblobber${i}/data/postgresql2"
        mkdir -p "eblobber${i}/data/tmp"
        mkdir -p "eblobber${i}/files"
        mkdir -p "eblobber${i}/log"
    done

    # Step 5: Start postgres containers first (--force-recreate for clean state)
    print_status "Starting postgres containers for enterprise blobbers..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        BLOBBER=$i docker compose -p "eblobber${i}" -f "eb0docker-compose.yml" up -d --force-recreate postgres 2>/dev/null || true
    done

    # Wait for postgres to initialize
    print_status "Waiting for postgres initialization (15s)..."
    sleep 15

    # Step 6: Create hdd_tablespace in each postgres (required by eblobber migrations)
    # The sql_init/000-init-db.sh should create it, but as a safety net, create manually
    print_status "Creating hdd_tablespace in enterprise postgres containers..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        local pg_container="postgres-eblob-${i}"
        if docker ps --format '{{.Names}}' | grep -q "^${pg_container}$"; then
            docker exec "$pg_container" bash -c "mkdir -p /var/lib/postgresql/hdd && chown postgres:postgres /var/lib/postgresql/hdd" 2>/dev/null || true
            docker exec "$pg_container" psql -U blobber_user -d blobber_meta -c \
                "CREATE TABLESPACE hdd_tablespace LOCATION '/var/lib/postgresql/hdd';" 2>/dev/null || true
            print_status "Created hdd_tablespace in $pg_container"
        else
            print_warning "$pg_container not running, skipping tablespace creation"
        fi
    done

    # Step 7: Fund a wallet for sending tokens to blobbers
    print_status "Funding deployment wallet for enterprise blobber funding..."
    $ZWALLET faucet --methodName pour --input '{PayerID:unused}' --tokens 100 \
        --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
    sleep 1

    # Step 8: Start enterprise blobber containers (--force-recreate for fresh registration)
    print_status "Starting enterprise blobber containers..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        BLOBBER=$i docker compose -p "eblobber${i}" -f "eb0docker-compose.yml" up -d --force-recreate eblobber 2>/dev/null || true
    done

    # Step 9: Wait for blobbers to start and discover their actual wallet IDs
    # IMPORTANT: eblobber computes client_id differently from regular blobber.
    # Same public key produces a different client_id hash. Must get from logs.
    print_status "Waiting for enterprise blobbers to start (30s)..."
    sleep 30

    # Step 10: Get actual wallet IDs from blobber logs and fund them
    # First ensure deploy wallet has enough balance (need 30 ZCN × ENTERPRISE_COUNT)
    local needed_balance=$((30 * ENTERPRISE_COUNT))
    print_status "Ensuring deploy wallet has >= ${needed_balance} ZCN for eblobber funding..."
    local deploy_bal
    deploy_bal=$($ZWALLET getbalance --json --silent \
        --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null | \
        python3 -c "import json,sys; d=json.load(sys.stdin); print(int(float(d.get('zcn', d.get('balance',0)/1e10 if isinstance(d.get('balance'),int) else 0))))" 2>/dev/null || echo "0")
    while [ "${deploy_bal:-0}" -lt "$needed_balance" ]; do
        $ZWALLET faucet --methodName pour --input '{PayerID:unused}' --tokens 100 \
            --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 3
        deploy_bal=$($ZWALLET getbalance --json --silent \
            --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null | \
            python3 -c "import json,sys; d=json.load(sys.stdin); print(int(float(d.get('zcn', d.get('balance',0)/1e10 if isinstance(d.get('balance'),int) else 0))))" 2>/dev/null || echo "0")
        print_status "  Deploy wallet balance: ${deploy_bal} ZCN (need ${needed_balance})"
    done

    print_status "Discovering actual enterprise blobber wallet IDs from logs..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        local container="eblobber-${i}"
        local actual_id=""

        # Extract the actual wallet ID from blobber logs (retry up to 30s if not yet available)
        for _lp in $(seq 1 6); do
            actual_id=$(docker logs "$container" 2>&1 | grep "^[[:space:]]*ID:" | head -1 | awk '{print $NF}')
            [ -n "$actual_id" ] && [ ${#actual_id} -eq 64 ] && break
            sleep 5
        done

        if [ -n "$actual_id" ] && [ ${#actual_id} -eq 64 ]; then
            print_status "Enterprise eblobber-$i actual wallet ID: ${actual_id:0:16}..."

            # Fund the blobber's actual wallet (30 ZCN)
            print_status "Funding enterprise eblobber-$i ($actual_id)..."
            $ZWALLET send --to_client_id "$actual_id" --tokens 30 --desc "fund eblobber $i" \
                --silent --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null || true
            sleep 2
        else
            print_warning "Could not determine actual wallet ID for eblobber-$i"
            print_warning "Check logs: docker logs $container 2>&1 | grep 'ID:'"
        fi
    done

    # Step 11: Restart blobbers so they can register with funded wallets
    print_status "Restarting enterprise blobbers for on-chain registration..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        docker restart "eblobber-${i}" 2>/dev/null || true
    done

    # Wait for registration
    print_status "Waiting for on-chain registration (45s)..."
    sleep 45

    # Step 11.5: Force URL update on-chain for enterprise blobbers.
    # Enterprise blobbers don't re-register their URL on restart even with --hosturl.
    # Must explicitly call bl-update to push the new public URL to the chain.
    local _ebw_domain="${NGINX_DOMAIN:-test.zus.network}"
    local _ebw_scheme="https"
    print_status "Forcing enterprise blobber URL update on chain (bl-update)..."
    for i in $(seq 1 $ENTERPRISE_COUNT); do
        local container="eblobber-${i}"
        local eb_id
        eb_id=$(docker logs "$container" 2>&1 | grep "^[[:space:]]*ID:" | head -1 | awk '{print $NF}')
        [ -z "$eb_id" ] && continue
        local new_url="${_ebw_scheme}://${_ebw_domain}/eblobber0${i}/"
        $ZBOX bl-update --blobber_id "$eb_id" --url "$new_url" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1 | \
            grep -Ev '^$' | head -3 || true
        print_status "  Updated eblobber-${i} URL → ${new_url}"
        sleep 2
    done

    # Step 12: Verify registration via chain API (getBlobber is authoritative, not docker logs)
    print_status "Verifying enterprise blobber registration via chain API..."
    local registered=0
    local eb_ids
    eb_ids=$(curl -s "http://${SHARDER_IP}:${SHARDER_PORT}/v1/screst/${STORAGE_SC_ADDRESS}/getblobbers?limit=20&offset=0" | \
        python3 -c "
import sys,json
d=json.load(sys.stdin)
blobbers=d.get('Nodes', d.get('nodes', []))
for b in blobbers:
    if b.get('is_enterprise') and not b.get('is_killed'):
        print(b['id'])
" 2>/dev/null || echo "")
    registered=$(echo "$eb_ids" | grep -c . 2>/dev/null | head -1)
    registered=${registered:-0}
    if [ "$registered" -ge "$ENTERPRISE_COUNT" ]; then
        print_status "Enterprise blobbers: $registered/$ENTERPRISE_COUNT registered on chain"
    else
        print_warning "Enterprise blobbers: only $registered/$ENTERPRISE_COUNT registered (wait or check logs)"
    fi

    # Step 13: Stake enterprise blobbers (10 ZCN each from local.json wallet)
    if [ "$registered" -gt 0 ] && [ -n "$eb_ids" ]; then
        print_status "Staking enterprise blobbers (10 ZCN each)..."
        while IFS= read -r eb_id; do
            [ -z "$eb_id" ] && continue
            print_status "  Staking eblobber ${eb_id:0:16}..."
            $ZBOX sp-lock --blobber_id "$eb_id" --tokens 10 \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1 | \
                grep -E 'locked|error|Error' || true
            sleep 2
        done <<< "$eb_ids"
        print_status "Enterprise blobber staking complete!"
    fi

    cd "$SCRIPT_DIR"
    print_status "Enterprise blobber deployment complete! ($registered/$ENTERPRISE_COUNT registered)"
}

# Crawler removed - not needed for local deployment

# Build blobber/validator Docker images and create containers
# Fix blobber config: set delegate_wallet to the SC owner wallet (same for all providers).
# CRITICAL: delegate_wallet is set on first registration and CANNOT be changed after.
# This must be called BEFORE first blobber start on a fresh chain.
fix_blobber_config() {
    print_header "Fixing Blobber Configuration (delegate_wallet + settings)"

    local BLOBBER_CONFIG="${BASE_DIR}/blobber/config/0chain_blobber.yaml"
    if [ ! -f "$BLOBBER_CONFIG" ]; then
        print_warning "Blobber config not found at $BLOBBER_CONFIG"
        return 0
    fi

    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local delegate_wallet=$(jq -r '.client_id' "$SC_OWNER_WALLET" 2>/dev/null)

    if [ -z "$delegate_wallet" ]; then
        print_warning "Could not read delegate wallet ID from $SC_OWNER_WALLET"
        return 0
    fi

    # Set delegate_wallet for regular blobbers to the same wallet used for all providers
    print_status "Setting blobber delegate_wallet to $delegate_wallet..."
    sed -i.bak "s|delegate_wallet:.*|delegate_wallet: '$delegate_wallet'|" "$BLOBBER_CONFIG"

    # Fix block_worker: use local 0dns (same as validators) so blobbers connect to
    # the LOCAL chain regardless of DNS resolution for the public domain.
    # Using https://<domain>/dns fails when the domain resolves to a different server
    # (e.g., test.zus.network pointing to 37.x while deploying on 65.x).
    print_status "Fixing blobber block_worker to local 0dns (http://198.18.0.100:9091)..."
    sed -i.bak "s|block_worker:.*|block_worker: http://198.18.0.100:9091|" "$BLOBBER_CONFIG"

    # Ensure storage_version is 1 (required for SDK v2 allocations)
    if grep -q 'storage_version:' "$BLOBBER_CONFIG"; then
        sed -i.bak 's/storage_version:.*/storage_version: 1/' "$BLOBBER_CONFIG"
    else
        sed -i.bak '/^capacity:/a\storage_version: 1' "$BLOBBER_CONFIG"
    fi

    # Set read_price to 0 (required for free allocation tests)
    sed -i.bak 's/read_price:.*/read_price: 0.00/' "$BLOBBER_CONFIG"

    # Set write_price to 0.001 ZCN/GB/time_unit (chain-enforced minimum for write_price)
    sed -i.bak 's/write_price:.*/write_price: 0.001/' "$BLOBBER_CONFIG"
    rm -f "${BLOBBER_CONFIG}.bak"

    # Set service_charge to 0.1 (10%) so blobber operators earn service fees
    if grep -q 'service_charge:' "$BLOBBER_CONFIG"; then
        sed -i.bak 's/service_charge:.*/service_charge: 0.1/' "$BLOBBER_CONFIG"
    else
        sed -i.bak '/^write_price:/a\service_charge: 0.1' "$BLOBBER_CONFIG"
    fi
    rm -f "${BLOBBER_CONFIG}.bak"
    print_status "Set service_charge to 0.1 (10%)"

    # Fix healthcheck frequency: 60m is too slow for local testing
    # Blobbers need to send healthchecks frequently to stay "active" on chain
    if grep -qE "frequency: (60m|1m)" "$BLOBBER_CONFIG"; then
        sed -i.bak -E 's/frequency: (60m|1m)/frequency: 5m/' "$BLOBBER_CONFIG"
        rm -f "${BLOBBER_CONFIG}.bak"
        print_status "Set healthcheck frequency to 5m"
    fi

    # Set finalize_allocations_interval to 168h (1 week). Blobbers finalize expired
    # allocations on startup and then again every finalize_allocations_interval.
    # Weekly is the right cadence — the blobber code also triggers early if the
    # expired-allocation count reaches 100 (count-based trigger).
    if grep -q 'finalize_allocations_interval:' "$BLOBBER_CONFIG"; then
        sed -i.bak 's/finalize_allocations_interval:.*/finalize_allocations_interval: 168h/' "$BLOBBER_CONFIG"
        rm -f "${BLOBBER_CONFIG}.bak"
        print_status "Set finalize_allocations_interval to 168h (weekly)"
    fi

    # Lower write marker redeem interval: default 10m is too slow for tests that
    # sleep 10 minutes waiting for MovedToChallenge to become non-zero. With 10m
    # interval (randomized to 5–15m), the write marker often isn't processed in
    # time. Setting to 1m ensures it's processed within ~2 minutes.
    if grep -q 'marker_redeem_interval:' "$BLOBBER_CONFIG"; then
        sed -i.bak 's/marker_redeem_interval:.*/marker_redeem_interval: 1m/' "$BLOBBER_CONFIG"
        rm -f "${BLOBBER_CONFIG}.bak"
        print_status "Set marker_redeem_interval to 1m"
    fi

    # Set 0box.public_key so blobbers can verify restricted-blobber auth tickets.
    # The TestRestrictedBlobbers test checks this key exists; without it all auth
    # ticket subtests are skipped.
    local zbox_pubkey
    zbox_pubkey=$(grep 'public_key:' "${BASE_DIR}/0box/docker.local/config/0box.yaml" 2>/dev/null | head -1 | sed 's/.*public_key: *["\x27]*//' | sed 's/["\x27].*//' | tr -d ' ')
    if [ -n "$zbox_pubkey" ]; then
        if grep -q "^0box:" "$BLOBBER_CONFIG"; then
            sed -i.bak "/^0box:/,/^[^ ]/ s|public_key:.*|public_key: ${zbox_pubkey}|" "$BLOBBER_CONFIG"
            rm -f "${BLOBBER_CONFIG}.bak"
        else
            printf '\n0box:\n  public_key: %s\n' "$zbox_pubkey" >> "$BLOBBER_CONFIG"
        fi
        print_status "Set blobber 0box.public_key (${zbox_pubkey:0:16}...)"
    else
        print_warning "Could not read 0box public key — 0box.public_key NOT set (TestRestrictedBlobbers subtests will skip)"
    fi

    # CRITICAL: Verify the config was actually updated. If delegate_wallet still
    # has the default placeholder (9c693cb1...), blobbers will register with the
    # wrong delegate_wallet which is IMMUTABLE after registration.
    local actual_dw=$(grep 'delegate_wallet:' "$BLOBBER_CONFIG" | head -1 | sed "s/.*delegate_wallet: *['\"]*//" | sed "s/['\"].*//")
    if [ "$actual_dw" != "$delegate_wallet" ]; then
        print_error "CRITICAL: delegate_wallet in config ($actual_dw) != expected ($delegate_wallet)"
        print_error "Blobbers will register with WRONG delegate_wallet (immutable after registration!)"
        return 1
    fi
    print_status "Blobber config fixed! delegate_wallet=${delegate_wallet:0:16}... (verified)"

    # Patch docker-compose files to advertise external URLs via nginx.
    # Blobbers register using --hostname (internal Docker IP) by default. Setting
    # --hosturl overrides the on-chain registered URL so browsers can reach blobbers
    # through the nginx reverse proxy.
    # Without this, gosdk WASM CheckAllocStatus fails and allocations show as "broken".
    #
    # DNS check: if the configured domain (e.g. test.zus.network) does NOT resolve to
    # this server's IP, we fall back to HTTP (port 80) and add a /etc/hosts override so
    # local processes (ZS3 server, go tests) can reach blobbers via the domain name.
    # This handles the case where two servers share a domain name but DNS only points
    # to one of them (e.g. secondary test server 65.x sharing test.zus.network with 37.x).
    local _blobber_domain="${NGINX_DOMAIN:-test.zus.network}"
    local _blobber_scheme="https"
    local _blobber_use_external=true  # Whether to set external --hosturl in docker-compose
    local _resolved_ip
    _resolved_ip=$(host "$_blobber_domain" 2>/dev/null | grep 'has address' | head -1 | awk '{print $NF}' || true)
    local _local_ips
    _local_ips=$(hostname -I 2>/dev/null || ip addr show | grep 'inet ' | awk '{print $2}' | cut -d/ -f1 | tr '\n' ' ')
    if [ -n "$_resolved_ip" ] && ! echo "$_local_ips" | grep -qw "$_resolved_ip"; then
        print_warning "Domain $_blobber_domain resolves to $_resolved_ip (not this server — local IPs: $_local_ips)"
        print_status "Using HTTP for blobber hosturls + adding local /etc/hosts override"
        _blobber_scheme="http"
        # Add /etc/hosts entry so local processes (ZS3 server, go tests) route through local nginx
        if ! grep -q "^127\.0\.0\.1.*$_blobber_domain" /etc/hosts; then
            echo "127.0.0.1 $_blobber_domain" >> /etc/hosts
            print_status "Added '127.0.0.1 $_blobber_domain' to /etc/hosts"
        else
            print_status "/etc/hosts already has local override for $_blobber_domain"
        fi
    elif [ -z "$_resolved_ip" ]; then
        # DNS doesn't resolve at all — remove any existing --hosturl so blobbers use --hostname (internal Docker IPs)
        print_warning "Domain $_blobber_domain does not resolve — removing --hosturl from compose files (using internal Docker IPs)"
        _blobber_use_external=false
        local _cleanup_compose_dir="${BASE_DIR}/blobber/docker.local"
        if [ -d "$_cleanup_compose_dir" ]; then
            find "$_cleanup_compose_dir" -name "*.yml" -exec sed -i 's|--hosturl [^[:space:]]* ||g' {} \;
            print_status "Removed --hosturl from blobber docker-compose files (blobbers will use --hostname)"
        fi
        local _cleanup_ecompose_dir="${BASE_DIR}/eblobber/docker.local"
        if [ -d "$_cleanup_ecompose_dir" ]; then
            find "$_cleanup_ecompose_dir" -name "*.yml" -exec sed -i 's|--hosturl [^[:space:]]* ||g' {} \;
            print_status "Removed --hosturl from enterprise blobber docker-compose files"
        fi
    else
        print_status "Domain $_blobber_domain resolves to this server — using HTTPS for blobbers"
    fi

    local COMPOSE_DIR="${BASE_DIR}/blobber/docker.local"
    if [ "$_blobber_use_external" = true ] && [ -d "$COMPOSE_DIR" ]; then
        print_status "Patching blobber docker-compose files with external --hosturl (${_blobber_scheme})..."

        # Template file: blobbers 3-9 use Docker Compose ${BLOBBER} variable
        local tmpl="${COMPOSE_DIR}/b0docker-compose.yml"
        if [ -f "$tmpl" ]; then
            if ! grep -q -- '--hosturl' "$tmpl"; then
                # Not patched yet — add hosturl
                sed -i "s|--hostname 198\.18\.0\.9\${BLOBBER}|--hosturl ${_blobber_scheme}://${_blobber_domain}/blobber0\${BLOBBER}/ --hostname 198.18.0.9\${BLOBBER}|" "$tmpl" \
                    && print_status "  Patched b0docker-compose.yml (blobbers 3-9)" \
                    || print_warning "  Could not patch b0docker-compose.yml"
            elif ! grep -q -- "--hosturl ${_blobber_scheme}://${_blobber_domain}/" "$tmpl"; then
                # Has --hosturl but wrong scheme or domain — replace the full hosturl URL prefix
                sed -i "s|--hosturl [^[:space:]]*/blobber|--hosturl ${_blobber_scheme}://${_blobber_domain}/blobber|g" "$tmpl" \
                    && print_status "  Updated b0docker-compose.yml hosturl to ${_blobber_scheme}://${_blobber_domain}" \
                    || print_warning "  Could not update b0docker-compose.yml hosturl"
            else
                print_status "  b0docker-compose.yml already has correct ${_blobber_scheme}://${_blobber_domain} hosturl"
            fi
        fi

        # Individual compose files for blobbers 1, 2, 10, 11, 12
        for n in 1 2 10 11 12; do
            local cf="${COMPOSE_DIR}/b0docker-compose-${n}.yml"
            [ -f "$cf" ] || continue
            if grep -q -- "--hosturl ${_blobber_scheme}://${_blobber_domain}/" "$cf"; then
                continue  # already has correct scheme+domain, skip
            fi
            local suffix
            [ $n -lt 10 ] && suffix="0${n}" || suffix="${n}"
            if grep -q -- '--hosturl' "$cf"; then
                # Has wrong scheme or domain — replace the full hosturl URL prefix
                sed -i "s|--hosturl [^[:space:]]*/blobber${suffix}/|--hosturl ${_blobber_scheme}://${_blobber_domain}/blobber${suffix}/|g" "$cf" \
                    && print_status "  Updated b0docker-compose-${n}.yml hosturl to ${_blobber_scheme}://${_blobber_domain}/blobber${suffix}/" \
                    || print_warning "  Could not update b0docker-compose-${n}.yml hosturl"
                continue
            fi
            local ext_url="${_blobber_scheme}://${_blobber_domain}/blobber${suffix}/"
            sed -i "s|--hostname |--hosturl ${ext_url} --hostname |" "$cf" \
                && print_status "  Patched b0docker-compose-${n}.yml (${ext_url})" \
                || print_warning "  Could not patch b0docker-compose-${n}.yml"
        done
    fi

    # Also fix enterprise blobber config if present
    local EBLOBBER_CONFIG="${BASE_DIR}/eblobber/config/0chain_blobber.yaml"
    if [ -f "$EBLOBBER_CONFIG" ]; then
        print_status "Setting enterprise blobber delegate_wallet to same wallet..."
        sed -i.bak "s|delegate_wallet:.*|delegate_wallet: '$delegate_wallet'|" "$EBLOBBER_CONFIG"
        rm -f "${EBLOBBER_CONFIG}.bak"

        # Fix enterprise blobber block_worker to point to deployment domain /dns
        # When DNS doesn't resolve, use internal 0dns URL so blobbers can reach the chain.
        local _ebw_block_worker
        if [ "${_blobber_use_external:-true}" = false ]; then
            _ebw_block_worker="http://198.18.0.100:9091"
        else
            local _ebw_domain="${_blobber_domain:-${NGINX_DOMAIN:-test.zus.network}}"
            local _ebw_scheme="${_blobber_scheme:-https}"
            _ebw_block_worker="${_ebw_scheme}://${_ebw_domain}/dns"
        fi
        sed -i.bak "s|block_worker:.*|block_worker: ${_ebw_block_worker}|" "$EBLOBBER_CONFIG"
        rm -f "${EBLOBBER_CONFIG}.bak"
        print_status "Fixed enterprise blobber block_worker to ${_ebw_block_worker}"

        # Set enterprise blobber read_price to 0
        sed -i.bak 's/read_price:.*/read_price: 0.00/' "$EBLOBBER_CONFIG"
        rm -f "${EBLOBBER_CONFIG}.bak"

        # Set enterprise blobber write_price to 0.001 ZCN/GB (chain-enforced minimum)
        sed -i.bak 's/write_price:.*/write_price: 0.001/' "$EBLOBBER_CONFIG"
        rm -f "${EBLOBBER_CONFIG}.bak"

        # Fix enterprise blobber healthcheck frequency too
        if grep -qE "frequency: (60m|1m)" "$EBLOBBER_CONFIG"; then
            sed -i.bak -E 's/frequency: (60m|1m)/frequency: 5m/' "$EBLOBBER_CONFIG"
            rm -f "${EBLOBBER_CONFIG}.bak"
            print_status "Set enterprise blobber healthcheck frequency to 5m"
        fi

        # Lower write marker redeem interval for enterprise blobbers too
        if grep -q 'marker_redeem_interval:' "$EBLOBBER_CONFIG"; then
            sed -i.bak 's/marker_redeem_interval:.*/marker_redeem_interval: 1m/' "$EBLOBBER_CONFIG"
            rm -f "${EBLOBBER_CONFIG}.bak"
            print_status "Set enterprise blobber marker_redeem_interval to 1m"
        fi

        # CRITICAL: Set 0box.public_key so enterprise blobbers can verify auth tickets.
        # The /v1/auth/generate endpoint checks Zbox-Signature against this public key.
        # Without it, auth ticket requests return 403 and ZS3/enterprise allocations fail.
        # Use local.json (the primary deploy wallet) as the auth ticket signer.
        local local_pubkey
        local_pubkey=$(python3 -c "import json; d=json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}')); print(d['keys'][0]['public_key'])" 2>/dev/null || \
                       jq -r '.keys[0].public_key' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null || echo "")
        if [ -n "$local_pubkey" ]; then
            if grep -q "^0box:" "$EBLOBBER_CONFIG"; then
                # Update existing 0box.public_key
                sed -i.bak "/^0box:/,/^[^ ]/ s|public_key:.*|public_key: ${local_pubkey}|" "$EBLOBBER_CONFIG"
                rm -f "${EBLOBBER_CONFIG}.bak"
            else
                # Append 0box section
                printf '\n0box:\n  public_key: %s\n' "$local_pubkey" >> "$EBLOBBER_CONFIG"
            fi
            print_status "Set enterprise blobber 0box.public_key to local.json public key (${local_pubkey:0:16}...)"
        else
            print_warning "Could not read local.json public key — 0box.public_key NOT set in eblobber config (auth tickets will fail)"
        fi

        # Patch eb0docker-compose.yml with --hosturl so enterprise blobbers register with
        # their public domain URL instead of internal 198.18.0.20X IPs.
        # Without this, Atlus cannot reach blobber /_stats from the browser.
        local ECOMPOSE="${BASE_DIR}/eblobber/docker.local/eb0docker-compose.yml"
        if [ -f "$ECOMPOSE" ]; then
            if ! grep -q -- '--hosturl' "$ECOMPOSE"; then
                sed -i "s|--hostname 198\.18\.0\.20\${BLOBBER}|--hosturl ${_ebw_scheme}://${_ebw_domain}/eblobber0\${BLOBBER}/ --hostname 198.18.0.20\${BLOBBER}|" "$ECOMPOSE" \
                    && print_status "  Patched eb0docker-compose.yml with --hosturl ${_ebw_scheme}://${_ebw_domain}/eblobber0N/" \
                    || print_warning "  Could not patch eb0docker-compose.yml with --hosturl"
            elif ! grep -q -- "--hosturl ${_ebw_scheme}://${_ebw_domain}/" "$ECOMPOSE"; then
                sed -i "s|--hosturl [^[:space:]]*/eblobber|--hosturl ${_ebw_scheme}://${_ebw_domain}/eblobber|g" "$ECOMPOSE" \
                    && print_status "  Updated eb0docker-compose.yml eblobber hosturl to ${_ebw_scheme}://${_ebw_domain}" \
                    || print_warning "  Could not update eb0docker-compose.yml eblobber hosturl"
            else
                print_status "  eb0docker-compose.yml already has correct ${_ebw_scheme}://${_ebw_domain} hosturl"
            fi
        fi
    fi
}

# Rebuild the blobber binary inside the blobber_base Docker image using the local
# gosdk at BASE_DIR/gosdk (with replace directive). This deploys the gosdk nonce fix
# (HealthyByLFB cold-start + GetNonce max-from-sharders) to running blobbers without
# requiring a git commit or Docker image rebuild.
#
# Steps:
#  1. Temporarily add "replace github.com/0chain/gosdk => /blobber_gosdk" to blobber go.mod
#  2. Build inside blobber_base (Alpine + Go + Herumi BLS) — produces a musl-linked binary
#  3. Restore go.mod
#  4. docker cp binary into each blobber container + docker start (preserves writable layer)
#
# Usage: bash scripts/deploy_local.sh fix-blobbers
rebuild_blobbers_with_local_gosdk() {
    print_header "Rebuilding Blobber Binary with Local gosdk (nonce fix)"

    local BLOBBER_SRC="${BASE_DIR}/blobber"
    local GOSDK_SRC="${BASE_DIR}/gosdk"
    local BINARY_OUT="${BLOBBER_SRC}/blobber_new"

    if [ ! -d "$BLOBBER_SRC" ]; then
        print_error "Blobber source not found at $BLOBBER_SRC"
        return 1
    fi
    if [ ! -d "$GOSDK_SRC" ]; then
        print_error "gosdk source not found at $GOSDK_SRC — cannot apply nonce fix"
        return 1
    fi
    if ! docker image inspect blobber_base >/dev/null 2>&1; then
        print_error "blobber_base Docker image not found — run a full blobber deploy first"
        return 1
    fi

    print_status "Building blobber binary with local gosdk replace directive..."
    # Backup go.mod, add replace directive, build, restore
    cp "${BLOBBER_SRC}/go.mod" "${BLOBBER_SRC}/go.mod.bak"
    echo "replace github.com/0chain/gosdk => /blobber_gosdk" >> "${BLOBBER_SRC}/go.mod"

    docker run --rm \
        -v "${BLOBBER_SRC}:/blobber_src" \
        -v "${GOSDK_SRC}:/blobber_gosdk:ro" \
        -v "/root/go/pkg/mod:/go/pkg/mod" \
        blobber_base \
        sh -c 'cd /blobber_src && go build -mod=mod -o /blobber_src/blobber_new ./code/go/0chain.net/blobber/ 2>&1'
    local build_exit=$?

    # Always restore go.mod
    mv "${BLOBBER_SRC}/go.mod.bak" "${BLOBBER_SRC}/go.mod"

    if [ $build_exit -ne 0 ] || [ ! -f "$BINARY_OUT" ]; then
        print_error "Blobber build FAILED (exit $build_exit)"
        return 1
    fi
    print_status "Build succeeded: $(du -sh $BINARY_OUT | cut -f1) binary at $BINARY_OUT"

    # Deploy to all running regular blobber containers (blobber-N, not eblobber-N)
    local blobber_containers
    blobber_containers=$(docker ps --format '{{.Names}}' | grep -E '^blobber-[0-9]+$' | sort -t- -k2 -n)
    print_status "Deploying new binary to: $(echo $blobber_containers | tr '\n' ' ')"
    for container in $blobber_containers; do
        docker cp "${BINARY_OUT}" "${container}:/blobber/bin/blobber" \
            && print_status "  Deployed to $container" \
            || print_warning "  Failed to deploy to $container"
    done

    # Restart blobbers (docker stop + start preserves writable layer)
    print_status "Restarting blobber containers..."
    for container in $blobber_containers; do
        docker stop "$container" >/dev/null 2>&1 || true
        sleep 1
        docker start "$container" >/dev/null 2>&1 \
            && print_status "  Restarted $container" \
            || print_warning "  Failed to restart $container"
    done

    print_status "Blobbers rebuilt and restarted with local gosdk nonce fix."
    print_status "Wait ~60s for blobbers to register on chain."
}

# Regenerate BLS keys for all blobbers and validators.
# Use this when blobbers have been killed on-chain and cannot re-register with old keys.
# Generates new BLS key pairs (public_key + private_key), computes client_id via SHA3-256,
# and writes both .txt and .json key files for each blobber and validator.
# Also handles enterprise blobbers in the eblobber repo.
#
# Usage: bash deploy_local.sh regen-keys
#        bash deploy_local.sh regen-keys --enterprise-only
#        bash deploy_local.sh regen-keys --regular-only
regenerate_blobber_keys() {
    print_header "Regenerating BLS Keys for Blobbers and Validators"

    local REGULAR_COUNT=15
    local ENTERPRISE_COUNT=5
    local REGEN_REGULAR=true
    local REGEN_ENTERPRISE=true

    # Parse args
    for arg in "$@"; do
        case "$arg" in
            --enterprise-only) REGEN_REGULAR=false ;;
            --regular-only) REGEN_ENTERPRISE=false ;;
        esac
    done

    # Build keygen tool if not present
    local KEYGEN_BIN="/tmp/0chain_keygen"
    if [ ! -f "$KEYGEN_BIN" ]; then
        local KEYGEN_SRC="${BASE_DIR}/0chain/code/go/0chain.net/core/encryption/keys"
        if [ -d "$KEYGEN_SRC" ]; then
            print_status "Building keygen tool..."
            cd "$KEYGEN_SRC"
            go build -o "$KEYGEN_BIN" . 2>/dev/null || {
                print_error "Failed to build keygen tool from $KEYGEN_SRC"
                return 1
            }
            print_status "Keygen tool built at $KEYGEN_BIN"
        else
            print_error "Keygen source not found at $KEYGEN_SRC"
            print_error "Clone 0chain repo to ${BASE_DIR}/0chain first"
            return 1
        fi
    fi

    # --- Regular blobbers + validators ---
    if [ "$REGEN_REGULAR" = true ]; then
        local KEYS_DIR="${BLOBBER_KEYS_DIR}"
        mkdir -p "$KEYS_DIR"

        # Back up existing keys
        local BACKUP_DIR="${KEYS_DIR}/backup_$(date +%Y%m%d_%H%M%S)"
        if ls "${KEYS_DIR}"/b0bnode*_keys.txt > /dev/null 2>&1; then
            mkdir -p "$BACKUP_DIR"
            cp "${KEYS_DIR}"/b0bnode*_keys.txt* "$BACKUP_DIR/" 2>/dev/null || true
            cp "${KEYS_DIR}"/b0vnode*_keys.txt* "$BACKUP_DIR/" 2>/dev/null || true
            print_status "Backed up existing keys to $BACKUP_DIR"
        fi

        print_status "Generating new keys for $REGULAR_COUNT regular blobbers..."
        for i in $(seq 1 $REGULAR_COUNT); do
            # Generate blobber keys
            "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                -keys_file_name "b0bnode${i}_keys.txt" \
                -keys_file_path "$KEYS_DIR" 2>/dev/null || {
                print_warning "Failed to generate keys for blobber-$i"
                continue
            }
            print_status "  Blobber-$i: new keys generated"
        done

        print_status "Generating new keys for $REGULAR_COUNT validators..."
        for i in $(seq 1 $REGULAR_COUNT); do
            # Generate validator keys
            "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                -keys_file_name "b0vnode${i}_keys.txt" \
                -keys_file_path "$KEYS_DIR" 2>/dev/null || {
                print_warning "Failed to generate keys for validator-$i"
                continue
            }
            print_status "  Validator-$i: new keys generated"
        done

        print_status "Regular blobber + validator keys regenerated ($REGULAR_COUNT each)"
    fi

    # --- Enterprise blobbers ---
    if [ "$REGEN_ENTERPRISE" = true ]; then
        local EKEYS_DIR="${BASE_DIR}/eblobber/docker.local/ekeys_config"
        mkdir -p "$EKEYS_DIR"

        # Back up existing enterprise keys
        local EBACKUP_DIR="${EKEYS_DIR}/backup_$(date +%Y%m%d_%H%M%S)"
        if ls "${EKEYS_DIR}"/b0bnode*_keys.txt > /dev/null 2>&1; then
            mkdir -p "$EBACKUP_DIR"
            cp "${EKEYS_DIR}"/b0bnode*_keys.txt* "$EBACKUP_DIR/" 2>/dev/null || true
            print_status "Backed up existing enterprise keys to $EBACKUP_DIR"
        fi

        print_status "Generating new keys for $ENTERPRISE_COUNT enterprise blobbers..."
        for i in $(seq 1 $ENTERPRISE_COUNT); do
            "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                -keys_file_name "b0bnode${i}_keys.txt" \
                -keys_file_path "$EKEYS_DIR" 2>/dev/null || {
                print_warning "Failed to generate keys for eblobber-$i"
                continue
            }
            print_status "  Enterprise blobber-$i: new keys generated"
        done

        print_status "Enterprise blobber keys regenerated ($ENTERPRISE_COUNT)"
    fi

    print_status "Key regeneration complete! Run 'deploy_local.sh blobbers' to redeploy with new keys."
}

build_and_create_blobbers() {
    print_header "Building and Creating Blobber/Validator Containers"

    # Checkout correct gosdk branch for blobber (lfb branch by default)
    checkout_gosdk_for_dependent "blobber"

    cd "${BASE_DIR}/blobber/docker.local"

    # Ensure BLS keys exist for all regular blobbers/validators (1-12).
    # Keys are pre-committed for 1-6; 7-12 must be generated on first deploy.
    local KEYS_DIR="${BASE_DIR}/blobber/docker.local/keys_config"
    local KEYGEN_BIN="/tmp/0chain_keygen"
    mkdir -p "$KEYS_DIR"
    for i in $(seq 1 12); do
        if [ ! -f "${KEYS_DIR}/b0bnode${i}_keys.txt" ] || [ ! -f "${KEYS_DIR}/b0vnode${i}_keys.txt" ]; then
            # Build keygen tool if not present
            if [ ! -f "$KEYGEN_BIN" ]; then
                local KEYGEN_SRC="${BASE_DIR}/0chain/code/go/0chain.net/core/encryption/keys"
                if [ -d "$KEYGEN_SRC" ]; then
                    print_status "Building keygen tool for blobber keys..."
                    (cd "$KEYGEN_SRC" && go build -o "$KEYGEN_BIN" . 2>/dev/null) || {
                        print_warning "Failed to build keygen tool — blobber keys may be missing"
                        break
                    }
                else
                    print_warning "Keygen source not found at $KEYGEN_SRC — blobber keys may be missing"
                    break
                fi
            fi
            [ ! -f "${KEYS_DIR}/b0bnode${i}_keys.txt" ] && {
                "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                    -keys_file_name "b0bnode${i}_keys.txt" -keys_file_path "$KEYS_DIR" 2>/dev/null \
                    && print_status "Generated keys for blobber-$i" \
                    || print_warning "Failed to generate keys for blobber-$i"
            }
            [ ! -f "${KEYS_DIR}/b0vnode${i}_keys.txt" ] && {
                "$KEYGEN_BIN" -generate_keys -signature_scheme bls0chain \
                    -keys_file_name "b0vnode${i}_keys.txt" -keys_file_path "$KEYS_DIR" 2>/dev/null \
                    && print_status "Generated keys for validator-$i" \
                    || print_warning "Failed to generate keys for validator-$i"
            }
        fi
    done

    # Build images (show output so build failures are visible)
    cd "${BASE_DIR}/blobber"
    inject_local_gosdk "${BASE_DIR}/blobber" \
        "${BASE_DIR}/blobber/docker.local/blobber.Dockerfile" \
        "${BASE_DIR}/blobber/docker.local/validatorDockerfile"

    # Fix incomplete go.sum (common after branch switches via checkout_branches).
    # Without this, Docker build fails with "missing go.sum entry for module".
    if [ -f go.mod ]; then
        print_status "Running go mod tidy for blobber (fix stale go.sum)..."
        go mod tidy 2>&1 || print_warning "go mod tidy failed (build may still work from cache)"
    fi

    if ! docker image inspect blobber_base > /dev/null 2>&1; then
        print_status "Building blobber base image..."
        local _bb_ok=false
        for _bb_try in 1 2 3; do
            if docker.local/bin/build.base.sh 2>&1; then _bb_ok=true; break; fi
            print_warning "blobber_base build failed (attempt $_bb_try/3), retrying in 15s..."
            sleep 15
        done
        $_bb_ok || print_error "Blobber base image build failed after 3 attempts"
    fi
    if ! docker image inspect blobber > /dev/null 2>&1; then
        print_status "Building blobber image..."
        local _bi_ok=false
        for _bi_try in 1 2 3; do
            if docker.local/bin/build.blobber.sh 2>&1; then _bi_ok=true; break; fi
            print_warning "blobber image build failed (attempt $_bi_try/3), retrying in 15s..."
            sleep 15
        done
        $_bi_ok || print_error "Blobber image build failed after 3 attempts"
    fi
    if ! docker image inspect validator > /dev/null 2>&1; then
        print_status "Building validator image..."
        local _vi_ok=false
        for _vi_try in 1 2 3; do
            if docker.local/bin/build.validator.sh 2>&1; then _vi_ok=true; break; fi
            print_warning "validator image build failed (attempt $_vi_try/3), retrying in 15s..."
            sleep 15
        done
        $_vi_ok || print_error "Validator image build failed after 3 attempts"
    fi
    cleanup_injected_gosdk "${BASE_DIR}/blobber"
    cd "${BASE_DIR}/blobber/docker.local"

    # Generate specific compose files for blobbers 10-12 if they don't exist.
    # The generic b0docker-compose.yml uses ${BLOBBER} in IPs (198.18.0.9${BLOBBER})
    # and ports (3150${BLOBBER}), which produces invalid values for double-digit N
    # (e.g. 198.18.0.910 or port 315010). These need hardcoded IPs and ports.
    # IP scheme: blobber=198.18.0.{100+N}, validator=198.18.0.{110+N}
    # Ports: blobber=506{N-10} (host 506{N-10}{N}), validator=507{N-10} (host 507{N-10}{N})
    local _bd="${NGINX_DOMAIN:-test.zus.network}"
    for _n in 10 11 12; do
        local _cf="b0docker-compose-${_n}.yml"
        [ -f "$_cf" ] && continue
        local _off=$((_n - 10))
        local _blobber_ip="198.18.0.$((100 + _n))"
        local _validator_ip="198.18.0.$((110 + _n))"
        local _blobber_port=$((5060 + _off))
        local _validator_port=$((5070 + _off))
        local _grpc_port=$((31510 + _off))
        local _blobber_host_port=$((50600 + _n))
        local _validator_host_port=$((50700 + _n))
        local _grpc_host_port=$((31500 + _n))
        # blobber-10 gets host port 50610 (not 5060 which may conflict)
        print_status "Generating ${_cf} (blobber IP ${_blobber_ip}, validator IP ${_validator_ip})..."
        cat > "$_cf" <<COMPOSEEOF
version: "3"
services:
  postgres:
    container_name: postgres-blob-${_n}
    image: postgres:14
    environment:
      POSTGRES_DB: blobber_meta
      POSTGRES_PORT: 5432
      POSTGRES_HOST: postgres-blob-${_n}
      POSTGRES_USER: blobber_user
      POSTGRES_PASSWORD: blobber
      POSTGRES_HOST_AUTH_METHOD: trust
      SLOW_TABLESPACE_PATH: /var/lib/postgresql/hdd
      SLOW_TABLESPACE: hdd_tablespace
    volumes:
      - ../config/postgresql.conf:/etc/postgresql/postgresql.conf
      - ./blobber${_n}/data/postgresql:/var/lib/postgresql/data
      - ./sql_init:/docker-entrypoint-initdb.d
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
    restart: unless-stopped
    networks:
      default:

  validator:
    container_name: validator-${_n}
    image: validator
    environment:
      - DOCKER=true
      - AWS_ACCESS_KEY_ID=key_id
      - AWS_SECRET_ACCESS_KEY=secret_key
      - VALIDATOR_SECRET_NAME=validator_secret_name
      - AWS_REGION=aws_region
    depends_on:
      - postgres
    volumes:
      - \${CONFIG_PATH:-../config}:/validator/config
      - ./blobber${_n}/data:/validator/data
      - ./blobber${_n}/log:/validator/log
      - ./keys_config:/validator/keysconfig
    ports:
      - "${_validator_host_port}:${_validator_port}"
    command: ./bin/validator --port ${_validator_port} --hostname ${_validator_ip} --deployment_mode 0 --keys_file keysconfig/b0vnode${_n}_keys.txt --log_dir /validator/log
    restart: unless-stopped
    networks:
      default:
      testnet0:
        ipv4_address: ${_validator_ip}

  blobber:
    container_name: blobber-${_n}
    image: blobber
    environment:
      - DOCKER=true
      - DB_HOST=postgres-blob-${_n}
      - DB_NAME=blobber_meta
      - DB_USER=blobber_user
      - DB_PASSWORD=blobber
      - DB_PORT=5432
      - AWS_ACCESS_KEY_ID=key_id
      - AWS_SECRET_ACCESS_KEY=secret_key
      - BLOBBER_SECRET_NAME=blobber_secret_name
      - AWS_REGION=aws_region
    depends_on:
      - validator
      - postgres
    links:
      - validator:validator
      - postgres:postgres
    volumes:
      - \${CONFIG_PATH:-../config}:/blobber/config
      - ./blobber${_n}/files:/blobber/files
      - ./blobber${_n}/data:/blobber/data
      - ./blobber${_n}/log:/blobber/log
      - ./keys_config:/blobber/keysconfig
      - ./blobber${_n}/data/tmp:/tmp
    ports:
      - "${_blobber_host_port}:${_blobber_port}"
      - "${_grpc_host_port}:${_grpc_port}"
    command: ./bin/blobber --port ${_blobber_port} --grpc_port ${_grpc_port} --hosturl https://${_bd}/blobber${_n}/ --hostname ${_blobber_ip} --deployment_mode 0 --keys_file keysconfig/b0bnode${_n}_keys.txt --files_dir /blobber/files --log_dir /blobber/log --db_dir /blobber/data
    restart: unless-stopped
    networks:
      default:
      testnet0:
        ipv4_address: ${_blobber_ip}

networks:
  default:
    driver: bridge
  testnet0:
    external: true
COMPOSEEOF
    done

    # Create containers for blobbers 1-6
    # --force-recreate ensures fresh containers with no stale state from previous deploys.
    for i in 1 2 3 4 5 6; do
        if [ -f "b0docker-compose-${i}.yml" ]; then
            print_status "Creating blobber-$i and validator-$i (specific compose)..."
            docker compose -p blobber$i -f b0docker-compose-${i}.yml up -d --force-recreate 2>/dev/null || true
        else
            print_status "Creating blobber-$i and validator-$i (generic compose)..."
            BLOBBER=$i docker compose -p blobber$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
        fi
    done

    # Create containers for blobbers 7-12
    # Blobbers 7-9 can use generic compose (single-digit N gives valid IPs 198.18.0.9N/6N)
    # Blobbers 10-12 use generated specific compose files (double-digit N gives invalid IPs with generic)
    for i in 7 8 9 10 11 12; do
        if [ -f "b0docker-compose-${i}.yml" ]; then
            print_status "Creating blobber-$i and validator-$i (specific compose)..."
            docker compose -p blobber$i -f b0docker-compose-${i}.yml up -d --force-recreate 2>/dev/null || true
        else
            print_status "Creating blobber-$i and validator-$i (generic compose)..."
            BLOBBER=$i docker compose -p blobber$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
        fi
    done

    print_status "Blobber containers created! (12 regular blobbers + validators)"
}

# Ensure hdd_tablespace exists in all blobber postgres containers
# This is needed because docker-entrypoint-initdb.d scripts only run on first init
# (when data directory is empty). On subsequent starts with existing data, the
# tablespace may be missing if the data was partially cleaned.
ensure_blobber_hdd_tablespace() {
    print_header "Ensuring hdd_tablespace in Blobber PostgreSQL"

    # Wait for postgres containers to be ready
    sleep 5

    local fixed=0
    for i in $(seq 1 15); do
        local container="postgres-blob-$i"
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            # Use 'postgres' db (always exists); blobber_meta may not exist yet on fresh init
            local ts_exists
            ts_exists=$(docker exec "$container" psql -U blobber_user -d postgres -tAc \
                "SELECT 1 FROM pg_tablespace WHERE spcname='hdd_tablespace';" 2>/dev/null)
            if [ "$ts_exists" != "1" ]; then
                # Fix host directory permissions for volume-mounted hdd dirs (owned by root by default)
                local host_hdd_dir="${BASE_DIR}/blobber/docker.local/blobber${i}/data/postgresql2"
                if [ -d "$host_hdd_dir" ]; then
                    chown 999:999 "$host_hdd_dir" 2>/dev/null || true
                fi
                docker exec "$container" bash -c \
                    "mkdir -p /var/lib/postgresql/hdd" 2>/dev/null
                if docker exec "$container" psql -U blobber_user -d postgres -c \
                    "CREATE TABLESPACE hdd_tablespace LOCATION '/var/lib/postgresql/hdd';" 2>/dev/null; then
                    print_status "Created hdd_tablespace in $container"
                    fixed=$((fixed + 1))
                fi
            fi
        fi
    done

    if [ $fixed -gt 0 ]; then
        print_status "Fixed hdd_tablespace in $fixed postgres containers"
    else
        print_status "All blobber postgres containers have hdd_tablespace"
    fi
}

# Patch zauth server.go to allow faucet "pour" method for split wallets.
# By default, AvailableRestrictions in gosdk doesn't include "pour",
# so zauth returns "permission denied" when a split wallet tries faucet.
patch_zauth_faucet() {
    local zauth_server_go="${BASE_DIR}/zauth-server/pkg/app/server.go"
    if [ ! -f "$zauth_server_go" ]; then
        print_error "FATAL: zauth server.go not found at $zauth_server_go — faucet will NOT work for split-key wallets"
        exit 1
    fi
    if grep -q 'Allow faucet pour for split wallets' "$zauth_server_go" 2>/dev/null; then
        print_status "zauth server.go: faucet pour patch already applied"
        return 0
    fi
    # Add init() that appends "pour" to the token_transfers restriction category
    python3 - "$zauth_server_go" << 'PYEOF'
import sys
with open(sys.argv[1], 'r') as f:
    content = f.read()
init_func = '''func init() {
\t// Allow faucet pour for split wallets
\tif transfers, ok := zcncore.AvailableRestrictions["token_transfers"]; ok {
\t\tzcncore.AvailableRestrictions["token_transfers"] = append(transfers, "pour")
\t}
}

'''
content = content.replace('func NewServer(', init_func + 'func NewServer(')
with open(sys.argv[1], 'w') as f:
    f.write(content)
PYEOF
    if [ $? -ne 0 ]; then
        print_error "FATAL: Failed to patch zauth server.go — faucet will NOT work for split-key wallets"
        exit 1
    fi
    print_status "zauth server.go: patched to allow faucet pour for split wallets"
}

# Patch zcn_blimp.js to force local WASM loading instead of CDN.
# The default behavior fetches from cdn.blimp.software which may have an older WASM
# without custom functions (e.g., faucet). Our deploy builds a fresh WASM from gosdk
# and copies it to each app's public/ dir, so we force loading from local /zcn.wasm.
patch_wasm_loader_local() {
    local WEB_APPS_DIR="$1"
    local WASM_LOADER="${WEB_APPS_DIR}/packages/shared/src/lib/wasm/zcn_blimp.js"
    if [ ! -f "$WASM_LOADER" ]; then
        print_error "FATAL: zcn_blimp.js not found at $WASM_LOADER — WASM will load from CDN (stale), faucet will NOT work"
        exit 1
    fi
    if grep -q 'FORCE LOCAL WASM' "$WASM_LOADER" 2>/dev/null; then
        print_status "zcn_blimp.js: local WASM patch already applied"
        return 0
    fi
    python3 - "$WASM_LOADER" << 'PYEOF'
import sys
wasm_loader = sys.argv[1]
with open(wasm_loader, 'r') as f:
    content = f.read()

# Replace getWasmUrl function to always return local path
old_start = 'const getWasmUrl = () => {'
new_func_marker = 'const getCachedWasmResponse'
start_idx = content.find(old_start)
end_idx = content.find(new_func_marker)
if start_idx == -1 or end_idx == -1:
    print('FATAL: Could not find getWasmUrl function boundaries in zcn_blimp.js — WASM will load from CDN')
    sys.exit(1)

new_func = """const getWasmUrl = () => {
  const isEnterpriseMode = getIsEnterpriseMode()
  let suffix = 'mainnet'
  const currentLocation = window?.location?.hostname
  const isHost = host => currentLocation?.includes(host)
  if (isHost('localhost') || isHost('mob')) suffix = 'mob'
  else if (isHost('dev') || isHost('mob.desktop')) suffix = 'dev'
  else if (isHost('demo')) suffix = 'demo'
  else if (isHost('staging')) suffix = 'staging'
  else if (isHost('test')) suffix = 'test'
  const wasmPath = isEnterpriseMode ? '/enterprise-zcn.wasm' : '/zcn.wasm'
  // FORCE LOCAL WASM - bypass CDN and caching (deploy_local.sh builds fresh WASM from gosdk)
  return { suffix, wasmUrl: wasmPath, wasmPath, defaultUrl: wasmPath }
}

"""
content = content[:start_idx] + new_func + content[end_idx:]

# Also make getCachedWasmResponse always return null to skip stale browser cache
old_cache = 'const getCachedWasmResponse = async ({ wasmCache, wasmPath }) => {'
if old_cache in content and 'return null // FORCE LOCAL WASM' not in content:
    content = content.replace(old_cache, old_cache + '\n  return null // FORCE LOCAL WASM - skip cache')

with open(wasm_loader, 'w') as f:
    f.write(content)
print('OK: zcn_blimp.js patched for local WASM')
PYEOF
    if [ $? -ne 0 ]; then
        print_error "FATAL: Failed to patch zcn_blimp.js — WASM will load from CDN (stale), faucet will NOT work"
        exit 1
    fi
    print_status "zcn_blimp.js: patched to load WASM from local /zcn.wasm (not CDN)"
}

# Start supporting services
start_zauth() {
    print_header "Starting zauth-server"

    local ZAUTH_CONFIG="${BASE_DIR}/zauth-server/config/zauthserver.yaml"
    if [ -f "$ZAUTH_CONFIG" ]; then
        # CRITICAL: JWT secret must match across 0box, zauth, and zvault.
        # 0box generates JWTs that zauth/zvault verify. If secrets differ,
        # zauth returns "jwt: invalid token: signature is invalid".
        local OBOX_CONFIG="${BASE_DIR}/0box/docker.local/config/0box.yaml"
        if [ -f "$OBOX_CONFIG" ]; then
            local obox_secret
            obox_secret=$(grep -A1 '^jwt:' "$OBOX_CONFIG" | grep 'secret_key:' | head -1 | awk '{print $2}' | tr -d '"')
            if [ -n "$obox_secret" ]; then
                local current_secret
                current_secret=$(grep 'jwt_secret:' "$ZAUTH_CONFIG" | head -1 | awk '{print $2}' | tr -d '"')
                if [ "$current_secret" != "$obox_secret" ]; then
                    print_status "Syncing zauth JWT secret with 0box..."
                    sed -i.bak "s|jwt_secret:.*|jwt_secret: \"${obox_secret}\"|" "$ZAUTH_CONFIG"
                    rm -f "${ZAUTH_CONFIG}.bak"
                fi
            fi
        fi
    fi

    # Patch: allow faucet pour for split wallets
    patch_zauth_faucet

    # Checkout correct gosdk branch for zauth-server.
    # NOTE: Do NOT inject local gosdk into zauth — it's a key management server, not a chain
    # client. The LFB-aware gosdk branch may be missing symbols (e.g. zcncore.AvailableRestrictions)
    # that zauth's staging branch expects. Let zauth use its own gosdk dependency.
    checkout_gosdk_for_dependent "zauth-server"
    # Tidy go.mod after gosdk version change so go.sum is correct for docker build
    (cd "${BASE_DIR}/zauth-server" && go mod tidy 2>&1) || print_warning "zauth go mod tidy failed"

    cd "${BASE_DIR}/zauth-server/docker.local"

    # Build zauthserver image (always rebuild to pick up patches)
    print_status "Building zauthserver Docker image..."
    docker build -f Dockerfile -t zauthserver ../ 2>&1 || print_error "zauthserver image build failed"
    cleanup_injected_gosdk "${BASE_DIR}/zauth-server"

    # Start only zauth and postgres (skip pgadmin — we manage pgadmin separately with auto-login)
    docker compose -p zauth up -d --force-recreate zauthserver postgres
    print_status "zauth-server started on port 8080"
}

start_zvault() {
    print_header "Starting zvault"

    local ZVAULT_CONFIG="${BASE_DIR}/zvault/config/zvault.yaml"
    if [ -f "$ZVAULT_CONFIG" ]; then
        # Fix postgres.host: must use docker-compose service name, not localhost
        if grep -q "host:.*localhost" "$ZVAULT_CONFIG"; then
            print_status "Fixing zvault postgres.host to postgreszv..."
            sed -i.bak 's/host:.*localhost.*/host: postgreszv/' "$ZVAULT_CONFIG"
            rm -f "${ZVAULT_CONFIG}.bak"
        fi

        # Fix zauth_server: must use Docker bridge gateway IP for container-to-container communication
        if grep -q "zauth_server:.*localhost" "$ZVAULT_CONFIG"; then
            print_status "Fixing zvault zauth_server to Docker bridge gateway..."
            sed -i.bak 's|zauth_server:.*|zauth_server: http://172.17.0.1:8080|' "$ZVAULT_CONFIG"
            rm -f "${ZVAULT_CONFIG}.bak"
        fi

        # CRITICAL: JWT secret must match across 0box, zauth, and zvault.
        local OBOX_CONFIG="${BASE_DIR}/0box/docker.local/config/0box.yaml"
        if [ -f "$OBOX_CONFIG" ]; then
            local obox_secret
            obox_secret=$(grep -A1 '^jwt:' "$OBOX_CONFIG" | grep 'secret_key:' | head -1 | awk '{print $2}' | tr -d '"')
            if [ -n "$obox_secret" ]; then
                local current_secret
                current_secret=$(grep 'jwt_secret:' "$ZVAULT_CONFIG" | head -1 | awk '{print $2}' | tr -d '"')
                if [ "$current_secret" != "$obox_secret" ]; then
                    print_status "Syncing zvault JWT secret with 0box..."
                    sed -i.bak "s|jwt_secret:.*|jwt_secret: \"${obox_secret}\"|" "$ZVAULT_CONFIG"
                    rm -f "${ZVAULT_CONFIG}.bak"
                fi
            fi
        fi
    fi

    # Fix zvault pgadmin port: 8083 (not 8082, which conflicts with 0box pgadmin)
    local ZVAULT_COMPOSE="${BASE_DIR}/zvault/docker.local/docker-compose.yml"
    if [ -f "$ZVAULT_COMPOSE" ] && grep -q "8082:80" "$ZVAULT_COMPOSE"; then
        print_status "Fixing zvault pgadmin port 8082 -> 8083 (avoid 0box conflict)..."
        sed -i 's|8082:80|8083:80|g' "$ZVAULT_COMPOSE"
    fi

    cd "${BASE_DIR}/zvault/docker.local"

    # Build zvault image if it doesn't exist (e.g. after docker system prune)
    if ! docker image inspect zvault > /dev/null 2>&1; then
        print_status "Building zvault Docker image..."
        docker build -f Dockerfile -t zvault ../ 2>&1 || print_error "zvault image build failed"
    fi

    # Start only zvault and postgres (skip pgadmin_zvault — we manage pgadmin separately with auto-login)
    docker compose -p zvault up -d --force-recreate zvault postgreszv
    print_status "zvault started on port 8090"
}

start_elasticsearch() {
    print_header "Starting Elasticsearch"

    # Remove any stale elasticsearch container to avoid name conflicts
    if docker ps -a --format '{{.Names}}' | grep -q "^elasticsearch$"; then
        docker rm -f elasticsearch 2>/dev/null || true
    fi

    # Limit Elasticsearch heap to 2GB. Without this, ES 7.17 auto-sizes to 50% of RAM
    # (e.g. 31GB on a 64GB server), causing severe memory pressure and swap thrashing.
    # 512MB heap: enough for test environment indexing, prevents OOM kills on shared servers.
    # Default ES heap (50% of RAM) can be >30GB on 62GB servers, causing OOM during startup.
    local ES_COMPOSE="${BASE_DIR}/0box/docker.local/docker-compose.yml"
    if [ -f "$ES_COMPOSE" ]; then
        if ! grep -q "ES_JAVA_OPTS" "$ES_COMPOSE"; then
            print_status "Adding ES_JAVA_OPTS=-Xms512m -Xmx512m to elasticsearch compose..."
            sed -i.bak '/discovery.type=single-node/a\      - ES_JAVA_OPTS=-Xms512m -Xmx512m' "$ES_COMPOSE"
            rm -f "${ES_COMPOSE}.bak"
        else
            # Update any existing ES_JAVA_OPTS value to 512m
            sed -i.bak 's|ES_JAVA_OPTS=.*|ES_JAVA_OPTS=-Xms512m -Xmx512m|' "$ES_COMPOSE"
            rm -f "${ES_COMPOSE}.bak"
        fi
    fi

    # Start from 0box docker.local which includes elasticsearch
    cd "${BASE_DIR}/0box/docker.local"
    docker compose up -d --force-recreate elasticsearch 2>/dev/null || true

    # Wait for elasticsearch to be healthy
    print_status "Waiting for Elasticsearch to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; then
            print_status "Elasticsearch is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    print_warning "Elasticsearch may not be fully ready"
}

start_kafka() {
    print_header "Starting Kafka (with SASL Authentication)"

    local KAFKA_DIR="${BASE_DIR}/0chain/docker.local/build.kafka"
    local KAFKA_CONFIG_DIR="${KAFKA_DIR}/config"
    local KAFKA_DATA_DIR="${BASE_DIR}/0chain/docker.local/kafka"

    # IMPORTANT: Both sharder and 0box HARDCODE SASL.Enable=true in their Kafka producers/consumers.
    # Source: 0chain.net/smartcontract/dbs/queueProvider/kafka.go:40 (sharder)
    # Source: 0box/code/zboxcore/entity/kafka.go:239 (0box)
    # Therefore Kafka MUST be configured with SASL_PLAINTEXT authentication.

    # Create kafka directories
    mkdir -p "${KAFKA_DIR}" "${KAFKA_CONFIG_DIR}" "${KAFKA_DATA_DIR}"

    # Create JAAS configuration for SASL authentication
    local KAFKA_USER="${KAFKA_USERNAME:-admin}"
    local KAFKA_PASS="${KAFKA_PASSWORD:-admin-secret}"
    cat > "${KAFKA_CONFIG_DIR}/kafka_server_jaas.conf" << JAASEOF
KafkaServer {
    org.apache.kafka.common.security.plain.PlainLoginModule required
    username="${KAFKA_USER}"
    password="${KAFKA_PASS}"
    user_${KAFKA_USER}="${KAFKA_PASS}";
};
JAASEOF

    # Create docker-compose.yml for Kafka with SASL (using apache/kafka since bitnami is unavailable)
    cat > "${KAFKA_DIR}/docker-compose.yml" << 'KAFKAEOF'
services:
  kafka:
    image: 'apache/kafka:3.7.0'
    container_name: kafka
    networks:
      default:
      testnet0:
        ipv4_address: 198.19.0.99
    volumes:
      - ../kafka:/tmp/kraft-combined-logs
      - ./config/kafka_server_jaas.conf:/opt/kafka/config/kafka_server_jaas.conf
    environment:
      KAFKA_NODE_ID: '0'
      KAFKA_PROCESS_ROLES: controller,broker
      KAFKA_LISTENERS: SASL_PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: SASL_PLAINTEXT://198.19.0.99:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,SASL_PLAINTEXT:SASL_PLAINTEXT
      KAFKA_CONTROLLER_QUORUM_VOTERS: 0@kafka:9093
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_INTER_BROKER_LISTENER_NAME: SASL_PLAINTEXT
      KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL: PLAIN
      KAFKA_SASL_ENABLED_MECHANISMS: PLAIN
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: '1'
      KAFKA_OPTS: -Djava.security.auth.login.config=/opt/kafka/config/kafka_server_jaas.conf
      CLUSTER_ID: MkU3OEVBNTcwNTJENDM2Qk
    restart: unless-stopped
networks:
  default:
    driver: bridge
  testnet0:
    external: true
KAFKAEOF

    # Ensure testnet0 network exists before starting Kafka.
    # Kafka's docker-compose uses testnet0 (external: true), which is normally created
    # by start_chain(). When Kafka starts first (before the chain), the network must
    # be pre-created here to avoid "network testnet0 not found" docker compose failure.
    if ! docker network ls --format '{{.Name}}' | grep -q "^testnet0$"; then
        print_status "Creating Docker network testnet0 (needed by Kafka before chain starts)..."
        docker network create --driver bridge --subnet 198.18.0.0/15 testnet0
    fi

    # Start Kafka
    cd "${KAFKA_DIR}"
    docker compose up -d --force-recreate 2>/dev/null || docker-compose up -d --force-recreate 2>/dev/null || {
        print_error "Failed to start Kafka"
        return 1
    }

    # Create SASL client properties for readiness check
    # The broker only has a SASL_PLAINTEXT listener, so the health check must authenticate
    docker exec kafka bash -c "cat > /tmp/client.properties << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username=\"${KAFKA_USER}\" password=\"${KAFKA_PASS}\";
EOF" 2>/dev/null

    # Wait for Kafka to be ready
    print_status "Waiting for Kafka to start..."
    local max_wait=60
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if docker exec kafka \
            /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list \
            --command-config /tmp/client.properties 2>/dev/null; then
            print_status "Kafka is ready at 198.19.0.99:9092 (SASL_PLAINTEXT)"
            print_status "SASL credentials: ${KAFKA_USER} / ****"
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
    done

    # Even if the check fails, Kafka may still be starting up
    if docker ps --format '{{.Names}}' | grep -q "kafka"; then
        print_status "Kafka container is running (may still be initializing)"
    else
        print_warning "Kafka container not found. Check: docker compose -f ${KAFKA_DIR}/docker-compose.yml logs"
    fi
}

# Configure Kafka settings in sharder (0chain.yaml) and 0box (0box.yaml).
# CRITICAL: Sharder Kafka config MUST be under server_chain.kafka (NOT root-level kafka:).
#   Sharder code reads: server_chain.kafka.host, server_chain.kafka.username, etc.
#   Placing kafka: at YAML root level causes PANIC: "Net.SASL.User must not be empty"
#   because the code finds SASL enabled but reads empty username from wrong path.
# 0box Kafka config is at root-level kafka: (different from sharder).
configure_kafka_in_configs() {
    print_header "Configuring Kafka in Service Configs"

    local KAFKA_HOST="198.19.0.99:9092"
    local KAFKA_USER="${KAFKA_USERNAME:-admin}"
    local KAFKA_PASS="${KAFKA_PASSWORD:-admin-secret}"
    local KAFKA_TOPIC="events"
    local KAFKA_TRIGGER_ROUND="1"

    # ---- Sharder: 0chain.yaml (server_chain.kafka) ----
    local CHAIN_YAML="${BASE_DIR}/0chain/docker.local/config/0chain.yaml"
    if [ -f "$CHAIN_YAML" ]; then
        print_status "Configuring Kafka in sharder 0chain.yaml (under server_chain.kafka)..."

        # Use Python to update BOTH server_chain.kafka AND root-level kafka blocks
        # Sharder reads config from mixed locations:
        #   - server_chain.kafka.host/username/password (nested under server_chain:)
        #   - kafka.enabled/topic/partition/write_timeout/trigger_round (root level)
        # Both blocks MUST be present and correct for Kafka to work
        python3 << PYEOF
import re, sys

yaml_file = "$CHAIN_YAML"
with open(yaml_file, 'r') as f:
    content = f.read()

server_chain_kafka_block = """  kafka:
    enabled: true
    host: "${KAFKA_HOST}"
    username: "${KAFKA_USER}"
    password: "${KAFKA_PASS}"
    topic: "${KAFKA_TOPIC}"
    write_timeout: 10s
    trigger_round: ${KAFKA_TRIGGER_ROUND}
"""

root_kafka_block = """kafka:
  enabled: true
  host: "${KAFKA_HOST}"
  username: "${KAFKA_USER}"
  password: "${KAFKA_PASS}"
  topic: "${KAFKA_TOPIC}"
  partition: 0
  write_timeout: 10s
  trigger_round: ${KAFKA_TRIGGER_ROUND}
"""

# Step 1: Remove any existing server_chain.kafka block (indented 2 spaces under server_chain)
content = re.sub(r'  kafka:\n(?:    [^\n]*\n)*', '', content)

# Step 2: Remove any existing root-level kafka: block (even if values are wrong)
content = re.sub(r'^kafka:\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)

# Step 3: Insert server_chain.kafka block inside server_chain section
# Find the end of server_chain section (next root-level key) and insert before it
lines = content.split('\n')
in_server_chain = False
insert_idx = -1
for i, line in enumerate(lines):
    if line.startswith('server_chain:'):
        in_server_chain = True
        continue
    if in_server_chain:
        # A non-empty, non-comment line at root level (no leading space) = end of server_chain
        if line and not line.startswith(' ') and not line.startswith('#'):
            insert_idx = i
            break

if insert_idx > 0:
    # Insert kafka block right before the next root-level section
    kafka_lines = server_chain_kafka_block.rstrip().split('\n')
    for j, kline in enumerate(kafka_lines):
        lines.insert(insert_idx + j, kline)
    # Add blank line after
    lines.insert(insert_idx + len(kafka_lines), '')
    content = '\n'.join(lines)
    print(f"Inserted server_chain.kafka at line {insert_idx}")
else:
    print("ERROR: Could not find end of server_chain section")
    sys.exit(1)

# Step 4: Append root-level kafka block at end of file
content = content.rstrip('\n') + '\n\n' + root_kafka_block
print("Added root-level kafka: block at end of file")

with open(yaml_file, 'w') as f:
    f.write(content)
PYEOF
        if [ $? -eq 0 ]; then
            print_status "Sharder 0chain.yaml Kafka config updated (server_chain.kafka + root kafka)"
        else
            print_warning "Failed to update sharder Kafka config"
        fi
    else
        print_warning "0chain.yaml not found at $CHAIN_YAML"
    fi

    # ---- 0box: 0box.yaml (root-level kafka:) ----
    local BOX_CONFIG="${BASE_DIR}/0box/docker.local/config/0box.yaml"
    if [ -f "$BOX_CONFIG" ]; then
        print_status "Configuring Kafka in 0box.yaml..."

        python3 << PYEOF
import re

yaml_file = "$BOX_CONFIG"
with open(yaml_file, 'r') as f:
    content = f.read()

kafka_block = """kafka:
  enabled: true
  host: "${KAFKA_HOST}"
  username: "${KAFKA_USER}"
  password: "${KAFKA_PASS}"
  eventsTopic: "${KAFKA_TOPIC}"
  eventsBlobberMonitorTopic: "monitor"
  eventsGroupId: "events-consumer"
  eventsRetryTopic: "events-retry"
  triggerRound: ${KAFKA_TRIGGER_ROUND}
"""

# Remove existing root-level kafka: block
content = re.sub(r'^kafka:\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)

# Append kafka block at end
content = content.rstrip() + '\n\n' + kafka_block

with open(yaml_file, 'w') as f:
    f.write(content)
print("0box Kafka config updated")
PYEOF
        if [ $? -eq 0 ]; then
            print_status "0box.yaml Kafka config updated"
        else
            print_warning "Failed to update 0box Kafka config"
        fi
    else
        print_warning "0box.yaml not found at $BOX_CONFIG"
    fi

    print_status "Kafka configuration complete for sharder and 0box"
}

# Fix 0box config for local deployment.
# MUST be called after any git pull/checkout of the 0box repo (swap-image, services, full deploy).
# The 0box repo's default config ships with dev.zus.network — calling this ensures the
# local chain URLs are always applied before starting/restarting the container.
patch_web_apps_for_dev() {
    # Applies three patches to web-apps source for dev/test environments:
    #
    # Fix 1: NEXTAUTH_SECRET in shared/.env
    #   NextAuth requires NEXTAUTH_SECRET or it returns 500 "There is a problem with the
    #   server configuration" on every /api/auth/* request including /api/auth/_log.
    #
    # Fix 2: WASM wallet init — !privateKey guard (shared/src/lib/wasm/index.js)
    #   Without this, wallets with a privateKey but no mnemonic skip setWallet() → all
    #   blobber ops fail with "invalid client" and login shows "mnemonic required".
    #
    # Fix 3: user_id in OTP verify body (shared/src/store/user/actions/user.js)
    #   0box's VerifyPhoneOTPSignup and VerifyPhoneOTPLogin handlers require user_id when
    #   IsDevelopmentNoAuth()=true (no Firebase token to derive UID from). The web app
    #   doesn't send it, causing 400 "user_id is required" → "Error verifying OTP".
    #   In production: 0box ignores user_id (derives UID from Firebase token instead).

    local WEB_APPS_DIR="${BASE_DIR}/web-apps"
    if [ ! -d "$WEB_APPS_DIR" ]; then
        print_warning "web-apps not found at $WEB_APPS_DIR — skipping dev patches"
        return 0
    fi

    local SHARED_ENV="${WEB_APPS_DIR}/packages/shared/.env"

    # Fix 1: NEXTAUTH_SECRET
    if [ -f "$SHARED_ENV" ] && ! grep -q "^NEXTAUTH_SECRET=" "$SHARED_ENV"; then
        local secret="${AUTH0_SECRET:-$(openssl rand -hex 32)}"
        local domain="${NGINX_DOMAIN:-test.zus.network}"
        local prefix="${APP_DOMAIN_PREFIX:-test}"
        printf '\nNEXTAUTH_SECRET=%s\nNEXTAUTH_URL=https://%s.vult.network\n' "$secret" "$prefix" >> "$SHARED_ENV"
        print_status "Added NEXTAUTH_SECRET + NEXTAUTH_URL to shared/.env"
    fi

    # Fix 2: WASM wallet initialization — add !privateKey guard (wasm/index.js)
    #   Without this, wallets with a privateKey but no cached mnemonic return early from
    #   getWasm() without calling setWallet() → WASM has no key material → all blobber
    #   operations fail with "invalid client" and login shows "mnemonic required".
    #   The upstream source is missing this guard; we patch it here before every build.
    local WASM_JS="${WEB_APPS_DIR}/packages/shared/src/lib/wasm/index.js"
    if [ -f "$WASM_JS" ] && grep -q "!mnemonic && !wallet?.is_split) {" "$WASM_JS"; then
        sed -i "s/if (!mnemonic && !wallet?.is_split) {/if (!mnemonic \&\& !wallet?.is_split \&\& !privateKey) {/" "$WASM_JS"
        print_status "web-apps wasm/index.js patched (!privateKey guard added)"
    else
        print_status "web-apps wasm/index.js: already patched or not found"
    fi

    # Fix 3: user_id in OTP body
    local USER_JS="${WEB_APPS_DIR}/packages/shared/src/store/user/actions/user.js"
    if [ -f "$USER_JS" ] && ! grep -q "devUserId" "$USER_JS"; then
        python3 - "$USER_JS" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    content = f.read()

old = "  if (userName) body.append('username', userName)\n"
new = old + (
    "\n"
    "  // Dev mode: 0box IsDevelopmentNoAuth() requires user_id for OTP verify endpoints.\n"
    "  // In production: 0box derives user ID from Firebase token; this field is ignored.\n"
    "  const devUserId = firebaseTokens?.uid ||\n"
    "    ('devnet_' + (phoneNumber || email || '').toLowerCase().replace(/[^a-z0-9]/g, ''))\n"
    "  body.append('user_id', devUserId)\n"
)
if old in content:
    with open(path, 'w') as f:
        f.write(content.replace(old, new))
    print("Patched user.js: added user_id to OTP verify body")
else:
    print("user.js: target not found — may already be patched or source changed")
PYEOF
        print_status "web-apps user.js patched (user_id for dev OTP flow)"
    else
        print_status "web-apps user.js already patched (user_id present)"
    fi

    # Fix 4: getZusBlogs fetch timeout — blog.zus.network may be unreachable from test
    # servers, causing next build SSG to hang indefinitely (fetch has no default timeout).
    local ORG_SCHEMES="${WEB_APPS_DIR}/packages/shared/src/lib/constants/orgSchemes.js"
    if [ -f "$ORG_SCHEMES" ] && ! grep -q "AbortSignal.timeout" "$ORG_SCHEMES"; then
        sed -i "s/const res = await fetch(GRAPH_ENDPOINT, {/const res = await fetch(GRAPH_ENDPOINT, { signal: AbortSignal.timeout(5000),/" "$ORG_SCHEMES"
        print_status "web-apps orgSchemes.js patched (5s fetch timeout for getZusBlogs)"
    fi

    # Fix 4b: Replace deprecated blog.zus.network with zus.network/blog
    if [ -f "$ORG_SCHEMES" ] && grep -q "blog\.zus\.network" "$ORG_SCHEMES"; then
        sed -i "s|blog\.zus\.network/graphql|zus.network/blog/graphql|g" "$ORG_SCHEMES"
        print_status "web-apps orgSchemes.js patched (blog URL: zus.network/blog/graphql)"
    fi

    # Fix 5: mock token dispatch on dev login (verifyOtpTwilio, user.js)
    # When 0box IsDevelopmentNoAuth() returns no customToken, dispatch synthetic firebaseTokens
    # so subsequent requests have valid X-App-User-ID and X-App-ID-TOKEN headers.
    if [ -f "$USER_JS" ] && ! grep -q "mock-token-" "$USER_JS"; then
        python3 - "$USER_JS" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    content = f.read()

old = (
    "    dispatch({ type: types.VERIFY_OTP_SUCCESS, payload: firebaseTokens })\n"
    "    return defaultResponse\n"
    "  } catch (e) {"
)
new = (
    "    // Dev mode: 0box returns no customToken when IsDevelopmentNoAuth() is true.\n"
    "    // Synthesize mock tokens so subsequent API calls have valid X-App-User-ID + X-App-ID-TOKEN.\n"
    "    const syntheticFirebaseData = {\n"
    "      uid: devUserId,\n"
    "      accessToken: 'mock-token-' + devUserId,\n"
    "      refreshToken: 'mock-token-' + devUserId,\n"
    "      expirationTime: Date.now() + 3600000,\n"
    "    }\n"
    "    dispatch({ type: types.VERIFY_OTP_SUCCESS, payload: syntheticFirebaseData })\n"
    "    return { data: syntheticFirebaseData }\n"
    "  } catch (e) {"
)
if old in content:
    with open(path, 'w') as f:
        f.write(content.replace(old, new, 1))
    print("Patched user.js: mock token dispatch for dev OTP flow")
else:
    print("user.js: mock-token target not found — may already be patched or source changed")
PYEOF
        print_status "web-apps user.js patched (mock token dispatch for dev)"
    else
        print_status "web-apps user.js mock token already patched"
    fi
}

patch_0box_auth_for_dev() {
    # Adds IsDevelopmentNoAuth() bypass to VerifyPhoneOTPLogin so that phone OTP login
    # works on test/dev deployments without requiring real Firebase tokens.
    # The signup handler already has this bypass; this applies the same pattern to login.
    local AUTH_FILE="${BASE_DIR}/0box/code/zboxcore/handler/auth.go"
    if [ ! -f "$AUTH_FILE" ]; then
        print_warning "0box auth.go not found — skipping dev login bypass"
        return 0
    fi
    if grep -q "dev-mode bypass (injected by deploy_local.sh)" "$AUTH_FILE" 2>/dev/null; then
        print_status "0box auth.go: dev login bypass already applied"
        return 0
    fi

    print_status "Patching 0box auth.go: adding IsDevelopmentNoAuth bypass to VerifyPhoneOTPLogin..."
    python3 - "$AUTH_FILE" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    code = f.read()

OLD = (
    '\tfirebaseUserByEmail, err := zboxFirebase.TokenVerifyEmail(ctx, firebaseTokenFromEmail, email)\n'
    '\tif err != nil {\n'
    '\t\treturn nil, err\n'
    '\t}\n'
    '\n'
    '\tres := common.JsonMessageDataResponse{}\n'
    '\n'
    '\tif config.IsMainNet() || config.Configuration.BlockWorker == "https://demo.zus.network/dns" {'
)
NEW = (
    '\t// dev-mode bypass (injected by deploy_local.sh): skip Firebase for test/dev environments\n'
    '\tif config.IsDevelopmentNoAuth() {\n'
    '\t\tuserID := verifyPhoneOTPDetails.UserID\n'
    '\t\tif userID == "" {\n'
    '\t\t\treturn nil, common.NewError("400", "user_id is required")\n'
    '\t\t}\n'
    '\t\townerFromPhone, _ := a.ownersRepo.GetByPhoneNumber(ctx, phoneNumber)\n'
    '\t\tif ownerFromPhone == nil {\n'
    '\t\t\tdevEmail := email\n'
    '\t\t\tif devEmail == "" || devEmail == "undefined" || devEmail == "null" {\n'
    '\t\t\t\tdevEmail = userID + "@dev.zus.network"\n'
    '\t\t\t}\n'
    '\t\t\t_, _ = a.ownersRepo.Create(ctx, &modelV2.OwnerEntity{UserName: userID, UserID: userID, PhoneNumber: phoneNumber, Email: devEmail})\n'
    '\t\t}\n'
    '\t\treturn common.JsonMessageDataResponse{Message: "OTP verified successfully"}, nil\n'
    '\t}\n'
    '\n'
    '\tfirebaseUserByEmail, err := zboxFirebase.TokenVerifyEmail(ctx, firebaseTokenFromEmail, email)\n'
    '\tif err != nil {\n'
    '\t\treturn nil, err\n'
    '\t}\n'
    '\n'
    '\tres := common.JsonMessageDataResponse{}\n'
    '\n'
    '\tif config.IsMainNet() || config.Configuration.BlockWorker == "https://demo.zus.network/dns" {'
)
if OLD not in code:
    print("ERROR: target block not found in auth.go (code may have changed)", file=sys.stderr)
    sys.exit(1)
with open(path, 'w') as f:
    f.write(code.replace(OLD, NEW, 1))
print("OK: patched VerifyPhoneOTPLogin")
PYEOF
    [ $? -eq 0 ] && print_status "0box auth.go: dev login bypass applied" || \
        print_warning "0box auth.go: patch failed — login may still require Firebase in dev mode"
}

patch_0box_jwt_refresh_for_dev() {
    # Two-part fix for PUT /v2/jwt/token in dev mode:
    # 1. handler.go: bypass SignatureHandler middleware (BLS signature check)
    # 2. auth.go: bypass user_id mismatch in RefreshJwtToken handler
    #    WASM SDK sends wallet client_id as X-App-User-ID, but JWT was created
    #    with Firebase UID — Check() compares them and fails. In dev mode, skip
    #    the strict validation and just refresh any valid JWT.
    # Production (mode 0) is unaffected — both checks remain active.

    # --- Part 1: handler.go — bypass SignatureHandler on PUT route ---
    local HANDLER_FILE="${BASE_DIR}/0box/code/zboxcore/router/handler.go"
    if [ -f "$HANDLER_FILE" ]; then
        if grep -q "jwt-refresh bypass (injected by deploy_local.sh)" "$HANDLER_FILE" 2>/dev/null; then
            print_status "0box handler.go: SignatureHandler bypass already applied"
        else
            print_status "Patching 0box handler.go: bypassing SignatureHandler on PUT /jwt/token..."
            python3 - "$HANDLER_FILE" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    code = f.read()

OLD = '\t\tv2jwtPublic.PUT("/jwt/token", middleware.SignatureHandler(), RefreshJwtToken)\n'
NEW = (
    '\t\t// jwt-refresh bypass (injected by deploy_local.sh): skip SignatureHandler in dev mode\n'
    '\t\tif !config.IsDevelopmentNoAuth() {\n'
    '\t\t\tv2jwtPublic.PUT("/jwt/token", middleware.SignatureHandler(), RefreshJwtToken)\n'
    '\t\t} else {\n'
    '\t\t\tv2jwtPublic.PUT("/jwt/token", RefreshJwtToken)\n'
    '\t\t}\n'
)
if OLD not in code:
    print("ERROR: target line not found in handler.go (already patched or code changed)", file=sys.stderr)
    sys.exit(1)
with open(path, 'w') as f:
    f.write(code.replace(OLD, NEW, 1))
print("OK: patched PUT /jwt/token route in handler.go")
PYEOF
            [ $? -eq 0 ] && print_status "0box handler.go: SignatureHandler bypass applied" || \
                print_warning "0box handler.go: SignatureHandler patch failed"
        fi
    else
        print_warning "0box handler.go not found — skipping SignatureHandler bypass"
    fi

    # --- Part 2: auth.go — bypass user_id mismatch in RefreshJwtToken handler ---
    # WASM SDK sends wallet client_id as X-App-User-ID, but JWT claims have Firebase UID.
    # In dev mode, use simpler headers (no hash_validate on client_id) and extract user_id
    # from the JWT itself instead of requiring the header to match.
    local AUTH_FILE="${BASE_DIR}/0box/code/zboxcore/router/auth.go"
    if [ -f "$AUTH_FILE" ]; then
        if grep -q "jwt-refresh-userid bypass (injected by deploy_local.sh)" "$AUTH_FILE" 2>/dev/null; then
            print_status "0box auth.go: JWT refresh user_id bypass already applied"
        else
            print_status "Patching 0box auth.go: bypassing user_id mismatch in RefreshJwtToken..."
            python3 - "$AUTH_FILE" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    code = f.read()

# Step 1: Add encoding/base64 and encoding/json imports if not present
if '"encoding/base64"' not in code:
    code = code.replace('"fmt"', '"encoding/base64"\n\t"encoding/json"\n\t"fmt"', 1)
    print("Added encoding/base64 and encoding/json imports")
elif '"encoding/json"' not in code:
    code = code.replace('"encoding/base64"', '"encoding/base64"\n\t"encoding/json"', 1)
    print("Added encoding/json import")

# Step 2: Patch RefreshJwtToken
OLD = '''func RefreshJwtToken(context *gin.Context) {

	headers := helpers.NewAuthHeader()
	if err := headers.Bind(context); err != nil {
		context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := jwtService.Refresh(headers.UserID, headers.JwtToken)'''

NEW = '''func RefreshJwtToken(context *gin.Context) {
	// jwt-refresh-userid bypass (injected by deploy_local.sh)
	if config.IsDevelopmentNoAuth() {
		// In dev mode, skip strict AuthHeader validation and user_id match.
		// WASM SDK sends wallet client_id as X-App-User-ID but JWT has Firebase UID.
		// Extract the real user_id from JWT claims and use that for refresh.
		jwtToken := context.GetHeader("X-Jwt-Token")
		if jwtToken == "" {
			jwtToken = context.GetHeader("X-JWT-TOKEN")
		}
		if jwtToken == "" {
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing X-Jwt-Token header"})
			return
		}
		// Decode JWT payload to get user_id from claims (base64url middle section)
		parts := strings.SplitN(jwtToken, ".", 3)
		if len(parts) != 3 {
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "malformed jwt token"})
			return
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid jwt payload: " + err.Error()})
			return
		}
		var claims struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(payload, &claims); err != nil || claims.UserID == "" {
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "cannot extract user_id from jwt"})
			return
		}
		response, err := jwtService.Refresh(claims.UserID, jwtToken)
		if err != nil {
			logging.Logger0box.Error("jwt: ", zap.String("user_id", claims.UserID), zap.Error(err))
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		context.JSON(http.StatusOK, jwt.JwtResponse{JwtToken: response})
		return
	}

	headers := helpers.NewAuthHeader()
	if err := headers.Bind(context); err != nil {
		context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := jwtService.Refresh(headers.UserID, headers.JwtToken)'''

if OLD not in code:
    print("ERROR: RefreshJwtToken target block not found in auth.go", file=sys.stderr)
    sys.exit(1)
with open(path, 'w') as f:
    f.write(code.replace(OLD, NEW, 1))
print("OK: patched RefreshJwtToken in auth.go")
PYEOF
            [ $? -eq 0 ] && print_status "0box auth.go: JWT refresh user_id bypass applied" || \
                print_warning "0box auth.go: JWT refresh patch failed"
        fi
    else
        print_warning "0box auth.go not found — skipping JWT refresh user_id bypass"
    fi
}

fix_0box_config() {
    local BOX_CONFIG="${BASE_DIR}/0box/docker.local/config/0box.yaml"
    if [ ! -f "$BOX_CONFIG" ]; then
        print_warning "0box config not found at $BOX_CONFIG — skipping fix"
        return 0
    fi

    # Patch 0box Go source for dev-mode auth bypass (login endpoint needs same bypass as signup)
    patch_0box_auth_for_dev
    # Patch 0box handler.go to bypass SignatureHandler on JWT refresh in dev mode (wallet recovery)
    patch_0box_jwt_refresh_for_dev

    # Backup before modifying
    cp "$BOX_CONFIG" "${BOX_CONFIG}.bak_$(date +%s)" 2>/dev/null || true

    # 1. block_worker: must point to local 0dns (chain discovery)
    print_status "Fixing 0box block_worker → http://198.18.0.100:9091"
    sed -i "s|block_worker:.*|block_worker: http://198.18.0.100:9091|" "$BOX_CONFIG"

    # 2. host and domain: match deployment domain or use localhost
    local DEPLOY_DOMAIN="${NGINX_DOMAIN:-}"
    if [ -n "$DEPLOY_DOMAIN" ]; then
        local NETWORK_NAME="${DEPLOY_DOMAIN%%.*}"
        print_status "Setting 0box host → https://0box.${DEPLOY_DOMAIN}"
        sed -i "s|^host:.*|host: https://0box.${DEPLOY_DOMAIN}|" "$BOX_CONFIG"
        sed -i "/^network:/,/^[a-z]/ s|^  name:.*|  name: ${NETWORK_NAME}|" "$BOX_CONFIG"
        sed -i "s|^\(  domain:\).*|  domain: ${DEPLOY_DOMAIN}|" "$BOX_CONFIG"
        # Fix kms.domain and nft_tracker.domain: code builds URLs as
        # "zvault.{network.name}.{kms.domain}" so these must use the base domain
        # (e.g. zus.network), NOT the full domain (e.g. test1.zus.network),
        # otherwise the URL becomes zvault.test1.test1.zus.network (DNS failure).
        local BASE_DOMAIN="${DEPLOY_DOMAIN#*.}"
        sed -i "/^kms:/,/^[a-z]/ s|^  domain:.*|  domain: ${BASE_DOMAIN}|" "$BOX_CONFIG"
        sed -i "/^nft_tracker:/,/^[a-z]/ s|^  domain:.*|  domain: ${BASE_DOMAIN}|" "$BOX_CONFIG"
    else
        # No domain configured — use localhost for local-only deployment
        print_status "Setting 0box host → http://localhost (no domain configured)"
        sed -i "s|^host:.*|host: http://localhost|" "$BOX_CONFIG"
        sed -i "/^network:/,/^[a-z]/ s|^  name:.*|  name: local|" "$BOX_CONFIG"
        sed -i "s|^\(  domain:\).*|  domain: localhost|" "$BOX_CONFIG"
    fi

    # 3. deployment_mode: 3 = DeploymentDevelopmentNoAuth — ONLY mode that bypasses Firebase + CSRF
    # IMPORTANT: deployment_mode is a CLI FLAG (--deployment_mode N) in docker-compose.yml command,
    # NOT a yaml config key. The yaml value is ignored. Must patch docker-compose.yml.
    # Mode 0 = DeploymentDevelopment (still enforces Firebase TokenVerify — NOT what we want)
    # Mode 3 = DeploymentDevelopmentNoAuth (skips AuthHandler + CSRFHandler + SignatureHandler)
    local BOX_COMPOSE_FILE="${BASE_DIR}/0box/docker.local/docker-compose.yml"
    if [ -f "$BOX_COMPOSE_FILE" ]; then
        if grep -q "deployment_mode" "$BOX_COMPOSE_FILE"; then
            sed -i "s|--deployment_mode [0-9]*|--deployment_mode 3|g" "$BOX_COMPOSE_FILE"
        else
            # Add the flag if not present
            sed -i "s|./bin/zbox |./bin/zbox --deployment_mode 3 |" "$BOX_COMPOSE_FILE"
        fi
        print_status "Set --deployment_mode 3 in docker-compose.yml (DeploymentDevelopmentNoAuth)"
    else
        print_warning "0box docker-compose.yml not found at $BOX_COMPOSE_FILE"
    fi

    # 4. elastic.host: docker service name is "elasticsearch", not "elastic"
    sed -i '/^elastic:/,/^[a-z]/ s|host:.*elastic.*|host: elasticsearch|' "$BOX_CONFIG"

    # 4b. jwt.token_timeout: default is 10m — expires during normal use, and Refresh() can't
    # recover expired tokens (Check() rejects them). Set to 720h for dev so users stay logged in.
    sed -i "s|token_timeout:.*|token_timeout: 720h # extended for dev: default 10m causes constant re-login|" "$BOX_CONFIG"
    print_status "Set jwt.token_timeout: 720h (prevents 10-minute JWT expiry forcing re-login)"

    # 4. Kafka: enable and point to local Kafka (event pipeline from sharder → 0box)
    print_status "Fixing 0box Kafka config → 198.19.0.99:9092 (SASL)"
    sed -i "/^kafka:/,/^[a-z]/ {
        s|enabled:.*|enabled: true|
        s|host:.*|host: \"198.19.0.99:9092\"|
        s|username:.*|username: \"admin\"|
        s|password:.*|password: \"admin-secret\"|
    }" "$BOX_CONFIG"

    # 5. server_chain.owner: must match the on-chain SC owner wallet
    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    if [ -f "$SC_OWNER_WALLET" ]; then
        local sc_owner_id sc_owner_pubkey
        sc_owner_id=$(jq -r '.client_id' "$SC_OWNER_WALLET" 2>/dev/null)
        sc_owner_pubkey=$(jq -r '.keys[0].public_key // .client_key' "$SC_OWNER_WALLET" 2>/dev/null)
        if [ -n "$sc_owner_id" ] && [ "$sc_owner_id" != "null" ]; then
            print_status "Setting 0box server_chain.owner → ${sc_owner_id:0:16}..."
            sed -i "s|owner:.*|owner: ${sc_owner_id}|" "$BOX_CONFIG"

            # NOTE: Do NOT overwrite wallets.free_storage_assginer or free_storage.public_key.
            # 0box.yaml ships with a dedicated keypair (free_storage.private_key / public_key)
            # that 0box uses to sign free storage markers. The chain SC is registered with the
            # matching public_key via fund_0box(). Overwriting with the SC owner key breaks
            # free allocation requests (chain verifies with SC owner key, 0box signs with
            # free_storage.private_key → signature mismatch).
        fi
    fi

    # 6. Firebase credentials: copy from .secrets.env path or scripts/ if available
    local FIREBASE_KEY_DST="${BASE_DIR}/0box/docker.local/config/0box_firebase_key.json"
    local FIREBASE_KEY_SRC="${SCRIPT_DIR}/0box_firebase_key.json"
    if [ -f "$FIREBASE_KEY_SRC" ]; then
        print_status "Copying Firebase key from scripts/ → 0box config"
        cp "$FIREBASE_KEY_SRC" "$FIREBASE_KEY_DST"
    elif [ -n "${FIREBASE_KEY_PATH:-}" ] && [ -f "$FIREBASE_KEY_PATH" ]; then
        print_status "Copying Firebase key from FIREBASE_KEY_PATH → 0box config"
        cp "$FIREBASE_KEY_PATH" "$FIREBASE_KEY_DST"
    else
        # Check if the existing key is valid (has a private_key field)
        if [ -f "$FIREBASE_KEY_DST" ]; then
            local fb_email
            fb_email=$(jq -r '.client_email' "$FIREBASE_KEY_DST" 2>/dev/null || echo "")
            if [ -n "$fb_email" ]; then
                print_status "Using existing Firebase key: $fb_email"
            else
                print_warning "Firebase key at $FIREBASE_KEY_DST may be invalid"
                print_warning "Place a valid key at ${FIREBASE_KEY_SRC} or set FIREBASE_KEY_PATH"
            fi
        else
            print_warning "No Firebase key found. OTP/phone auth will not work."
            print_warning "Place a key at ${FIREBASE_KEY_SRC} or set FIREBASE_KEY_PATH"
        fi
    fi

    # 7. Render server (Gotenberg): set gotenberg_host to container name on testnet0 network
    #    Gotenberg provides LibreOffice+Chromium for document/image-to-PDF conversion.
    #    Used by /v2/convert-to-pdf endpoint for file preview in Vult/Blimp/Bolt.
    sed -i 's|gotenberg_host:.*|gotenberg_host: "gotenberg:3000"|' "$BOX_CONFIG"
    print_status "Set gotenberg_host → gotenberg:3000"

    # 8. Ensure jwt.test_numbers includes common developer numbers.
    # Numbers in test_numbers bypass Twilio OTP — 0box accepts any 6-digit code for them.
    # Useful for dev login when real Twilio credentials are not available.
    # Twilio test credentials (ACxxxx with test auth_token) only work with +15005550006, +15005550001 etc.
    # For US developer logins, add numbers here or add via: jwt > test_numbers in 0box.yaml.
    if [ -f "$BOX_CONFIG" ] && grep -q 'test_numbers:' "$BOX_CONFIG"; then
        # Add US developer test number if not already present
        if ! grep -q '"+14087723543"' "$BOX_CONFIG" 2>/dev/null; then
            sed -i '/test_numbers:/a\    - "+14087723543"' "$BOX_CONFIG"
        fi
        print_status "jwt.test_numbers: numbers in list bypass Twilio (use any 6-digit OTP code)"
    fi

    print_status "0box config updated for local deployment"
}

start_gotenberg() {
    print_header "Starting Gotenberg (render server)"
    # Gotenberg provides LibreOffice + Chromium for rendering documents/images to PDF.
    # Used by 0box /v2/convert-to-pdf endpoint (file preview in web apps).
    if docker ps --format '{{.Names}}' | grep -q "^gotenberg$"; then
        print_status "Gotenberg already running — skipping"
        return 0
    fi
    docker rm -f gotenberg 2>/dev/null || true
    docker run -d --name gotenberg \
        --network testnet0 \
        -p 3010:3000 \
        --restart unless-stopped \
        gotenberg/gotenberg:8 \
        gotenberg --api-port=3000
    print_status "Gotenberg started on port 3010 (network: testnet0)"
}

start_0box() {
    print_header "Starting 0box"

    local BOX_CONFIG="${BASE_DIR}/0box/docker.local/config/0box.yaml"
    local BOX_COMPOSE="${BASE_DIR}/0box/docker.local/docker-compose.yml"

    # Checkout correct gosdk branch for 0box and inject local gosdk
    checkout_gosdk_for_dependent "0box"
    inject_local_gosdk "${BASE_DIR}/0box" "${BASE_DIR}/0box/docker.local/Dockerfile"

    # Fix config: The 0box repo ships with dev.zus.network — always apply local settings.
    fix_0box_config

    # Build 0box image from checked-out source using the repo's own build scripts.
    # Without this, docker compose uses the stale 0chaindev/0box:staging image from Docker Hub
    # instead of the local branch, causing 0box to run months-old code.
    local LOCAL_IMAGE="0chaindev/0box:local-build"
    print_status "Building 0box image from source as ${LOCAL_IMAGE}..."
    cd "${BASE_DIR}/0box"
    ./docker.local/bin/build.base.sh 2>&1 || { cleanup_injected_gosdk "${BASE_DIR}/0box"; print_error "Failed to build zbox_base"; return 1; }
    ./docker.local/bin/build.zbox.sh 2>&1 || { cleanup_injected_gosdk "${BASE_DIR}/0box"; print_error "0box docker build failed"; return 1; }
    docker tag zbox "${LOCAL_IMAGE}"
    cleanup_injected_gosdk "${BASE_DIR}/0box"
    if [ -f "$BOX_COMPOSE" ]; then
        sed -i "s|image: 0chaindev/0box:.*|image: ${LOCAL_IMAGE}|g" "$BOX_COMPOSE"
        print_status "Updated docker-compose.yml to use ${LOCAL_IMAGE}"
    fi

    # Fix Redis image: pin to redis:7-alpine to avoid redis:alpine pulling v8.4.0 which crashes
    if [ -f "$BOX_COMPOSE" ] && grep -q "redis:alpine" "$BOX_COMPOSE"; then
        print_status "Pinning Redis image to redis:7-alpine (avoid v8.4.0 crash)..."
        sed -i 's|redis:alpine|redis:7-alpine|g' "$BOX_COMPOSE"
    fi

    # Fix redis.conf: remove pidfile and rename-command directives that cause crashes
    local REDIS_CONF="${BASE_DIR}/0box/docker.local/config/redis.conf"
    if [ -f "$REDIS_CONF" ]; then
        if grep -qE '^pidfile|^rename-command' "$REDIS_CONF"; then
            print_status "Removing problematic redis.conf directives (pidfile, rename-command)..."
            sed -i '/^pidfile/d' "$REDIS_CONF"
            sed -i '/^rename-command/d' "$REDIS_CONF"
        fi
    fi

    # Fix Elasticsearch heap: default (50% of RAM) can be >30GB on 62GB servers → OOM kill.
    # 512MB is sufficient for test environment indexing.
    if [ -f "$BOX_COMPOSE" ] && ! grep -q "ES_JAVA_OPTS" "$BOX_COMPOSE"; then
        print_status "Adding ES_JAVA_OPTS=-Xms512m -Xmx512m to elasticsearch (prevent OOM)"
        sed -i 's/- discovery.type=single-node/- discovery.type=single-node\n      - ES_JAVA_OPTS=-Xms512m -Xmx512m/' "$BOX_COMPOSE"
    fi

    # Pin Redis image to 7.4.3-alpine — redis:alpine (v8+) crashes with SIGSEGV (exit 139)
    if grep -q 'redis:alpine' "$BOX_COMPOSE" 2>/dev/null; then
        print_status "Pinning Redis image to redis:7.4.3-alpine (prevents SIGSEGV)..."
        sed -i 's|"redis:alpine"|"redis:7.4.3-alpine"|g; s|image: redis:alpine|image: redis:7.4.3-alpine|g' "$BOX_COMPOSE"
    fi

    # Fix redis host: use IP (198.18.9.11) instead of hostname to avoid DNS lookup failure
    # when testnet0 is an external network — Docker DNS may not resolve 'redis' reliably
    if [ -f "$BOX_CONFIG" ] && grep -q 'host: "redis:6379"' "$BOX_CONFIG" 2>/dev/null; then
        print_status "Fixing 0box redis host → 198.18.9.11:6379 (bypass testnet0 DNS)"
        sed -i 's|host: "redis:6379"|host: "198.18.9.11:6379"|g' "$BOX_CONFIG"
    fi

    # Remove stale containers that may conflict with 0box compose service names
    docker rm -f elasticsearch 2>/dev/null || true
    docker rm -f 0box-redis 2>/dev/null || true
    docker rm -f postgres-0box 2>/dev/null || true

    cd "${BASE_DIR}/0box/docker.local"

    # Step 1: Start postgres + redis FIRST (without 0box) to clean stale DB state.
    # On a clean deploy the chain starts from round 1, but postgres may retain
    # last_processed_round from a previous deployment. If 0box reads a stale
    # last_processed_round > 0 it skips all early events including provider
    # registrations (TagAddMiner, TagAddSharder, TagAddBlobber) → Atlus shows 0 providers.
    print_status "Starting postgres + redis (without 0box yet)..."
    docker compose -p 0box up -d --force-recreate postgres redis
    print_status "Waiting for postgres to accept connections..."
    local pg_wait=0
    while [ $pg_wait -lt 30 ]; do
        if docker exec ${ZBOX_PG_CONTAINER:-postgres-0box} pg_isready -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} 2>/dev/null | grep -q "accepting"; then
            break
        fi
        sleep 2
        pg_wait=$((pg_wait + 2))
    done

    # Step 2: Truncate all 0box tables to remove stale last_processed_round and old provider data.
    # This ensures 0box starts with last_processed_round=0 and processes all events from round 1.
    print_status "Truncating 0box database (remove stale state from previous deploy)..."
    local all_tables
    all_tables=$(docker exec ${ZBOX_PG_CONTAINER:-postgres-0box} psql -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} -t -c \
        "SELECT string_agg('\"' || tablename || '\"', ', ')
         FROM pg_tables
         WHERE schemaname = 'public'
           AND tablename NOT IN ('goose_db_version');" 2>/dev/null | tr -d ' \n')
    if [ -n "$all_tables" ]; then
        docker exec ${ZBOX_PG_CONTAINER:-postgres-0box} psql -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} -c \
            "TRUNCATE TABLE ${all_tables} CASCADE;" 2>/dev/null \
            && print_status "0box database truncated (clean slate for fresh deploy)" \
            || print_warning "Truncate had issues (non-critical)"
    fi

    # Step 3: Delete Kafka consumer group so 0box starts from earliest offset (round 1).
    # On a clean deploy, Kafka data dirs are wiped but the consumer group may be recreated
    # by 0box reading __consumer_offsets. Explicitly delete it as a safety net.
    print_status "Deleting Kafka consumer group 'events-consumer' (fresh start from offset 0)..."
    docker exec kafka bash -c '
cat > /tmp/sasl.props << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";
EOF
/opt/kafka/bin/kafka-consumer-groups.sh \
    --bootstrap-server 198.19.0.99:9092 \
    --command-config /tmp/sasl.props \
    --delete --group events-consumer' 2>/dev/null \
        && print_status "Kafka consumer group deleted" \
        || print_warning "Consumer group delete failed (may not exist yet — OK)"

    # Step 4: Start 0box. With truncated DB (last_processed_round=0) and no consumer group,
    # 0box will start consuming from Kafka's earliest offset and process ALL events from round 1,
    # including provider registrations (TagAddMiner, TagAddSharder, TagAddBlobber).
    print_status "Starting 0box..."
    docker compose -p 0box up -d --force-recreate 0box
    print_status "0box started on port 9081 (processing events from round 1)"
}

# Sync a wallet file's nonce to match the on-chain nonce.
# Required on long-running chains where nonce in file (0) is stale vs on-chain (N).
# Usage: sync_wallet_nonce_from_chain <wallet_file_path>
sync_wallet_nonce_from_chain() {
    local wallet_file="$1"
    [ -f "$wallet_file" ] || return 0
    local client_id
    client_id=$(jq -r '.client_id' "$wallet_file" 2>/dev/null) || return 0
    [ -n "$client_id" ] || return 0
    local on_chain_nonce
    on_chain_nonce=$(curl -s "http://198.18.0.82:7172/v1/client/get/balance?client_id=$client_id" -m 5 2>/dev/null \
        | python3 -c 'import json,sys; print(json.load(sys.stdin).get("nonce",0))' 2>/dev/null || echo "0")
    local file_nonce
    file_nonce=$(jq '.nonce // 0' "$wallet_file" 2>/dev/null || echo "0")
    if [ "$on_chain_nonce" != "$file_nonce" ]; then
        jq ".nonce = $on_chain_nonce" "$wallet_file" > "${wallet_file}.tmp" \
            && mv "${wallet_file}.tmp" "$wallet_file" \
            && print_status "Synced nonce for $(basename $wallet_file): $file_nonce → $on_chain_nonce" || true
    fi
}

# Fund 0box for free allocations
fund_0box() {
    print_header "Funding 0box for Free Allocations"

    # Wait for 0box to start and get its wallet ID
    print_status "Waiting for 0box to initialize..."
    sleep 10

    local BOX_YAML="${BASE_DIR}/0box/docker.local/config/0box.yaml"

    # Get 0box assigner wallet ID for funding (wallets.free_storage_assginer)
    local box_wallet_id=""
    if [ -f "$BOX_YAML" ]; then
        box_wallet_id=$(grep -A5 '^wallets:' "$BOX_YAML" | grep 'free_storage_assginer:' | head -1 | awk '{print $2}')
    fi
    if [ -z "$box_wallet_id" ]; then
        box_wallet_id=$(docker logs 0box 2>&1 | grep -o "0box wallet client id: [a-f0-9]*" | head -1 | awk '{print $NF}')
    fi
    if [ -z "$box_wallet_id" ]; then
        print_error "FATAL: Could not find 0box wallet ID (wallets.free_storage_assginer) in 0box.yaml or docker logs."
        print_error "Free allocations will NOT work. Check 0box.yaml and 0box container."
        exit 1
    fi

    print_status "0box wallet ID: ${box_wallet_id:0:16}..."

    # Check current assigner balance — only send if below threshold
    local assigner_balance
    assigner_balance=$(curl -s "http://198.18.0.82:7172/v1/client/get/balance?client_id=$box_wallet_id" -m 5 2>/dev/null \
        | python3 -c 'import json,sys; print(json.load(sys.stdin).get("balance",0))' 2>/dev/null || echo "0")
    if [ "${assigner_balance:-0}" -lt 50000000000 ]; then  # < 5 ZCN
        # Use faucet to top up the SC owner wallet, then send to assigner
        # SC owner (ZCN_WALLET_FILE) may have low balance on long-running chains
        sync_wallet_nonce_from_chain "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
        print_status "Topping up SC owner wallet via faucet..."
        $ZWALLET faucet --methodName pour --input '{"pour_amount": 20}' \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 3
        sync_wallet_nonce_from_chain "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
        print_status "Sending 10 ZCN to 0box assigner wallet..."
        $ZWALLET send \
            --to_client_id "$box_wallet_id" \
            --tokens 10 \
            --desc "Fund 0box for free allocations" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 3
    else
        local assigner_balance_zcn; assigner_balance_zcn=$(python3 -c "print(${assigner_balance}/1e10, 'ZCN')" 2>/dev/null || echo "${assigner_balance} SAS")
        print_status "Assigner wallet already funded (balance=${assigner_balance_zcn})"
    fi

    # Get free_storage.name and free_storage.public_key from 0box.yaml.
    # The SC registers assigners by NAME (free_storage.name, e.g. "0chain"), not wallet_id.
    # The marker 0box signs includes "assigner":"<name>" — SC looks up assigner by this name.
    # The public_key registered must match the key 0box uses to sign markers (free_storage.private_key).
    local box_assigner_name=""
    local box_public_key=""
    if [ -f "$BOX_YAML" ]; then
        box_assigner_name=$(grep -A3 '^free_storage:' "$BOX_YAML" | grep '^\s*name:' | head -1 | awk '{print $2}' | tr -d '"')
        box_public_key=$(grep -A5 '^free_storage:' "$BOX_YAML" | grep 'public_key:' | head -1 | awk -F'"' '{print $2}')
    fi

    if [ -z "$box_public_key" ] || [ "$box_public_key" = "null" ]; then
        print_error "FATAL: Could not find free_storage.public_key in 0box.yaml."
        print_error "Free allocations will NOT work. Check 0box.yaml free_storage section."
        exit 1
    fi
    box_assigner_name="${box_assigner_name:-0chain}"
    print_status "Registering assigner name='${box_assigner_name}' key=${box_public_key:0:16}..."

    # Add 0box as free storage assigner using zbox add command
    # --name must match free_storage.name in 0box.yaml (the "assigner" field in markers)
    # Must use SC owner wallet (owner.json) — only owner can call add_free_storage_assigner
    # CRITICAL: This is a single point of cascading failure — without it, ALL free allocations
    # fail with "not enough blobbers available". Retry aggressively.
    local assigner_registered=false
    local max_retries=5
    for attempt in $(seq 1 $max_retries); do
        sync_wallet_nonce_from_chain "${ZCN_CONFIG_DIR}/owner.json"
        print_status "Adding 0box as free storage assigner (attempt $attempt/$max_retries)..."
        local add_output
        add_output=$($ZBOX add \
            --name "$box_assigner_name" \
            --key "$box_public_key" \
            --limit 100 \
            --max 10000 \
            --wallet owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>&1 || true)
        if echo "$add_output" | grep -qi "added as free storage assigner\|already exist\|already registered"; then
            print_status "Free storage assigner registered: ${box_assigner_name}"
            assigner_registered=true
            break
        fi
        print_warning "Attempt $attempt failed: $add_output"
        if [ "$attempt" -lt "$max_retries" ]; then
            sleep 5
        fi
    done

    # zbox add already confirms the transaction on-chain internally before returning success.
    # No secondary SC state check needed — it uses an unreliable internal key format.

    if ! $assigner_registered; then
        print_error "FATAL: Free storage assigner could NOT be registered after $max_retries attempts!"
        print_error "Free allocations will NOT work until this is fixed."
        print_error "Manual fix: $ZBOX add --name $box_assigner_name --key $box_public_key --limit 100 --max 10000 --wallet owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
        exit 1
    fi

    print_status "0box funding complete!"
}

# Register the "0box" user in the owner/wallet tables so that the Render (convert-to-pdf)
# service works for encrypted/private files. The frontend shares encrypted files with the
# "0box" user's pub_encryption_key. Without this user, Render for private files fails with
# "User not found" because getPublicEncryptionKey({ userName: '0box' }) returns nothing.
register_0box_render_user() {
    print_header "Registering 0box Render User"

    # Get 0box server identity from its status page
    local server_client_id server_public_key
    server_client_id=$(curl -s http://localhost:9081/ 2>/dev/null | grep -oP '(?<=id:)[^<]*' | head -1)
    server_public_key=$(curl -s http://localhost:9081/ 2>/dev/null | grep -oP '(?<=public_key:)[^<]*' | head -1)

    if [ -z "$server_client_id" ] || [ -z "$server_public_key" ]; then
        print_warning "Could not get 0box server wallet info — skipping render user registration"
        return
    fi
    print_status "0box server: ${server_client_id:0:16}..."

    # Compute pub_encryption_key (ed25519 from sha3(publicKey+clientID))
    mkdir -p /tmp/genkey
    cat > /tmp/genkey/main.go << 'GENEOF'
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"golang.org/x/crypto/sha3"
)

func main() {
	data := os.Args[1] + os.Args[2]
	hash := sha3.Sum256([]byte(data))
	privKey := ed25519.NewKeyFromSeed(hash[:32])
	pubKey := privKey.Public().(ed25519.PublicKey)
	fmt.Println(base64.StdEncoding.EncodeToString(pubKey))
}
GENEOF
    cd /tmp/genkey
    go mod init genkey 2>/dev/null
    go get golang.org/x/crypto/sha3 2>/dev/null
    local pub_enc_key
    pub_enc_key=$(go run main.go "$server_public_key" "$server_client_id" 2>/dev/null)
    cd - >/dev/null

    if [ -z "$pub_enc_key" ]; then
        print_warning "Could not compute pub_encryption_key — skipping render user registration"
        return
    fi
    print_status "Encryption key: ${pub_enc_key:0:20}..."

    local PG="docker exec ${ZBOX_PG_CONTAINER:-postgres-0box} psql -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox}"

    # Create "0box" owner (idempotent — rename 0box-server if transcoder branch created it)
    $PG -c "UPDATE owner SET username = '0box' WHERE username = '0box-server' AND user_id = '$server_client_id';" 2>/dev/null
    $PG -c "INSERT INTO owner (username, user_id, app_types, created_at, updated_at) \
            VALUES ('0box', '$server_client_id', '{vult,blimp,bolt}', NOW(), NOW()) \
            ON CONFLICT DO NOTHING;" 2>/dev/null

    local owner_id
    owner_id=$($PG -t -c "SELECT id FROM owner WHERE username = '0box';" | tr -d ' ')
    if [ -z "$owner_id" ]; then
        print_warning "Failed to create 0box owner — render for private files may not work"
        return
    fi

    # Create wallet with pub_encryption_key
    $PG -c "INSERT INTO wallet (owner_id, client_id, public_key, public_encryption_key, updated_at) \
            VALUES ($owner_id, '$server_client_id', '$server_public_key', '$pub_enc_key', NOW()) \
            ON CONFLICT (client_id) DO UPDATE SET public_encryption_key = '$pub_enc_key', owner_id = $owner_id, updated_at = NOW();" 2>/dev/null

    # Verify
    local verify
    verify=$($PG -t -c "SELECT o.username FROM owner o JOIN wallet w ON w.client_id = o.user_id WHERE o.username = '0box' AND w.public_encryption_key IS NOT NULL;" | tr -d ' ')
    if [ "$verify" = "0box" ]; then
        print_status "0box render user registered successfully (owner_id=$owner_id)"
    else
        print_warning "0box render user verification failed — check owner/wallet tables"
    fi
}

# Configure 0box public key in blobber config.
# Blobbers use 0box.public_key to verify Zbox-Signature headers on auth ticket requests.
# Without this, blobber's Authenticate0Box middleware rejects all requests with "Invalid signature".
# The public key is extracted from 0box's wallet.json after 0box starts.
configure_0box_public_key_in_blobbers() {
    print_header "Configuring 0box Public Key in Blobber Config"

    # Get 0box public key from container, fall back to SC owner wallet
    local box_public_key=""
    box_public_key=$(docker exec 0box cat /0box/config/wallet.json 2>/dev/null | jq -r '.client_key' 2>/dev/null || true)

    if [ -z "$box_public_key" ] || [ "$box_public_key" = "null" ]; then
        box_public_key=$(docker logs 0box 2>&1 | grep -o "public_key: [a-f0-9]*" | head -1 | awk '{print $NF}')
    fi
    # Fall back to SC owner wallet public key (any valid key works)
    if [ -z "$box_public_key" ] || [ "$box_public_key" = "null" ]; then
        box_public_key=$(jq -r '.keys[0].public_key // .client_key' "$ZCN_WALLET_FILE" 2>/dev/null || true)
        if [ -n "$box_public_key" ] && [ "$box_public_key" != "null" ]; then
            print_status "Using SC owner wallet public key as 0box key"
        fi
    fi

    if [ -z "$box_public_key" ] || [ "$box_public_key" = "null" ]; then
        print_warning "Could not find any public key for blobber 0box config."
        return 0
    fi

    print_status "0box public key: ${box_public_key:0:32}..."

    # Add to regular blobber config
    local BLOBBER_CONFIG="${BASE_DIR}/blobber/config/0chain_blobber.yaml"
    if [ -f "$BLOBBER_CONFIG" ]; then
        # Remove existing 0box block if present
        python3 << PYEOF
import re

config_file = "$BLOBBER_CONFIG"
with open(config_file, 'r') as f:
    content = f.read()

# Remove existing "0box": block (quoted key)
content = re.sub(r'^"0box":\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)
# Remove existing 0box: block (unquoted key)
content = re.sub(r'^0box:\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)

# Append 0box config
obox_block = '''"0box":
  public_key: "${box_public_key}"
'''
content = content.rstrip() + '\n\n' + obox_block

with open(config_file, 'w') as f:
    f.write(content)
print("Updated blobber config with 0box.public_key")
PYEOF
        print_status "Added 0box.public_key to regular blobber config"
    fi

    # Add to enterprise blobber config
    local EBLOBBER_CONFIG="${BASE_DIR}/eblobber/config/0chain_blobber.yaml"
    if [ -f "$EBLOBBER_CONFIG" ]; then
        python3 << PYEOF
import re

config_file = "$EBLOBBER_CONFIG"
with open(config_file, 'r') as f:
    content = f.read()

content = re.sub(r'^"0box":\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)
content = re.sub(r'^0box:\n(?:  [^\n]*\n)*', '', content, flags=re.MULTILINE)

obox_block = '''"0box":
  public_key: "${box_public_key}"
'''
content = content.rstrip() + '\n\n' + obox_block

with open(config_file, 'w') as f:
    f.write(content)
print("Updated eblobber config with 0box.public_key")
PYEOF
        print_status "Added 0box.public_key to enterprise blobber config"
    fi

    # Restart blobbers to pick up the new config
    print_status "Restarting blobbers to load 0box.public_key..."
    for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
        docker restart "blobber-$i" 2>/dev/null || true
    done
    for i in 1 2 3; do
        docker restart "eblobber-$i" 2>/dev/null || true
    done
    sleep 5

    print_status "0box public key configured in all blobbers!"
}

# Seed 0box provider tables from sharder events_db for Explorer
# Provider data only arrives via Kafka health check events (~hourly). This function
# bootstraps the tables immediately by copying from the sharder's events_db.
seed_0box_providers() {
    print_header "Seeding 0box Provider Tables"

    local ZBOX_PG=${ZBOX_PG_CONTAINER:-postgres-0box}
    local ZBOX_USER=${ZBOX_DB_USER:-zbox_user}
    local ZBOX_PASS=${ZBOX_DB_PASS:-zbox_server}
    local ZBOX_DB=${ZBOX_DB_NAME:-zbox}
    local SHARDER_PG="sharder-postgres-2"
    local SHARDER_USER="zchain_user"
    local SHARDER_PASS="zchian"
    local SHARDER_DB="events_db"

    # Helper to run SQL on 0box postgres
    run_0box_sql() {
        docker exec -e PGPASSWORD="$ZBOX_PASS" "$ZBOX_PG" psql -U "$ZBOX_USER" -d "$ZBOX_DB" -t -c "$1" 2>/dev/null
    }

    # Helper to run SQL on sharder postgres
    run_sharder_sql() {
        docker exec -e PGPASSWORD="$SHARDER_PASS" "$SHARDER_PG" psql -U "$SHARDER_USER" -d "$SHARDER_DB" -t -c "$1" 2>/dev/null
    }

    # Ensure provider_brand table has "Zus" entry (required for blobber JOIN in allocation queries)
    run_0box_sql "INSERT INTO provider_brand (brand_name) VALUES ('Zus') ON CONFLICT DO NOTHING;" 2>/dev/null || true
    local zus_brand_id
    zus_brand_id=$(run_0box_sql "SELECT id FROM provider_brand WHERE brand_name='Zus' LIMIT 1;" | tr -d '[:space:]')

    # Get current epoch for last_health_check
    local now_epoch=$(date +%s)

    # Seed blobbers from events_db
    local blobber_count=0
    while IFS='|' read -r bid url capacity allocated offers_total write_price read_price delegate_wallet is_enterprise; do
        bid=$(echo "$bid" | tr -d ' ')
        url=$(echo "$url" | tr -d ' ')
        [ -z "$bid" ] && continue
        capacity=$(echo "$capacity" | tr -d ' ')
        allocated=$(echo "$allocated" | tr -d ' ')
        offers_total=$(echo "$offers_total" | tr -d ' ')
        write_price=$(echo "$write_price" | tr -d ' ')
        read_price=$(echo "$read_price" | tr -d ' ')
        delegate_wallet=$(echo "$delegate_wallet" | tr -d ' ')
        is_enterprise=$(echo "$is_enterprise" | tr -d ' ')

        local btype='HotMinus'
        local is_ent_bool='false'
        [ "$is_enterprise" = "t" ] && btype='HotPlus' && is_ent_bool='true'

        run_0box_sql "INSERT INTO blobbers (id, base_url, capacity, allocated, offers_total, write_price, read_price, delegate_wallet, last_health_check, not_available, is_killed, is_shutdown, brand_id, blobber_type, is_enterprise, created_at, updated_at)
            VALUES ('$bid', '$url', ${capacity:-0}, ${allocated:-0}, ${offers_total:-0}, ${write_price:-0}, ${read_price:-0}, '$delegate_wallet', $now_epoch, false, false, false, ${zus_brand_id:-1}, '$btype', $is_ent_bool, to_timestamp($now_epoch), to_timestamp($now_epoch))
            ON CONFLICT (id) DO UPDATE SET base_url='$url', offers_total=COALESCE(EXCLUDED.offers_total,0), last_health_check=$now_epoch, not_available=false, brand_id=${zus_brand_id:-1}, updated_at=to_timestamp($now_epoch);" 2>/dev/null
        blobber_count=$((blobber_count + 1))
    done < <(run_sharder_sql "SELECT id, base_url, capacity, allocated, offers_total, write_price, read_price, delegate_wallet, is_enterprise FROM blobbers WHERE is_killed=false AND is_shutdown=false ORDER BY id;")
    print_status "Seeded $blobber_count blobbers"

    # Ensure challenge columns have DEFAULT 0 NOT NULL — older 0box schema may have them as nullable,
    # causing NULL+integer = NULL in Kafka aggregation which makes challenge counts silently stay at 0.
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN challenges_passed SET DEFAULT 0;" 2>/dev/null || true
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN challenges_passed SET NOT NULL;" 2>/dev/null || true
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN challenges_completed SET DEFAULT 0;" 2>/dev/null || true
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN challenges_completed SET NOT NULL;" 2>/dev/null || true
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN open_challenges SET DEFAULT 0;" 2>/dev/null || true
    run_0box_sql "ALTER TABLE blobbers ALTER COLUMN open_challenges SET NOT NULL;" 2>/dev/null || true
    # Fill any existing NULL rows
    run_0box_sql "UPDATE blobbers SET challenges_passed=0 WHERE challenges_passed IS NULL;" 2>/dev/null || true
    run_0box_sql "UPDATE blobbers SET challenges_completed=0 WHERE challenges_completed IS NULL;" 2>/dev/null || true
    run_0box_sql "UPDATE blobbers SET open_challenges=0 WHERE open_challenges IS NULL;" 2>/dev/null || true

    # Seed miners from events_db
    # Note: events_db miners uses n2n_host+port (no base_url); 0box miners table uses n2n_host+host+port
    local miner_count=0
    while IFS='|' read -r mid n2n_host mport delegate_wallet service_charge num_delegates; do
        mid=$(echo "$mid" | tr -d '[:space:]')
        n2n_host=$(echo "$n2n_host" | tr -d '[:space:]')
        mport=$(echo "$mport" | tr -d '[:space:]')
        [ -z "$mid" ] && continue
        delegate_wallet=$(echo "$delegate_wallet" | tr -d '[:space:]')
        service_charge=$(echo "$service_charge" | tr -d '[:space:]')
        num_delegates=$(echo "$num_delegates" | tr -d '[:space:]')

        run_0box_sql "INSERT INTO miners (id, n2n_host, host, port, delegate_wallet, service_charge, num_delegates, last_health_check, is_killed, is_shutdown, created_at, updated_at)
            VALUES ('$mid', '$n2n_host', '$n2n_host', ${mport:-0}, '$delegate_wallet', ${service_charge:-0}, ${num_delegates:-0}, $now_epoch, false, false, to_timestamp($now_epoch), to_timestamp($now_epoch))
            ON CONFLICT (id) DO UPDATE SET n2n_host='$n2n_host', host='$n2n_host', port=${mport:-0}, last_health_check=$now_epoch, updated_at=to_timestamp($now_epoch);" 2>/dev/null
        miner_count=$((miner_count + 1))
    done < <(run_sharder_sql "SELECT id, n2n_host, port, delegate_wallet, service_charge, num_delegates FROM miners WHERE is_killed=false AND is_shutdown=false ORDER BY id;")
    print_status "Seeded $miner_count miners"

    # Seed sharders from events_db
    # Note: events_db sharders uses n2n_host+port (no base_url); 0box sharders table uses n2n_host+host+port
    local sharder_count=0
    while IFS='|' read -r sid n2n_host sport delegate_wallet service_charge num_delegates; do
        sid=$(echo "$sid" | tr -d '[:space:]')
        n2n_host=$(echo "$n2n_host" | tr -d '[:space:]')
        sport=$(echo "$sport" | tr -d '[:space:]')
        [ -z "$sid" ] && continue
        delegate_wallet=$(echo "$delegate_wallet" | tr -d '[:space:]')
        service_charge=$(echo "$service_charge" | tr -d '[:space:]')
        num_delegates=$(echo "$num_delegates" | tr -d '[:space:]')

        run_0box_sql "INSERT INTO sharders (id, n2n_host, host, port, delegate_wallet, service_charge, num_delegates, last_health_check, is_killed, is_shutdown, created_at, updated_at)
            VALUES ('$sid', '$n2n_host', '$n2n_host', ${sport:-0}, '$delegate_wallet', ${service_charge:-0}, ${num_delegates:-0}, $now_epoch, false, false, to_timestamp($now_epoch), to_timestamp($now_epoch))
            ON CONFLICT (id) DO UPDATE SET n2n_host='$n2n_host', host='$n2n_host', port=${sport:-0}, last_health_check=$now_epoch, updated_at=to_timestamp($now_epoch);" 2>/dev/null
        sharder_count=$((sharder_count + 1))
    done < <(run_sharder_sql "SELECT id, n2n_host, port, delegate_wallet, service_charge, num_delegates FROM sharders WHERE is_killed=false AND is_shutdown=false ORDER BY id;")
    print_status "Seeded $sharder_count sharders"

    # Update sharder n2n_host to public nginx domain URL.
    # Atlus Explorer fetches last_finalized_round via: n2n_host + '/sharder01/v1/sharder/get/stats'
    # Internal IPs (198.18.0.81/82) are unreachable from browsers; public nginx domain routes correctly.
    local public_domain="${NGINX_DOMAIN:-}"
    if [ -n "$public_domain" ]; then
        local public_url="https://${public_domain}"
        run_0box_sql "UPDATE sharders SET n2n_host='$public_url', updated_at=to_timestamp($now_epoch) WHERE is_killed=false AND is_shutdown=false;" 2>/dev/null
        print_status "Updated sharder n2n_host to public URL: $public_url (enables Atlus last_finalized_round)"
    fi

    # Seed validators from events_db
    local validator_count=0
    while IFS='|' read -r vid url delegate_wallet service_charge num_delegates total_stake; do
        vid=$(echo "$vid" | tr -d ' ')
        url=$(echo "$url" | tr -d ' ')
        [ -z "$vid" ] && continue
        delegate_wallet=$(echo "$delegate_wallet" | tr -d ' ')
        service_charge=$(echo "$service_charge" | tr -d ' ')
        num_delegates=$(echo "$num_delegates" | tr -d ' ')
        total_stake=$(echo "$total_stake" | tr -d ' ')

        run_0box_sql "INSERT INTO validators (id, base_url, delegate_wallet, service_charge, num_delegates, total_stake, last_health_check, is_killed, is_shutdown, created_at, updated_at)
            VALUES ('$vid', '$url', '$delegate_wallet', ${service_charge:-0}, ${num_delegates:-0}, ${total_stake:-0}, $now_epoch, false, false, to_timestamp($now_epoch), to_timestamp($now_epoch))
            ON CONFLICT (id) DO UPDATE SET base_url='$url', total_stake=${total_stake:-0}, last_health_check=$now_epoch, updated_at=to_timestamp($now_epoch);" 2>/dev/null
        validator_count=$((validator_count + 1))
    done < <(run_sharder_sql "SELECT id, base_url, delegate_wallet, service_charge, num_delegates, total_stake FROM validators WHERE is_killed=false AND is_shutdown=false ORDER BY id;")
    print_status "Seeded $validator_count validators"

    print_status "Provider seeding complete: $blobber_count blobbers, $miner_count miners, $sharder_count sharders, $validator_count validators"

    # Seed provider_elastic_search_index in Elasticsearch
    # 0box only indexes providers when TagAddOrOverwriteBlobber/Miner/Sharder events come through Kafka.
    # On fresh deploys, those events are in the past and not replayed, so we must seed ES directly.
    local es_url="http://localhost:9200"
    local es_index="provider_elastic_search_index"
    local es_count=0
    docker exec postgres-0box psql -U zbox_user zbox -t -c "SELECT id FROM blobbers WHERE not_available=false;" 2>/dev/null | tr -d '[:space:]' | grep -v '^$' | while read -r bid; do
        curl -s -X PUT "$es_url/$es_index/_doc/$bid" \
          -H "Content-Type: application/json" \
          -d "{\"provider_id\":\"$bid\",\"provider_name\":\"Blobber ${bid:0:8}\",\"provider_type\":\"blobber\"}" -o /dev/null
    done
    docker exec postgres-0box psql -U zbox_user zbox -t -c "SELECT id FROM miners;" 2>/dev/null | tr -d '[:space:]' | grep -v '^$' | while read -r mid; do
        curl -s -X PUT "$es_url/$es_index/_doc/$mid" \
          -H "Content-Type: application/json" \
          -d "{\"provider_id\":\"$mid\",\"provider_name\":\"Miner ${mid:0:8}\",\"provider_type\":\"miner\"}" -o /dev/null
    done
    docker exec postgres-0box psql -U zbox_user zbox -t -c "SELECT id FROM sharders;" 2>/dev/null | tr -d '[:space:]' | grep -v '^$' | while read -r sid; do
        curl -s -X PUT "$es_url/$es_index/_doc/$sid" \
          -H "Content-Type: application/json" \
          -d "{\"provider_id\":\"$sid\",\"provider_name\":\"Sharder ${sid:0:8}\",\"provider_type\":\"sharder\"}" -o /dev/null
    done
    local indexed
    indexed=$(curl -s "$es_url/$es_index/_count" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('count',0))" 2>/dev/null || echo "?")
    print_status "Seeded provider_elastic_search_index: $indexed docs"

    # Update 0box latest snapshot with correct provider counts and capacity.
    # The initial seeded snapshot has zeros for blobber_count/max_capacity_storage because
    # TagAddOrOverwriteBlobber events happened before 0box started processing Kafka events.
    local snap_blobber_cnt snap_miner_cnt snap_sharder_cnt snap_cap
    snap_blobber_cnt=$(run_sharder_sql "SELECT COUNT(*) FROM blobbers WHERE is_killed=false AND is_shutdown=false AND is_enterprise=false;" | tr -d '[:space:]')
    snap_miner_cnt=$(run_sharder_sql "SELECT COUNT(*) FROM miners WHERE is_killed=false AND is_shutdown=false;" | tr -d '[:space:]')
    snap_sharder_cnt=$(run_sharder_sql "SELECT COUNT(*) FROM sharders WHERE is_killed=false AND is_shutdown=false;" | tr -d '[:space:]')
    snap_cap=$(run_sharder_sql "SELECT COALESCE(SUM(capacity),0) FROM blobbers WHERE is_killed=false AND is_shutdown=false AND is_enterprise=false;" | tr -d '[:space:]')
    run_0box_sql "UPDATE snapshots SET max_capacity_storage = ${snap_cap:-0}, blobber_count = ${snap_blobber_cnt:-0}, miner_count = ${snap_miner_cnt:-0}, sharder_count = ${snap_sharder_cnt:-0} WHERE round = (SELECT MAX(round) FROM snapshots);" 2>/dev/null
    print_status "Updated latest snapshot: ${snap_blobber_cnt} blobbers, ${snap_cap} total capacity"

    # Set active=true for miners/sharders with recent health checks.
    # The 'active' column is only set by full miner/sharder update events (TagUpdateMiner),
    # NOT by TagMinerHealthCheck events. When 0box is seeded from events_db, the initial
    # rows are inserted without active=true. Atlus queries with active=true filter → blank.
    # Fix: set active=true for any miner/sharder that has a last_health_check timestamp.
    local NOW
    NOW=$(date +%s)
    run_0box_sql "UPDATE miners SET active = true WHERE last_health_check IS NOT NULL AND last_health_check > 0;" 2>/dev/null
    run_0box_sql "UPDATE sharders SET active = true WHERE last_health_check IS NOT NULL AND last_health_check > 0;" 2>/dev/null
    local active_miners active_sharders
    active_miners=$(run_0box_sql "SELECT COUNT(*) FROM miners WHERE active = true;" | tr -d '[:space:]')
    active_sharders=$(run_0box_sql "SELECT COUNT(*) FROM sharders WHERE active = true;" | tr -d '[:space:]')
    print_status "Set active=true: ${active_miners:-0} miners, ${active_sharders:-0} sharders"

    # Fix 0box graph endpoints with DataPoints=1 (no from/to params).
    # 0box graph queries with data-points=1 generate SQL that looks for round=0.
    # If no snapshot exists at round=0, ALL graph endpoints return [0].
    # Fix: create a trigger that copies each new snapshot to round=0, keeping it in sync.
    run_0box_sql "
        CREATE OR REPLACE FUNCTION sync_snapshot_round_zero() RETURNS TRIGGER AS \$\$
        BEGIN
            INSERT INTO snapshots (round, total_mint, total_challenge_pools, active_allocated_delta,
                zcn_supply, total_read_pool_locked, client_locks, total_staked, storage_token_stake,
                total_rewards, successful_challenges, total_challenges, allocated_storage,
                max_capacity_storage, staked_storage, used_storage, transactions_count,
                unique_addresses, block_count, total_txn_fee, created_at, blobber_count,
                miner_count, sharder_count, validator_count, authorizer_count,
                miner_total_rewards, sharder_total_rewards, blobber_total_rewards, total_allocations)
            VALUES (0, NEW.total_mint, NEW.total_challenge_pools, NEW.active_allocated_delta,
                NEW.zcn_supply, NEW.total_read_pool_locked, NEW.client_locks, NEW.total_staked,
                NEW.storage_token_stake, NEW.total_rewards, NEW.successful_challenges,
                NEW.total_challenges, NEW.allocated_storage, NEW.max_capacity_storage,
                NEW.staked_storage, NEW.used_storage, NEW.transactions_count, NEW.unique_addresses,
                NEW.block_count, NEW.total_txn_fee, NEW.created_at, NEW.blobber_count,
                NEW.miner_count, NEW.sharder_count, NEW.validator_count, NEW.authorizer_count,
                NEW.miner_total_rewards, NEW.sharder_total_rewards, NEW.blobber_total_rewards,
                NEW.total_allocations)
            ON CONFLICT (round) DO UPDATE SET
                total_mint=EXCLUDED.total_mint, total_challenge_pools=EXCLUDED.total_challenge_pools,
                active_allocated_delta=EXCLUDED.active_allocated_delta, zcn_supply=EXCLUDED.zcn_supply,
                total_read_pool_locked=EXCLUDED.total_read_pool_locked, client_locks=EXCLUDED.client_locks,
                total_staked=EXCLUDED.total_staked, storage_token_stake=EXCLUDED.storage_token_stake,
                total_rewards=EXCLUDED.total_rewards, successful_challenges=EXCLUDED.successful_challenges,
                total_challenges=EXCLUDED.total_challenges, allocated_storage=EXCLUDED.allocated_storage,
                max_capacity_storage=EXCLUDED.max_capacity_storage, staked_storage=EXCLUDED.staked_storage,
                used_storage=EXCLUDED.used_storage, transactions_count=EXCLUDED.transactions_count,
                unique_addresses=EXCLUDED.unique_addresses, block_count=EXCLUDED.block_count,
                total_txn_fee=EXCLUDED.total_txn_fee, created_at=EXCLUDED.created_at,
                blobber_count=EXCLUDED.blobber_count, miner_count=EXCLUDED.miner_count,
                sharder_count=EXCLUDED.sharder_count, validator_count=EXCLUDED.validator_count,
                authorizer_count=EXCLUDED.authorizer_count, miner_total_rewards=EXCLUDED.miner_total_rewards,
                sharder_total_rewards=EXCLUDED.sharder_total_rewards,
                blobber_total_rewards=EXCLUDED.blobber_total_rewards,
                total_allocations=EXCLUDED.total_allocations;
            RETURN NEW;
        END;
        \$\$ LANGUAGE plpgsql;
        DROP TRIGGER IF EXISTS trg_sync_round_zero ON snapshots;
        CREATE TRIGGER trg_sync_round_zero AFTER INSERT ON snapshots
            FOR EACH ROW EXECUTE FUNCTION sync_snapshot_round_zero();
    " 2>/dev/null || true

    # Seed round=0 from current latest snapshot
    run_0box_sql "
        INSERT INTO snapshots (round, total_mint, total_challenge_pools, active_allocated_delta,
            zcn_supply, total_read_pool_locked, client_locks, total_staked, storage_token_stake,
            total_rewards, successful_challenges, total_challenges, allocated_storage,
            max_capacity_storage, staked_storage, used_storage, transactions_count,
            unique_addresses, block_count, total_txn_fee, created_at, blobber_count,
            miner_count, sharder_count, validator_count, authorizer_count,
            miner_total_rewards, sharder_total_rewards, blobber_total_rewards, total_allocations)
        SELECT 0, total_mint, total_challenge_pools, active_allocated_delta,
            zcn_supply, total_read_pool_locked, client_locks, total_staked, storage_token_stake,
            total_rewards, successful_challenges, total_challenges, allocated_storage,
            max_capacity_storage, staked_storage, used_storage, transactions_count,
            unique_addresses, block_count, total_txn_fee, created_at, blobber_count,
            miner_count, sharder_count, validator_count, authorizer_count,
            miner_total_rewards, sharder_total_rewards, blobber_total_rewards, total_allocations
        FROM snapshots WHERE round = (SELECT MAX(round) FROM snapshots WHERE round > 0)
        ON CONFLICT (round) DO UPDATE SET
            total_mint=EXCLUDED.total_mint, total_challenge_pools=EXCLUDED.total_challenge_pools,
            allocated_storage=EXCLUDED.allocated_storage, used_storage=EXCLUDED.used_storage,
            successful_challenges=EXCLUDED.successful_challenges, total_challenges=EXCLUDED.total_challenges,
            blobber_count=EXCLUDED.blobber_count, miner_count=EXCLUDED.miner_count,
            sharder_count=EXCLUDED.sharder_count, max_capacity_storage=EXCLUDED.max_capacity_storage;
    " 2>/dev/null || true
    print_status "Created snapshot round-0 sync trigger (fixes graph endpoints with DataPoints=1)"

    # Fix snapshot created_at timestamps.
    # 0box uses gorm:"autoCreateTime" which sets created_at to INSERT time.
    # After DB truncation + re-processing, all snapshots get the same created_at.
    # Graph endpoints that use created_at for time-based bucketing return zeros.
    # Fix: spread created_at based on round number (3 seconds per round approximation).
    local max_round
    max_round=$(run_0box_sql "SELECT COALESCE(MAX(round),0) FROM snapshots WHERE round > 0;" | tr -d '[:space:]')
    if [ "${max_round:-0}" -gt 0 ]; then
        local now_ts
        now_ts=$(date +%s)
        run_0box_sql "UPDATE snapshots SET created_at = $now_ts - (($max_round - round) * 3) WHERE round > 0 AND created_at = (SELECT created_at FROM snapshots WHERE round > 0 ORDER BY round DESC LIMIT 1);" 2>/dev/null || true
        print_status "Spread snapshot created_at timestamps across ${max_round} rounds"
    fi

}

# Reset 0box to current chain round — clears stale data, resets Kafka offset to latest,
# sets triggerRound in config, restarts services, and re-seeds provider tables.
# Use this after a chain restart when 0box has stale data from a previous chain.
# Usage: bash scripts/deploy_local.sh reset-0box
reset_0box_data() {
    print_header "Reset 0box Data to Current Chain Round"

    # Step 1: Stop 0box (required before resetting Kafka consumer group)
    print_status "Stopping 0box..."
    docker stop 0box 2>/dev/null || docker stop 0box-0box-1 2>/dev/null || true

    # Wait for Kafka consumer group to become inactive (required for offset reset)
    # Session timeout is 45s by default, so we wait up to 90s
    print_status "Waiting for Kafka consumer group 'events-consumer' to become Empty..."
    local kafka_wait=0
    while [ $kafka_wait -lt 90 ]; do
        local cg_state
        cg_state=$(docker exec kafka bash -c '
cat > /tmp/sasl.props << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";
EOF
/opt/kafka/bin/kafka-consumer-groups.sh \
    --bootstrap-server 198.19.0.99:9092 \
    --command-config /tmp/sasl.props \
    --describe --group events-consumer --state 2>/dev/null' 2>/dev/null || true)
        if echo "$cg_state" | grep -qE 'Empty|Dead|#MEMBERS.*0'; then
            print_status "Consumer group is Empty — proceeding with offset reset"
            break
        fi
        sleep 5
        kafka_wait=$((kafka_wait + 5))
        if [ $kafka_wait -ge 90 ]; then
            print_warning "Consumer group did not become Empty after 90s — offset reset may fail"
        fi
    done

    # Step 2: Truncate ALL 0box database tables (except schema version tracking)
    print_status "Truncating 0box database tables..."
    if docker ps --format '{{.Names}}' | grep -q "postgres-0box"; then
        local all_tables
        all_tables=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT string_agg('\"' || tablename || '\"', ', ')
             FROM pg_tables
             WHERE schemaname = 'public'
               AND tablename NOT IN ('goose_db_version');" 2>/dev/null | tr -d ' \n')
        if [ -n "$all_tables" ]; then
            docker exec postgres-0box psql -U zbox_user -d zbox -c \
                "TRUNCATE TABLE ${all_tables} CASCADE;" 2>/dev/null \
                && print_status "0box database truncated" \
                || print_warning "Truncate had issues (non-critical)"
        else
            print_warning "No tables found in 0box database"
        fi
    else
        print_warning "postgres-0box not running — starting postgres first..."
        cd "${BASE_DIR}/0box/docker.local"
        docker compose -p 0box up -d postgres
        sleep 5
    fi

    # Step 3: Reset Kafka consumer group offset to LATEST (skip all old chain events)
    # This prevents 0box from replaying 192k rounds of old chain data
    print_status "Resetting Kafka consumer group 'events-consumer' to latest..."
    docker exec kafka bash -c '
cat > /tmp/sasl.props << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";
EOF
/opt/kafka/bin/kafka-consumer-groups.sh \
    --bootstrap-server 198.19.0.99:9092 \
    --command-config /tmp/sasl.props \
    --group events-consumer \
    --topic events \
    --reset-offsets --to-latest --execute' 2>/dev/null \
        && print_status "Kafka consumer group reset to latest" \
        || print_warning "Kafka offset reset failed (non-critical)"

    # CRITICAL: Query chain round AFTER Kafka reset (not before), so the bootstrap round
    # is aligned with what Kafka will actually deliver next. Querying before the 90s wait
    # means the chain advances ~300-700 rounds between the query and the bootstrap insert,
    # creating a permanent gap where 0box expects round N+1 but Kafka delivers N+666.
    local reset_round
    reset_round=$(curl -s http://198.18.0.82:7172/v1/chain/get/stats -m 5 2>/dev/null \
        | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("current_round", d.get("latest_finalized_round", 0)))' 2>/dev/null || echo "0")
    reset_round=${reset_round:-0}
    if [ "$reset_round" -eq 0 ]; then
        print_error "Could not get current chain round — aborting"
        return 1
    fi
    print_status "Current chain round (post-Kafka-reset): $reset_round"

    # Step 4: Set 0box triggerRound in config to current chain round
    local box_cfg="${BASE_DIR}/0box/docker.local/config/0box.yaml"
    if [ -f "$box_cfg" ]; then
        sed -i "s/triggerRound: [0-9]*/triggerRound: ${reset_round}/" "$box_cfg" \
            && print_status "Updated 0box.yaml triggerRound to $reset_round" \
            || print_warning "Could not update triggerRound in 0box.yaml"
    fi

    # Step 5: Insert a bootstrap snapshot row so 0box reads last_processed_round=reset_round on startup.
    # Without this: last_processed_round=0, and 0box skips all rounds because round != 0+1.
    print_status "Inserting bootstrap snapshot at round $reset_round..."
    local now_epoch
    now_epoch=$(date +%s)
    docker exec postgres-0box psql -U zbox_user -d zbox -c "
        INSERT INTO snapshots (
          round, total_mint, total_challenge_pools, active_allocated_delta,
          zcn_supply, total_read_pool_locked, client_locks, total_staked,
          storage_token_stake, total_rewards, successful_challenges, total_challenges,
          allocated_storage, max_capacity_storage, staked_storage, used_storage,
          transactions_count, unique_addresses, block_count, total_txn_fee,
          created_at, blobber_count, miner_count, sharder_count, validator_count,
          authorizer_count, miner_total_rewards, sharder_total_rewards,
          blobber_total_rewards, total_allocations
        ) VALUES (
          ${reset_round}, 0,0,0,0,0,0,0,0,0,0,0,0,
          0, 0,0,0,0,0,0,${now_epoch},9,4,2,9,0,0,0,0,0
        ) ON CONFLICT DO NOTHING;" 2>/dev/null \
        && print_status "Bootstrap snapshot inserted at round $reset_round" \
        || print_warning "Could not insert bootstrap snapshot"

    # Fix negative open_challenges in blobbers table — causes uint64 scan error in 0box.
    # The chain can produce open_challenges < 0 (challenges_completed > total_challenges bug).
    # 0box's Go code uses uint64 for this field; reading a negative int64 causes:
    # "converting driver.Value type int64 ("-3") to a uint64: invalid syntax"
    # This prevents 0box from processing ANY events, silently stalling last_processed_round.
    print_status "Clamping negative open_challenges in blobbers table..."
    docker exec postgres-0box psql -U zbox_user -d zbox -c \
        "UPDATE blobbers SET open_challenges = 0 WHERE open_challenges < 0;" 2>/dev/null \
        && print_status "Negative open_challenges clamped to 0" \
        || print_warning "Could not clamp open_challenges (non-critical)"
    # Also install trigger to prevent future negative values (persists across 0box restarts)
    docker exec postgres-0box psql -U zbox_user -d zbox -c "
        CREATE OR REPLACE FUNCTION clamp_open_challenges()
        RETURNS TRIGGER AS \$\$
        BEGIN
            IF NEW.open_challenges IS NOT NULL AND NEW.open_challenges < 0 THEN
                NEW.open_challenges := 0;
            END IF;
            RETURN NEW;
        END;
        \$\$ LANGUAGE plpgsql;
        DROP TRIGGER IF EXISTS trg_clamp_open_challenges ON blobbers;
        CREATE TRIGGER trg_clamp_open_challenges
            BEFORE INSERT OR UPDATE ON blobbers
            FOR EACH ROW EXECUTE FUNCTION clamp_open_challenges();" 2>/dev/null \
        && print_status "DB trigger to clamp open_challenges installed" \
        || print_warning "Could not install open_challenges trigger (non-critical)"

    # Step 6: Restart 0box so it reads last_processed_round from the bootstrap snapshot
    print_status "Restarting 0box service (reads bootstrap round from DB)..."
    cd "${BASE_DIR}/0box/docker.local"
    docker compose -p 0box up -d --force-recreate 0box redis
    sleep 8

    # Step 6b: Truncate zvault and zauth split key tables to prevent stale wallet mismatch.
    # When 0box wallets are truncated but zvault/zauth retain old split keys, the browser
    # can cache a stale wallet that no longer exists in 0box, causing "public key mismatch"
    # and "Network redeployed: Invalid wallet id accessed" errors in Vult.
    print_status "Clearing zvault split_keys and zauth split_wallets..."
    docker exec zvault-postgreszv-1 psql -U zvault_user -d zvault -c \
        "TRUNCATE TABLE split_keys, keys CASCADE;" 2>/dev/null \
        && print_status "zvault split_keys + keys truncated" \
        || print_warning "Could not truncate zvault tables (non-critical)"
    docker exec zauth-postgres-1 psql -U zauth_user -d zauth -c \
        "TRUNCATE TABLE split_wallets CASCADE;" 2>/dev/null \
        && print_status "zauth split_wallets truncated" \
        || print_warning "Could not truncate zauth tables (non-critical)"

    # Step 7: Restart zauth and zvault to clear stale auth tokens
    print_status "Restarting zauth..."
    local ZAUTH_COMPOSE="${BASE_DIR}/zauth-server/docker.local/docker-compose.yml"
    if [ -f "$ZAUTH_COMPOSE" ]; then
        docker compose -p zauth -f "$ZAUTH_COMPOSE" restart 2>/dev/null || true
    else
        docker restart zauth-zauth-1 2>/dev/null || true
    fi
    sleep 3

    print_status "Restarting zvault..."
    local ZVAULT_COMPOSE="${BASE_DIR}/zvault/docker.local/docker-compose.yml"
    if [ -f "$ZVAULT_COMPOSE" ]; then
        docker compose -p zvault -f "$ZVAULT_COMPOSE" restart 2>/dev/null || true
    else
        docker restart zvault-zvault-1 2>/dev/null || true
    fi
    sleep 3

    # Step 8: Re-seed 0box provider tables so Atlus/Explorer shows data immediately
    # (without waiting for first Kafka health check event ~60-90 min from now)
    print_status "Re-seeding 0box provider tables from sharder events_db..."
    seed_0box_providers || print_warning "Provider seeding had issues (non-critical)"

    # Step 9: Seed provider_rewards + challenges + blobber stats from sharder events_db
    # Without this, provider_rewards table starts empty and reward charts show 0 on Atlus.
    # (On a fresh deploy the UPDATE-only path in RewardProviders() has no rows to update.)
    print_status "Seeding provider_rewards + historical data from sharder events_db..."
    seed_0box_historical_data || print_warning "Historical data seeding had issues (non-critical)"

    print_status "reset-0box complete!"
    print_status "  0box now starts from chain round $reset_round"
    print_status "  Provider tables seeded — Atlus/Explorer should show data immediately"
    print_status "  Graph/aggregate endpoints will populate as new blocks arrive"
}

# Workaround: seed 0box historical data (challenges, provider_rewards, blobber stats) from events_db.
# Only needed when 0box was started AFTER the chain (missed historical Kafka events).
# In a correct deploy, 0box starts before the chain and receives all events naturally.
# Usage: bash scripts/deploy_local.sh fix-0box-data
seed_0box_historical_data() {
    print_header "Seeding 0box Historical Data from events_db (workaround)"

    local ZBOX_PG=${ZBOX_PG_CONTAINER:-postgres-0box}
    local ZBOX_USER=${ZBOX_DB_USER:-zbox_user}
    local ZBOX_PASS=${ZBOX_DB_PASS:-zbox_server}
    local ZBOX_DB=${ZBOX_DB_NAME:-zbox}
    local SHARDER_PG="sharder-postgres-1"
    local SHARDER_USER="zchain_user"

    run_0box_psql() {
        docker exec -e PGPASSWORD="$ZBOX_PASS" -i "$ZBOX_PG" psql -U "$ZBOX_USER" -d "$ZBOX_DB" "$@" 2>/dev/null
    }

    # --- Challenges ---
    print_status "  Seeding challenges from events_db..."
    docker exec "$SHARDER_PG" psql -U "$SHARDER_USER" -d events_db --csv -t -c \
        "SELECT challenge_id, allocation_id, blobber_id, validators_id, seed, allocation_root, responded, passed, round_created_at, round_responded, timestamp FROM challenges" \
        > /tmp/zbox_challenges_seed.csv 2>/dev/null || true
    local ch_count
    ch_count=$(wc -l < /tmp/zbox_challenges_seed.csv 2>/dev/null || echo 0)
    if [ "${ch_count:-0}" -gt 0 ]; then
        docker cp /tmp/zbox_challenges_seed.csv "$ZBOX_PG":/tmp/zbox_challenges_seed.csv 2>/dev/null
        run_0box_psql << 'PSQL'
CREATE TEMP TABLE ch_staging (
  challenge_id text, allocation_id text, blobber_id text, validators_id text,
  seed bigint, allocation_root text, responded bigint, passed boolean,
  round_created_at bigint, round_responded bigint, timestamp bigint
);
\copy ch_staging FROM '/tmp/zbox_challenges_seed.csv' CSV
INSERT INTO challenges (challenge_id, allocation_id, blobber_id, validators_id, seed, allocation_root, responded, passed, round_created_at, round_responded, timestamp)
SELECT * FROM ch_staging
ON CONFLICT (challenge_id) DO NOTHING;
PSQL
        local ch_imported
        ch_imported=$(run_0box_psql -t -c "SELECT COUNT(*) FROM challenges;" | tr -d '[:space:]')
        print_status "  Challenges: seeded ${ch_count} rows, 0box now has ${ch_imported:-0}"
    else
        print_warning "  No challenges found in events_db"
    fi

    # --- Provider Rewards ---
    print_status "  Seeding provider_rewards from events_db..."
    docker exec "$SHARDER_PG" psql -U "$SHARDER_USER" -d events_db --csv -t -c \
        "SELECT provider_id, rewards, total_rewards, round_service_charge_last_updated FROM provider_rewards WHERE total_rewards > 0" \
        > /tmp/zbox_provider_rewards_seed.csv 2>/dev/null || true
    local pr_count
    pr_count=$(wc -l < /tmp/zbox_provider_rewards_seed.csv 2>/dev/null || echo 0)
    if [ "${pr_count:-0}" -gt 0 ]; then
        docker cp /tmp/zbox_provider_rewards_seed.csv "$ZBOX_PG":/tmp/zbox_provider_rewards_seed.csv 2>/dev/null
        run_0box_psql << 'PSQL'
CREATE TEMP TABLE pr_staging (
  provider_id text, rewards bigint, total_rewards bigint, round_service_charge_last_updated bigint
);
\copy pr_staging FROM '/tmp/zbox_provider_rewards_seed.csv' CSV
INSERT INTO provider_rewards (provider_id, rewards, total_rewards, round_service_charge_last_updated)
SELECT provider_id, rewards, total_rewards, round_service_charge_last_updated FROM pr_staging
ON CONFLICT (provider_id) DO UPDATE SET
  total_rewards = GREATEST(provider_rewards.total_rewards, EXCLUDED.total_rewards),
  rewards = EXCLUDED.rewards,
  round_service_charge_last_updated = EXCLUDED.round_service_charge_last_updated;
PSQL
        local pr_total
        pr_total=$(run_0box_psql -t -c "SELECT COALESCE(SUM(total_rewards),0) FROM provider_rewards;" | tr -d '[:space:]')
        print_status "  Provider rewards: seeded ${pr_count} rows, total_rewards=${pr_total:-0}"
    else
        print_warning "  No provider_rewards found in events_db"
    fi

    # --- Blobber Usage Stats ---
    print_status "  Updating blobber usage/challenge stats from events_db..."
    docker exec "$SHARDER_PG" psql -U "$SHARDER_USER" -d events_db --csv -t -c \
        "SELECT id, saved_data, challenges_passed, challenges_completed, open_challenges, total_storage_income, total_read_income, total_slashed_stake, total_block_rewards FROM blobbers" \
        > /tmp/zbox_blobbers_stats.csv 2>/dev/null || true
    local bl_count
    bl_count=$(wc -l < /tmp/zbox_blobbers_stats.csv 2>/dev/null || echo 0)
    if [ "${bl_count:-0}" -gt 0 ]; then
        docker cp /tmp/zbox_blobbers_stats.csv "$ZBOX_PG":/tmp/zbox_blobbers_stats.csv 2>/dev/null
        run_0box_psql << 'PSQL'
CREATE TEMP TABLE bl_staging (
  id text, saved_data bigint, challenges_passed bigint, challenges_completed bigint,
  open_challenges bigint, total_storage_income bigint, total_read_income bigint,
  total_slashed_stake bigint, total_block_rewards bigint
);
\copy bl_staging FROM '/tmp/zbox_blobbers_stats.csv' CSV
UPDATE blobbers b SET
  saved_data = bl.saved_data,
  used = bl.saved_data,
  challenges_passed = bl.challenges_passed,
  challenges_completed = bl.challenges_completed,
  open_challenges = GREATEST(0, bl.open_challenges),
  total_storage_income = bl.total_storage_income,
  total_read_income = bl.total_read_income,
  total_slashed_stake = bl.total_slashed_stake,
  total_block_rewards = bl.total_block_rewards
FROM bl_staging bl WHERE b.id = bl.id;
-- Clamp any negative open_challenges (uint64 scan error in 0box Kafka consumer)
UPDATE blobbers SET open_challenges = 0 WHERE open_challenges < 0;
PSQL
        local bl_used
        bl_used=$(run_0box_psql -t -c "SELECT COALESCE(SUM(saved_data),0) FROM blobbers;" | tr -d '[:space:]')
        print_status "  Blobbers updated: ${bl_count} rows, total saved_data=${bl_used:-0}"
    else
        print_warning "  No blobber data found in events_db"
    fi

    fix_snapshot_aggregates
}

# Fix 0box snapshot aggregates that are stuck at 0.
# NOTE: In a correct fresh deploy (redeploy from block 0), this is NOT needed —
# 0box's Kafka consumer reads from offset 0 and accumulates all events naturally.
# This patch is only required when 0box is started/restarted against an existing
# chain (Kafka consumer missed historical challenge/reward/user events).
# events_db has all the data (provider_rewards has per-provider totals) but
# 0box never received those Kafka events due to ordering at deploy time.
fix_snapshot_aggregates() {
    print_header "Fixing 0box Snapshot Aggregates from events_db"

    local ZBOX_PG=${ZBOX_PG_CONTAINER:-postgres-0box}
    local ZBOX_USER=${ZBOX_DB_USER:-zbox_user}
    local ZBOX_PASS=${ZBOX_DB_PASS:-zbox_server}
    local ZBOX_DB=${ZBOX_DB_NAME:-zbox}
    local SHARDER_PG="sharder-postgres-1"
    local SHARDER_USER="zchain_user"
    local SHARDER_PASS="zchian"
    local SHARDER_DB="events_db"

    run_snap_0box_sql() {
        docker exec -e PGPASSWORD="$ZBOX_PASS" "$ZBOX_PG" psql -U "$ZBOX_USER" -d "$ZBOX_DB" -t -c "$1" 2>/dev/null
    }
    run_snap_sharder_sql() {
        docker exec -e PGPASSWORD="$SHARDER_PASS" "$SHARDER_PG" psql -U "$SHARDER_USER" -d "$SHARDER_DB" -t -c "$1" 2>/dev/null
    }

    # Pull real counts from events_db
    local total_challenges successful_challenges unique_addresses total_rewards
    total_challenges=$(run_snap_sharder_sql "SELECT COUNT(*) FROM challenges;" | tr -d '[:space:]')
    successful_challenges=$(run_snap_sharder_sql "SELECT COUNT(*) FROM challenges WHERE passed=true;" | tr -d '[:space:]')
    unique_addresses=$(run_snap_sharder_sql "SELECT COUNT(*) FROM users;" | tr -d '[:space:]')
    total_rewards=$(run_snap_sharder_sql "SELECT COALESCE(SUM(total_rewards),0) FROM provider_rewards;" | tr -d '[:space:]')
    local used_storage
    used_storage=$(run_snap_0box_sql "SELECT COALESCE(SUM(saved_data),0) FROM blobbers;" | tr -d '[:space:]')

    print_status "  events_db: unique_addresses=${unique_addresses}, challenges=${total_challenges}, passed=${successful_challenges}, rewards=${total_rewards}, used_storage=${used_storage}"

    if [ -z "$total_challenges" ] || [ "$total_challenges" = "0" ]; then
        print_warning "  No challenge data found in events_db — skipping snapshot patch"
        return 0
    fi

    # Get per-provider-type reward totals from 0box provider_rewards (joined with provider tables)
    local miner_rewards sharder_rewards blobber_rewards
    miner_rewards=$(run_snap_0box_sql "SELECT COALESCE(SUM(pr.total_rewards),0) FROM provider_rewards pr WHERE pr.provider_id IN (SELECT id FROM miners);" | tr -d '[:space:]')
    sharder_rewards=$(run_snap_0box_sql "SELECT COALESCE(SUM(pr.total_rewards),0) FROM provider_rewards pr WHERE pr.provider_id IN (SELECT id FROM sharders);" | tr -d '[:space:]')
    blobber_rewards=$(run_snap_0box_sql "SELECT COALESCE(SUM(pr.total_rewards),0) FROM provider_rewards pr WHERE pr.provider_id IN (SELECT id FROM blobbers);" | tr -d '[:space:]')

    # Patch the latest snapshot row
    run_snap_0box_sql "UPDATE snapshots SET
        unique_addresses      = ${unique_addresses:-0},
        total_challenges      = ${total_challenges:-0},
        successful_challenges = ${successful_challenges:-0},
        total_rewards         = ${total_rewards:-0},
        miner_total_rewards   = ${miner_rewards:-0},
        sharder_total_rewards = ${sharder_rewards:-0},
        blobber_total_rewards = ${blobber_rewards:-0},
        used_storage          = ${used_storage:-0}
        WHERE round = (SELECT MAX(round) FROM snapshots);" 2>/dev/null \
        && print_status "  Patched latest snapshot row with real aggregates" \
        || print_warning "  Failed to patch snapshot (table may have different schema)"

    # Also update all-zero historical rows so graphs show data
    run_snap_0box_sql "UPDATE snapshots SET
        unique_addresses    = ${unique_addresses:-0},
        total_challenges    = ${total_challenges:-0},
        successful_challenges = ${successful_challenges:-0},
        total_rewards       = ${total_rewards:-0}
        WHERE unique_addresses = 0 AND total_challenges = 0;" 2>/dev/null \
        && print_status "  Patched historical zero-rows" \
        || true
}

# Start blobber postgres containers
start_blobber_postgres() {
    print_header "Starting Blobber PostgreSQL Containers"

    for i in $(seq 1 15); do
        if docker ps -a --format '{{.Names}}' | grep -q "postgres-blob-$i"; then
            docker start "postgres-blob-$i" 2>/dev/null || true
            print_status "Started postgres-blob-$i"
        fi
    done

    # Wait for postgres to be ready
    sleep 5

    # Ensure hdd_tablespace exists in all running blobber postgres containers
    # This is needed because docker-entrypoint-initdb.d scripts only run on first init
    # and a clean deploy may leave postgres data intact but without the tablespace
    for i in $(seq 1 15); do
        local container="postgres-blob-$i"
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            local ts_exists=$(docker exec "$container" psql -U blobber_user -d blobber_meta -tAc "SELECT 1 FROM pg_tablespace WHERE spcname='hdd_tablespace';" 2>/dev/null)
            if [ "$ts_exists" != "1" ]; then
                docker exec "$container" bash -c "rm -rf /var/lib/postgresql/hdd/PG_* && mkdir -p /var/lib/postgresql/hdd && chown postgres:postgres /var/lib/postgresql/hdd" 2>/dev/null
                docker exec "$container" psql -U blobber_user -d blobber_meta -c "CREATE TABLESPACE hdd_tablespace LOCATION '/var/lib/postgresql/hdd';" 2>/dev/null && \
                    print_status "Created hdd_tablespace in $container" || true
            fi
        fi
    done
}

# Start blobbers and validators (15 regular + enterprise)
start_blobbers() {
    print_header "Starting Blobbers and Validators"

    for i in $(seq 1 15); do
        if docker ps -a --format '{{.Names}}' | grep -q "blobber-$i"; then
            docker start "blobber-$i" 2>/dev/null || true
            print_status "Started blobber-$i"
        fi

        if docker ps -a --format '{{.Names}}' | grep -q "validator-$i"; then
            docker start "validator-$i" 2>/dev/null || true
            print_status "Started validator-$i"
        fi
    done
}

# Fund blobbers and validators using actual wallet IDs from running containers
# NOTE: key config files (b0bnode*_keys.txt.json) may be stale/out-of-sync after
# a fresh postgres wipe — blobbers generate new wallets on first start. Always
# read actual IDs from docker logs (same as enterprise blobbers).
fund_blobbers_and_validators() {
    print_header "Funding Blobbers and Validators"

    # Ensure funder wallet has enough ZCN before sending to blobbers/validators.
    # Need at least 12 blobbers × 100 ZCN + 12 validators × 5 ZCN + 5 eblobbers × 100 ZCN = 1760 ZCN
    local funder_id
    funder_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)
    local funder_bal_raw
    funder_bal_raw=$(curl -s "http://198.18.0.81:7171/v1/client/get/balance?client_id=${funder_id}" | \
        python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('balance',0))" 2>/dev/null || echo "0")
    local funder_zcn
    funder_zcn=$(python3 -c "print(int('${funder_bal_raw}') // 10000000000)" 2>/dev/null || echo "0")
    print_status "Funder wallet balance: ${funder_zcn} ZCN (need ~1800 ZCN)"
    if [ "${funder_zcn}" -lt 1800 ] 2>/dev/null; then
        local needed=$(( 1800 - funder_zcn ))
        local pours=$(( (needed + 99) / 100 ))  # faucet pour_limit=100 ZCN per call
        [ "$pours" -lt 2 ] && pours=2
        [ "$pours" -gt 30 ] && pours=30  # cap at 30 pours = 3000 ZCN max
        print_status "Topping up funder wallet via faucet (${pours} pours of 100 ZCN each, need ${needed} more)..."
        for p in $(seq 1 "$pours"); do
            $ZWALLET faucet --methodName pour --tokens 100 --input '{}' \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 2
        done
        funder_bal_raw=$(curl -s "http://198.18.0.81:7171/v1/client/get/balance?client_id=${funder_id}" | \
            python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('balance',0))" 2>/dev/null || echo "0")
        funder_zcn=$(python3 -c "print(int('${funder_bal_raw}') // 10000000000)" 2>/dev/null || echo "0")
        print_status "Funder wallet after faucet: ${funder_zcn} ZCN"
    fi

    # Fund blobbers: get actual wallet IDs from container logs (NOT key files)
    for i in $(seq 1 12); do
        if ! docker ps --format '{{.Names}}' | grep -q "^blobber-${i}$"; then
            continue
        fi
        # Actual wallet ID: try several log formats blobbers use
        local blobber_id
        blobber_id=$(docker logs "blobber-${i}" 2>&1 | \
            grep -oP '(?<=client_id=)[a-f0-9]{64}' | head -1)
        # Fallback: "ID:  <hash>" format (staging blobber binary)
        if [ -z "$blobber_id" ]; then
            blobber_id=$(docker logs "blobber-${i}" 2>&1 | grep 'ID:  ' | head -1 | grep -oP '[a-f0-9]{64}')
        fi
        # Fallback: JSON log format "blobber":"<hash>" (challenge sync logs)
        if [ -z "$blobber_id" ]; then
            blobber_id=$(docker logs "blobber-${i}" 2>&1 | grep -oP '(?<="blobber":")[a-f0-9]{64}' | head -1)
        fi
        # Fallback: compute client_id = sha3-256(public_key) from line 1 of .txt key file
        # NOTE: use .txt (authoritative, updated by build scripts) before .txt.json (may be stale)
        if [ -z "$blobber_id" ]; then
            local key_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt"
            if [ -f "$key_file" ]; then
                local pub_key
                pub_key=$(awk 'NR==1' "$key_file" 2>/dev/null | tr -d '[:space:]')
                [ -n "$pub_key" ] && blobber_id=$(python3 -c \
                    "import sys,hashlib; print(hashlib.sha3_256(bytes.fromhex(sys.argv[1])).hexdigest())" \
                    "$pub_key" 2>/dev/null)
            fi
        fi
        # Last resort: .txt.json wallet file (may have stale keys if .txt was regenerated)
        if [ -z "$blobber_id" ]; then
            local wallet_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt.json"
            [ -f "$wallet_file" ] && blobber_id=$(jq -r '.client_id' "$wallet_file" 2>/dev/null)
        fi
        if [ -n "$blobber_id" ] && [ ${#blobber_id} -ge 60 ]; then
            print_status "Funding blobber-$i: ${blobber_id:0:16}... (100 ZCN)"
            $ZWALLET send \
                --to_client_id "$blobber_id" \
                --tokens 100 \
                --desc "Fund blobber" \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        else
            print_warning "blobber-$i: could not get wallet ID from logs or key files"
        fi
    done

    # Fund validators: same approach — actual IDs from logs, fallback to key files
    for i in $(seq 1 12); do
        if ! docker ps --format '{{.Names}}' | grep -q "^validator-${i}$"; then
            continue
        fi
        local validator_id
        validator_id=$(docker logs "validator-${i}" 2>&1 | \
            grep -oP '(?<=client_id=)[a-f0-9]{64}' | head -1)
        # Fallback: "ID:  <hash>" format
        if [ -z "$validator_id" ]; then
            validator_id=$(docker logs "validator-${i}" 2>&1 | grep 'ID:  ' | head -1 | grep -oP '[a-f0-9]{64}')
        fi
        if [ -z "$validator_id" ]; then
            local wallet_file="${BLOBBER_KEYS_DIR}/b0vnode${i}_keys.txt.json"
            [ -f "$wallet_file" ] && validator_id=$(jq -r '.client_id' "$wallet_file" 2>/dev/null)
        fi
        # Last resort: compute client_id = sha3-256(public_key) from line 1 of .txt key file
        if [ -z "$validator_id" ]; then
            local key_file="${BLOBBER_KEYS_DIR}/b0vnode${i}_keys.txt"
            if [ -f "$key_file" ]; then
                local pub_key
                pub_key=$(awk 'NR==1' "$key_file" 2>/dev/null | tr -d '[:space:]')
                [ -n "$pub_key" ] && validator_id=$(python3 -c \
                    "import sys,hashlib; print(hashlib.sha3_256(bytes.fromhex(sys.argv[1])).hexdigest())" \
                    "$pub_key" 2>/dev/null)
            fi
        fi
        if [ -n "$validator_id" ] && [ ${#validator_id} -ge 60 ]; then
            print_status "Funding validator-$i: ${validator_id:0:16}... (5 ZCN)"
            $ZWALLET send \
                --to_client_id "$validator_id" \
                --tokens 5 \
                --desc "Fund validator" \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        else
            print_warning "validator-$i: could not get wallet ID from logs or key files"
        fi
    done

    # Fund enterprise blobbers (eblobber-1..5).
    # Try to get wallet ID from docker logs first; fall back to the key file inside the container.
    for i in 1 2 3 4 5; do
        if ! docker ps --format '{{.Names}}' | grep -q "^eblobber-${i}$"; then
            continue  # Skip if container is not running
        fi
        local eblobber_id=$(docker logs eblobber-$i 2>&1 | grep "ID:" | head -1 | awk '{print $NF}')
        # Fallback: read client_id from key file inside the container (second line of keys.txt)
        if [ -z "$eblobber_id" ] || [ ${#eblobber_id} -lt 60 ]; then
            eblobber_id=$(docker exec "eblobber-${i}" sed -n '2p' "keysconfig/b0bnode${i}_keys.txt" 2>/dev/null | tr -d '[:space:]')
        fi
        if [ -n "$eblobber_id" ] && [ ${#eblobber_id} -ge 60 ]; then
            print_status "Funding eblobber-$i: ${eblobber_id:0:16}... (100 ZCN)"
            $ZWALLET send \
                --to_client_id "$eblobber_id" \
                --tokens 100 \
                --desc "Fund eblobber $i" \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        else
            print_warning "eblobber-$i: could not get wallet ID from logs or key file"
        fi
    done

    print_status "Blobber and validator funding complete!"
}

# Restart blobbers/validators that failed to register due to insufficient balance at first start.
# Must be called AFTER fund_blobbers_and_validators() so wallets are funded.
restart_failed_blobbers() {
    print_header "Restarting Failed Blobbers/Validators"

    local restarted=0
    for i in $(seq 1 12); do
        if docker ps --format '{{.Names}}' | grep -q "^blobber-${i}$"; then
            # Check if blobber failed to register (insufficient balance or not registered)
            if docker logs "blobber-${i}" 2>&1 | grep -q "insufficient balance to pay fee"; then
                print_status "blobber-$i: failed to register — restarting after funding..."
                docker restart "blobber-${i}" 2>/dev/null || true
                restarted=$((restarted + 1))
            fi
        fi
        if docker ps --format '{{.Names}}' | grep -q "^validator-${i}$"; then
            if docker logs "validator-${i}" 2>&1 | grep -q "insufficient balance to pay fee"; then
                print_status "validator-$i: failed to register — restarting after funding..."
                docker restart "validator-${i}" 2>/dev/null || true
                restarted=$((restarted + 1))
            fi
        fi
    done

    # Also restart enterprise blobbers that failed to register
    for i in 1 2 3 4 5; do
        if docker ps --format '{{.Names}}' | grep -q "^eblobber-${i}$"; then
            if docker logs "eblobber-${i}" 2>&1 | grep -q "insufficient balance\|max retries exceeded"; then
                print_status "eblobber-$i: failed to register — restarting after funding..."
                docker restart "eblobber-${i}" 2>/dev/null || true
                restarted=$((restarted + 1))
            fi
        fi
    done

    if [ "$restarted" -eq 0 ]; then
        print_status "All blobbers/validators registered OK — no restarts needed"
    else
        print_status "Restarted $restarted blobber/validator/eblobber container(s)"
    fi
}

# Check and top up service provider balances (one-shot)
# Funds any blobber/validator/miner/sharder/0box wallet below the minimum threshold
check_and_fund_providers() {
    local MIN_BLOBBER_BALANCE=${1:-5}   # ZCN - minimum before top-up
    local TOP_UP_AMOUNT=${2:-20}         # ZCN - amount to send when low
    local MIN_VALIDATOR_BALANCE=2
    local VALIDATOR_TOP_UP=5
    local sharder_url="http://198.18.0.82:7172"

    print_header "Checking and Funding Service Providers"

    # Helper to get balance in ZCN from client_id
    _get_balance() {
        local cid="$1"
        local bal_raw=$(curl -s --max-time 5 "${sharder_url}/v1/client/get/balance?client_id=${cid}" 2>/dev/null | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("balance",0))' 2>/dev/null || echo "0")
        python3 -c "print(round(${bal_raw}/1e10, 2))" 2>/dev/null || echo "0"
    }

    # Get funder wallet balance first
    local funder_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)
    local funder_bal=$(_get_balance "$funder_id")
    print_status "Funder wallet balance: ${funder_bal} ZCN"

    # If funder balance is low, top up from faucet first
    local funder_low=$(python3 -c "print(1 if float('${funder_bal}') < 50 else 0)" 2>/dev/null || echo "0")
    if [ "$funder_low" = "1" ]; then
        print_status "Funder balance low, pouring from faucet..."
        $ZWALLET faucet --methodName pour --tokens 100 --input "fund providers" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 2
        print_status "Faucet pours complete"
        funder_bal=$(_get_balance "$funder_id")
    fi

    # ===== Collect all balances first, then print summary table =====
    # Arrays to store provider balances for the summary table
    local -a bal_names=()
    local -a bal_values=()
    local -a bal_statuses=()
    local -a fund_queue_ids=()
    local -a fund_queue_amounts=()
    local -a fund_queue_names=()

    # Blobbers (regular 1-12)
    for i in $(seq 1 12); do
        local wallet_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt.json"
        local txt_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt"
        local client_id=""
        if [ -f "$wallet_file" ]; then
            client_id=$(jq -r '.client_id' "$wallet_file")
        elif [ -f "$txt_file" ]; then
            # .txt format: line1=pubkey, line2=client_id, line3=host, line4=port
            client_id=$(sed -n '2p' "$txt_file")
        else
            continue
        fi
        [ -z "$client_id" ] && continue
        local bal=$(_get_balance "$client_id")
        bal_names+=("blobber-${i}")
        bal_values+=("$bal")
        local is_low=$(python3 -c "print(1 if float('${bal}') < ${MIN_BLOBBER_BALANCE} else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$client_id")
            fund_queue_amounts+=("$TOP_UP_AMOUNT")
            fund_queue_names+=("blobber-${i}")
        else
            bal_statuses+=("OK")
        fi
    done

    # Enterprise blobbers — get IDs from chain (is_enterprise=true), not docker logs
    local _eblob_json
    _eblob_json=$(curl -s --max-time 8 "${sharder_url}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers?limit=20" 2>/dev/null)
    local _eblob_ids
    _eblob_ids=$(echo "$_eblob_json" | python3 -c "
import sys,json
d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
for b in nodes:
    if b.get('is_enterprise',False): print(b['id'])
" 2>/dev/null || true)
    local _eblob_idx=1
    for eblobber_id in $_eblob_ids; do
        [ -z "$eblobber_id" ] && continue
        local bal=$(_get_balance "$eblobber_id")
        bal_names+=("eblobber-${_eblob_idx}")
        bal_values+=("$bal")
        local is_low=$(python3 -c "print(1 if float('${bal}') < ${MIN_BLOBBER_BALANCE} else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$eblobber_id")
            fund_queue_amounts+=("$TOP_UP_AMOUNT")
            fund_queue_names+=("eblobber-${_eblob_idx}")
        else
            bal_statuses+=("OK")
        fi
        _eblob_idx=$((_eblob_idx + 1))
    done

    # Validators (1-12)
    for i in $(seq 1 12); do
        local wallet_file="${BLOBBER_KEYS_DIR}/b0vnode${i}_keys.txt.json"
        local txt_file="${BLOBBER_KEYS_DIR}/b0vnode${i}_keys.txt"
        local client_id=""
        if [ -f "$wallet_file" ]; then
            client_id=$(jq -r '.client_id' "$wallet_file")
        elif [ -f "$txt_file" ]; then
            # .txt format: line1=pubkey, line2=client_id, line3=host, line4=port
            client_id=$(sed -n '2p' "$txt_file")
        else
            continue
        fi
        [ -z "$client_id" ] && continue
        local bal=$(_get_balance "$client_id")
        bal_names+=("validator-${i}")
        bal_values+=("$bal")
        local is_low=$(python3 -c "print(1 if float('${bal}') < ${MIN_VALIDATOR_BALANCE} else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$client_id")
            fund_queue_amounts+=("$VALIDATOR_TOP_UP")
            fund_queue_names+=("validator-${i}")
        else
            bal_statuses+=("OK")
        fi
    done

    # Miners — get IDs from SC REST API (logs use JSON format, not "ID: <id>" text)
    local miner_ids_sc=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['simple_miner']['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null || true)
    local miner_idx=1
    for miner_id in $miner_ids_sc; do
        if [ -z "$miner_id" ] || [ "$miner_id" = "null" ]; then continue; fi
        local bal=$(_get_balance "$miner_id")
        bal_names+=("miner-${miner_idx}")
        bal_values+=("$bal")
        # Miners need ZCN to pay for health_check transactions (cost=149, fee≈0.097 ZCN each).
        # Fund with 10 ZCN to cover many health checks (every 90 min = ~16/day = ~1.5 ZCN/day).
        local is_low=$(python3 -c "print(1 if float('${bal}') < 5 else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$miner_id")
            fund_queue_amounts+=("10")
            fund_queue_names+=("miner-${miner_idx}")
        else
            bal_statuses+=("OK")
        fi
        miner_idx=$((miner_idx + 1))
    done

    # Sharders — get IDs from SC REST API (same reason as miners)
    local sharder_ids_sc=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['simple_miner']['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null || true)
    local sharder_idx=1
    for sharder_id in $sharder_ids_sc; do
        if [ -z "$sharder_id" ] || [ "$sharder_id" = "null" ]; then continue; fi
        local bal=$(_get_balance "$sharder_id")
        bal_names+=("sharder-${sharder_idx}")
        bal_values+=("$bal")
        local is_low=$(python3 -c "print(1 if float('${bal}') < 5 else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$sharder_id")
            fund_queue_amounts+=("10")
            fund_queue_names+=("sharder-${sharder_idx}")
        else
            bal_statuses+=("OK")
        fi
        sharder_idx=$((sharder_idx + 1))
    done

    # Crawler wallet — needs ZCN to create and renew allocations.
    # Uses the same wallet as local.json (ZCN_WALLET_FILE).
    local crawler_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null || true)
    if [ -n "$crawler_id" ] && [ "$crawler_id" != "null" ] && [ "$crawler_id" != "$funder_id" ]; then
        local bal=$(_get_balance "$crawler_id")
        bal_names+=("crawler-wallet")
        bal_values+=("$bal")
        local is_low=$(python3 -c "print(1 if float('${bal}') < 15 else 0)" 2>/dev/null || echo "0")
        if [ "$is_low" = "1" ]; then
            bal_statuses+=("LOW")
            fund_queue_ids+=("$crawler_id")
            fund_queue_amounts+=("50")
            fund_queue_names+=("crawler-wallet")
        else
            bal_statuses+=("OK")
        fi
    fi

    # ===== Print Balance Summary Table =====
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}  ${CYAN}Provider Balance Summary${NC}  $(date '+%H:%M:%S')                   ${BLUE}║${NC}"
    echo -e "${BLUE}╠════════════════════╦══════════════╦════════════════╣${NC}"
    echo -e "${BLUE}║${NC} Provider           ${BLUE}║${NC} Balance (ZCN)${BLUE}║${NC} Status         ${BLUE}║${NC}"
    echo -e "${BLUE}╠════════════════════╬══════════════╬════════════════╣${NC}"
    echo -e "${BLUE}║${NC} ${YELLOW}Funder Wallet${NC}      ${BLUE}║${NC} $(printf '%12s' "$funder_bal") ${BLUE}║${NC} ${GREEN}FUNDING SRC${NC}    ${BLUE}║${NC}"
    echo -e "${BLUE}╠════════════════════╬══════════════╬════════════════╣${NC}"
    for idx in "${!bal_names[@]}"; do
        local name="${bal_names[$idx]}"
        local val="${bal_values[$idx]}"
        local status="${bal_statuses[$idx]}"
        local status_color="${GREEN}"
        [ "$status" = "LOW" ] && status_color="${RED}"
        echo -e "${BLUE}║${NC} $(printf '%-18s' "$name") ${BLUE}║${NC} $(printf '%12s' "$val") ${BLUE}║${NC} ${status_color}$(printf '%-14s' "$status")${NC} ${BLUE}║${NC}"
    done
    echo -e "${BLUE}╚════════════════════╩══════════════╩════════════════╝${NC}"
    echo ""

    # ===== Fund providers that are LOW =====
    local funded_count=0
    for idx in "${!fund_queue_ids[@]}"; do
        local cid="${fund_queue_ids[$idx]}"
        local amount="${fund_queue_amounts[$idx]}"
        local name="${fund_queue_names[$idx]}"
        print_warning "${name} balance LOW - funding ${amount} ZCN..."
        $ZWALLET send \
            --to_client_id "$cid" \
            --tokens "$amount" \
            --desc "Auto-fund ${name}" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        funded_count=$((funded_count + 1))
        sleep 1
    done

    if [ "$funded_count" -gt 0 ]; then
        print_status "Funded $funded_count providers"
    else
        print_status "All provider balances healthy"
    fi

    print_status "Provider funding check complete!"

    # ===== Stake on miners and sharders (if not already staked) =====
    # This ensures miners/sharders have non-zero stake for atlus/explorer dashboard.
    # Uses magic block to discover miner/sharder IDs (works on all chain versions,
    # unlike getMinerList/getSharderList which don't exist on the lfb combined-SC branch).
    # Uses "zbox sp-lock" (not "zwallet mn-lock" which is deprecated on lfb branch).
    print_status "Ensuring wallet has enough ZCN for staking..."
    local _w_bal_raw
    _w_bal_raw=$(curl -s --max-time 5 "${sharder_url}/v1/client/get/balance?client_id=$(python3 -c "import json; print(json.load(open('${ZCN_WALLET_FILE}'))['client_id'])" 2>/dev/null)" 2>/dev/null \
        | python3 -c 'import json,sys; print(json.load(sys.stdin).get("balance",0))' 2>/dev/null || echo "0")
    local _w_bal_zcn=$(python3 -c "print(int(${_w_bal_raw:-0}) // 10000000000)" 2>/dev/null || echo "0")
    if [ "${_w_bal_zcn:-0}" -lt 500 ]; then
        print_status "  Wallet balance ~${_w_bal_zcn} ZCN < 500 ZCN — topping up via faucet (5 x 100 ZCN)..."
        for _i in $(seq 1 5); do
            $ZBOX faucet --methodName pour --input '{}' --tokens 100 \
                --wallet ${ZCN_WALLET_FILE} --configDir ${ZCN_CONFIG_DIR} --config ${ZCN_CONFIG_FILE} --silent 2>/dev/null || \
            ${ZWALLET} faucet --methodName pour --input '{}' --tokens 100 \
                --wallet ${ZCN_WALLET_FILE} --configDir ${ZCN_CONFIG_DIR} --config ${ZCN_CONFIG_FILE} --silent 2>/dev/null || true
        done
        sleep 5
    fi

    print_status "Checking miner/sharder stake pools..."
    # Get miner/sharder IDs from the magic block (works on all chain versions).
    local _mb_json
    _mb_json=$(curl -s --max-time 10 "${sharder_url}/v1/block/get/latest_finalized_magic_block" 2>/dev/null)

    local miner_ids
    miner_ids=$(echo "$_mb_json" | python3 -c "
import json,sys
try:
    d=json.load(sys.stdin)
    mb = d.get('magic_block', d)
    for m in mb.get('miners',{}).get('nodes',[]):
        mid = m if isinstance(m, str) else m.get('id','')
        if mid: print(mid)
except: pass
" 2>/dev/null || true)

    local sharder_ids
    sharder_ids=$(echo "$_mb_json" | python3 -c "
import json,sys
try:
    d=json.load(sys.stdin)
    mb = d.get('magic_block', d)
    for s in mb.get('sharders',{}).get('nodes',[]):
        sid = s if isinstance(s, str) else s.get('id','')
        if sid: print(sid)
except: pass
" 2>/dev/null || true)

    local staked_count=0
    local already_staked=0
    local _minersc_addr="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
    while IFS= read -r mid; do
        [ -z "$mid" ] && continue
        # Check current stake via nodeStat on MinerSC
        local _cur_stake
        _cur_stake=$(curl -s --max-time 5 "${sharder_url}/v1/screst/${_minersc_addr}/nodeStat?id=${mid}" 2>/dev/null \
            | python3 -c 'import json,sys; d=json.load(sys.stdin); sp=d.get("stake_pool",{}); print(sp.get("total_stake",0))' 2>/dev/null || echo "0")
        if [ "${_cur_stake:-0}" -gt 0 ]; then
            already_staked=$((already_staked + 1))
            continue
        fi
        print_status "  Staking 10 ZCN on miner ${mid:0:16}..."
        local _sp_out
        _sp_out=$($ZBOX sp-lock --miner_id "$mid" --tokens 10 \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1) || true
        if echo "$_sp_out" | grep -q "genesis miner used"; then
            print_warning "  Miner ${mid:0:16}: genesis miner — stake pool not yet initialised."
        elif echo "$_sp_out" | grep -q "locked"; then
            staked_count=$((staked_count + 1))
        else
            print_warning "  Miner ${mid:0:16}: sp-lock failed: $_sp_out"
        fi
        sleep 2
    done <<< "$miner_ids"

    while IFS= read -r sid; do
        [ -z "$sid" ] && continue
        local _cur_stake
        _cur_stake=$(curl -s --max-time 5 "${sharder_url}/v1/screst/${_minersc_addr}/nodeStat?id=${sid}" 2>/dev/null \
            | python3 -c 'import json,sys; d=json.load(sys.stdin); sp=d.get("stake_pool",{}); print(sp.get("total_stake",0))' 2>/dev/null || echo "0")
        if [ "${_cur_stake:-0}" -gt 0 ]; then
            already_staked=$((already_staked + 1))
            continue
        fi
        print_status "  Staking 10 ZCN on sharder ${sid:0:16}..."
        $ZBOX sp-lock --sharder_id "$sid" --tokens 10 \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        staked_count=$((staked_count + 1))
        sleep 2
    done <<< "$sharder_ids"

    if [ "$staked_count" -gt 0 ]; then
        print_status "Staked on $staked_count miners/sharders ($already_staked already had stake)"
    else
        print_status "All miners/sharders already have stake ($already_staked checked)"
    fi

    # ===== Stake all storage providers (blobbers + enterprise blobbers + validators) =====
    # Idempotent: queries current stake from chain, only tops up if below MIN_PROVIDER_STAKE.
    # Run once on fresh deploy; subsequent runs are fast no-ops if already staked.
    # Why 20 ZCN minimum:
    #   - blobbers with total_stake=0 are excluded from GetAllocationBlobbers entirely
    #   - block rewards are proportional to stake (zeta formula): more stake = more rewards
    #   - validators need stake to be selected for challenges (validators_per_challenge filter)
    local MIN_PROVIDER_STAKE=${3:-20}  # ZCN — third arg overrides, default 20
    local storage_sc_addr="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

    print_status "Checking storage provider stake pools (blobbers + validators, min ${MIN_PROVIDER_STAKE} ZCN)..."

    # Ensure funder has enough ZCN to top up all providers
    # Worst case: (12 blobbers + 5 eblobbers + 12 validators) x 20 ZCN = 580 ZCN
    local _sp_bal
    _sp_bal=$($ZWALLET getbalance --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR \
        --config $ZCN_CONFIG_FILE --silent 2>/dev/null | grep -oP '[\d.]+(?= ZCN)' | head -1 || echo "0")
    local _sp_bal_int=${_sp_bal%.*}
    if [ "${_sp_bal_int:-0}" -lt 400 ]; then
        print_status "  Wallet ${_sp_bal} ZCN — topping up for provider staking (100 ZCN)..."
        $ZWALLET faucet --methodName pour --input '{}' --tokens 100 \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE \
            --silent 2>/dev/null || true
    fi

    local sp_staked_count=0

    # ---- Blobbers (regular + enterprise) ----
    while IFS=$'\t' read -r _bid _gap _type; do
        [ -z "$_bid" ] || [ "$_gap" -le 0 ] 2>/dev/null && continue
        print_status "  sp-lock ${_type} ${_bid:0:16}... adding ${_gap} ZCN to reach ${MIN_PROVIDER_STAKE} ZCN"
        $ZBOX sp-lock --blobber_id "$_bid" --tokens "$_gap" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE \
            --silent 2>/dev/null || true
        sp_staked_count=$((sp_staked_count + 1))
        sleep 2
    done < <(curl -s --max-time 15 \
        "${sharder_url}/v1/screst/${storage_sc_addr}/getblobbers?limit=30" 2>/dev/null | \
        python3 -c "
import json,sys,math
MIN=int('${MIN_PROVIDER_STAKE}')
d=json.load(sys.stdin); nodes=d.get('Nodes',d.get('nodes',[]))
for b in nodes:
    bid=b.get('id',''); stake_zcn=b.get('total_stake',0)/1e10
    gap=math.ceil(max(0, MIN-stake_zcn))
    if bid and gap>0:
        t='eblobber' if b.get('is_enterprise',False) else 'blobber'
        print(f'{bid}\t{gap}\t{t}')
" 2>/dev/null)

    # ---- Validators ----
    while IFS=$'\t' read -r _vid _gap; do
        [ -z "$_vid" ] || [ "$_gap" -le 0 ] 2>/dev/null && continue
        print_status "  sp-lock validator ${_vid:0:16}... adding ${_gap} ZCN to reach ${MIN_PROVIDER_STAKE} ZCN"
        $ZBOX sp-lock --validator_id "$_vid" --tokens "$_gap" \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE \
            --silent 2>/dev/null || true
        sp_staked_count=$((sp_staked_count + 1))
        sleep 2
    done < <(curl -s --max-time 15 \
        "${sharder_url}/v1/screst/${storage_sc_addr}/validators?limit=20" 2>/dev/null | \
        python3 -c "
import json,sys,math
MIN=int('${MIN_PROVIDER_STAKE}')
d=json.load(sys.stdin)
nodes=d if isinstance(d,list) else d.get('nodes',[])
for v in nodes:
    vid=v.get('validator_id',''); stake_zcn=v.get('stake_total',0)/1e10
    gap=math.ceil(max(0, MIN-stake_zcn))
    if vid and gap>0:
        print(f'{vid}\t{gap}')
" 2>/dev/null)

    if [ "$sp_staked_count" -gt 0 ]; then
        print_status "Topped up stake on $sp_staked_count storage provider(s)"
    else
        print_status "All storage providers at or above ${MIN_PROVIDER_STAKE} ZCN stake — no staking needed"
    fi

    # Configure enterprise blobbers: ensure correct write_price, service_charge, not_available.
    # Idempotent — safe to re-run. Also handles blobbers that registered after the initial deploy.
    local _eb_ids
    _eb_ids=$($ZBOX ls-blobbers --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null | \
        jq -r '.[] | select(.is_enterprise == true or (.url | test("198\\.18\\.0\\.20[1-5]|:507[1-5]|eblobber"))) | .id' 2>/dev/null | grep -v '^$' || true)
    if [ -n "$_eb_ids" ]; then
        local _eb_count=0
        _eb_count=$(echo "$_eb_ids" | wc -l | tr -d ' ')
        print_status "Configuring $_eb_count enterprise blobber(s) (write_price=0.001, service_charge=0.3, not_available=true)..."
        for _ebid in $_eb_ids; do
            $ZBOX bl-update \
                --blobber_id "$_ebid" \
                --read_price 0 \
                --write_price 0.001 \
                --service_charge 0.3 \
                --storage_version 1 \
                --num_delegates 100 \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
            $ZBOX bl-update \
                --blobber_id "$_ebid" \
                --not_available=true \
                --wallet owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        done
        print_status "Enterprise blobber configuration complete."
    fi
}

# Check enterprise blobbers for stale health checks and restart if needed.
# Enterprise blobbers register once at startup but their periodic health check loop
# sometimes stops (known eblobber code behaviour). This detects staleness and
# restarts the container so it re-registers on the next startup.
check_and_restart_stale_eblobbers() {
    local STALE_THRESHOLD=${1:-7200}  # seconds — 2 hours
    local sharder_url="http://198.18.0.82:7172"
    local storage_sc="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local current_time=$(date +%s)

    local blobbers_json
    blobbers_json=$(curl -s --max-time 10 \
        "${sharder_url}/v1/screst/${storage_sc}/getblobbers" 2>/dev/null)
    [ -z "$blobbers_json" ] && return

    for i in 1 2 3 4 5; do
        # Skip if container is not running
        docker inspect --format='{{.State.Running}}' eblobber-$i 2>/dev/null | grep -q true || continue

        local eblobber_id
        eblobber_id=$(docker logs eblobber-$i 2>&1 | grep "ID:" | head -1 | awk '{print $NF}' 2>/dev/null)
        [ -z "$eblobber_id" ] || [ ${#eblobber_id} -lt 60 ] && continue

        local last_hc
        last_hc=$(echo "$blobbers_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
blobbers = data.get('nodes', []) or []
for b in blobbers:
    if b.get('id','') == '$eblobber_id':
        print(b.get('last_health_check', 0))
        break
else:
    print(0)
" 2>/dev/null || echo "0")

        # Only act if blobber is registered on chain (last_hc > 0)
        [ "$last_hc" -le 0 ] && continue

        local age=$(( current_time - last_hc ))
        if [ "$age" -gt "$STALE_THRESHOLD" ]; then
            echo "$(date '+%Y-%m-%d %H:%M:%S') eblobber-${i} health check stale (${age}s since last HC), restarting container..."
            docker restart eblobber-$i 2>/dev/null || true
            sleep 15  # Give it time to re-register before next iteration
        fi
    done
}

# Background funding daemon - continuously monitors and funds providers
# Runs in a loop, checking every INTERVAL seconds
start_funding_daemon() {
    local INTERVAL=${1:-300}  # Default: check every 5 minutes
    local LOG_FILE="/tmp/funding_daemon.log"
    local PID_FILE="/tmp/funding_daemon.pid"

    # Check if already running
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        print_warning "Funding daemon already running (PID: $(cat "$PID_FILE"))"
        return 0
    fi

    print_header "Starting Background Funding Daemon"
    print_status "Interval: ${INTERVAL}s, Log: ${LOG_FILE}"

    (
        echo $$ > "$PID_FILE"
        while true; do
            echo "$(date '+%Y-%m-%d %H:%M:%S') === Funding check started ===" >> "$LOG_FILE"

            # Check chain health first
            local round=$(curl -s --max-time 5 "http://198.18.0.82:7172/v1/chain/get/stats" 2>/dev/null | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("current_round",0))' 2>/dev/null || echo "0")
            if [ "$round" = "0" ]; then
                echo "$(date '+%Y-%m-%d %H:%M:%S') Chain not responding, skipping funding cycle" >> "$LOG_FILE"
                sleep "$INTERVAL"
                continue
            fi
            echo "$(date '+%Y-%m-%d %H:%M:%S') Chain healthy at round $round" >> "$LOG_FILE"

            # Run funding check (capture output to log)
            check_and_fund_providers 5 20 >> "$LOG_FILE" 2>&1

            # Restart enterprise blobbers whose health check loop has stalled
            check_and_restart_stale_eblobbers 7200 >> "$LOG_FILE" 2>&1

            echo "$(date '+%Y-%m-%d %H:%M:%S') === Funding check complete, sleeping ${INTERVAL}s ===" >> "$LOG_FILE"
            sleep "$INTERVAL"
        done
    ) &

    local daemon_pid=$!
    echo "$daemon_pid" > "$PID_FILE"
    print_status "Funding daemon started (PID: ${daemon_pid})"
    print_status "View logs: tail -f ${LOG_FILE}"
    print_status "Stop: kill \$(cat ${PID_FILE})"
}

# Stop funding daemon
stop_funding_daemon() {
    local PID_FILE="/tmp/funding_daemon.pid"
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$PID_FILE"
            print_status "Funding daemon stopped (PID: ${pid})"
        else
            rm -f "$PID_FILE"
            print_warning "Funding daemon was not running"
        fi
    else
        print_warning "No funding daemon PID file found"
    fi
}

# Start a background daemon that regenerates the test results info page every 10 minutes
start_results_daemon() {
    local INTERVAL=${1:-600}  # Default: regenerate every 10 minutes
    local LOG_FILE="/tmp/results_daemon.log"
    local PID_FILE="/tmp/results_daemon.pid"

    # Check if already running
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        print_warning "Results daemon already running (PID: $(cat "$PID_FILE"))"
        return 0
    fi

    # Ensure the refresh script exists
    if [ ! -f /usr/local/bin/refresh-0chain-logs.sh ]; then
        print_error "refresh-0chain-logs.sh not found. Run 'bash $0 nginx' first to install it."
        return 1
    fi

    print_header "Starting Background Results Daemon"
    print_status "Interval: ${INTERVAL}s, Log: ${LOG_FILE}"

    (
        trap 'rm -f "'"$PID_FILE"'"; exit 0' SIGTERM SIGINT
        echo $$ > "$PID_FILE"
        while true; do
            echo "$(date '+%Y-%m-%d %H:%M:%S') === Results page regeneration started ===" >> "$LOG_FILE"

            # Call the existing refresh script which regenerates test_results.html and log snapshots
            /usr/local/bin/refresh-0chain-logs.sh >> "$LOG_FILE" 2>&1 || true

            echo "$(date '+%Y-%m-%d %H:%M:%S') === Results page regeneration complete, sleeping ${INTERVAL}s ===" >> "$LOG_FILE"
            sleep "$INTERVAL"
        done
    ) &

    local daemon_pid=$!
    echo "$daemon_pid" > "$PID_FILE"
    print_status "Results daemon started (PID: ${daemon_pid})"
    print_status "View logs: tail -f ${LOG_FILE}"
    print_status "Stop: bash $0 results-daemon-stop"
}

# Stop results daemon
stop_results_daemon() {
    local PID_FILE="/tmp/results_daemon.pid"
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$PID_FILE"
            print_status "Results daemon stopped (PID: ${pid})"
        else
            rm -f "$PID_FILE"
            print_warning "Results daemon was not running"
        fi
    else
        print_warning "No results daemon PID file found"
    fi
}

# Stake blobbers and update read_price to 0
stake_and_configure_blobbers() {
    print_header "Staking and Configuring Blobbers"

    # Detect whether domain resolves to this server (same logic as fix_blobber_config)
    local _stake_domain="${NGINX_DOMAIN:-test.zus.network}"
    local _stake_scheme="https"
    local _stake_use_external=true  # Whether to update blobber on-chain URLs via bl-update
    local _stake_resolved_ip
    _stake_resolved_ip=$(host "$_stake_domain" 2>/dev/null | grep 'has address' | head -1 | awk '{print $NF}' || true)
    local _stake_local_ips
    _stake_local_ips=$(hostname -I 2>/dev/null || ip addr show | grep 'inet ' | awk '{print $2}' | cut -d/ -f1 | tr '\n' ' ')
    if [ -n "$_stake_resolved_ip" ] && ! echo "$_stake_local_ips" | grep -qw "$_stake_resolved_ip"; then
        _stake_scheme="http"
    elif [ -z "$_stake_resolved_ip" ]; then
        # DNS doesn't resolve — don't update on-chain blobber URLs, keep internal Docker IPs
        print_warning "Domain ${_stake_domain} does not resolve — skipping blobber URL update (keeping internal Docker IPs)"
        _stake_use_external=false
    fi

    # Wait for blobbers to register on chain (poll up to 90 seconds)
    print_status "Waiting for blobbers to register on chain..."
    local blobber_json=""
    local blobber_ids=""
    local poll_attempt=0
    local max_poll=18  # 18 * 10s = 180 seconds

    while [ $poll_attempt -lt $max_poll ]; do
        blobber_json=$($ZBOX ls-blobbers --json --silent \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null || echo "[]")
        blobber_ids=$(echo "$blobber_json" | jq -r '.[].id' 2>/dev/null || true)
        local count=$(echo "$blobber_ids" | wc -w | tr -d ' ')
        if [ "$count" -ge 6 ]; then
            print_status "Found $count blobbers on chain"
            break
        fi
        poll_attempt=$((poll_attempt + 1))
        print_status "  Found $count blobbers, waiting... (attempt $poll_attempt/$max_poll)"
        sleep 10
    done

    if [ -z "$blobber_ids" ]; then
        print_warning "No blobbers found on chain after polling. Falling back to key config files..."
        for i in $(seq 1 12); do
            local wallet_file="${BLOBBER_KEYS_DIR}/b0bnode${i}_keys.txt.json"
            if [ -f "$wallet_file" ]; then
                blobber_ids="$blobber_ids $(jq -r '.client_id' "$wallet_file")"
            fi
        done
    fi

    # Stake each blobber with retry — "too less sharders to confirm" is transient at chain startup
    local SHARDER_STORAGE_URL="http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    for blobber_id in $blobber_ids; do
        local stake_ok=0
        for attempt in 1 2 3; do
            print_status "Staking on blobber: ${blobber_id:0:16}... (attempt $attempt/3)"
            local sp_out
            sp_out=$($ZBOX sp-lock \
                --blobber_id "$blobber_id" \
                --tokens 1 \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>&1 | \
                grep -v "^0chain-core-sdk" || true)
            echo "$sp_out"
            if echo "$sp_out" | grep -q "tokens locked\|already has stake\|already locked"; then
                stake_ok=1
                break
            fi
            # Check if stake already applied (e.g. previous attempt went through)
            local cur_stake
            cur_stake=$(curl -s --max-time 10 \
                "${SHARDER_STORAGE_URL}/getBlobber?blobber_id=${blobber_id}" 2>/dev/null | \
                python3 -c "import json,sys; print(json.load(sys.stdin).get('total_stake',0))" 2>/dev/null || echo "0")
            if [ "${cur_stake:-0}" -gt 0 ]; then
                print_status "  Blobber ${blobber_id:0:16} already has stake ($cur_stake SAS)"
                stake_ok=1
                break
            fi
            [ "$attempt" -lt 3 ] && { print_warning "  sp-lock failed, retrying in 10s..."; sleep 10; }
        done
        [ "$stake_ok" = "0" ] && print_warning "  WARNING: Could not stake blobber ${blobber_id:0:16} after 3 attempts"
        sleep 3
    done

    # Verify staking worked — blobbers with 0 stake cannot serve allocations
    sleep 5
    local unstaked_after
    unstaked_after=$(curl -s --max-time 15 \
        "${SHARDER_STORAGE_URL}/getblobbers?active=true" \
        2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)
    unstaked=[b['id'][:16] for b in data.get('Nodes',[]) if not b.get('is_enterprise') and b.get('total_stake',0)==0]
    if unstaked: print(f'{len(unstaked)} blobbers still have 0 stake after sp-lock: {unstaked}')
    else: print('OK: all blobbers have stake')
except Exception as e: print(f'verify error: {e}')
" 2>/dev/null || echo "Could not verify blobber stake")
    print_status "Blobber stake check: $unstaked_after"

    # Note: Enterprise blobbers are staked separately after Phase 8 (build_and_deploy_enterprise_blobbers)
    # via stake_enterprise_blobbers() since they haven't been deployed yet at this point.

    # Update all blobber read_prices to 0, storage_version to 1, and num_delegates to 200
    # read_price=0 is required for free allocation tests
    # storage_version=1 is required for SDK v2 allocations (SDK always uses StorageV2=1)
    # num_delegates=200 is required so tokenomics tests can stake from multiple wallets
    # Without storage_version=1, the alloc_blobbers query filters on storage_version=1 and finds no blobbers
    #
    # Also sets external HTTPS URL (https://test.zus.network/blobberNN/) so browsers
    # can reach blobbers via nginx reverse proxy instead of internal Docker IPs.
    # Blobbers register with internal IPs (198.18.0.9X:505X) on first start; this
    # overrides the on-chain URL so gosdk WASM CheckAllocStatus calls succeed.
    # Port mapping: 5051-5059 → blobber01-09, 50610-50612 → blobber10-12.
    #
    # IMPORTANT: bl-update requires the delegate_wallet owner to sign the txn.
    # If the on-chain delegate_wallet doesn't match ZCN_WALLET_FILE, these calls
    # will silently fail with "access denied, allowed for delegate_wallet owner only".
    print_status "Setting blobber read_price=0, write_price=0.001, storage_version=1, num_delegates=200, service_charge=0.1, external URL for all blobbers..."
    local update_failures=0
    for blobber_id in $blobber_ids; do
        # Determine external URL from registered port (extracted from blobber_json URL field)
        local registered_url=""
        local external_url=""
        if [ -n "$blobber_json" ]; then
            registered_url=$(echo "$blobber_json" | jq -r --arg id "$blobber_id" '.[] | select(.id == $id) | .url // ""' 2>/dev/null || true)
        fi
        if [ -n "$registered_url" ]; then
            local port
            port=$(echo "$registered_url" | grep -oE '[0-9]+$' || true)
            case "$port" in
                5051) external_url="${_stake_scheme}://${_stake_domain}/blobber01/" ;;
                5052) external_url="${_stake_scheme}://${_stake_domain}/blobber02/" ;;
                5053) external_url="${_stake_scheme}://${_stake_domain}/blobber03/" ;;
                5054) external_url="${_stake_scheme}://${_stake_domain}/blobber04/" ;;
                5055) external_url="${_stake_scheme}://${_stake_domain}/blobber05/" ;;
                5056) external_url="${_stake_scheme}://${_stake_domain}/blobber06/" ;;
                5057) external_url="${_stake_scheme}://${_stake_domain}/blobber07/" ;;
                5058) external_url="${_stake_scheme}://${_stake_domain}/blobber08/" ;;
                5059) external_url="${_stake_scheme}://${_stake_domain}/blobber09/" ;;
                5060) external_url="${_stake_scheme}://${_stake_domain}/blobber10/" ;;
                5061) external_url="${_stake_scheme}://${_stake_domain}/blobber11/" ;;
                5062) external_url="${_stake_scheme}://${_stake_domain}/blobber12/" ;;
            esac
        fi

        local url_flag=""
        if [ "${_stake_use_external:-true}" = true ] && [ -n "$external_url" ]; then
            url_flag="--url $external_url"
            print_status "Updating blobber ${blobber_id:0:16}... read_price=0, write_price=0.001, storage_version=1, num_delegates=200, service_charge=0.1, url=$external_url"
        else
            print_status "Updating blobber ${blobber_id:0:16}... read_price=0, write_price=0.001, storage_version=1, num_delegates=200, service_charge=0.1"
        fi

        local update_output
        update_output=$($ZBOX bl-update \
            --blobber_id "$blobber_id" \
            --read_price 0 \
            --write_price 0.001 \
            --service_charge 0.1 \
            --storage_version 1 \
            --num_delegates 200 \
            --not_available=false \
            $url_flag \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1) || true
        if echo "$update_output" | grep -qi "access denied\|delegate_wallet"; then
            print_error "  bl-update FAILED for ${blobber_id:0:16}... (delegate_wallet mismatch!)"
            # Step 1: Find the current on-chain delegate_wallet for this blobber
            local _onchain_dw
            _onchain_dw=$(curl -s "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getblobbers?limit=50" \
                | python3 -c "
import sys,json
d=json.load(sys.stdin)
for b in d.get('Nodes',d.get('nodes',[])):
    if b.get('id','') == '$blobber_id':
        print(b.get('stake_pool_settings',{}).get('delegate_wallet',''))
        break
" 2>/dev/null || true)
            # Step 2: Search wallets.json pool for the old delegate wallet
            local _wallets_pool="${SYSTEM_TEST_DIR}/tests/cli_tests/config/wallets/wallets.json"
            local _old_dw_fixed=false
            if [ -n "$_onchain_dw" ] && [ -f "$_wallets_pool" ]; then
                print_status "  On-chain delegate_wallet: ${_onchain_dw:0:16}... — searching wallets.json pool"
                local _tmp_dw_wallet="/tmp/zcn_bl_dw_fix/wallet.json"
                mkdir -p /tmp/zcn_bl_dw_fix
                cp "${ZCN_CONFIG_DIR}/${ZCN_CONFIG_FILE}" /tmp/zcn_bl_dw_fix/config.yaml 2>/dev/null || true
                python3 -c "
import json, sys
wallets = json.load(open('$_wallets_pool'))
for w in wallets if isinstance(wallets, list) else [wallets]:
    if w.get('client_id','') == '$_onchain_dw':
        json.dump(w, open('$_tmp_dw_wallet', 'w'))
        print('found')
        break
" 2>/dev/null | grep -q "found" && {
                    print_status "  Found old delegate wallet — attempting bl-update with it"
                    local _fix_output
                    _fix_output=$($ZBOX bl-update \
                        --blobber_id "$blobber_id" \
                        --delegate_wallet "$sc_owner_id" \
                        --wallet wallet.json --configDir /tmp/zcn_bl_dw_fix \
                        --silent 2>&1) && {
                        print_status "  bl-update with old delegate wallet succeeded — delegate_wallet fixed to owner"
                        _old_dw_fixed=true
                    } || print_warning "  bl-update with old delegate wallet failed: $_fix_output"
                } || print_warning "  Old delegate wallet ${_onchain_dw:0:16}... not found in wallets.json pool"
            fi
            if ! $_old_dw_fixed; then
                print_warning "  Falling back: restarting blobber container to force re-registration"
                for _cname in $(docker ps --format '{{.Names}}' | grep -E '^blobber-[0-9]+$'); do
                    local _cid
                    _cid=$(docker logs "$_cname" 2>&1 | grep -oP '(?<=client_id=)[a-f0-9]{64}' | head -1)
                    [ -z "$_cid" ] && _cid=$(docker logs "$_cname" 2>&1 | grep -oP '(?i)(?:ID:|blobber_id:)\s*\K[a-f0-9]{64}' | head -1)
                    if [ "$_cid" = "$blobber_id" ]; then
                        docker restart "$_cname" 2>/dev/null || true
                        print_status "  Restarted $_cname"
                        sleep 15
                        break
                    fi
                done
                update_failures=$((update_failures + 1))
            fi
        fi
        sleep 2
    done
    if [ "$update_failures" -gt 0 ]; then
        print_error "$update_failures blobber(s) could not be updated due to delegate_wallet mismatch!"
        print_error "These blobbers were registered with a different delegate_wallet than expected."
        print_error "Fix: Redeploy these blobbers from scratch, or use a wallet matching their delegate_wallet."
    fi

    # Update validators: num_delegates=100 + external HTTPS URL via nginx proxy.
    # Same problem as blobbers: validators register with internal Docker IPs.
    # Port mapping: 5041→validator01, 5042→validator02, 5063-5070→validator03-10.
    print_status "Updating validators: num_delegates=100 and external URL..."
    local validator_json=$($ZBOX ls-validators --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null || echo "[]")
    local validator_ids=$(echo "$validator_json" | jq -r '.[].id' 2>/dev/null || true)

    for validator_id in $validator_ids; do
        # Skip empty/null IDs (happens when validators haven't registered yet)
        if [ -z "$validator_id" ] || [ "$validator_id" = "null" ]; then
            continue
        fi

        # Determine external URL from registered port
        local val_registered_url=""
        local val_external_url=""
        if [ -n "$validator_json" ]; then
            val_registered_url=$(echo "$validator_json" | jq -r --arg id "$validator_id" '.[] | select(.id == $id) | .base_url // .url // ""' 2>/dev/null || true)
        fi
        if [ -n "$val_registered_url" ]; then
            local val_port
            val_port=$(echo "$val_registered_url" | grep -oE '[0-9]+$' || true)
            local _val_domain="${NGINX_DOMAIN:-test.zus.network}"
            case "$val_port" in
                5041) val_external_url="https://${_val_domain}/validator01/" ;;
                5042) val_external_url="https://${_val_domain}/validator02/" ;;
                5061) val_external_url="https://${_val_domain}/validator01/" ;;
                5062) val_external_url="https://${_val_domain}/validator02/" ;;
                5063) val_external_url="https://${_val_domain}/validator03/" ;;
                5064) val_external_url="https://${_val_domain}/validator04/" ;;
                5065) val_external_url="https://${_val_domain}/validator05/" ;;
                5066) val_external_url="https://${_val_domain}/validator06/" ;;
                5067) val_external_url="https://${_val_domain}/validator07/" ;;
                5068) val_external_url="https://${_val_domain}/validator08/" ;;
                5069) val_external_url="https://${_val_domain}/validator09/" ;;
                5070) val_external_url="https://${_val_domain}/validator10/" ;;
            esac
        fi

        local val_url_flag=""
        if [ "${_stake_use_external:-true}" = true ] && [ -n "$val_external_url" ]; then
            val_url_flag="--base_url $val_external_url"
            print_status "Updating validator ${validator_id:0:16}... num_delegates=100, url=$val_external_url"
        else
            print_status "Updating validator ${validator_id:0:16}... num_delegates=100"
        fi

        $ZBOX validator-update \
            --validator_id "$validator_id" \
            --num_delegates 100 \
            $val_url_flag \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 2
    done

    print_status "Blobber staking and configuration complete!"
}

# Stake enterprise blobbers (called after Phase 8 when they're deployed and registered)
stake_enterprise_blobbers() {
    print_header "Staking Enterprise Blobbers"

    # Poll for enterprise blobbers to appear on chain (up to 120 seconds).
    # NOTE: The chain may not return is_enterprise in the blobber JSON, so we
    # identify enterprise blobbers by their URL pattern (ports 5071-5075,
    # IPs 198.18.0.201-205) which matches our compose template.
    local enterprise_ids=""
    local poll=0
    local max_poll=12  # 12 * 10s = 120 seconds

    while [ $poll -lt $max_poll ]; do
        local eblobber_json=$($ZBOX ls-blobbers --json --silent \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null || echo "[]")
        enterprise_ids=$(echo "$eblobber_json" | jq -r '.[] | select(.is_enterprise == true or (.url | test("198\\.18\\.0\\.20[1-5]|:507[1-5]|eblobber"))) | .id' 2>/dev/null | grep -v '^$' || true)
        local count=0
        if [ -n "$enterprise_ids" ]; then
            count=$(echo "$enterprise_ids" | wc -l | tr -d ' ')
        fi
        local expected_count
        expected_count=$(yq_get "enterprise_blobbers.count" 2>/dev/null || echo "5")
        if [ "$count" -ge "${expected_count:-5}" ]; then
            print_status "Found all $count enterprise blobbers on chain"
            break
        elif [ "$count" -ge 1 ]; then
            print_status "  Found $count/$expected_count enterprise blobbers — waiting for all to register..."
        fi
        poll=$((poll + 1))
        print_status "  Enterprise blobbers: $count/$expected_count found, waiting... (attempt $poll/$max_poll)"
        sleep 10
    done

    if [ -n "$enterprise_ids" ]; then
        for eblobber_id in $enterprise_ids; do
            print_status "Staking on enterprise blobber: ${eblobber_id:0:16}..."
            $ZBOX sp-lock \
                --blobber_id "$eblobber_id" \
                --tokens 20 \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        done
        print_status "Enterprise blobber staking complete!"

        # Configure enterprise blobbers: read_price=0, write_price=0.001, service_charge=0.1, storage_version=1, num_delegates=100
        # Retry each blobber up to 3 times to handle transient nonce/fee errors
        print_status "Configuring enterprise blobbers (read_price=0, write_price=0.001, service_charge=0.1, storage_version=1, num_delegates=100)..."
        for eblobber_id in $enterprise_ids; do
            local _eb_ok=false
            for _eb_try in 1 2 3; do
                if $ZBOX bl-update \
                    --blobber_id "$eblobber_id" \
                    --read_price 0 \
                    --write_price 0.001 \
                    --service_charge 0.1 \
                    --storage_version 1 \
                    --num_delegates 100 \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>&1 | grep -qiE 'updated|success'; then
                    _eb_ok=true
                    print_status "  Configured enterprise blobber ${eblobber_id:0:16}..."
                    break
                fi
                print_warning "  bl-update for ${eblobber_id:0:16} attempt $_eb_try failed, retrying..."
                sleep 3
            done
            if ! $_eb_ok; then
                # Try once more without checking output (some versions don't print "updated")
                $ZBOX bl-update \
                    --blobber_id "$eblobber_id" \
                    --read_price 0 \
                    --write_price 0.001 \
                    --service_charge 0.1 \
                    --storage_version 1 \
                    --num_delegates 100 \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
                print_status "  Configured enterprise blobber ${eblobber_id:0:16} (no confirmation)"
            fi
            sleep 2
        done
        print_status "Enterprise blobber configuration complete!"

        # Mark enterprise blobbers as not_available=true so regular alloc_blobbers
        # queries don't select them. GetBlobbersFromParams orders by (capacity-allocated) ASC,
        # enterprise blobbers have large existing allocations → less free space → appear first
        # → selected for regular client allocations → rejected with invalid_signature.
        print_status "Setting enterprise blobbers not_available=true (prevent selection for regular allocations)..."
        for eblobber_id in $enterprise_ids; do
            $ZBOX bl-update \
                --blobber_id "$eblobber_id" \
                --not_available=true \
                --wallet owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
            sleep 1
        done
        print_status "Enterprise blobbers marked not_available."

        # Verify enterprise blobber configuration is correct
        sleep 5
        print_status "Verifying enterprise blobber configuration..."
        for eblobber_id in $enterprise_ids; do
            local eb_wp
            eb_wp=$(curl -s "http://${SHARDER_IP}:${SHARDER_PORT}/v1/screst/${STORAGE_SC_ADDRESS}/getBlobber?blobber_id=${eblobber_id}" 2>/dev/null | \
                python3 -c "import json,sys; b=json.load(sys.stdin); wp=b.get('terms',{}).get('write_price',0); print(wp)" 2>/dev/null || echo "0")
            local expected_wp=10000000  # 0.001 ZCN = 10,000,000 SAS
            if [ "${eb_wp:-0}" -ne "$expected_wp" ] && [ "${eb_wp:-0}" -gt 0 ]; then
                print_warning "  eblobber ${eblobber_id:0:16} write_price=${eb_wp} (expected ${expected_wp}) — retrying bl-update..."
                $ZBOX bl-update --blobber_id "$eblobber_id" --write_price 0.001 \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
                sleep 2
            else
                print_status "  eblobber ${eblobber_id:0:16} write_price OK (${eb_wp})"
            fi
        done
    else
        print_warning "No enterprise blobbers registered on chain after polling - skipping staking"
    fi
}


# Start a persistent DNS proxy on port 9099 that injects sharder URLs into the /network response.
# This fixes "network has no miners/sharders" gosdk initialization failures when 0dns has no
# sharders (because sharders never made it into the magic block due to DKG deadlock).
#
# The proxy is transparent: it passes all requests to 0dns (port 9091) unchanged, except
# /network where it ensures the sharders list is populated from the Miner SC.
#
# Writes /tmp/dns_proxy.py and /root/.zcn/vc_proxy.yaml.
# Idempotent: kills any existing proxy before starting a new one.
start_dns_proxy() {
    local PROXY_SCRIPT="/tmp/dns_proxy.py"

    # Get sharder URLs from Miner SC (canonical source of truth)
    local sharder_hosts
    sharder_hosts=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList" 2>/dev/null \
        | python3 -c "import json,sys; [print('http://'+n['simple_miner']['n2n_host']+':'+str(n['simple_miner']['port'])) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)

    if [ -z "$sharder_hosts" ]; then
        print_warning "start_dns_proxy: no sharder URLs from Miner SC — proxy will not inject sharders"
        sharder_hosts="http://198.18.0.81:7171
http://198.18.0.82:7172"
    fi

    local sharder_list
    sharder_list=$(echo "$sharder_hosts" | python3 -c "import sys,json; urls=[l.strip() for l in sys.stdin if l.strip()]; print(json.dumps(urls))")

    # Kill any existing proxy
    pkill -f "$PROXY_SCRIPT" 2>/dev/null || true
    sleep 1

    # Write proxy script
    cat > "$PROXY_SCRIPT" << 'PYEOF'
import sys,json,urllib.request
from http.server import HTTPServer,BaseHTTPRequestHandler
sharders = json.loads(sys.argv[1])
class P(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path=='/network':
            try: d=json.loads(urllib.request.urlopen('http://127.0.0.1:9091/network').read())
            except: d={}
            if not d.get('sharders'):
                d['sharders']=sharders
            b=json.dumps(d).encode()
            self.send_response(200); self.send_header('Content-Type','application/json'); self.end_headers(); self.wfile.write(b)
        else:
            try: resp=urllib.request.urlopen('http://127.0.0.1:9091'+self.path); b=resp.read(); self.send_response(200); self.end_headers(); self.wfile.write(b)
            except: self.send_response(502); self.end_headers()
    def log_message(self,*a): pass
HTTPServer(('127.0.0.1',9099),P).serve_forever()
PYEOF

    # Start proxy in background (disown so it survives the shell)
    python3 "$PROXY_SCRIPT" "$sharder_list" &
    disown
    sleep 2

    # Write proxy config for zwallet/zbox commands that need gosdk init
    cat > "${ZCN_CONFIG_DIR}/vc_proxy.yaml" << EOF
block_worker: http://127.0.0.1:9099
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 10
confirmation_chain_length: 3
max_txn_query: 10
query_sleep_time: 5
EOF

    print_status "DNS proxy started on port 9099 (injects sharder IPs into /network)"
}

# Add sharders that are registered in Miner SC but not yet in the magic block.
# This fixes DKG restart loops where 0dns returns sharders:[] causing API/SDK tests to fail.
# Safe to call multiple times (vc-add is idempotent when sharder already in MB).
vc_add_sharders() {
    print_header "Adding Sharders to View Change Register List"
    local ZWALLET_BIN="${BASE_DIR}/system_test/tests/cli_tests/zwallet"
    [ ! -f "$ZWALLET_BIN" ] && ZWALLET_BIN="$ZWALLET"

    # Check if 0dns returns sharders already
    local dns_sharders
    dns_sharders=$(curl -s "http://127.0.0.1:9091/network" 2>/dev/null \
        | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d.get('sharders') or []))" 2>/dev/null)

    # Determine config: use proxy if 0dns has no sharders (gosdk requires sharders to init)
    local VC_CONFIG="$ZCN_CONFIG_FILE"
    if [ "${dns_sharders:-0}" -eq 0 ]; then
        print_status "0dns has no sharders — ensuring DNS proxy is running on port 9099..."
        start_dns_proxy
        VC_CONFIG="vc_proxy.yaml"
    fi

    local WO="--wallet $ZCN_SC_OWNER_WALLET --configDir $ZCN_CONFIG_DIR --config $VC_CONFIG"

    local sharder_ids
    sharder_ids=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['simple_miner']['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)
    local mb_sharders
    mb_sharders=$(curl -s "http://198.18.0.82:7172/v1/block/get/latest_finalized_magic_block" 2>/dev/null \
        | python3 -c "import json,sys; mb=json.load(sys.stdin).get('magic_block',{}); [print(k) for k in mb.get('sharders',{}).get('nodes',{}).keys()]" 2>/dev/null)
    local added=0
    for sid in $sharder_ids; do
        if echo "$mb_sharders" | grep -q "$sid"; then
            print_status "Sharder ${sid:0:16}... already in MB, skipping"
        else
            print_status "Adding Sharder ${sid:0:16}... to view change register list"
            "$ZWALLET_BIN" vc-add --id "$sid" --provider-type sharder $WO --silent 2>&1 || \
                print_warning "vc-add failed for ${sid:0:16} (may already be registered)"
            added=$((added + 1))
        fi
    done

    if [ "$added" -gt 0 ]; then
        print_status "Added $added sharder(s). They will join the MB in the next view change cycle (~100 rounds)."
    else
        print_status "All sharders already in MB."
    fi
}

# Idempotent chain configuration -- safe to run multiple times.
# Sets all SC configs, blobber settings, and verifies the result.
# Can be called standalone: ./deploy_local.sh ensure-config
ensure_chain_config() {
    print_header "Ensuring Chain Configuration (Idempotent)"

    # Reset wallet nonces to 0 so zwallet re-fetches from chain on next use.
    # This prevents stale cached nonces from causing "invalid transaction nonce" errors.
    reset_wallet_nonces

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
    local WO="--wallet $ZCN_SC_OWNER_WALLET --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
    # MinerSC may have a different owner on chains with historical state (miner_sc_owner.json).
    # Verify client_id matches on-chain MinerSC owner_id — stale files cause auth failures on fresh chains.
    local WOM="$WO"
    local _miner_sc2="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
    local _sharder_wom2="${SHARDER_BASE_URL:-http://198.18.0.81:7171}"
    local _onchain_miner_owner2
    _onchain_miner_owner2=$(curl -sf "${_sharder_wom2}/v1/screst/${_miner_sc2}/configs" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); f=d.get('fields',d); print(f.get('owner_id',''))" 2>/dev/null || echo "")
    if [ -f "${ZCN_CONFIG_DIR}/miner_sc_owner.json" ]; then
        local _wom_id2
        _wom_id2=$(python3 -c "import json; print(json.load(open('${ZCN_CONFIG_DIR}/miner_sc_owner.json')).get('client_id',''))" 2>/dev/null || echo "")
        if [ -n "$_onchain_miner_owner2" ] && [ "$_wom_id2" = "$_onchain_miner_owner2" ]; then
            WOM="--wallet miner_sc_owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
            print_status "Using miner_sc_owner.json for mn-update-config (verified MinerSC owner match)"
        else
            print_status "miner_sc_owner.json (${_wom_id2:0:16}...) != on-chain MinerSC owner (${_onchain_miner_owner2:0:16}...) — using owner.json"
        fi
    fi

    # ========== Restore StorageSC owner if a test changed it ==========
    # TestOwnerUpdate changes StorageSC owner_id and restores it in Cleanup.
    # If Cleanup fails (view change, "too less sharders"), the owner is left as a test wallet.
    # Detect this and restore before running SC config updates (which require the correct owner).
    local storage_sc="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local sharder_url="${SHARDER_BASE_URL:-http://198.18.0.81:7171}"
    local expected_sc_owner
    expected_sc_owner=$(jq -r '.client_id' "${BASE_DIR}/system_test/tests/cli_tests/config/wallets/sc_owner_wallet.json" 2>/dev/null)
    local current_sc_owner
    current_sc_owner=$(curl -sf "${sharder_url}/v1/screst/${storage_sc}/storage-config" 2>/dev/null | jq -r '.fields.owner_id // .owner_id // empty')
    if [ -n "$expected_sc_owner" ] && [ -n "$current_sc_owner" ] && [ "$current_sc_owner" != "$expected_sc_owner" ]; then
        print_warning "StorageSC owner mismatch: on-chain=${current_sc_owner:0:16}... expected=${expected_sc_owner:0:16}... — attempting restore..."
        local restore_wallet_file=""
        # Check ~/.zcn wallets first (miner_sc_owner.json is often the culprit since wallets[500] = miner_sc_owner)
        for candidate in "${ZCN_CONFIG_DIR}/miner_sc_owner.json" "${ZCN_CONFIG_DIR}/miner_sc_owner_wallet.json"; do
            if [ -f "$candidate" ]; then
                local cid
                cid=$(jq -r '.client_id // empty' "$candidate" 2>/dev/null)
                if [ "$cid" = "$current_sc_owner" ]; then
                    restore_wallet_file="$candidate"
                    break
                fi
            fi
        done
        # Fall back: search test config dir (skip wallets.json array file)
        if [ -z "$restore_wallet_file" ]; then
            while IFS= read -r f; do
                if [ "$(basename "$f")" = "wallets.json" ]; then continue; fi
                local cid
                cid=$(jq -r '.client_id // empty' "$f" 2>/dev/null)
                if [ "$cid" = "$current_sc_owner" ]; then
                    restore_wallet_file="$f"
                    break
                fi
            done < <(find "${BASE_DIR}/system_test/tests/cli_tests/config/" -name '*.json' 2>/dev/null)
        fi
        if [ -n "$restore_wallet_file" ]; then
            print_status "Using wallet: $restore_wallet_file"
            jq '.nonce = 0' "$restore_wallet_file" > "${ZCN_CONFIG_DIR}/restore_sc_owner.json"
            local WO_RESTORE="--wallet restore_sc_owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
            if $ZWALLET sc-update-config --keys owner_id --values "$expected_sc_owner" $WO_RESTORE 2>&1; then
                print_status "StorageSC owner restored to ${expected_sc_owner:0:16}..."
                sleep 5
            else
                print_error "Failed to restore StorageSC owner — SC config updates may fail!"
            fi
        else
            print_error "No wallet file found with client_id=${current_sc_owner:0:16}... — SC config updates may fail!"
        fi
    fi

    # ========== Faucet Settings (enforce idempotently on running chains) ==========
    # patch_sc_yaml() sets genesis values but CANNOT fix a chain that started with old sc.yaml.
    # Call updateSettings here so the correct limits are enforced regardless of genesis config.
    # NOTE: faucet owner_id is set to owner.json by patch_sc_yaml(), so $WO is correct.
    print_status "Enforcing faucet limits (pour=100, max_pour=10000, periodic=10000, global=10M, reset=1h)..."
    $ZWALLET faucet --methodName updateSettings \
        --input '{"pour_limit":100,"max_pour_amount":10000,"periodic_limit":10000,"global_limit":10000000,"individual_reset":"1h"}' \
        $WO 2>&1 || print_warning "Faucet updateSettings failed (chain may not be ready yet; genesis patch still applies)"
    sleep 3

    # ========== Fund owner wallet (idempotent -- faucet tops up) ==========
    print_status "Funding owner wallet (100 ZCN top-up)..."
    $ZWALLET faucet --methodName pour --input '{}' --tokens 100 $W 2>&1 || print_warning "Faucet pour failed (wallet may already have funds)"
    sleep 3

    # ========== Miner SC Config (requires SC owner wallet) ==========
    # NOTE: vc_rounds MUST be set before min_n. Setting vc_rounds migrates the GlobalNode
    # entity from DefaultOriginVersion to v2 (hermes hardfork path). Without this migration,
    # setInt for min_n/min_s is silently dropped (DefaultOriginVersion only handles VCPhaseRounds).
    print_status "Setting miner SC: VC round durations (10,20,10,10,20)"
    run_cmd "mn-update-config vc_rounds" \
        "$ZWALLET mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' --values '10,20,10,10,20' $WOM"

    print_status "Setting miner SC: k_percent=0.6, x_percent=0.6"
    run_cmd "mn-update-config k_percent,x_percent" \
        "$ZWALLET mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' $WOM"

    print_status "Setting miner SC: min_n=3, min_s=1"
    run_cmd "mn-update-config min_n,min_s" \
        "$ZWALLET mn-update-config --keys 'min_n,min_s' --values '3,1' $WOM"

    print_status "Setting miner SC: cost.vc_add=361"
    run_cmd "mn-update-config cost.vc_add" \
        "$ZWALLET mn-update-config --keys 'cost.vc_add' --values '361' $WOM"

    print_status "Setting miner SC: num_sharders_rewarded=2"
    run_cmd "mn-update-config num_sharders_rewarded" \
        "$ZWALLET mn-update-config --keys 'num_sharders_rewarded' --values '2' $WOM"

    print_status "Setting miner SC: max_delegates=200"
    run_cmd "mn-update-config max_delegates" \
        "$ZWALLET mn-update-config --keys 'max_delegates' --values '200' $WOM"

    # share_ratio must be < 1 — prevents "uint64 minus overflow" in payfees on every block
    print_status "Setting miner SC: share_ratio=0.16"
    run_cmd "mn-update-config share_ratio" \
        "$ZWALLET mn-update-config --keys 'share_ratio' --values '0.16' $WOM"

    # health_check_period=60m gives a 30-min buffer vs the Atlus isGoodHealth threshold of 90m.
    # Default 90m == threshold → any missed cycle makes miner appear unhealthy in Atlus.
    print_status "Setting miner SC: health_check_period=60m"
    run_cmd "mn-update-config health_check_period" \
        "$ZWALLET mn-update-config --keys 'health_check_period' --values '60m' $WOM"

    # ========== Global Config (requires SC owner wallet) ==========
    # view_change=false: prevents split-DKG bug where miners restarted at different times
    # load different DKG key states from disk → VRF share verification fails → chain stuck.
    # The 0chain.yaml also has view_change: false as local default. Both must be false.
    # If view_change was ever enabled (e.g., old deploy), disable it now.
    print_status "Setting global: view_change=false (prevents split-DKG chain-stuck)"
    run_cmd "global-update-config view_change" \
        "$ZWALLET global-update-config --keys 'server_chain.view_change' --values 'false' $WO"

    print_status "Setting global: max_wait_time=500ms"
    run_cmd "global-update-config max_wait_time" \
        "$ZWALLET global-update-config --keys 'server_chain.block.proposal.max_wait_time' --values '500ms' $WO"

    # fee_SAS=cost×10^12/coeff so HIGHER=cheaper; default 1000, set 10000000 = 10000x cheaper
    # At 10000000: challenge_response=0.00728 ZCN, new_allocation=0.019 ZCN (enterprise tests pass)
    # Miners read this from local 0chain.yaml, NOT globalSettings — must patch yaml file too.
    print_status "Setting global: cost_fee_coeff=10000000 (fees 10000x cheaper)"
    local MINER_CFG="${BASE_DIR}/0chain/docker.local/config/0chain.yaml"
    if [ -f "$MINER_CFG" ]; then
        sed -i "s/cost_fee_coeff: [0-9]*/cost_fee_coeff: 10000000/" "$MINER_CFG"
        print_status "Patched 0chain.yaml: cost_fee_coeff=10000000"
    fi
    run_cmd "global-update-config cost_fee_coeff" \
        "$ZWALLET global-update-config --keys 'server_chain.transaction.cost_fee_coeff' --values '10000000' $WO"
    # CRITICAL: Set max_fee low so sharder health check transactions can go through.
    # Sharder binary uses coeff=10000000 (fee=145,000 SAS) but miners may validate with higher coeff
    # giving minFee >> 145,000 SAS. Cap max_fee at 100,000 SAS so minFee is capped below sharder's fee.
    run_cmd "global-update-config max_fee" \
        "$ZWALLET global-update-config --keys 'server_chain.transaction.max_fee' --values '0.00001' $WO"

    # ========== Storage SC Config (must use SC owner wallet) ==========
    print_status "Setting storage SC: time_unit=720h (30 days — chain uses hours not days)"
    run_cmd "sc-update-config time_unit" \
        "$ZWALLET sc-update-config --keys 'time_unit' --values '720h' $WO"

    print_status "Setting storage SC: min_alloc_size=1024"
    run_cmd "sc-update-config min_alloc_size" \
        "$ZWALLET sc-update-config --keys 'min_alloc_size' --values '1024' $WO"

    print_status "Setting storage SC: min_write_price=0.001 (chain-enforced minimum; 0chain code floor)"
    run_cmd "sc-update-config min_write_price" \
        "$ZWALLET sc-update-config --keys 'min_write_price' --values '0.001' $WO"

    print_status "Resetting free_allocation write_price_range.min=0"
    run_cmd "sc-update-config free_allocation write_price_range.min" \
        "$ZWALLET sc-update-config --keys 'free_allocation_settings.write_price_range.min' --values '0' $WO"

    print_status "Setting storage SC: challenge_enabled=true"
    run_cmd "sc-update-config challenge_enabled" \
        "$ZWALLET sc-update-config --keys 'challenge_enabled' --values 'true' $WO"

    # validators_per_challenge: set to 2 (only 2 validators reliably stake on this chain).
    # Most validators fail sp-lock with "can't get stake pool: value not present" — only 2 succeed.
    # num_validators_rewarded: all 9 validators rewarded per challenge.
    print_status "Setting storage SC: validators_per_challenge=2, num_validators_rewarded=9"
    run_cmd "sc-update-config validators_per_challenge" \
        "$ZWALLET sc-update-config --keys 'validators_per_challenge,num_validators_rewarded' --values '2,9' $WO"

    print_status "Setting storage SC: free_allocation read_price_range=[0,0] (read_price always 0)"
    run_cmd "sc-update-config free_allocation read_price_range" \
        "$ZWALLET sc-update-config --keys 'free_allocation_settings.read_price_range.min,free_allocation_settings.read_price_range.max' --values '0,0' $WO"

    # Use 60m to match validator healthcheck.frequency=50m in 0chain_validator.yaml.
    # SC requires health_check within this period; validators check every 50m so 30m fails.
    print_status "Setting storage SC: health_check_period=60m"
    run_cmd "sc-update-config health_check_period" \
        "$ZWALLET sc-update-config --keys 'health_check_period' --values '60m' $WO"

    # Set validator_reward to 0.025 (2.5% of challenge rewards go to validators).
    # Validators provide data integrity proofs for challenges and receive 2.5% of
    # the challenge reward pool per the tokenomics design.
    print_status "Setting storage SC: validator_reward=0.025 (2.5% of challenge rewards to validators)"
    run_cmd "sc-update-config validator_reward" \
        "$ZWALLET sc-update-config --keys 'validator_reward' --values '0.025' $WO"

    # Fix SC function costs to match gosdk fee estimation (gosdk estimates cost=N-1, chain uses N).
    print_status "Setting cost.blobber_health_check=97 (gosdk estimates 97, chain default 98)"
    run_cmd "sc-update-config cost.blobber_health_check" \
        "$ZWALLET sc-update-config --keys 'cost.blobber_health_check' --values '97' $WO"
    print_status "Setting cost.commit_connection=743 (gosdk estimates 743, chain default 744)"
    run_cmd "sc-update-config cost.commit_connection" \
        "$ZWALLET sc-update-config --keys 'cost.commit_connection' --values '743' $WO"

    # ========== Miner/Sharder num_delegates + delegate_wallet ==========
    local sc_owner_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)
    print_status "Setting miner/sharder num_delegates=200, delegate_wallet=${sc_owner_id:0:16}..."

    local miner_ids=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)
    for miner_id in $miner_ids; do
        print_status "  Setting miner ${miner_id:0:16}... num_delegates=200"
        $ZWALLET mn-update-settings \
            --id "$miner_id" \
            --num_delegates 200 \
            --delegate_wallet "$sc_owner_id" \
            $W --silent 2>&1 || print_warning "  mn-update-settings failed for ${miner_id:0:16}"
        sleep 1
    done

    local sharder_ids=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList" 2>/dev/null \
        | python3 -c "import json,sys; [print(n['id']) for n in json.load(sys.stdin).get('Nodes',[])]" 2>/dev/null)
    for sharder_id in $sharder_ids; do
        print_status "  Setting sharder ${sharder_id:0:16}... num_delegates=200"
        $ZWALLET mn-update-settings \
            --id "$sharder_id" \
            --num_delegates 200 \
            --delegate_wallet "$sc_owner_id" \
            $W --silent 2>&1 || print_warning "  mn-update-settings failed for ${sharder_id:0:16}"
        sleep 1
    done

    # ========== Blobber Config via bl-update ==========
    print_status "Configuring all blobbers (write_price=0.001, read_price=0, service_charge=0.1, num_delegates=200, storage_version=1)..."

    local blobber_json=$($ZBOX ls-blobbers --json --silent $W 2>/dev/null || echo "[]")
    local blobber_ids=$(echo "$blobber_json" | jq -r '.[].id' 2>/dev/null || true)

    local count=0
    for blobber_id in $blobber_ids; do
        count=$((count + 1))
        print_status "  Updating blobber $count: ${blobber_id:0:16}..."
        $ZBOX bl-update \
            --blobber_id "$blobber_id" \
            --write_price 0.001 \
            --read_price 0 \
            --service_charge 0.1 \
            --num_delegates 200 \
            --storage_version 1 \
            $W --silent 2>&1 || print_warning "  bl-update failed for ${blobber_id:0:16}"
        sleep 1
    done
    print_status "Updated $count blobbers"

    # ========== Validator Config ==========
    print_status "Configuring all validators (num_delegates=200)..."
    local validator_json=$($ZBOX ls-validators --json --silent $W 2>/dev/null || echo "[]")
    local validator_ids=$(echo "$validator_json" | jq -r '.[] | (.validator_id // .id)' 2>/dev/null || true)

    local vcount=0
    for validator_id in $validator_ids; do
        if [ -z "$validator_id" ] || [ "$validator_id" = "null" ]; then
            continue
        fi
        vcount=$((vcount + 1))
        print_status "  Updating validator $vcount: ${validator_id:0:16}..."
        $ZBOX validator-update \
            --validator_id "$validator_id" \
            --num_delegates 200 \
            $W --silent 2>/dev/null || print_warning "  validator-update failed for ${validator_id:0:16}"
        sleep 1
    done
    print_status "Updated $vcount validators"

    # ========== Validator Staking ==========
    # Stake validators so they can participate in challenges.
    # Without staking, validators are filtered out and challenges cannot be generated.
    # validators_per_challenge=2, so at least 2 validators need stake > 0.
    # 20 ZCN per validator. With 12+ validators, we need 240+ ZCN locked.
    # Fund owner wallet for validator staking (100 ZCN per pour, 1 pour enough for ~5 validators)
    print_status "Funding owner wallet for validator staking (100 ZCN)..."
    $ZWALLET faucet --methodName pour --input '{}' --tokens 100 $W 2>/dev/null || true
    sleep 0.5
    print_status "Staking all validators (20 ZCN each, with retries)..."
    local vstake_count=0
    local vstake_failed=0
    for validator_id in $validator_ids; do
        if [ -z "$validator_id" ] || [ "$validator_id" = "null" ]; then
            continue
        fi
        vstake_count=$((vstake_count + 1))
        print_status "  Staking validator $vstake_count: ${validator_id:0:16}..."
        local stake_output stake_ok=0
        for _retry in 1 2 3; do
            stake_output=$($ZBOX sp-lock \
                --validator_id "$validator_id" \
                --tokens 20 \
                $W --silent 2>&1) && { stake_ok=1; break; }
            if echo "$stake_output" | grep -qi "already.*stak\|already.*locked\|stake_pool_lock"; then
                print_status "  Validator ${validator_id:0:16} already staked (skipping)"
                stake_ok=1; break
            fi
            print_warning "  sp-lock attempt $_retry failed for ${validator_id:0:16}: $stake_output"
            sleep 3
        done
        [ $stake_ok -eq 0 ] && vstake_failed=$((vstake_failed + 1))
        sleep 1
    done
    print_status "Staked $vstake_count validators ($vstake_failed failed)"
    if [ $vstake_failed -gt 0 ]; then
        print_warning "WARNING: $vstake_failed validator(s) failed to stake — challenges may fail (need validators_per_challenge=2 staked validators)"
    fi

    # ========== Verification ==========
    print_status "Verifying chain configuration..."

    local sc_addr="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local mn_addr="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"

    # Verify storage SC config
    curl -s "http://198.18.0.82:7172/v1/screst/${sc_addr}/storage-config" 2>/dev/null | python3 -c "
import sys,json
try:
    f=json.load(sys.stdin).get('fields',{})
    print(f'  time_unit:           {f.get("time_unit", "N/A")}')
    print(f'  min_alloc_size:      {f.get("min_alloc_size", "N/A")}')
    print(f'  min_write_price:     {f.get("min_write_price", "N/A")}')
    print(f'  challenge_enabled:   {f.get("challenge_enabled", "N/A")}')
    print(f'  health_check_period: {f.get("health_check_period", "N/A")}')
except Exception as e:
    print(f'  Could not parse storage config: {e}')
" 2>/dev/null || print_warning "Could not verify storage config"

    # Verify miner SC config
    curl -s "http://198.18.0.82:7172/v1/screst/${mn_addr}/configs" 2>/dev/null | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    fields = d if isinstance(d, dict) else {}
    print(f'  min_n:               {fields.get("min_n", "N/A")}')
    print(f'  min_s:               {fields.get("min_s", "N/A")}')
    print(f'  k_percent:           {fields.get("k_percent", "N/A")}')
except Exception as e:
    print(f'  Could not parse miner config: {e}')
" 2>/dev/null || print_warning "Could not verify miner config"

    # Verify blobber count and prices
    local active_blobbers=$($ZBOX ls-blobbers --json --silent $W 2>/dev/null || echo "[]")
    local active_count=$(echo "$active_blobbers" | jq 'length' 2>/dev/null || echo "0")
    local zero_write=$(echo "$active_blobbers" | jq '[.[] | select(.terms.write_price <= 10000000)] | length' 2>/dev/null || echo "0")
    print_status "  Active blobbers: $active_count (low write_price: $zero_write)"

    print_status "Chain configuration ensured successfully!"
}

# Start zs3server
# Write .env files for web-apps packages (Vult, Blimp, etc.)
# These env files configure Firebase, blockchain bridges, Auth0, and app settings
# for the dev environment. They are required for building and running the apps.
setup_web_app_env_files() {
    local WEB_APPS_DIR="$1"

    # Get deployment domain (defaults to test.zus.network)
    local DEPLOY_DOMAIN="${NGINX_DOMAIN:-}"
    if [ -z "$DEPLOY_DOMAIN" ] && [ -f "$CONFIG_FILE" ]; then
        DEPLOY_DOMAIN=$(grep "^  domain:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    fi
    DEPLOY_DOMAIN="${DEPLOY_DOMAIN:-test.zus.network}"

    # Generic .env template — same keys for all apps, only ZBOX_APP differs per app
    # Secrets are loaded from .secrets.env (via env vars) — NOT hardcoded here.
    # Non-secret values (contract addresses, chain IDs) are kept inline.
    # NOTE: FBASE_AUTH_DOMAIN is the Firebase project's auth domain (*.firebaseapp.com),
    # NOT the deployment domain. Set it in .secrets.env for your Firebase project.
    local ENV_TEMPLATE
    read -r -d '' ENV_TEMPLATE << ENVEOF || true
NFT_CHAIN_ID=0x89
ETH_CHAIN_ID=0x1
FBASE_API_KEY=${FBASE_API_KEY}
FBASE_AUTH_DOMAIN=${FBASE_AUTH_DOMAIN}
FBASE_DB_URL=${FBASE_DB_URL:-https://box-dev-ce8bf.firebaseio.com}
FBASE_PROJECT_ID=${FBASE_PROJECT_ID:-box-dev-ce8bf}
FBASE_STORAGE_BUCKET=${FBASE_STORAGE_BUCKET:-box-dev-ce8bf.appspot.com}
FBASE_MESSAGING_SENDER_ID=${FBASE_MESSAGING_SENDER_ID:-893964718514}
FBASE_APP_ID=${FBASE_APP_ID:-1:893964718514:web:6d6f2ee9f96211e64954ab}
FBASE_SHARE_LINK=${FBASE_SHARE_LINK:-https://zuspublicdev.page.link}
NODE_ENV=development
APP_ENV=local
WEBHOOK_API_TOKEN=${WEBHOOK_API_TOKEN:?".secrets.env missing WEBHOOK_API_TOKEN"}
ZENDESK_KEY=${ZENDESK_KEY:?".secrets.env missing ZENDESK_KEY"}
ETH_TOKEN=0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE
BANCOR_NETWORK=0xeEF417e1D5CC832e619ae18D2F140De2999dD4fB
ETH_NODE_URL=${ETH_NODE_URL:?".secrets.env missing ETH_NODE_URL"}
NFT_NODE_URL=${NFT_NODE_URL:?".secrets.env missing NFT_NODE_URL"}
RECAPTCHA_KEY=${RECAPTCHA_KEY:?".secrets.env missing RECAPTCHA_KEY"}
DOMAIN=DOMAIN_PLACEHOLDER
ZBOX_APP=APP_PLACEHOLDER
JWT_ENABLED=false
BE_URL=https://lp-backend.zus.network
ZCN_TOKEN=0xb9EF770B6A5e12E45983C5D80545258aA38F3B78
MOCK_ZCN_TOKEN=0xb9EF770B6A5e12E45983C5D80545258aA38F3B78
UNISWAP_V2_SWAP_ADDRESS=0x2d899d91a7ccd2126e03459ea08f6a8ad3342289
UNISWAP_V2_ROUTER02_ADDRESS=0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D
BRIDGE=0x7700D773022b19622095118Fadf46f7B9448Be9b
AUTHORIZER_ADDRESS=0x481daB4407b9880DE0A68dc62E6aF611c4949E42
REVERSE_BRIDGE=0x7700D773022b19622095118Fadf46f7B9448Be9b
BANCOR_TOKEN_ADDRESS=0x1F573D6Fb3F13d689FF844B4cE37794d79a7FF1C
USDC_TOKEN_ADDRESS=0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48
EURC_TOKEN_ADDRESS=0x1aBaEA1f7C830bD89Acc67eC4af516284b1bC33c
ALCHEMY_API_KEY=${ALCHEMY_API_KEY:?".secrets.env missing ALCHEMY_API_KEY"}
NFT_FACTORY_ADDRESS=0x87747794eFD812a0e4783818941359f44a48940a
NFT_FACTORY_MODULE_ERC_721_ADDRESS=0x2f9f0d893f00e633e552fb6c7c57fb29a8c61ede
NFT_FACTORY_MODULE_ERC_721_FIXED_ADDRESS=0xe3cDc03417b8B713271C714f1D4886B80C7B86e6
NFT_FACTORY_MODULE_ERC_721_PACK_ADDRESS=0xa27719459ee42AC07a625Ef874478a2bec984308
NFT_FACTORY_MODULE_ERC_721_RANDOM_ADDRESS=0x016aa145ae9f896cbB3611D5fA98c9F6b4A52EBd
APP_VERSION=1.15.4
AUTH0_SECRET=${AUTH0_SECRET:?".secrets.env missing AUTH0_SECRET"}
AUTH0_BASE_URL=https://APP_DOMAIN_PLACEHOLDER
AUTH0_ISSUER_BASE_URL=${AUTH0_ISSUER_BASE_URL:?".secrets.env missing AUTH0_ISSUER_BASE_URL"}
AUTH0_CLIENT_ID=${AUTH0_CLIENT_ID:?".secrets.env missing AUTH0_CLIENT_ID"}
AUTH0_CLIENT_SECRET=${AUTH0_CLIENT_SECRET:?".secrets.env missing AUTH0_CLIENT_SECRET"}
GOSDK_VERSION=1.20.4
EGOSDK_VERSION=1.19.0
USE_CACHED_WASM=false
DATALAKE_API_URL=https://datalake.blimp.software
ENVEOF

    # Per-app domains for AUTH0_BASE_URL (each app has its own subdomain)
    local _prefix="${APP_DOMAIN_PREFIX:-test}"
    local -A APP_DOMAIN_MAP=(
        [vult]="${_prefix}.vult.network"
        [bolt]="${_prefix}.bolt.holdings"
        [blimp]="${_prefix}.blimp.software"
        [explorer]="${_prefix}.atlus.cloud"
        [chimney]="${_prefix}.chimney.software"
    )

    # Write .env for each web app, substituting ZBOX_APP, DOMAIN (backend), and per-app AUTH0_BASE_URL
    local WEB_APPS_LIST="vult bolt blimp explorer chimney"
    for app in $WEB_APPS_LIST; do
        local app_dir="${WEB_APPS_DIR}/packages/${app}"
        if [ -d "$app_dir" ]; then
            local app_domain="${APP_DOMAIN_MAP[$app]:-${DEPLOY_DOMAIN}}"
            # NOTE: APP_DOMAIN_PLACEHOLDER must be replaced BEFORE DOMAIN_PLACEHOLDER,
            # because DOMAIN_PLACEHOLDER is a substring of APP_DOMAIN_PLACEHOLDER.
            # Wrong order would turn "https://APP_DOMAIN_PLACEHOLDER" into "https://APP_test1.zus.network".
            echo "$ENV_TEMPLATE" | \
                sed "s/APP_DOMAIN_PLACEHOLDER/${app_domain}/g" | \
                sed "s/DOMAIN_PLACEHOLDER/${DEPLOY_DOMAIN}/g" | \
                sed "s/APP_PLACEHOLDER/${app}/g" > "${app_dir}/.env"
            print_status "Wrote ${app}/.env (backend: ${DEPLOY_DOMAIN}, app_url: ${app_domain})"
        fi
    done

    # Remove any .env.local files that may override our generated .env.
    # In Next.js, .env.local has HIGHEST priority (overrides .env completely).
    # Stale .env.local files from previous deploys can contain wrong domains (e.g. ent.zus.network).
    for app in $WEB_APPS_LIST; do
        local env_local="${WEB_APPS_DIR}/packages/${app}/.env.local"
        if [ -f "$env_local" ]; then
            rm -f "$env_local"
            print_status "Removed stale ${app}/.env.local (would override generated .env)"
        fi
    done

    # SimpleLocalize fix: Remove NEXT_PUBLIC_ORIGIN if present in any app .env
    # When NEXT_PUBLIC_ORIGIN points to a dead port (e.g. localhost:3430),
    # the i18n translation loader fails silently and raw keys show on the page.
    # Without this var, apps correctly fall through to the SimpleLocalize CDN.
    for app in $WEB_APPS_LIST; do
        local app_env="${WEB_APPS_DIR}/packages/${app}/.env"
        if [ -f "$app_env" ] && grep -q "NEXT_PUBLIC_ORIGIN" "$app_env"; then
            sed -i.tmp '/^NEXT_PUBLIC_ORIGIN=/d' "$app_env"
            rm -f "${app_env}.tmp"
            print_status "Removed NEXT_PUBLIC_ORIGIN from ${app}/.env (SimpleLocalize fix)"
        fi
    done
}

# Build web-apps (Vult, Bolt, Blimp, Explorer, Chimney)
# Chalk is excluded — it's not part of the test deployment.
# Requires: gosdk repo for zcn.wasm build, Node.js/yarn
build_web_apps() {
    print_header "Building Web Apps"

    local WEB_APPS_DIR="${BASE_DIR}/web-apps"
    if [ ! -d "$WEB_APPS_DIR" ]; then
        print_warning "web-apps repo not found at $WEB_APPS_DIR, skipping"
        return 0
    fi

    # Pull latest web-apps from the stable test branch.
    # Force-checkout discards local changes (e.g. zcn.wasm binary, stale .env edits)
    # before pulling — we regenerate .env and recopy WASM below anyway.
    print_status "Updating web-apps to fix/atlus-explorer-improvements..."
    (
        cd "$WEB_APPS_DIR"
        git fetch origin fix/atlus-explorer-improvements 2>/dev/null || true
        git checkout -f fix/atlus-explorer-improvements 2>/dev/null || true
        git reset --hard origin/fix/atlus-explorer-improvements 2>/dev/null || \
            git pull origin fix/atlus-explorer-improvements 2>/dev/null || \
            print_warning "web-apps git pull failed (using existing code)"
        print_status "web-apps at: $(git log --oneline -1 2>/dev/null)"
    )

    # Check for required secrets before attempting env file setup
    # The setup_web_app_env_files function uses ${:?} which kills the shell if secrets are missing
    local SECRETS_FILE="${SCRIPT_DIR}/.secrets.env"
    if [ -f "$SECRETS_FILE" ]; then
        source "$SECRETS_FILE"
    fi
    if [ -z "${FBASE_API_KEY:-}" ]; then
        print_warning "web-apps secrets not found (.secrets.env missing FBASE_API_KEY), skipping web-apps build"
        return 0
    fi

    # Write generic .env files for all apps (Firebase, Auth0, blockchain config)
    # Domain is set from deploy_config.yaml or NGINX_DOMAIN env var
    setup_web_app_env_files "$WEB_APPS_DIR"

    # Apply dev-only patches to web-apps source (NEXTAUTH_SECRET, user_id in OTP body)
    patch_web_apps_for_dev

    # Build zcn.wasm from gosdk (required by all web apps)
    # Uses gosdk_branch from web-apps config section, falls back to gosdk.branch, then master
    local GOSDK_DIR="${BASE_DIR}/gosdk"

    if [ -d "$GOSDK_DIR" ]; then
        # Checkout the correct gosdk branch for web-apps WASM build
        checkout_gosdk_for_dependent "web-apps"
        cd "$GOSDK_DIR"

        # Patch: allow setWallet with privateKey but no mnemonic (fixes "mnemonic is required" on login/download)
        if grep -q 'mnemonic == "" && !isSplit {' wasmsdk/wallet.go 2>/dev/null; then
            sed -i 's/if mnemonic == "" \&\& !isSplit {/if mnemonic == "" \&\& !isSplit \&\& privateKey == "" {/' wasmsdk/wallet.go
            print_status "gosdk wasmsdk/wallet.go patched (privateKey guard added)"
        fi

        # Build WASM
        print_status "Building zcn.wasm from gosdk ($(git log --oneline -1 2>/dev/null))..."
        CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-s -w" -buildvcs=false -o zcn.wasm ./wasmsdk 2>/dev/null || {
            print_warning "zcn.wasm build failed (may need specific gosdk branch)"
        }

        if [ -f "$GOSDK_DIR/zcn.wasm" ]; then
            local wasm_size=$(du -h "$GOSDK_DIR/zcn.wasm" | cut -f1)
            print_status "Built zcn.wasm (${wasm_size})"
            # Copy to web-apps packages that need it (src dirs for nft-core-js/zus-sdk)
            for pkg_dir in "${WEB_APPS_DIR}/packages/nft-core-js/src/wasm" \
                           "${WEB_APPS_DIR}/packages/zus-sdk/src/wasm"; do
                if [ -d "$(dirname "$pkg_dir")" ]; then
                    mkdir -p "$pkg_dir"
                    cp "$GOSDK_DIR/zcn.wasm" "$pkg_dir/"
                    print_status "Copied zcn.wasm to $pkg_dir"
                fi
            done
            # Copy to public dirs so browsers get local WASM
            for app in blimp vult bolt shared chimney explorer; do
                local pub_dir="${WEB_APPS_DIR}/packages/${app}/public"
                if [ -d "$pub_dir" ]; then
                    cp "$GOSDK_DIR/zcn.wasm" "$pub_dir/zcn.wasm"
                    print_status "Copied zcn.wasm to ${app}/public/"
                fi
            done
        fi
    else
        print_warning "gosdk not found at $GOSDK_DIR, cannot build zcn.wasm"
    fi

    # Patch WASM loader to use local /zcn.wasm instead of CDN
    patch_wasm_loader_local "$WEB_APPS_DIR"

    # Install dependencies and build web apps
    cd "$WEB_APPS_DIR"

    # This monorepo uses yarn workspaces — yarn MUST be installed.
    # The npm run scripts (vult:build, explorer:build, etc.) call "yarn workspace ..." internally,
    # so falling back to npm alone is not sufficient. Install yarn if missing.
    if ! command -v yarn &>/dev/null; then
        print_status "yarn not found, installing via npm..."
        npm install -g yarn 2>/dev/null || {
            print_warning "Failed to install yarn, cannot build web-apps"
            return 0
        }
    fi
    local PKG_MGR="yarn"

    print_status "Installing web-app dependencies with ${PKG_MGR}..."
    $PKG_MGR install 2>/dev/null || {
        print_warning "${PKG_MGR} install failed for web-apps"
        return 0
    }

    # Build shared package first (dependency for other apps)
    # Sync shared/.env from vult .env so shared components use correct network config
    if [ -f "${WEB_APPS_DIR}/packages/vult/.env" ]; then
        cp "${WEB_APPS_DIR}/packages/vult/.env" "${WEB_APPS_DIR}/packages/shared/.env"
        print_status "Synced shared/.env from vult/.env (domain: ${DEPLOY_DOMAIN})"
    fi
    print_status "Building shared package..."
    yarn workspace shared build 2>/dev/null || $PKG_MGR run shared:build 2>/dev/null || true

    # Remove basePath from each app's next.config.js (apps serve from / on their own subdomain)
    local WEB_APPS_TO_BUILD="vult bolt blimp explorer chimney"
    for app in $WEB_APPS_TO_BUILD; do
        local next_config="${WEB_APPS_DIR}/packages/${app}/next.config.js"
        if [ -f "$next_config" ] && grep -q "basePath" "$next_config"; then
            sed -i.tmp "/^[[:space:]]*basePath:/d" "$next_config"
            rm -f "${next_config}.tmp"
            print_status "Removed basePath from ${app}/next.config.js (subdomain mode)"
        fi
    done

    # Build each app individually (excludes chalk)
    for app in $WEB_APPS_TO_BUILD; do
        print_status "Building ${app}..."
        yarn workspace "$app" build 2>/dev/null || $PKG_MGR run "${app}:build" 2>/dev/null || {
            print_warning "${app} build failed (non-critical)"
        }
    done

    print_status "Web apps build complete (built: ${WEB_APPS_TO_BUILD})"

    # Fix prerender-manifest.json for each app: the .js source only stores the preview section.
    # Next.js 13.x requires the full JSON (version, routes, dynamicRoutes, preview, notFoundRoutes).
    for app in $WEB_APPS_TO_BUILD; do
        local jsonf="${WEB_APPS_DIR}/packages/${app}/.next/prerender-manifest.json"
        local jsf="${WEB_APPS_DIR}/packages/${app}/.next/prerender-manifest.js"
        if [ ! -f "$jsonf" ] || ! python3 -c "import json; d=json.load(open('$jsonf')); assert 'version' in d" 2>/dev/null; then
            # Extract preview section from .js source or existing malformed JSON, write full format
            python3 - "$jsf" "$jsonf" << 'PYEOF'
import json, sys, re

js_file = sys.argv[1]
json_file = sys.argv[2]
preview = {}
try:
    # Try reading existing JSON first
    with open(json_file) as f:
        d = json.load(f)
    if 'preview' in d:
        preview = d['preview']
except:
    pass
if not preview:
    try:
        # Extract from .js: self.__PRERENDER_MANIFEST="..."
        with open(js_file) as f:
            content = f.read()
        m = re.search(r'self\.__PRERENDER_MANIFEST=["\'](.*)["\']', content)
        if m:
            d = json.loads(m.group(1).replace('\\"', '"'))
            preview = d.get('preview', {})
    except:
        pass
manifest = {"version": 4, "routes": {}, "dynamicRoutes": {}, "preview": preview, "notFoundRoutes": []}
with open(json_file, 'w') as f:
    json.dump(manifest, f)
PYEOF
            print_status "Fixed prerender-manifest.json for ${app}"
        fi
    done

    # Start web apps as Next.js servers (much faster than Docker build)
    start_web_apps "$WEB_APPS_DIR"
}

# Start web apps via next start (no Docker required)
start_web_apps() {
    local WEB_APPS_DIR="${1:-${BASE_DIR}/web-apps}"
    print_header "Starting Web App Servers"

    # Use PM2 for process management (auto-restart, logging)
    if ! command -v pm2 &>/dev/null; then
        npm install -g pm2 2>/dev/null || {
            print_warning "PM2 not available, falling back to nohup"
        }
    fi

    # App -> Port mapping (matches nginx proxy_pass config)
    local -A APP_PORTS=( [vult]=3003 [bolt]=3002 [blimp]=3006 [explorer]=3001 [chimney]=3005 )

    # NextAuth requires NEXTAUTH_SECRET in production. Use AUTH0_SECRET from .secrets.env
    # (same value, already exported). pm2 stores env at start time in dump.pm2, so we
    # must pass it explicitly here — pm2 restart does NOT re-read .env files.
    export NEXTAUTH_SECRET="${AUTH0_SECRET:-$(openssl rand -hex 32)}"

    # Stop existing PM2 processes
    if command -v pm2 &>/dev/null; then
        pm2 delete all 2>/dev/null || true
        sleep 1
    else
        pkill -f "next start" 2>/dev/null || true
        sleep 1
    fi

    # SimpleLocalize fix: Remove NEXT_PUBLIC_ORIGIN if set to a wrong port (e.g. localhost:3430).
    # The i18n loadLocaleFrom runs server-side and reads this var at runtime; remove it so
    # apps fall through to the SimpleLocalize CDN instead of trying a dead local URL.
    for app in vult bolt blimp explorer chimney; do
        local app_env="${WEB_APPS_DIR}/packages/${app}/.env"
        if [ -f "$app_env" ] && grep -q "^NEXT_PUBLIC_ORIGIN=" "$app_env"; then
            sed -i.tmp '/^NEXT_PUBLIC_ORIGIN=/d' "$app_env"
            rm -f "${app_env}.tmp"
            print_status "Removed NEXT_PUBLIC_ORIGIN from ${app}/.env (SimpleLocalize fix)"
        fi
    done

    # Kill stale next-router-worker processes from previous deploys.
    # After clean.sh + redeploy, old Next.js worker processes can survive and hold ports
    # (3001-3006), causing EADDRINUSE when PM2 tries to start the new apps.
    print_status "Killing stale next-router-worker processes..."
    if command -v pm2 &>/dev/null; then
        pm2 delete all 2>/dev/null || true
    fi
    pkill -f 'next-router-worker' 2>/dev/null || true
    pkill -f 'next start -p 300' 2>/dev/null || true
    sleep 2

    for app in vult bolt blimp explorer chimney; do
        local port=${APP_PORTS[$app]}
        local app_dir="${WEB_APPS_DIR}/packages/${app}"

        if [ ! -d "${app_dir}/.next" ]; then
            print_warning "${app}: no .next build output found, skipping"
            continue
        fi

        if command -v pm2 &>/dev/null; then
            pm2 start "npx next start -p ${port}" --name "$app" --cwd "$app_dir" 2>/dev/null || {
                print_warning "${app}: PM2 start failed, trying nohup"
                cd "$app_dir"
                nohup npx next start -p "$port" > "/tmp/webapp_${app}.log" 2>&1 &
            }
        else
            cd "$app_dir"
            nohup npx next start -p "$port" > "/tmp/webapp_${app}.log" 2>&1 &
        fi
        print_status "${app} started on port ${port}"
    done

    # Save PM2 config for persistence across reboots
    if command -v pm2 &>/dev/null; then
        pm2 save 2>/dev/null || true
    fi

    # Wait and verify servers
    sleep 5
    for app in vult bolt blimp explorer chimney; do
        local port=${APP_PORTS[$app]}
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "http://localhost:${port}/${app}/" 2>/dev/null || echo "000")
        if [ "$code" = "200" ]; then
            print_status "${app}: HTTP 200 on port ${port}"
        else
            print_warning "${app}: HTTP ${code} on port ${port} (may need a moment to start)"
        fi
    done
}

# Build rclone with Zus backend
# Requires: rclone_zus repo, CGO_ENABLED=1
build_rclone_zus() {
    print_header "Building rclone-zus"

    local RCLONE_DIR="${BASE_DIR}/rclone_zus"
    local RCLONE_ZUS_BRANCH="${RCLONE_ZUS_BRANCH:-feat/zus-backend}"

    # Clone repo if not present
    if [ ! -d "$RCLONE_DIR" ]; then
        print_status "Cloning rclone_zus repo..."
        git clone https://github.com/0chain/rclone.git "$RCLONE_DIR" 2>&1 || {
            print_warning "Failed to clone rclone_zus repo — skipping rclone-zus build"
            return 0
        }
    fi

    cd "$RCLONE_DIR"

    # Ensure we're on the branch that includes the Zus backend.
    # master branch does NOT have the zus backend — feat/zus-backend does.
    # Use full refspec fetch + create local tracking branch if not present.
    local current_branch
    current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    if [ "$current_branch" != "$RCLONE_ZUS_BRANCH" ]; then
        print_status "Switching rclone_zus to $RCLONE_ZUS_BRANCH (currently on $current_branch)..."
        git fetch origin "refs/heads/${RCLONE_ZUS_BRANCH}:refs/remotes/origin/${RCLONE_ZUS_BRANCH}" 2>/dev/null || true
        git checkout "$RCLONE_ZUS_BRANCH" 2>/dev/null \
            || git checkout -b "$RCLONE_ZUS_BRANCH" "origin/${RCLONE_ZUS_BRANCH}" 2>/dev/null || {
            print_warning "Failed to switch rclone_zus to $RCLONE_ZUS_BRANCH — build may fail"
        }
    else
        git pull origin "$RCLONE_ZUS_BRANCH" 2>/dev/null || true
    fi

    # Checkout correct gosdk branch for rclone_zus
    checkout_gosdk_for_dependent "rclone_zus"

    print_status "Building rclone with Zus backend..."
    CGO_ENABLED=1 go build -tags bn256 -o rclone rclone.go 2>/dev/null || {
        print_warning "rclone-zus build failed"
        return 0
    }

    if [ -f "$RCLONE_DIR/rclone" ]; then
        print_status "rclone-zus binary built successfully"

        # Install to /usr/local/bin as rclone-zus (tests expect this name)
        cp "$RCLONE_DIR/rclone" /usr/local/bin/rclone-zus
        print_status "Installed rclone-zus to /usr/local/bin/rclone-zus"

        # Setup rclone config for automation testing
        local RCLONE_CONFIG_DIR="${HOME}/.config/rclone"
        mkdir -p "$RCLONE_CONFIG_DIR"

        # Create rclone config pointing to local chain (if allocation exists)
        if [ -f "${ZCN_CONFIG_DIR}/allocation.txt" ]; then
            local alloc_id=$(cat "${ZCN_CONFIG_DIR}/allocation.txt")
            cat > "${RCLONE_CONFIG_DIR}/rclone.conf" <<EOF
[automation]
type = zus
allocation_id = ${alloc_id}
config_dir = ${ZCN_CONFIG_DIR}
EOF
            print_status "rclone config written to ${RCLONE_CONFIG_DIR}/rclone.conf"
        else
            print_warning "No allocation.txt found, rclone config not written"
        fi
    fi
}

# Check if the ZS3 server's allocation is still valid; recreate + restart if expired.
# Reads allocation ID from /root/.zcn/allocation.txt, checks expiry via sharder REST API.
renew_zs3_allocation() {
    local ZS3_ALLOC_FILE="${ZCN_CONFIG_DIR}/allocation.txt"
    local now
    now=$(date +%s)

    # Get current allocation ID
    local alloc_id=""
    if [ -f "$ZS3_ALLOC_FILE" ]; then
        alloc_id=$(cat "$ZS3_ALLOC_FILE")
    fi

    if [ -z "$alloc_id" ]; then
        print_status "No ZS3 allocation found — creating one..."
        start_zs3server
        return
    fi

    # Check expiry via sharder REST
    local expiry
    expiry=$(curl -sf "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation?allocation=${alloc_id}" 2>/dev/null | \
        python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('expiration_date',0))" 2>/dev/null || echo "0")

    local margin=$((2 * 24 * 3600))  # 2 days (allocation may only last ~25 days total)
    if [ "$expiry" -gt "$((now + margin))" ] 2>/dev/null; then
        print_status "ZS3 allocation ${alloc_id:0:16}... is valid (expires $(date -d @$expiry 2>/dev/null || date -r $expiry 2>/dev/null || echo $expiry))"

        # Verify the running ZS3 server is using THIS allocation; restart if mismatched or not running.
        local running_alloc
        running_alloc=$(pgrep -a -f "minio gateway zcn" 2>/dev/null | grep -o -- '--allocationId [a-f0-9]\{64\}' | awk '{print $2}' | head -1)
        if [ -z "$running_alloc" ]; then
            print_status "ZS3 server not running — starting it..."
            pkill -f "minio gateway zcn" 2>/dev/null || true
            sleep 2
            start_zs3server
            return
        elif [ "$running_alloc" = "$alloc_id" ]; then
            # Allocation matches. Check for write_marker_verification_failed errors in recent logs:
            # these occur when minio was restarted with an existing allocation that has blobber state
            # it can't reconcile (prev_allocation_root mismatch). A fresh allocation fixes this.
            local write_marker_errors=0
            write_marker_errors=$(tail -100 /var/log/zs3server.log 2>/dev/null | grep -c "write_marker_verification_failed" 2>/dev/null || true)
            write_marker_errors="${write_marker_errors:-0}"
            if (( write_marker_errors > 0 )); then
                print_status "ZS3 server has write_marker errors (stale blobber state) — creating fresh allocation..."
                pkill -f "minio gateway zcn" 2>/dev/null || true
                sleep 2
                start_zs3server
                return
            fi
            # Check for repair_required errors from last test run — means blobbers have inconsistent
            # state from a partially-committed write; a fresh allocation clears this.
            # Only check recently-modified logs (within 1 hour) to avoid false positives from
            # previous sessions.
            local repair_required_count=0
            local zs3_log="/tmp/test_zs3.log"
            if [ -f "$zs3_log" ] && (( $(date +%s) - $(stat -c %Y "$zs3_log" 2>/dev/null || echo 0) < 3600 )); then
                repair_required_count=$(grep -c "repair_required" "$zs3_log" 2>/dev/null || true)
                repair_required_count="${repair_required_count:-0}"
            fi
            if (( repair_required_count > 0 )); then
                print_status "ZS3 allocation has repair_required state (${repair_required_count} occurrences) — creating fresh allocation..."
                pkill -f "minio gateway zcn" 2>/dev/null || true
                sleep 2
                start_zs3server
                return
            fi
            print_status "ZS3 server running with correct allocation — OK"
            return
        else
            # Running minio uses a different allocation — zcnconfig state is stale.
            # Create a fresh allocation to avoid write_marker_verification_failed errors.
            print_status "ZS3 server running with wrong allocation (${running_alloc:0:16}... ≠ ${alloc_id:0:16}...) — creating fresh allocation..."
            pkill -f "minio gateway zcn" 2>/dev/null || true
            sleep 2
            start_zs3server
            return
        fi
    fi

    print_status "ZS3 allocation expired or expiring soon (expiry=$expiry, now=$now) — creating new allocation and restarting zs3server..."
    # Stop existing zs3server
    pkill -f "minio gateway zcn" 2>/dev/null || true
    sleep 2
    # start_zs3server creates a new allocation and starts the server
    start_zs3server
}

# Setup zs3/mc/warp test tools in CLI test directory
# Downloads mc and warp binaries for Linux, creates hosts.yaml config
setup_zs3_test_tools() {
    print_header "Setting Up zs3/mc/warp Test Tools"

    local CLI_TEST_DIR="${BASE_DIR}/system_test/tests/cli_tests"

    # Download mc (MinIO client) for Linux if not present or wrong architecture.
    # Check ELF magic bytes (7f 45 4c 46) via python3 — portable, no `file` command required.
    local mc_needs_download=false
    if [ ! -f "$CLI_TEST_DIR/mc" ]; then
        mc_needs_download=true
    elif ! python3 -c "
import sys
with open('$CLI_TEST_DIR/mc', 'rb') as f:
    magic = f.read(4)
sys.exit(0 if magic == b'\x7fELF' else 1)
" 2>/dev/null; then
        print_status "mc binary is wrong architecture (not ELF) — re-downloading Linux amd64..."
        mc_needs_download=true
    fi

    if [ "$mc_needs_download" = true ]; then
        print_status "Downloading mc (MinIO client) for linux-amd64..."
        curl -sL "https://dl.min.io/client/mc/release/linux-amd64/mc" \
            -o "$CLI_TEST_DIR/mc" 2>/dev/null && chmod +x "$CLI_TEST_DIR/mc" || {
            print_warning "Failed to download mc binary"
        }
    else
        print_status "mc binary already exists and works"
    fi

    # Also copy mc to mc_tests directory (tests expect mc there too)
    local MC_BIN_DIR="$CLI_TEST_DIR/mc_tests"
    if [ -d "$MC_BIN_DIR" ] && [ -f "$CLI_TEST_DIR/mc" ]; then
        cp "$CLI_TEST_DIR/mc" "$MC_BIN_DIR/mc" 2>/dev/null && chmod +x "$MC_BIN_DIR/mc" || true
    fi

    # Download warp (MinIO benchmark tool) for Linux if not present
    if [ ! -f "$CLI_TEST_DIR/warp" ]; then
        print_status "Downloading warp..."
        # Try direct download from MinIO releases first (most reliable)
        curl -sfL "https://dl.min.io/aistor/warp/release/linux-amd64/warp" -o "$CLI_TEST_DIR/warp" 2>/dev/null && \
            chmod +x "$CLI_TEST_DIR/warp" && \
            print_status "warp downloaded from dl.min.io" || {
            # Fallback to go install
            if command -v go &>/dev/null; then
                GOBIN="$CLI_TEST_DIR" go install github.com/minio/warp@latest 2>/dev/null || {
                    print_warning "Failed to install warp (both dl.min.io and go install failed)"
                }
            else
                print_warning "Failed to download warp and go not available"
            fi
        }
    else
        print_status "warp binary already exists"
    fi

    local MINIO_USER="${MINIO_ROOT_USER:-rootroot}"
    local MINIO_PASS="${MINIO_ROOT_PASSWORD:-rootroot}"

    # Create hosts.yaml for zs3server warp tests (flat key format required by ReadFile())
    local ZS3_TEST_DIR="$CLI_TEST_DIR/zs3server_tests"
    if [ -d "$ZS3_TEST_DIR" ]; then
        cat > "$ZS3_TEST_DIR/hosts.yaml" <<EOF
server: "localhost"
port: 9100
access_key: "${MINIO_USER}"
secret_key: "${MINIO_PASS}"
concurrent: 4
object_size: "1KiB"
object_count: 100
EOF
        print_status "Created hosts.yaml for zs3server tests"
    fi

    # Create mc_hosts.yaml for mc tests (flat key format required by ReadFileMC())
    local MC_TEST_DIR="$CLI_TEST_DIR/mc_tests"
    if [ -d "$MC_TEST_DIR" ]; then
        cat > "$MC_TEST_DIR/mc_hosts.yaml" <<EOF
server: "localhost"
port: 9100
secondary_server: "localhost"
secondary_port: 9100
access_key: "${MINIO_USER}"
secret_key: "${MINIO_PASS}"
concurrent: 4
use_command: false
EOF
        print_status "Created mc_hosts.yaml for mc tests"
    fi

    # Symlink rclone-zus binary (installed at /usr/local/bin) to cli_tests dir
    # rclone_zus_tests expect "../rclone-zus" relative to the test directory
    if [ -f "/usr/local/bin/rclone-zus" ] && [ ! -f "$CLI_TEST_DIR/rclone-zus" ]; then
        ln -sf /usr/local/bin/rclone-zus "$CLI_TEST_DIR/rclone-zus"
        print_status "Symlinked rclone-zus to cli_tests directory"
    elif [ -f "$CLI_TEST_DIR/rclone-zus" ]; then
        print_status "rclone-zus binary already exists in cli_tests"
    else
        print_warning "rclone-zus not found at /usr/local/bin/rclone-zus"
    fi

    # Set up mc alias for zs3server (required by mc_tests and zs3server_tests)
    if [ -f "$CLI_TEST_DIR/mc" ] && curl -sf http://localhost:9100/minio/health/live > /dev/null 2>&1; then
        print_status "Configuring mc aliases for zs3server..."
        # 'play' alias used by mc_tests
        "$CLI_TEST_DIR/mc" alias set play "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" --api S3v2 2>/dev/null || true
        # 'warp-test' alias used by zs3server_tests
        "$CLI_TEST_DIR/mc" alias set warp-test "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" --api S3v2 2>/dev/null || true
        # 'zus' alias used by some tokenomics/CLI tests
        "$CLI_TEST_DIR/mc" alias set zus "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" 2>/dev/null || true
        print_status "mc aliases configured (play, warp-test, zus)"
        # Also configure aliases for mc in mc_tests dir
        if [ -f "$MC_BIN_DIR/mc" ]; then
            "$MC_BIN_DIR/mc" alias set play "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" --api S3v2 2>/dev/null || true
            "$MC_BIN_DIR/mc" alias set warp-test "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" --api S3v2 2>/dev/null || true
            "$MC_BIN_DIR/mc" alias set zus "http://localhost:9100" "$MINIO_USER" "$MINIO_PASS" 2>/dev/null || true
        fi
    else
        print_warning "Skipping mc alias setup (mc binary or zs3server not available)"
    fi

    print_status "zs3/mc/warp/rclone test tools setup complete"
}

start_zs3server() {
    print_header "Starting zs3server"

    # Try to create an enterprise allocation first (backed by enterprise blobbers).
    # Auth ticket generation requires BLS signing via the Go helper in scripts/gen_auth_tickets/.
    local alloc_id=""
    local use_enterprise=false

    if [ -f "${SYSTEM_TEST_DIR}/scripts/gen_auth_tickets/main.go" ]; then
        print_status "Attempting enterprise allocation for zs3server..."
        # Always use local.json (the primary deploy wallet) for enterprise allocation auth tickets.
        # This is the same wallet used for all chain operations and is guaranteed to exist.
        local zbox_team_wallet="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"

        if [ -f "$zbox_team_wallet" ]; then
            local client_id
            client_id=$(python3 -c "import json,sys; print(json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}'))['client_id'])" 2>/dev/null || \
                        python3 -c "import json,sys; d=json.load(sys.stdin); print(d['client_id'])" < "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null || echo "")
            if [ -n "$client_id" ]; then
                local auth_info
                auth_info=$(cd "${BASE_DIR}/system_test" && \
                    go run ./scripts/gen_auth_tickets/ \
                    "http://localhost:7171" \
                    "$zbox_team_wallet" \
                    "$client_id" 2>/dev/null || echo "")
                if [ -n "$auth_info" ]; then
                    local eb_ids eb_tickets
                    eb_ids=$(echo "$auth_info" | cut -d'|' -f1)
                    eb_tickets=$(echo "$auth_info" | cut -d'|' -f2)
                    print_status "Got auth tickets for $(echo "$eb_ids" | tr ',' '\n' | wc -l) enterprise blobbers"
                    for attempt in 1 2 3; do
                        local alloc_output
                        alloc_output=$($ZBOX newallocation \
                            --enterprise \
                            --blobber_auth_tickets "$eb_tickets" \
                            --preferred_blobbers "$eb_ids" \
                            --auth_round_expiry 999999999 \
                            --data 2 --parity 1 \
                            --size 1073741824 \
                            --lock 10 \
                            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>&1 || true)
                        alloc_id=$(echo "$alloc_output" | grep -i 'Allocation created' | grep -o '[a-f0-9]\{64\}' | head -1)
                        if [ -z "$alloc_id" ]; then
                            alloc_id=$(echo "$alloc_output" | grep -o '[a-f0-9]\{64\}' | grep -v "$client_id" | tail -1)
                        fi
                        if [ -n "$alloc_id" ] && [ "$alloc_id" != "$client_id" ]; then
                            use_enterprise=true
                            break
                        fi
                        alloc_id=""
                        print_status "Enterprise allocation attempt $attempt failed, retrying in 10s..."
                        sleep 10
                    done
                fi
            fi
        fi
    fi

    # Fall back to regular allocation if enterprise allocation failed
    if [ -z "$alloc_id" ]; then
        if [ "$use_enterprise" = false ]; then
            print_status "Enterprise allocation not available — using regular allocation for zs3server"
        else
            print_warning "Enterprise allocation failed — falling back to regular allocation"
        fi
        # Get wallet client_id so we can exclude it from allocation ID parsing
        # (zbox prints wallet ID in SDK init logs — same 64-hex format as allocation IDs)
        local wallet_client_id
        wallet_client_id=$(python3 -c "import json; print(json.load(open('${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}'))['client_id'])" 2>/dev/null || echo "")
        for attempt in 1 2 3; do
            local alloc_output
            alloc_output=$($ZBOX newallocation \
                --data 2 --parity 2 \
                --size 1073741824 \
                --lock 10 \
                --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>&1 || true)
            # Parse allocation ID from "Allocation created: <hash>" line specifically.
            # Fallback: last 64-hex string that isn't the wallet client_id.
            alloc_id=$(echo "$alloc_output" | grep -i 'Allocation created' | grep -o '[a-f0-9]\{64\}' | head -1)
            if [ -z "$alloc_id" ]; then
                alloc_id=$(echo "$alloc_output" | grep -o '[a-f0-9]\{64\}' | grep -v "$wallet_client_id" | tail -1)
            fi
            if [ -n "$alloc_id" ] && [ "$alloc_id" != "$wallet_client_id" ]; then
                break
            fi
            alloc_id=""
            print_status "Allocation attempt $attempt failed, retrying in 10s..."
            sleep 10
        done
    fi

    if [ -n "$alloc_id" ]; then
        echo "$alloc_id" > "${ZCN_CONFIG_DIR}/allocation.txt"
        if [ "$use_enterprise" = true ]; then
            print_status "Created ENTERPRISE allocation for zs3server: ${alloc_id:0:16}..."
        else
            print_status "Created regular allocation for zs3server: ${alloc_id:0:16}..."
        fi
        # NOTE: --extend sets expiry = block_time + 1_time_unit (not cumulative).
        # Duration is determined by initial --lock amount: lock / cost_per_time_unit.
        # With empirical cost ~5 ZCN/time_unit and time_unit=720h, --lock 3000 gives ~600 time_units = 25 days.
        # Wait for blobbers to sync the new allocation from chain events before starting minio.
        # Without this, minio immediately queries blobbers which return "record not found".
        print_status "Waiting 30s for blobbers to sync new allocation..."
        sleep 30
        # Verify allocation exists on chain (vc.sh can cause tx to fail silently).
        local verified=false
        for verify_attempt in 1 2 3 4 5; do
            local alloc_on_chain
            alloc_on_chain=$(curl -sf "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation?allocation=${alloc_id}" 2>/dev/null | \
                python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id',''))" 2>/dev/null || echo "")
            if [ -n "$alloc_on_chain" ]; then
                verified=true
                print_status "Allocation ${alloc_id:0:16}... confirmed on chain"
                break
            fi
            print_status "Allocation not yet on chain (attempt $verify_attempt/5), retrying in 15s..."
            sleep 15
        done
        if [ "$verified" != "true" ]; then
            print_warning "Allocation ${alloc_id:0:16}... not confirmed on chain — tx may have failed. Retrying allocation creation..."
            alloc_id=""
            for attempt in 1 2 3; do
                local alloc_output2
                alloc_output2=$($ZBOX newallocation \
                    --data 2 --parity 2 \
                    --size 1073741824 \
                    --lock 10 \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>&1 || true)
                alloc_id=$(echo "$alloc_output2" | grep -i 'Allocation created' | grep -o '[a-f0-9]\{64\}' | head -1)
                if [ -z "$alloc_id" ]; then
                    alloc_id=$(echo "$alloc_output2" | grep -o '[a-f0-9]\{64\}' | grep -v "$wallet_client_id" | tail -1)
                fi
                if [ -n "$alloc_id" ] && [ "$alloc_id" != "${wallet_client_id:-}" ]; then
                    sleep 30
                    echo "$alloc_id" > "${ZCN_CONFIG_DIR}/allocation.txt"
                    print_status "Created replacement allocation: ${alloc_id:0:16}..."
                    break
                fi
                print_status "Replacement allocation attempt $attempt failed, retrying in 10s..."
                sleep 10
            done
        fi
    else
        print_warning "Could not create allocation. zs3server may not work correctly."
        if [ -f "${ZCN_CONFIG_DIR}/allocation.txt" ]; then
            print_status "Using existing allocation from ${ZCN_CONFIG_DIR}/allocation.txt"
        fi
    fi

    # Kill any existing minio gateway process
    pkill -f "minio gateway zcn" 2>/dev/null || true
    sleep 2

    # Start zs3server as a background process (not Docker)
    cd "${BASE_DIR}/zs3server"

    # Sync zs3server wallet to match the primary deploy wallet (local.json).
    # Mismatched wallets cause "allocation not found" when allocation was created
    # by local.json but zs3server runs with a different wallet.json.
    local ZS3_CONFIG_DIR="${BASE_DIR}/zs3server/zcnconfig"
    mkdir -p "$ZS3_CONFIG_DIR"
    cp "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" "${ZS3_CONFIG_DIR}/wallet.json" 2>/dev/null \
        && print_status "Synced zs3server wallet to ${ZCN_WALLET_FILE}" \
        || print_warning "Could not sync zs3server wallet — allocation mismatch may occur"
    # Also copy network config so zs3server can reach miners/sharders
    [ -f "${ZCN_CONFIG_DIR}/config.yaml" ] && cp "${ZCN_CONFIG_DIR}/config.yaml" "${ZS3_CONFIG_DIR}/config.yaml" 2>/dev/null || true
    local alloc_flags=""
    if [ -f "${ZCN_CONFIG_DIR}/allocation.txt" ]; then
        local saved_alloc_id=$(cat "${ZCN_CONFIG_DIR}/allocation.txt")
        alloc_flags="--configDir ${ZS3_CONFIG_DIR} --allocationId ${saved_alloc_id}"
    elif [ -f "${ZS3_CONFIG_DIR}/allocation.txt" ]; then
        local saved_alloc_id=$(cat "${ZS3_CONFIG_DIR}/allocation.txt")
        alloc_flags="--configDir ${ZS3_CONFIG_DIR} --allocationId ${saved_alloc_id}"
    fi

    # Free port 9100 if occupied by a non-minio process (e.g. prometheus-node-exporter)
    if ss -tlnp 2>/dev/null | grep -q ':9100 '; then
        local _port_pid
        _port_pid=$(ss -tlnp 2>/dev/null | grep ':9100 ' | grep -oP 'pid=\K[0-9]+' | head -1)
        local _port_proc
        _port_proc=$(ss -tlnp 2>/dev/null | grep ':9100 ' | grep -oP '"\K[^"]+(?=",pid)' | head -1)
        if [ -n "$_port_pid" ] && ! echo "$_port_proc" | grep -qE 'minio|zs3'; then
            print_warning "Port 9100 occupied by ${_port_proc} (pid=${_port_pid}) — stopping it to free port for zs3server"
            kill "$_port_pid" 2>/dev/null || true
            sleep 2
        fi
    fi

    MINIO_ROOT_USER="${MINIO_ROOT_USER:-rootroot}" MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-rootroot}" MINIO_BROWSER=OFF \
        nohup ./minio gateway zcn --address :9100 --console-address :9101 ${alloc_flags} 200>&- > /var/log/zs3server.log 2>&1 </dev/null &
    disown
    sleep 5

    # Health check — use -f so curl fails on non-2xx (prevents false-positive from node-exporter HTML)
    if curl -sf http://localhost:9100/minio/health/live > /dev/null 2>&1; then
        print_status "zs3server started on port 9100 (healthy)"
    else
        print_warning "zs3server health check failed — check /var/log/zs3server.log"
    fi

    # Start background renewal loop to keep the allocation alive.
    # Each --extend call resets expiry to block_time + 1 time_unit (720h = 30 days).
    # We extend periodically to maintain an indefinite lifetime.
    local renewal_pid_file="${ZCN_CONFIG_DIR}/zs3_renewal.pid"
    if [ -f "$renewal_pid_file" ]; then
        kill "$(cat "$renewal_pid_file" 2>/dev/null)" 2>/dev/null || true
        rm -f "$renewal_pid_file"
    fi
    local _ZBOX="$ZBOX" _WALLET="$ZCN_WALLET_FILE" _CONFDIR="$ZCN_CONFIG_DIR" _CONFFILE="$ZCN_CONFIG_FILE"
    (
        RENEWAL_INTERVAL=$((50 * 60))
        while true; do
            sleep "$RENEWAL_INTERVAL"
            CURRENT_ALLOC=$(cat "${_CONFDIR}/allocation.txt" 2>/dev/null || echo "")
            [ -z "$CURRENT_ALLOC" ] && break
            echo "$(date -u): Extending ZS3 allocation ${CURRENT_ALLOC:0:16}..." >> /var/log/zs3_renewal.log
            "$_ZBOX" updateallocation \
                --allocation "$CURRENT_ALLOC" \
                --extend \
                --lock 10 \
                --wallet "$_WALLET" --configDir "$_CONFDIR" --config "$_CONFFILE" \
                >> /var/log/zs3_renewal.log 2>&1 \
                || echo "$(date -u): Extension failed (will retry next cycle)" >> /var/log/zs3_renewal.log
        done
    # Redirect all I/O away from parent pipe and close lock fd so run_tests.sh flock releases
    ) 200>&- >> /var/log/zs3_renewal.log 2>&1 </dev/null &
    ZS3_RENEWAL_PID=$!
    disown "$ZS3_RENEWAL_PID"
    echo "$ZS3_RENEWAL_PID" > "$renewal_pid_file"
    print_status "ZS3 renewal loop started (pid=$ZS3_RENEWAL_PID, interval=50m)"
}

# Setup nginx reverse proxy with SSL
# Creates subdomain-based proxy config for all 0chain services
# Requires: domain configured in deploy_config.yaml or passed via NGINX_DOMAIN env var
setup_nginx() {
    print_header "Setting Up Nginx Reverse Proxy (Path-Based Routing)"

    # Remove stale subdomain configs from previous deploys (they may have ssl directives
    # without certificates, which causes nginx -t to fail and blocks the whole deploy)
    for stale_conf in /etc/nginx/sites-enabled/*.test.zus.network; do
        [ -e "$stale_conf" ] && rm -f "$stale_conf" 2>/dev/null || true
    done
    for stale_conf in /etc/nginx/sites-available/*.test.zus.network; do
        [ -e "$stale_conf" ] && rm -f "$stale_conf" 2>/dev/null || true
    done

    # Get domain from config or environment
    local DOMAIN="${NGINX_DOMAIN:-}"
    if [ -z "$DOMAIN" ]; then
        DOMAIN=$(grep "^  domain:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    fi

    if [ -z "$DOMAIN" ]; then
        print_warning "No domain configured. Set nginx.domain in deploy_config.yaml or NGINX_DOMAIN env var"
        print_warning "Skipping nginx setup"
        return 0
    fi

    # Get domain prefix for web-app subdomains (test.bolt.holdings, test1.bolt.holdings, etc.)
    # Priority: APP_DOMAIN_PREFIX env → nginx.domain_prefix in yaml → first segment of domain
    if [ -z "${APP_DOMAIN_PREFIX:-}" ]; then
        local yaml_prefix
        yaml_prefix=$(grep "^  domain_prefix:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
        if [ -n "$yaml_prefix" ]; then
            export APP_DOMAIN_PREFIX="$yaml_prefix"
        else
            export APP_DOMAIN_PREFIX="${DOMAIN%%.*}"
        fi
    fi
    print_status "App domain prefix: ${APP_DOMAIN_PREFIX}"

    local EMAIL="${NGINX_EMAIL:-admin@zus.network}"
    local ENABLE_SSL="${NGINX_SSL:-true}"

    print_status "Domain: $DOMAIN"
    print_status "Email: $EMAIL"
    print_status "SSL: $ENABLE_SSL"

    # Install nginx if not present
    if ! command -v nginx &> /dev/null; then
        print_status "Installing nginx..."
        if command -v apt-get &> /dev/null; then
            apt-get update -qq && apt-get install -y -qq nginx
        elif command -v yum &> /dev/null; then
            yum install -y nginx
        else
            print_error "Cannot install nginx - unsupported package manager"
            return 1
        fi
    fi

    # Install certbot if SSL enabled and not present
    if [ "$ENABLE_SSL" = "true" ] && ! command -v certbot &> /dev/null; then
        print_status "Installing certbot..."
        if command -v apt-get &> /dev/null; then
            apt-get install -y -qq certbot python3-certbot-nginx
        elif command -v yum &> /dev/null; then
            yum install -y certbot python3-certbot-nginx
        fi
    fi

    # Open ports 80 and 443 in UFW firewall (required for nginx and certbot ACME challenges)
    if command -v ufw &> /dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
        if ! ufw status 2>/dev/null | grep -q "^80/tcp.*ALLOW"; then
            ufw allow 80/tcp >/dev/null 2>&1 && print_status "Opened port 80/tcp in UFW"
        fi
        if ! ufw status 2>/dev/null | grep -q "^443/tcp.*ALLOW"; then
            ufw allow 443/tcp >/dev/null 2>&1 && print_status "Opened port 443/tcp in UFW"
        fi
        ufw reload >/dev/null 2>&1 || true
    fi

    # Create nginx config directory
    mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled /etc/nginx/snippets

    # Create shared CORS snippet for blobbers/validators/providers.
    # Blobbers send Access-Control-Allow-Origin: * which browsers reject for
    # credentialed requests. Strip backend CORS and replace with $http_origin.
    cat > /etc/nginx/snippets/provider-cors.conf << 'CORSEOF'
proxy_hide_header Access-Control-Allow-Origin;
proxy_hide_header Access-Control-Allow-Credentials;
proxy_hide_header Access-Control-Allow-Methods;
proxy_hide_header Access-Control-Allow-Headers;
proxy_hide_header Access-Control-Expose-Headers;
if ($request_method = OPTIONS) {
    add_header 'Access-Control-Allow-Origin' $http_origin;
    add_header 'Access-Control-Allow-Credentials' 'true';
    add_header 'Access-Control-Allow-Methods' 'GET,POST,PUT,DELETE,OPTIONS';
    add_header 'Access-Control-Allow-Headers' $http_access_control_request_headers;
    add_header 'Access-Control-Max-Age' 1728000;
    add_header 'Cross-Origin-Resource-Policy' 'cross-origin';
    return 204;
}
add_header 'Access-Control-Allow-Origin' $http_origin always;
add_header 'Access-Control-Allow-Credentials' 'true' always;
add_header 'Access-Control-Allow-Methods' 'GET,POST,PUT,DELETE,OPTIONS' always;
add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-Client-Signature-V2,X-App-Timestamp,ALLOCATION-ID,ALLOCATION-TX,X-Connection-Id,x-connection-id,x-mode,X-App-ID-Token,X-App-Type,X-App-User-ID,Range' always;
add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range,X-App-Error-Code' always;
add_header 'Cross-Origin-Resource-Policy' 'cross-origin' always;
CORSEOF

    # Create connection upgrade map for conditional WebSocket upgrade
    cat > /etc/nginx/conf.d/connection-upgrade-map.conf << 'MAPEOF'
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      '';
}
MAPEOF

    # Remove old subdomain-based configs and any stale configs
    rm -f /etc/nginx/sites-enabled/default
    # Remove any stale .bak files from sites-enabled (they cause "conflicting server name" warnings
    # and may serve stale config if nginx picks them up over the canonical config file)
    rm -f /etc/nginx/sites-enabled/*.bak* /etc/nginx/sites-enabled/*.backup*

    local CONF_FILE="/etc/nginx/sites-available/${DOMAIN}"

    print_status "Creating path-based nginx config at ${CONF_FILE}"

    # Generate a single server block with path-based location routing
    # Uses /service/ -> localhost:PORT proxying (all containers expose ports on 0.0.0.0)
    # Also exposes docker logs and config files for each service via /service/log and /service/config
    # Monitoring logs (vc, chaos, deploy, tests) are served as static files

    # Create log directory for nginx-served log snapshots
    mkdir -p /var/log/0chain

    cat > "$CONF_FILE" << 'NGINXEOF'
server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;

    # Increase body size for file uploads
    client_max_body_size 100m;

    # Common proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # =====================================================================
    #  CHAIN: Miners, Sharders (live API proxy)
    #  sub_filter rewrites absolute http://localhost:PORT links in the
    #  diagnostics HTML so they resolve through the nginx reverse proxy.
    #  CORS: exact-match blocks handle bare /minerNN (no trailing slash).
    #  IMPORTANT: add_header must be OUTSIDE the if() block with 'always'
    #  so headers are present on both OPTIONS (204) and GET/POST (301) responses.
    #  If add_header is only inside if{}, the 301 redirect response has no CORS
    #  headers and browsers refuse to follow the cross-origin redirect.
    # =====================================================================
    location = /miner01 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /miner01/; }
    location = /miner02 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /miner02/; }
    location = /miner03 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /miner03/; }
    location = /miner04 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /miner04/; }
    location = /sharder01 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /sharder01/; }
    location = /sharder02 { add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always; add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always; add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always; if ($request_method = OPTIONS) { return 204; } return 301 /sharder02/; }
    location /miner01/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7071/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }
    location /miner02/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7072/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }
    location /miner03/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7073/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }
    location /miner04/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7074/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }

    location /sharder01/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7171/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }
    location /sharder02/ {
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always; add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET,POST,OPTIONS,PUT,DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type,X-App-Client-Id,X-App-Client-Key,X-App-Client-Signature,X-App-Timestamp,Access-Control-Allow-Origin' always;
        if ($request_method = OPTIONS) { return 204; }
        proxy_pass http://localhost:7172/;
        proxy_set_header Accept-Encoding "";
        sub_filter_once off;
        sub_filter_types application/json text/plain;
        sub_filter 'http://localhost:7071' '/miner01/';
        sub_filter 'http://localhost:7072' '/miner02/';
        sub_filter 'http://localhost:7073' '/miner03/';
        sub_filter 'http://localhost:7074' '/miner04/';
        sub_filter 'http://localhost:7171' '/sharder01/';
        sub_filter 'http://localhost:7172' '/sharder02/';
        sub_filter 'http://198.18.0.71:7071' '/miner01/';
        sub_filter 'http://198.18.0.72:7072' '/miner02/';
        sub_filter 'http://198.18.0.73:7073' '/miner03/';
        sub_filter 'http://198.18.0.74:7074' '/miner04/';
        sub_filter 'http://198.18.0.81:7171' '/sharder01/';
        sub_filter 'http://198.18.0.82:7172' '/sharder02/';
    }

    # =====================================================================
    #  BLOBBERS (port = 505N for 1-9, 10=5060, 11=50611, 12=50612)
    #  CORS: blobbers send Access-Control-Allow-Origin: * which browsers
    #  reject for credentialed requests. Strip and replace with $http_origin.
    # =====================================================================
    location /blobber01/ { proxy_pass http://localhost:5051/; include snippets/provider-cors.conf; }
    location /blobber02/ { proxy_pass http://localhost:5052/; include snippets/provider-cors.conf; }
    location /blobber03/ { proxy_pass http://localhost:5053/; include snippets/provider-cors.conf; }
    location /blobber04/ { proxy_pass http://localhost:5054/; include snippets/provider-cors.conf; }
    location /blobber05/ { proxy_pass http://localhost:5055/; include snippets/provider-cors.conf; }
    location /blobber06/ { proxy_pass http://localhost:5056/; include snippets/provider-cors.conf; }
    location /blobber07/ { proxy_pass http://localhost:5057/; include snippets/provider-cors.conf; }
    location /blobber08/ { proxy_pass http://localhost:5058/; include snippets/provider-cors.conf; }
    location /blobber09/ { proxy_pass http://localhost:5059/; include snippets/provider-cors.conf; }
    location /blobber10/ { proxy_pass http://localhost:50610/; include snippets/provider-cors.conf; }
    location /blobber11/ { proxy_pass http://localhost:50611/; include snippets/provider-cors.conf; }
    location /blobber12/ { proxy_pass http://localhost:50612/; include snippets/provider-cors.conf; }

    # =====================================================================
    #  VALIDATORS (1-2: 504N, 3-9: 506N, 10=5070, 11=50711, 12=50712)
    # =====================================================================
    location /validator01/ { proxy_pass http://localhost:5041/; include snippets/provider-cors.conf; }
    location /validator02/ { proxy_pass http://localhost:5042/; include snippets/provider-cors.conf; }
    location /validator03/ { proxy_pass http://localhost:5063/; include snippets/provider-cors.conf; }
    location /validator04/ { proxy_pass http://localhost:5064/; include snippets/provider-cors.conf; }
    location /validator05/ { proxy_pass http://localhost:5065/; include snippets/provider-cors.conf; }
    location /validator06/ { proxy_pass http://localhost:5066/; include snippets/provider-cors.conf; }
    location /validator07/ { proxy_pass http://localhost:5067/; include snippets/provider-cors.conf; }
    location /validator08/ { proxy_pass http://localhost:5068/; include snippets/provider-cors.conf; }
    location /validator09/ { proxy_pass http://localhost:5069/; include snippets/provider-cors.conf; }
    location /validator10/ { proxy_pass http://localhost:50710/; include snippets/provider-cors.conf; }
    location /validator11/ { proxy_pass http://localhost:50711/; include snippets/provider-cors.conf; }
    location /validator12/ { proxy_pass http://localhost:50712/; include snippets/provider-cors.conf; }

    # =====================================================================
    #  ENTERPRISE BLOBBERS (ports 5071-5075)
    # =====================================================================
    location /eblobber01/ { proxy_pass http://localhost:5071/; include snippets/provider-cors.conf; }
    location /eblobber02/ { proxy_pass http://localhost:5072/; include snippets/provider-cors.conf; }
    location /eblobber03/ { proxy_pass http://localhost:5073/; include snippets/provider-cors.conf; }
    location /eblobber04/ { proxy_pass http://localhost:5074/; include snippets/provider-cors.conf; }
    location /eblobber05/ { proxy_pass http://localhost:5075/; include snippets/provider-cors.conf; }

    # =====================================================================
    #  KAFKA (no host port - use docker logs)
    # =====================================================================
    location /kafka { default_type text/plain; alias /var/log/0chain/kafka.log; }

    # =====================================================================
    #  SERVICES (live proxy only - logs/configs available via /logs/)
    # =====================================================================
    location /dns/        { proxy_pass http://localhost:9091/; }
    location /0dns/       { proxy_pass http://localhost:9091/; }
    # --- 0box / zauth / zvault with CORS ---
    # Backends send Access-Control-Allow-Origin: * which browsers reject for
    # credentialed requests (cookies, CSRF). We strip backend CORS headers and
    # add centralized ones using the actual request origin.
    location /0box/ {
        proxy_pass http://localhost:9081/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        # Strip backend CORS headers
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Expose-Headers;
        # Add centralized CORS
        add_header 'Access-Control-Allow-Origin' $http_origin always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck' always;
        add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range' always;
        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' $http_origin;
            add_header 'Access-Control-Allow-Credentials' 'true';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            add_header 'Content-Length' 0;
            return 204;
        }
    }
    location /zauth/ {
        proxy_pass http://localhost:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Expose-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck' always;
        add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range' always;
        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' $http_origin;
            add_header 'Access-Control-Allow-Credentials' 'true';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            add_header 'Content-Length' 0;
            return 204;
        }
    }
    location /zvault/ {
        proxy_pass http://localhost:8090/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Expose-Headers;
        add_header 'Access-Control-Allow-Origin' $http_origin always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck' always;
        add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range' always;
        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' $http_origin;
            add_header 'Access-Control-Allow-Credentials' 'true';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            add_header 'Content-Length' 0;
            return 204;
        }
    }
    location /zs3/        { proxy_pass http://localhost:9100/; }
    location /elasticsearch/ { proxy_pass http://localhost:9200/; }

    # Docker IP rewriting for browser SDK (rewrites 198.18.0.x to public URLs)
    include /etc/nginx/snippets/docker-ip-rewrite.conf;

    # =====================================================================
    #  ADMIN TOOLS
    # =====================================================================
    location /pgadmin/ { proxy_pass http://localhost:5050; proxy_set_header X-Script-Name /pgadmin; proxy_redirect off; }
    location /pgadmin-sharder/ { proxy_pass http://localhost:5434; proxy_set_header X-Script-Name /pgadmin-sharder; proxy_redirect off; }
    location /pgadmin-0box/ { proxy_pass http://localhost:5435; proxy_set_header X-Script-Name /pgadmin-0box; proxy_redirect off; }
    location /pgadmin-zauth/ { proxy_pass http://localhost:8081; proxy_set_header X-Script-Name /pgadmin-zauth; proxy_redirect off; }
    location /pgadmin-zvault/ { proxy_pass http://localhost:8083; proxy_set_header X-Script-Name /pgadmin-zvault; proxy_redirect off; }
    location /cadvisor/ {
        proxy_pass http://localhost:8085/;
        proxy_set_header Host localhost:8085;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header Accept-Encoding "";
        proxy_redirect /containers/ /cadvisor/containers/;
        sub_filter_once off;
        sub_filter_types application/javascript text/css;
        sub_filter '/api/' '/cadvisor/api/';
        sub_filter '"/containers' '"/cadvisor/containers';
        sub_filter '"/docker' '"/cadvisor/docker';
    }
    location /cadvisor/api/ { proxy_pass http://localhost:8085/api/; proxy_set_header Host localhost:8085; }
    location /cadvisor/containers/ { proxy_pass http://localhost:8085/containers/; proxy_set_header Host localhost:8085; }
    location /cadvisor/docker/ { proxy_pass http://localhost:8085/docker/; proxy_set_header Host localhost:8085; }
    location /static/ { proxy_pass http://localhost:8085/static/; }

    # =====================================================================
    #  MONITORING & BACKGROUND PROCESS LOGS (with ANSI color rendering)
    # =====================================================================
    location /vc      { default_type text/plain; alias /tmp/vc.log; }
    location /chaos   { default_type text/plain; alias /tmp/chaos.log; }
    location /funding { default_type text/plain; alias /tmp/auto_fund_daemon.log; }
    location /deploy  { default_type text/plain; alias /tmp/deploy_local.log; }
    location /dkg     { default_type text/plain; alias /tmp/monitor.log; }

    # HTML-rendered versions with ANSI color support (append /html)
    location /chaos/html   { default_type text/html; alias /var/log/0chain/chaos.html; }
    location /vc/html      { default_type text/html; alias /var/log/0chain/vc.html; }
    location /funding/html { default_type text/html; alias /var/log/0chain/funding_daemon.html; }
    location /dkg/html     { default_type text/html; alias /var/log/0chain/monitor.html; }
    location /deploy/html  { default_type text/html; alias /var/log/0chain/deploy.html; }
    location /smoke        { default_type text/plain; alias /tmp/smoke_test.log; }
    location /smoke/html   { default_type text/html; alias /var/log/0chain/smoke.html; }

    # =====================================================================
    #  TEST OUTPUT LOGS
    # =====================================================================
    location /test/results    { default_type text/html; alias /var/log/0chain/test_results.html; }
    location /test/subtests   { default_type text/html; alias /var/log/0chain/test_subtests.html; }
    location /test/api        { default_type text/plain; alias /tmp/test_api.log; }
    location /test/cli        { default_type text/plain; alias /tmp/test_cli.log; }
    location /test/sdk        { default_type text/plain; alias /tmp/test_sdk.log; }
    location /test/tokenomics { default_type text/plain; alias /tmp/test_tokenomics.log; }
    location /test/zs3        { default_type text/plain; alias /tmp/test_zs3.log; }
    location /test/mc         { default_type text/plain; alias /tmp/test_mc.log; }
    location /test/rclone     { default_type text/plain; alias /tmp/test_rclone.log; }
    location /test/cypress    { default_type text/plain; alias /tmp/test_cypress.log; }

    # =====================================================================
    #  ALL LOG SNAPSHOTS (browsable directory - single source for all logs)
    # =====================================================================
    location /logs/ {
        alias /var/log/0chain/;
        default_type text/plain;
        autoindex on;
    }

    # Container status summary
    location /containers { default_type text/plain; alias /var/log/0chain/containers.txt; }

    # Deployment info page (branch table, service links)
    location /info { default_type text/html; alias /var/www/html/deploy-info.html; }

    # =====================================================================
    #  WEB APPS (proxy to Next.js Docker containers)
    #  Apps are built WITH basePath (e.g. basePath: '/vult'), so requests
    #  are passed as-is (no trailing slash on proxy_pass = no prefix strip).
    # =====================================================================
    location /vult {
        proxy_pass http://localhost:3003;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
    location /bolt {
        proxy_pass http://localhost:3002;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
    location /blimp {
        proxy_pass http://localhost:3006;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
    location /explorer {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
    location /chimney {
        proxy_pass http://localhost:3005;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }

    # =====================================================================
    # =====================================================================
    #  REFERER-BASED ASSET ROUTING
    #
    #  Problem: All Next.js apps reference /assets/*, /zcn.js, /bg.png,
    #  /manifest.json, /api/*, /nocache/* WITHOUT their basePath prefix.
    #
    #  Solution: Use $webapp_port and $webapp_prefix (from map directive
    #  in /etc/nginx/conf.d/webapp-maps.conf) to route based on the
    #  Referer header to the correct app backend.
    #
    #  This replaces the old chimney-first + blimp-fallback pattern
    #  which caused cross-app asset contamination (e.g. chimney logo
    #  appearing on blimp pages).
    # =====================================================================

    # --- Explorer-specific: /assets/geographies/ (no other app has this) ---
    location /assets/geographies/ {
        proxy_pass http://localhost:3001/explorer/assets/geographies/;
        proxy_http_version 1.1;
    }

    # --- Blimp-specific root images (referenced in CSS without basePath) ---
    location = /bg.png {
        proxy_pass http://localhost:3006/blimp/bg.png;
        proxy_http_version 1.1;
    }
    location = /cta-bg.png {
        proxy_pass http://localhost:3006/blimp/cta-bg.png;
        proxy_http_version 1.1;
    }

    # --- Shared /zcn.js (referer-based routing) ---
    location = /zcn.js {
        proxy_pass http://localhost:$webapp_port$webapp_prefix/zcn.js;
        proxy_http_version 1.1;
    }

    # --- Shared /zcn.wasm (referer-based routing) ---
    # Note: no $webapp_prefix here — apps serve zcn.wasm at root (no basePath)
    location = /zcn.wasm {
        proxy_pass http://localhost:$webapp_port/zcn.wasm;
        proxy_http_version 1.1;
        add_header Content-Type application/wasm;
    }

    # --- Shared /manifest.json (referer-based routing) ---
    location = /manifest.json {
        proxy_pass http://localhost:$webapp_port$webapp_prefix/manifest.json;
        proxy_http_version 1.1;
    }

    # --- API routes without basePath (auth, clarity, etc.) ---
    location /api/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    # --- /nocache/ fonts (shared across all apps; referer-based) ---
    location /nocache/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    # --- Chimney-specific public assets ---
    location /country-flags/ {
        proxy_pass http://localhost:3005/chimney/country-flags/;
        proxy_http_version 1.1;
    }
    location /libarchive.js/ {
        proxy_pass http://localhost:3005/chimney/libarchive.js/;
        proxy_http_version 1.1;
    }

    # --- Referer-based /assets/ routing ---
    # Each app gets its OWN assets from its OWN backend.
    # No more chimney-first + blimp-fallback pattern.

    location /assets/svg/partners/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    location /assets/svg/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    location /assets/png/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    location /assets/js/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    # Generic /assets/ fallback (referer-based routing)
    location /assets/ {
        proxy_pass http://localhost:$webapp_port$webapp_prefix$uri;
        proxy_http_version 1.1;
    }

    # =====================================================================
    #  LANDING PAGE (dashboard.html)
    # =====================================================================
    location = / {
        root /var/www/html;
        try_files /dashboard.html =404;
    }
}
NGINXEOF

    # Replace placeholders with actual values (can't use shell vars inside 'NGINXEOF')
    sed -i "s/DOMAIN_PLACEHOLDER/${DOMAIN}/g" "$CONF_FILE"
    sed -i "s|WEB_APPS_DIR_PLACEHOLDER|${BASE_DIR}/web-apps|g" "$CONF_FILE"

    # Enable site (single config file for all path-based routes)
    ln -sf "$CONF_FILE" "/etc/nginx/sites-enabled/${DOMAIN}"

    # Remove default site if it exists
    rm -f /etc/nginx/sites-enabled/default

    # Create webapp-maps.conf for referer-based asset routing ($webapp_port, $webapp_prefix)
    create_webapp_maps_conf

    # Create docker-ip-rewrite snippet (must exist before nginx -t)
    create_docker_ip_rewrite_snippet "$DOMAIN"

    # Test nginx config
    if nginx -t 2>/dev/null; then
        print_status "Nginx config test passed"
    else
        print_error "Nginx config test failed"
        nginx -t
        return 1
    fi

    # Reload nginx
    systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
    print_status "Nginx reloaded with path-based routing config"

    # Setup SSL with certbot if enabled
    if [ "$ENABLE_SSL" = "true" ]; then
        local CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
        local CERT_FULLCHAIN="${CERT_DIR}/fullchain.pem"
        local CERT_KEY="${CERT_DIR}/privkey.pem"

        if [ -f "$CERT_FULLCHAIN" ] && [ -f "$CERT_KEY" ]; then
            # Cert already exists: add listen 443 ssl + cert directives to the
            # existing server block (alongside listen 80). This is simpler and
            # more reliable than splitting into separate HTTP redirect + HTTPS blocks.
            print_status "SSL cert already exists — adding SSL to server block"

            if grep -q 'listen 443' "$CONF_FILE" 2>/dev/null; then
                print_status "HTTPS already in config, skipping injection"
            else
                sed -i "/listen 80;/a\\
    listen 443 ssl;\\
    ssl_certificate ${CERT_FULLCHAIN};\\
    ssl_certificate_key ${CERT_KEY};\\
    include /etc/letsencrypt/options-ssl-nginx.conf;\\
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;" "$CONF_FILE"
                print_status "SSL directives injected into ${CONF_FILE}"
            fi
        else
            print_status "Requesting SSL certificate for ${DOMAIN}..."
            certbot --nginx --non-interactive --agree-tos --email "$EMAIL" \
                --redirect \
                -d "${DOMAIN}" \
                2>&1 || {
                print_warning "Certbot SSL setup failed. Services are still available via HTTP."
                print_warning "You can retry manually: certbot --nginx -d ${DOMAIN}"
            }
        fi

        # Fix certbot redirect bug: certbot adds 'if ($host) { return 301 https:// }'
        # to the HTTPS server block (listen 443), causing infinite redirect loops.
        # The HTTP->HTTPS redirect is handled by the separate port 80 server block
        # that certbot also creates, so remove the redundant if-redirect from the
        # HTTPS block.
        if grep -q 'listen 443 ssl' "$CONF_FILE" 2>/dev/null; then
            # Remove the 'if ($host = DOMAIN) { return 301 ... } # managed by Certbot' block
            # from the first server block (which is the HTTPS block after certbot runs)
            sed -i '/^server {$/,/listen 443 ssl;/{
                /if (\$host = '"${DOMAIN}"') {/,/} # managed by Certbot/d
            }' "$CONF_FILE"
            print_status "Removed certbot redirect from HTTPS block (prevents infinite loop)"
        fi

        # Setup subdomain configs for 0box, zauth, zvault (browser SDK needs these)
        setup_nginx_subdomains "$DOMAIN" "$EMAIL"

        # Setup custom domain configs for web apps (test.bolt.holdings, etc.)
        setup_app_domain_nginx "$DOMAIN" "$EMAIL"
    fi

    # Generate dashboard landing page with custom domain links
    generate_dashboard "$DOMAIN"

    # Setup log snapshot refresh cron (dumps docker logs and configs every 60s)
    setup_log_snapshots

    print_status "Nginx reverse proxy setup complete!"
    echo ""
    echo "  Landing Page: https://${DOMAIN}/"
    echo "  All Logs:     https://${DOMAIN}/logs/"
    echo "  Containers:   https://${DOMAIN}/containers"
    echo "  Web Apps:     https://${DOMAIN}/vult/  /bolt/  /blimp/  /explorer/  /chimney/"
    echo "  Subdomains:   https://0box.${DOMAIN}/  https://zauth.${DOMAIN}/  https://zvault.${DOMAIN}/"
}

# Create /etc/nginx/conf.d/webapp-maps.conf for referer-based asset routing
# Maps the Referer header to the correct web app backend port and basePath prefix
# Used by location blocks that serve root-level assets (zcn.js, manifest.json, etc.)
create_webapp_maps_conf() {
    local MAPS_CONF="/etc/nginx/conf.d/webapp-maps.conf"

    print_status "Creating webapp-maps.conf at $MAPS_CONF"

    cat > "$MAPS_CONF" << 'MAPSEOF'
# Map referer to the correct web app backend port and basePath prefix
# Used for root-level asset requests that lack the app basePath

map $http_referer $webapp_port {
    default         3005;    # chimney as default
    "~*/blimp"      3006;
    "~*/chimney"    3005;
    "~*/bolt"       3002;
    "~*/vult"       3003;
    "~*/explorer"   3001;
}

map $http_referer $webapp_prefix {
    default         /chimney;
    "~*/blimp"      /blimp;
    "~*/chimney"    /chimney;
    "~*/bolt"       /bolt;
    "~*/vult"       /vult;
    "~*/explorer"   /explorer;
}
MAPSEOF

    print_status "webapp-maps.conf created"
}

# Create the docker-ip-rewrite.conf snippet for nginx sub_filter
# This rewrites Docker IPs (198.18.0.x) in API responses to public URLs
# so browser-based wasm SDKs can reach miners/sharders/blobbers
create_docker_ip_rewrite_snippet() {
    local DOMAIN="$1"
    local SNIPPET="/etc/nginx/snippets/docker-ip-rewrite.conf"

    mkdir -p /etc/nginx/snippets

    print_status "Creating docker-ip-rewrite snippet at $SNIPPET"

    cat > "$SNIPPET" << EOF
# Auto-generated by deploy_local.sh - rewrites Docker IPs to public URLs
# This allows browser wasm SDKs to reach services through nginx
proxy_set_header Accept-Encoding "";
sub_filter_once off;
sub_filter_types application/json text/plain;

# Miners
sub_filter 'http://198.18.0.71:7071' 'https://${DOMAIN}/miner01';
sub_filter 'http://198.18.0.72:7072' 'https://${DOMAIN}/miner02';
sub_filter 'http://198.18.0.73:7073' 'https://${DOMAIN}/miner03';
sub_filter 'http://198.18.0.74:7074' 'https://${DOMAIN}/miner04';

# Sharders
sub_filter 'http://198.18.0.81:7171' 'https://${DOMAIN}/sharder01';
sub_filter 'http://198.18.0.82:7172' 'https://${DOMAIN}/sharder02';

# Blobbers 1-9
sub_filter 'http://198.18.0.91:5051' 'https://${DOMAIN}/blobber01';
sub_filter 'http://198.18.0.92:5052' 'https://${DOMAIN}/blobber02';
sub_filter 'http://198.18.0.93:5053' 'https://${DOMAIN}/blobber03';
sub_filter 'http://198.18.0.94:5054' 'https://${DOMAIN}/blobber04';
sub_filter 'http://198.18.0.95:5055' 'https://${DOMAIN}/blobber05';
sub_filter 'http://198.18.0.96:5056' 'https://${DOMAIN}/blobber06';
sub_filter 'http://198.18.0.97:5057' 'https://${DOMAIN}/blobber07';
sub_filter 'http://198.18.0.98:5058' 'https://${DOMAIN}/blobber08';
sub_filter 'http://198.18.0.99:5059' 'https://${DOMAIN}/blobber09';

# Blobbers 10-12
sub_filter 'http://198.18.0.110:5060' 'https://${DOMAIN}/blobber10';
sub_filter 'http://198.18.0.111:5061' 'https://${DOMAIN}/blobber11';
sub_filter 'http://198.18.0.112:5062' 'https://${DOMAIN}/blobber12';


# Enterprise blobbers
sub_filter 'http://198.18.0.201:5071' 'https://${DOMAIN}/eblobber01';
sub_filter 'http://198.18.0.202:5072' 'https://${DOMAIN}/eblobber02';
sub_filter 'http://198.18.0.203:5073' 'https://${DOMAIN}/eblobber03';
sub_filter 'http://198.18.0.204:5074' 'https://${DOMAIN}/eblobber04';
sub_filter 'http://198.18.0.205:5075' 'https://${DOMAIN}/eblobber05';

# 0dns
sub_filter 'http://198.18.0.100:9091' 'https://${DOMAIN}/dns';

# Bare IPs in 0box API JSON responses (0box returns "url":"198.18.0.x" without http:// or port)
# The Atlus service-providers page calls /v2/miners and /v2/sharders and uses item.url directly
sub_filter '"url":"198.18.0.81"' '"url":"https://${DOMAIN}/sharder01"';
sub_filter '"url":"198.18.0.82"' '"url":"https://${DOMAIN}/sharder02"';
sub_filter '"url":"198.18.0.71"' '"url":"https://${DOMAIN}/miner01"';
sub_filter '"url":"198.18.0.72"' '"url":"https://${DOMAIN}/miner02"';
sub_filter '"url":"198.18.0.73"' '"url":"https://${DOMAIN}/miner03"';
sub_filter '"url":"198.18.0.74"' '"url":"https://${DOMAIN}/miner04"';
EOF

    print_status "Docker IP rewrite snippet created with $(grep -c sub_filter "$SNIPPET") rules"
}

# Setup nginx subdomain configs for 0box, zauth, zvault
# Browser SDK uses 0box.DOMAIN, zauth.DOMAIN, zvault.DOMAIN pattern
# Requires DNS A records pointing to this server for each subdomain
setup_nginx_subdomains() {
    local DOMAIN="$1"
    local EMAIL="$2"

    print_status "Setting up subdomain nginx configs..."

    # Remove stale manually-created subdomain configs (e.g. subdomains.conf)
    rm -f /etc/nginx/sites-enabled/subdomains.conf /etc/nginx/sites-available/subdomains.conf 2>/dev/null || true

    # Create subdomain configs: name -> backend port
    local subdomain_name subdomain_port
    for subdomain_entry in "0box:9081" "zauth:8080" "zvault:8090"; do
        subdomain_name="${subdomain_entry%%:*}"
        subdomain_port="${subdomain_entry##*:}"
        local fqdn="${subdomain_name}.${DOMAIN}"
        local conf="/etc/nginx/sites-available/${fqdn}"

        print_status "  Creating config for ${fqdn} -> localhost:${subdomain_port}"

        # HTTP-only config with full location block.
        # Certbot --nginx will add 443/ssl block and redirect automatically.
        cat > "$conf" << SUBEOF
server {
    listen 80;
    server_name ${fqdn};

    client_max_body_size 100m;

    # Docker IP rewriting for browser SDK
    include /etc/nginx/snippets/docker-ip-rewrite.conf;

    location / {
        proxy_pass http://127.0.0.1:${subdomain_port};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \$connection_upgrade;
        proxy_connect_timeout 60s;
        proxy_send_timeout 3600s;
        proxy_read_timeout 3600s;

        # Hide backend CORS headers — nginx handles CORS centrally
        # to support credentialed requests (cookies/CSRF tokens).
        # Backends send Access-Control-Allow-Origin: * which browsers
        # reject for credentialed requests; we replace with the actual origin.
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        proxy_hide_header Access-Control-Allow-Credentials;
        proxy_hide_header Access-Control-Expose-Headers;

        # CORS headers for all proxied responses (including 404, 500, etc.)
        add_header 'Access-Control-Allow-Origin' \$http_origin always;
        add_header 'Access-Control-Allow-Credentials' 'true' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck,Access-Control-Allow-Origin' always;
        add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range' always;

        # Prevent browser caching of API responses — all API data must always be fresh.
        # Without this, browsers cache 0box responses and show stale data (e.g. empty
        # "Top Blobbers" chart) until private/incognito mode clears the cache.
        add_header 'Cache-Control' 'no-store' always;

        # Handle OPTIONS preflight directly (don't proxy to backend)
        if (\$request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' \$http_origin;
            add_header 'Access-Control-Allow-Credentials' 'true';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,X-App-Client-ID,X-App-Client-Key,X-App-Client-Signature,X-App-ID-Token,X-App-ID-TOKEN,X-App-Phone-Number,X-App-Signature,X-App-Timestamp,X-App-Type,X-APP-TYPE,X-App-User-ID,X-App-Alloc-ID,X-App-Alloc-Type,X-CSRF-TOKEN,X-CSRF-Token,X-Jwt-Token,X-JWT-TOKEN,X-Access-Token,X-Client-ID,X-Client-Version,X-User-ID,X-Admin-User-ID,X-Organization-User-ID,X-Peer-Public-Key,X-Firebase-AppCheck,Access-Control-Allow-Origin';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain; charset=utf-8';
            add_header 'Content-Length' 0;
            return 204;
        }
    }
}
SUBEOF

        ln -sf "$conf" "/etc/nginx/sites-enabled/${fqdn}"
    done

    # Test and reload nginx
    if nginx -t 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
    else
        print_warning "Nginx config test failed after adding subdomains - removing them to avoid blocking nginx"
        nginx -t 2>&1 || true
        # Remove the broken subdomain configs so nginx stays functional
        for subdomain_entry in "0box:9081" "zauth:8080" "zvault:8090"; do
            subdomain_name="${subdomain_entry%%:*}"
            local fqdn="${subdomain_name}.${DOMAIN}"
            rm -f "/etc/nginx/sites-enabled/${fqdn}" "/etc/nginx/sites-available/${fqdn}"
        done
        # Reload with clean config
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
        return 0
    fi

    # Request SSL certs for each subdomain
    for subdomain_entry in "0box:9081" "zauth:8080" "zvault:8090"; do
        subdomain_name="${subdomain_entry%%:*}"
        local fqdn="${subdomain_name}.${DOMAIN}"
        print_status "  Requesting SSL cert for ${fqdn}..."
        certbot --nginx --non-interactive --agree-tos --email "$EMAIL" \
            -d "${fqdn}" \
            2>&1 || {
            print_warning "  SSL cert for ${fqdn} failed. Ensure DNS A record points to this server."
            print_warning "  Retry: certbot --nginx -d ${fqdn}"
        }
    done

    print_status "Subdomain setup complete"
}

# Setup nginx server blocks for app custom domains
# Maps test.bolt.holdings, test.blimp.software, etc. to the correct Next.js app port
# IMPORTANT: After adding custom domains, you must MANUALLY add them to Firebase Console:
# Firebase Console -> project box-dev-ce8bf -> Authentication -> Settings -> Authorized domains
# Add: test.bolt.holdings, test.blimp.software, test.vult.network, test.chimney.software, test.atlus.cloud
# Without this, Firebase auth (Google sign-in) will fail with "auth/unauthorized-domain" error
setup_app_domain_nginx() {
    local DOMAIN="${1:-test.zus.network}"
    local EMAIL="${2:-admin@zus.network}"

    print_header "Setting up App Domain Nginx Configs"

    # Domain -> port / basePath mapping
    # Each app runs on its own port with a basePath prefix
    # APP_DOMAIN_PREFIX controls the subdomain (e.g. "test" -> test.bolt.holdings, "test1" -> test1.bolt.holdings)
    local PREFIX="${APP_DOMAIN_PREFIX:-test}"
    local -A APP_DOMAINS=(
        [${PREFIX}.bolt.holdings]="3002:bolt"
        [${PREFIX}.blimp.software]="3006:blimp"
        [${PREFIX}.chimney.software]="3005:chimney"
        [${PREFIX}.vult.network]="3003:vult"
        [${PREFIX}.atlus.cloud]="3001:explorer"
    )

    for fqdn in "${!APP_DOMAINS[@]}"; do
        local port_prefix="${APP_DOMAINS[$fqdn]}"
        local port="${port_prefix%%:*}"
        local conf="/etc/nginx/sites-available/${fqdn}"

        print_status "  Creating config for ${fqdn} -> localhost:${port}/"

        cat > "$conf" << APPEOF
server {
    listen 80;
    server_name ${fqdn};

    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:${port}/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \$connection_upgrade;
    }
}
APPEOF

        ln -sf "$conf" "/etc/nginx/sites-enabled/${fqdn}"
    done

    # Test and reload nginx
    if nginx -t 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
        print_status "App domain nginx configs loaded"
    else
        print_warning "Nginx config test failed after adding app domains — removing them"
        nginx -t 2>&1 || true
        for fqdn in "${!APP_DOMAINS[@]}"; do
            rm -f "/etc/nginx/sites-enabled/${fqdn}" "/etc/nginx/sites-available/${fqdn}"
        done
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
        return 0
    fi

    # Request SSL certs for each app domain
    if [ "${ENABLE_SSL:-false}" = "true" ]; then
        for fqdn in "${!APP_DOMAINS[@]}"; do
            print_status "  Requesting SSL cert for ${fqdn}..."
            certbot --nginx --non-interactive --agree-tos --email "$EMAIL" \
                -d "${fqdn}" \
                2>&1 || {
                print_warning "  SSL cert for ${fqdn} failed. Ensure DNS A record points to this server."
                print_warning "  Retry: certbot --nginx -d ${fqdn}"
            }
            # Fix certbot redirect bug: remove 'if ($host) { return 301 }' from HTTPS block
            local app_conf="/etc/nginx/sites-available/${fqdn}"
            if grep -q 'listen 443 ssl' "$app_conf" 2>/dev/null; then
                sed -i "/^server {$/,/listen 443 ssl;/{
                    /if (\$host = ${fqdn}) {/,/} # managed by Certbot/d
                }" "$app_conf" 2>/dev/null
            fi
        done
        nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
    fi

    print_status "App domain setup complete"
}

# Setup auto-funding daemon and watchdog cron
# Copies auto_fund_daemon.sh and ensure_fund_daemon.sh into place and sets up cron
setup_auto_funding() {
    local SCRIPT_DIR="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"

    print_header "Setting up Auto-Funding Daemon"

    local DAEMON_SRC="${SCRIPT_DIR}/auto_fund_daemon.sh"
    local WATCHDOG_SRC="${SCRIPT_DIR}/ensure_fund_daemon.sh"
    local INSTALL_DIR="/usr/local/bin"

    if [ ! -f "$DAEMON_SRC" ]; then
        return 0
    fi

    # Install scripts
    cp "$DAEMON_SRC" "${INSTALL_DIR}/auto_fund_daemon.sh"
    chmod +x "${INSTALL_DIR}/auto_fund_daemon.sh"
    print_status "Installed auto_fund_daemon.sh to ${INSTALL_DIR}"

    if [ -f "$WATCHDOG_SRC" ]; then
        cp "$WATCHDOG_SRC" "${INSTALL_DIR}/ensure_fund_daemon.sh"
        chmod +x "${INSTALL_DIR}/ensure_fund_daemon.sh"
        print_status "Installed ensure_fund_daemon.sh to ${INSTALL_DIR}"
    fi

    # Setup cron job for watchdog (every 5 minutes)
    local CRON_LINE="*/5 * * * * /usr/local/bin/ensure_fund_daemon.sh >> /tmp/ensure_fund_daemon.log 2>&1"
    if crontab -l 2>/dev/null | grep -qF "ensure_fund_daemon"; then
        print_status "Watchdog cron job already exists"
    else
        (crontab -l 2>/dev/null; echo "$CRON_LINE") | crontab -
        print_status "Added watchdog cron: ${CRON_LINE}"
    fi

    # Start the daemon now if not running
    if [ -f /tmp/auto_fund_daemon.pid ] && kill -0 "$(cat /tmp/auto_fund_daemon.pid)" 2>/dev/null; then
        print_status "Auto-fund daemon already running (PID: $(cat /tmp/auto_fund_daemon.pid))"
    else
        print_status "Starting auto-fund daemon..."
        nohup bash "${INSTALL_DIR}/auto_fund_daemon.sh" 120 5 20 >> /dev/null 2>&1 &
        print_status "Auto-fund daemon started"
    fi

    print_status "Auto-funding setup complete"
}

# Setup block pruning cron — runs daily at 3am, keeps blocks newer than 1 week.
# Idempotent: skips if cron entry already exists.
setup_block_pruning() {
    local SCRIPT_DIR="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
    local SRC="${SCRIPT_DIR}/prune_blocks.sh"
    local DEST="/root/prune_blocks.sh"

    if [ ! -f "$SRC" ]; then
        return 0
    fi

    cp "$SRC" "$DEST"
    chmod +x "$DEST"
    print_status "Installed prune_blocks.sh to ${DEST}"

    local CRON_LINE="0 3 * * * ${DEST} >> /var/log/prune_blocks.log 2>&1"
    if crontab -l 2>/dev/null | grep -qF "prune_blocks"; then
        print_status "Block pruning cron already exists"
    else
        (crontab -l 2>/dev/null; echo "$CRON_LINE") | crontab -
        print_status "Added block pruning cron: ${CRON_LINE}"
    fi
}

# Refresh log snapshots for nginx serving
# Dumps last 500 lines of docker logs and config files to /var/log/0chain/
setup_log_snapshots() {
    local LOG_DIR="/var/log/0chain"
    mkdir -p "$LOG_DIR"

    # Create the log refresh script
    # NOTE: Uses BASE_DIR from deploy to locate config files. Defaults to ~/Code.
    cat > /usr/local/bin/refresh-0chain-logs.sh << 'LOGEOF'
#!/bin/bash
LOG_DIR="/var/log/0chain"
BASE="${OCHAIN_BASE_DIR:-$HOME/Code}"
mkdir -p "$LOG_DIR"

# Docker container logs (last 500 lines each)
for container in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2 \
    blobber-1 blobber-2 blobber-3 blobber-4 blobber-5 blobber-6 \
    blobber-7 blobber-8 blobber-9 \
    validator-1 validator-2 validator-3 validator-4 validator-5 validator-6 \
    validator-7 validator-8 validator-9 \
    eblobber-1 eblobber-2 eblobber-3 \
    kafka elasticsearch 0box zauth-zauthserver-1 zvault-zvault-1; do
    docker logs --tail 500 "$container" > "$LOG_DIR/${container}.log" 2>&1 || true
done

# Also capture eblobber containers with old naming pattern
for container in eblobber1-blobber-1 eblobber2-blobber-1 eblobber3-blobber-1; do
    num=$(echo "$container" | grep -o '[0-9]' | head -1)
    docker logs --tail 500 "$container" > "$LOG_DIR/eblobber-${num}.log" 2>&1 || true
done

# zs3server log (runs as a process, not Docker)
if [ -f "$BASE/zs3server/cmd.log" ]; then
    tail -500 "$BASE/zs3server/cmd.log" > "$LOG_DIR/zs3server.log" 2>/dev/null || true
fi

# Crawler log (Docker container)
docker logs --tail 500 crawler > "$LOG_DIR/crawler.log" 2>&1 || true

# Render (Gotenberg) log
docker logs --tail 500 gotenberg > "$LOG_DIR/render.log" 2>&1 || true

# 0dns log (service name varies)
for dns_container in 0dns 0dns-0dns-1 dockerlocal-0dns-1; do
    docker logs --tail 500 "$dns_container" > "$LOG_DIR/0dns.log" 2>&1 && break
done

# zauth / zvault logs (container names vary by compose project name)
docker logs --tail 500 zauth-zauthserver-1 > "$LOG_DIR/zauth.log" 2>&1 || \
    docker logs --tail 500 zauth > "$LOG_DIR/zauth.log" 2>&1 || true
docker logs --tail 500 zvault-zvault-1 > "$LOG_DIR/zvault.log" 2>&1 || \
    docker logs --tail 500 zvault > "$LOG_DIR/zvault.log" 2>&1 || true

# Kafka config snapshot
docker exec kafka cat /etc/kafka/docker/server.properties > "$LOG_DIR/kafka_config.txt" 2>/dev/null || \
    docker exec kafka cat /opt/kafka/config/server.properties > "$LOG_DIR/kafka_config.txt" 2>/dev/null || \
    echo "Kafka config not found" > "$LOG_DIR/kafka_config.txt"

# Service configs (use $BASE for portability)
for svc_config in \
    "$BASE/0box/docker.local/config/0box.yaml:0box_config.txt" \
    "$BASE/zvault/config/zvault.yaml:zvault_config.txt" \
    "$BASE/zauth-server/config/zauthserver.yaml:zauth_config.txt" \
    "$BASE/0dns/docker.local/config/0dns.yaml:0dns_config.txt" \
    "$BASE/blobber/config/0chain_blobber.yaml:blobber_config.txt" \
    "$BASE/blobber/config/0chain_validator.yaml:validator_config.txt" \
    "$BASE/eblobber/config/0chain_blobber.yaml:eblobber_config.txt" \
    "$BASE/0chain/docker.local/config/0chain.yaml:miner_config.txt" \
    "$BASE/0chain/docker.local/config/0chain.yaml:sharder_config.txt"; do
    src="${svc_config%%:*}"
    dst="${svc_config##*:}"
    cp "$src" "$LOG_DIR/$dst" 2>/dev/null || echo "Config not found: $src" > "$LOG_DIR/$dst"
done

# Container status summary
docker ps -a --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' > "$LOG_DIR/containers.txt" 2>/dev/null

# Convert ANSI color logs to HTML for browser viewing
# Use full file for smoke tests, tail -2000 for others
for log_html in \
    "/tmp/chaos.log:$LOG_DIR/chaos.html" \
    "/tmp/vc.log:$LOG_DIR/vc.html" \
    "/tmp/auto_fund_daemon.log:$LOG_DIR/funding_daemon.html" \
    "/tmp/monitor.log:$LOG_DIR/monitor.html" \
    "/tmp/deploy_local.log:$LOG_DIR/deploy.html" \
    "/tmp/smoke_test.log:$LOG_DIR/smoke.html"; do
    src="${log_html%%:*}"
    dst="${log_html##*:}"
    if [ -f "$src" ]; then
        # Use full file for smoke tests, tail -5000 for others
        if echo "$src" | grep -q "smoke_test"; then
            log_content="$(cat "$src")"
        else
            log_content="$(tail -5000 "$src")"
        fi
        # Strip control chars but preserve UTF-8 multibyte (box-drawing etc) and ANSI escapes
        log_content="$(echo "$log_content" | LC_ALL=C tr -d '\000-\010\016-\032\177')"
        if command -v aha &>/dev/null; then
            echo "$log_content" | aha --no-header 2>/dev/null | \
                sed '1i<html><head><meta charset="UTF-8"><style>body{background:#fff;padding:10px;color:#333}pre{color:#333;font-size:13px;white-space:pre-wrap;word-wrap:break-word}span{background:transparent !important;color:inherit}</style></head><body><pre>' | \
                sed '$a</pre></body></html>' > "$dst" 2>/dev/null || true
        else
            {
                echo '<html><head><meta charset="UTF-8"><style>body{background:#fff;padding:10px;color:#333}pre{color:#333;font-size:13px;white-space:pre-wrap;word-wrap:break-word}span{background:transparent !important;color:inherit}</style></head><body><pre>'
                echo "$log_content" | sed 's/\x1b\[[0-9;]*m//g'
                echo '</pre></body></html>'
            } > "$dst" 2>/dev/null || true
        fi
    fi
done

# Generate test results HTML using the unified format from run_tests.sh.
# Sources run_tests.sh to reuse generate_results_snapshot() which applies
# cumulative best-result tallying (PASS in any rerun counts as PASS overall).
generate_test_results() {
    local run_tests_sh="${SCRIPT_DIR}/run_tests.sh"
    if [ ! -f "$run_tests_sh" ]; then
        print_warning "run_tests.sh not found at ${run_tests_sh}; skipping results generation"
        return
    fi
    # Prevent run_tests.sh from redirecting stdout/stderr when sourced
    local _saved="${_RUN_TESTS_LOGGING_SET}"
    export _RUN_TESTS_LOGGING_SET=1
    # Set paths before sourcing (run_tests.sh derives RESULTS_DIR from its own SCRIPT_DIR)
    RESULTS_DIR="${BASE_DIR}/system_test/test_results"
    RESULTS_HTML="${LOG_DIR}/test_results.html"
    source "$run_tests_sh"
    mkdir -p "${LOG_DIR}"
    generate_results_snapshot
    export _RUN_TESTS_LOGGING_SET="${_saved}"
}
generate_test_results

# Generate detailed subtests HTML page with filtering
generate_subtests_page() {
    local SUBTESTS_HTML="$LOG_DIR/test_subtests.html"
    local total_pass=0 total_fail=0 total_skip=0
    local tmprows=$(mktemp)
    local suite_summary=""

    for suite_entry in "sdk:/tmp/test_sdk.log" "api:/tmp/test_api.log" "cli:/tmp/test_cli.log"; do
        local suite="${suite_entry%%:*}"
        local logfile="${suite_entry##*:}"
        [ ! -f "$logfile" ] && continue
        [ ! -s "$logfile" ] && continue

        local sp sf ss st
        sp=$(grep -caE "^[[:space:]]*--- PASS:" "$logfile" 2>/dev/null || true)
        sf=$(grep -caE "^[[:space:]]*--- FAIL:" "$logfile" 2>/dev/null || true)
        ss=$(grep -caE "^[[:space:]]*--- SKIP:" "$logfile" 2>/dev/null || true)
        sp=${sp:-0}; sf=${sf:-0}; ss=${ss:-0}
        st=$((sp + sf + ss))
        local SUITE_UPPER
        SUITE_UPPER=$(echo "$suite" | tr 'a-z' 'A-Z')
        suite_summary="${suite_summary}${SUITE_UPPER}: ${sp}P/${sf}F/${ss}S (${st})<br>"
        total_pass=$((total_pass + sp))
        total_fail=$((total_fail + sf))
        total_skip=$((total_skip + ss))

        grep -aE "^[[:space:]]*--- (PASS|FAIL|SKIP):" "$logfile" 2>/dev/null | while IFS= read -r line; do
            local status=""
            case "$line" in
                *PASS:*) status="PASS" ;;
                *FAIL:*) status="FAIL" ;;
                *SKIP:*) status="SKIP" ;;
            esac
            local test_name
            test_name=$(echo "$line" | sed "s/^[[:space:]]*--- ${status}: //" | sed 's/ ([0-9.]*s)$//')
            local duration
            duration=$(echo "$line" | grep -oE '\([0-9.]+s\)' | tr -d '()')
            : "${duration:=0.00s}"
            local safe_name
            safe_name=$(echo "$test_name" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
            local color="green" cls="pass"
            [ "$status" = "FAIL" ] && color="red" && cls="fail"
            [ "$status" = "SKIP" ] && color="gray" && cls="skip"
            echo "<tr class=\"${cls}\"><td>${suite}</td><td title=\"${safe_name}\">${safe_name}</td><td style=\"color:${color};font-weight:bold\">${status}</td><td>${duration}</td></tr>"
        done >> "$tmprows"
    done

    local total=$((total_pass + total_fail + total_skip))
    local now
    now=$(date '+%Y-%m-%d %H:%M:%S %Z')

    {
        echo '<!DOCTYPE html><html><head><title>Test Subtests Detail</title>'
        echo '<meta charset="UTF-8">'
        echo '<style>'
        echo 'body{font-family:monospace;margin:10px;font-size:12px;background:#fafafa}'
        echo 'table{border-collapse:collapse;width:100%;margin-bottom:15px;table-layout:fixed}'
        echo 'th{background:#1a73e8;color:#fff;padding:4px 8px;text-align:left;position:sticky;top:0}'
        echo 'td{padding:3px 8px;border-bottom:1px solid #eee}'
        echo 'th:nth-child(1),td:nth-child(1){width:5%}'
        echo 'th:nth-child(2),td:nth-child(2){width:40%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}'
        echo 'th:nth-child(3),td:nth-child(3){width:7%}'
        echo 'th:nth-child(4),td:nth-child(4){width:8%}'
        echo 'tr:hover{background:#e8f0fe}'
        echo 'tr.pass td:nth-child(3){color:green}'
        echo 'tr.fail td:nth-child(3){color:red;font-weight:bold}'
        echo 'tr.skip td:nth-child(3){color:gray}'
        echo 'input[type=text]{padding:4px 8px;width:300px;font-family:monospace;font-size:12px}'
        echo 'label{margin-right:10px;cursor:pointer}'
        echo '</style>'
        echo '<script>'
        echo 'function applyFilters(){'
        echo '  var q=document.getElementById("filter").value.toLowerCase();'
        echo '  var sp=document.getElementById("showPass").checked;'
        echo '  var sf=document.getElementById("showFail").checked;'
        echo '  var ss=document.getElementById("showSkip").checked;'
        echo '  var rows=document.querySelectorAll("tbody tr");'
        echo '  for(var i=0;i<rows.length;i++){'
        echo '    var r=rows[i];var cls=r.className;var txt=r.textContent.toLowerCase();'
        echo '    var show=(cls==="pass"&&sp)||(cls==="fail"&&sf)||(cls==="skip"&&ss);'
        echo '    if(show&&q&&txt.indexOf(q)<0)show=false;'
        echo '    r.style.display=show?"":"none";'
        echo '  }'
        echo '}'
        echo '</script></head><body>'
        echo "<h3>0Chain System Test &mdash; Subtests Detail &mdash; ${now}</h3>"
        echo '<p><a href="/test/results">&larr; Back to summary</a></p>'
        echo "<p>Pass: ${total_pass} | Fail: ${total_fail} | Skip: ${total_skip} | Total: ${total}</p>"
        echo "<p>${suite_summary}</p>"
        echo '<p>Filter: <input type="text" id="filter" onkeyup="applyFilters()" placeholder="type to search...">'
        echo '<label><input type="checkbox" id="showPass" checked onchange="applyFilters()"> Pass</label>'
        echo '<label><input type="checkbox" id="showFail" checked onchange="applyFilters()"> Fail</label>'
        echo '<label><input type="checkbox" id="showSkip" checked onchange="applyFilters()"> Skip</label></p>'
        echo '<table><thead><tr><th>Suite</th><th>Test</th><th>Status</th><th>Duration</th></tr></thead>'
        echo '<tbody>'
        cat "$tmprows"
        echo '</tbody></table>'
        echo '</body></html>'
    } > "$SUBTESTS_HTML"

    rm -f "$tmprows"
}
generate_subtests_page

# Override with fix_results.py — supports run history, retry badges, consolidated results
FIXER="${BASE}/system_test/scripts/fix_results.py"
if [ -f "$FIXER" ]; then
    python3 "$FIXER" 2>/dev/null || true
fi
LOGEOF
    chmod +x /usr/local/bin/refresh-0chain-logs.sh

    # Run once immediately
    /usr/local/bin/refresh-0chain-logs.sh

    # Setup cron to refresh every 60 seconds
    local CRON_ENTRY="* * * * * /usr/local/bin/refresh-0chain-logs.sh"
    (crontab -l 2>/dev/null | grep -v "refresh-0chain-logs" ; echo "$CRON_ENTRY") | crontab -
    print_status "Log snapshot refresh cron installed (every 60s)"
}

# Clean up stale/unreachable blobbers from previous deployments or test runs.
# This prevents test failures caused by blobbers with non-local URLs (e.g. devnet)
# or random URLs created by TestRegisterBlobber.
cleanup_stale_blobbers() {
    cleanup_stale_providers
}

# Kill all blobbers and validators on-chain that are NOT part of this deployment.
# Junk providers are registered by API tests (TestRegisterBlobber, etc.) with random
# hostnames and pollute the blobber list, causing allocation creation failures.
# This function identifies providers by URL — any provider whose URL doesn't match
# the local docker network (198.18.* or 198.19.*) is considered junk and killed.
# Safe to run multiple times — already-killed providers are skipped.
cleanup_stale_providers() {
    print_header "Cleaning Up Stale Blobbers & Validators"

    local SHARDER_URL="http://127.0.0.1:7171"
    local SC_ADDRESS="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local total_killed=0

    # ---- 1. Clean up stale blobbers ----
    print_status "Checking for stale blobbers..."

    local blobbers_json
    blobbers_json=$(curl -s "${SHARDER_URL}/v1/screst/${SC_ADDRESS}/getblobbers" 2>/dev/null)

    if [ -z "$blobbers_json" ] || ! echo "$blobbers_json" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
        print_warning "Could not fetch blobber list from sharder - skipping blobber cleanup"
    else
        local INFRA_WALLET_ID
        INFRA_WALLET_ID=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/local.json" 2>/dev/null || echo "")
        local NGINX_DOMAIN_LOCAL
        NGINX_DOMAIN_LOCAL="${NGINX_DOMAIN:-}"
        if [ -z "$NGINX_DOMAIN_LOCAL" ] && [ -f "$CONFIG_FILE" ]; then
            NGINX_DOMAIN_LOCAL=$(grep "^  domain:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
        fi

        local stale_blobbers
        stale_blobbers=$(echo "$blobbers_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
infra_wallet = sys.argv[1] if len(sys.argv) > 1 else ''
nginx_domain = sys.argv[2] if len(sys.argv) > 2 else ''
for b in data.get('Nodes', []):
    bid = b.get('id', '')
    url = b.get('url', '')
    killed = b.get('is_killed', False)
    shutdown = b.get('is_shutdown', False)
    if killed or shutdown:
        continue
    # Skip infrastructure blobbers (registered with the infra wallet)
    delegate = b.get('stake_pool_settings', {}).get('delegate_wallet', '')
    if infra_wallet and delegate == infra_wallet:
        continue
    host = url.replace('https://', '').replace('http://', '').split(':')[0].split('/')[0]
    # Skip local IPs or the configured nginx domain
    if host.startswith('198.18.') or host.startswith('198.19.') or host.startswith('127.0.0.') or host == 'localhost':
        continue
    if nginx_domain and (host == nginx_domain or host.endswith('.' + nginx_domain)):
        continue
    print(f'{bid}|{url}')
" "$INFRA_WALLET_ID" "$NGINX_DOMAIN_LOCAL" 2>/dev/null)

        if [ -z "$stale_blobbers" ]; then
            print_status "  No stale blobbers found - all URLs are local"
        else
            local blobber_count=0
            while IFS='|' read -r blobber_id blobber_url; do
                [ -z "$blobber_id" ] && continue
                print_warning "  Killing stale blobber: ${blobber_id:0:16}... (URL: $blobber_url)"
                if $ZBOX kill-blobber --id "$blobber_id" \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null; then
                    print_status "    Killed successfully"
                    blobber_count=$((blobber_count + 1))
                else
                    print_warning "    Failed to kill (may need SC owner wallet or already killed)"
                fi
                sleep 2
            done <<< "$stale_blobbers"
            total_killed=$((total_killed + blobber_count))
            print_status "  Killed $blobber_count stale blobber(s)"
        fi
    fi

    # ---- 2. Clean up stale validators ----
    print_status "Checking for stale validators..."

    local validators_json
    validators_json=$(curl -s "${SHARDER_URL}/v1/screst/${SC_ADDRESS}/validators" 2>/dev/null)

    if [ -z "$validators_json" ] || ! echo "$validators_json" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
        print_warning "Could not fetch validator list from sharder - skipping validator cleanup"
    else
        local stale_validators
        stale_validators=$(echo "$validators_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
infra_wallet = sys.argv[1] if len(sys.argv) > 1 else ''
nginx_domain = sys.argv[2] if len(sys.argv) > 2 else ''
nodes = data if isinstance(data, list) else data.get('Nodes', data.get('validators', []))
for v in nodes:
    vid = v.get('id', v.get('validator_id', ''))
    url = v.get('url', v.get('base_url', ''))
    killed = v.get('is_killed', False)
    shutdown = v.get('is_shutdown', False)
    if killed or shutdown:
        continue
    delegate = v.get('stake_pool_settings', {}).get('delegate_wallet', '')
    if infra_wallet and delegate == infra_wallet:
        continue
    host = url.replace('https://', '').replace('http://', '').split(':')[0].split('/')[0]
    if host.startswith('198.18.') or host.startswith('198.19.') or host.startswith('127.0.0.') or host == 'localhost':
        continue
    if nginx_domain and (host == nginx_domain or host.endswith('.' + nginx_domain)):
        continue
    print(f'{vid}|{url}')
" "$INFRA_WALLET_ID" "$NGINX_DOMAIN_LOCAL" 2>/dev/null)

        if [ -z "$stale_validators" ]; then
            print_status "  No stale validators found - all URLs are local"
        else
            local validator_count=0
            while IFS='|' read -r validator_id validator_url; do
                [ -z "$validator_id" ] && continue
                print_warning "  Killing stale validator: ${validator_id:0:16}... (URL: $validator_url)"
                if $ZBOX kill-validator --id "$validator_id" \
                    --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null; then
                    print_status "    Killed successfully"
                    validator_count=$((validator_count + 1))
                else
                    print_warning "    Failed to kill (may need SC owner wallet or already killed)"
                fi
                sleep 2
            done <<< "$stale_validators"
            total_killed=$((total_killed + validator_count))
            print_status "  Killed $validator_count stale validator(s)"
        fi
    fi

    if [ $total_killed -gt 0 ]; then
        print_status "Total killed: $total_killed stale provider(s)"
        sleep 5
    else
        print_status "Chain is clean - no stale providers found"
    fi
}

# Wait until the alloc_blobbers SC REST endpoint returns enough eligible blobbers.
# This ensures blobbers have registered AND sent at least one health check before tests run.
# Usage: wait_for_blobbers_ready [min_blobbers] [timeout_seconds]
wait_for_blobbers_ready() {
    local min_blobbers="${1:-4}"
    local timeout_sec="${2:-600}"  # 10 minutes default
    local SC_ADDRESS="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local sharder_url="http://127.0.0.1:7171"
    local ALLOC_DATA='{"data_shards":2,"parity_shards":2,"size":10485760,"read_price_range":{"min":0,"max":9223372036854775807},"write_price_range":{"min":0,"max":9223372036854775807}}'

    print_header "Waiting for Blobbers to be Ready (need $min_blobbers eligible)"
    local start_time
    start_time=$(date +%s)
    local attempt=0

    while true; do
        attempt=$((attempt + 1))
        local elapsed=$(( $(date +%s) - start_time ))

        if [ "$elapsed" -ge "$timeout_sec" ]; then
            print_error "Timed out after ${timeout_sec}s waiting for $min_blobbers blobbers (attempt $attempt)"
            return 1
        fi

        # Try alloc_blobbers endpoint
        local result
        result=$(curl -s -G "${sharder_url}/v1/screst/${SC_ADDRESS}/alloc_blobbers" \
            --data-urlencode "allocation_data=${ALLOC_DATA}" 2>/dev/null)

        # Check if it returned an error or a valid array
        local count=0
        if echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null | grep -q '^[0-9]'; then
            count=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null || echo "0")
        fi

        if [ "$count" -ge "$min_blobbers" ]; then
            print_status "  Blobbers ready: $count eligible blobbers returned (attempt $attempt, elapsed ${elapsed}s)"
            return 0
        fi

        # Show progress every 30 seconds
        if [ $((attempt % 6)) -eq 0 ] || [ "$attempt" -eq 1 ]; then
            local err_msg
            err_msg=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error','?') if isinstance(d,dict) else 'OK')" 2>/dev/null || echo "?")
            print_warning "  Attempt $attempt: $count/$min_blobbers blobbers ready (${elapsed}s elapsed, msg: $err_msg)"
        fi

        sleep 5
    done
}

# Verify chain infrastructure health: sharders, miners, blobbers, containers.
# Reports any issues that could cause test failures.
# This is the primary deployment verification tool - run after any deployment or restart.
verify_chain_health() {
    print_header "Verifying Chain Health"

    local all_ok=true
    local warnings=0
    local SC_ADDRESS="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local sharder_url=""

    # ---- 1. Container Health (crash detection) ----
    print_status "Checking container health..."
    local crash_detected=false
    for container in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2; do
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            local restarts=$(docker inspect "$container" --format '{{.RestartCount}}' 2>/dev/null || echo "0")
            local uptime=$(docker inspect "$container" --format '{{.State.StartedAt}}' 2>/dev/null)
            local status=$(docker inspect "$container" --format '{{.State.Status}}' 2>/dev/null)
            if [ "$restarts" -gt 2 ]; then
                print_error "  $container: CRASH-LOOPING (restarts: $restarts, status: $status)"
                crash_detected=true
                all_ok=false
            elif [ "$restarts" -gt 0 ]; then
                print_warning "  $container: restarted $restarts time(s), status: $status"
                warnings=$((warnings + 1))
            else
                print_status "  $container: OK (restarts: 0)"
            fi
        else
            print_error "  $container: NOT RUNNING"
            all_ok=false
        fi
    done
    if $crash_detected; then
        print_error "  ACTION NEEDED: Crash-looping containers detected. Check logs with 'docker logs <container>'"
    fi

    # ---- 2. Sharder Health (direct check) ----
    print_status "Checking sharders..."
    local sharder_rounds=()
    for port in 7171 7172; do
        local stats
        stats=$(curl -s "http://127.0.0.1:${port}/v1/chain/get/stats" -m 5 2>/dev/null)
        if [ -n "$stats" ]; then
            local round=$(echo "$stats" | python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")
            print_status "  Sharder ($port): OK (round $round)"
            sharder_rounds+=("$round")
            if [ -z "$sharder_url" ]; then
                sharder_url="http://127.0.0.1:${port}"
            fi
        else
            print_error "  Sharder ($port): NOT RESPONDING"
            all_ok=false
        fi
    done

    # ---- 3. Miner Health (individual check + 0dns) ----
    print_status "Checking miners..."
    local miner_ok=0
    for i in 1 2 3 4; do
        local ip="198.18.0.7${i}"
        local port="707${i}"
        local stats
        stats=$(curl -s "http://${ip}:${port}/v1/chain/get/stats" -m 3 2>/dev/null)
        if [ -n "$stats" ]; then
            local round=$(echo "$stats" | python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "?")
            print_status "  Miner-$i ($ip:$port): OK (round $round)"
            miner_ok=$((miner_ok + 1))
        else
            print_error "  Miner-$i ($ip:$port): NOT RESPONDING"
        fi
    done
    if [ "$miner_ok" -lt 3 ]; then
        print_error "  Only $miner_ok/4 miners responding (need 3+ for consensus)"
        all_ok=false
    fi

    # Check 0dns miner list
    local miner_count
    miner_count=$(curl -s http://127.0.0.1:9091/network 2>/dev/null | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('miners',[])))" 2>/dev/null || echo "0")
    print_status "  0dns reports $miner_count miners"

    # ---- 4. Chain Progress Check ----
    if [ -n "$sharder_url" ]; then
        print_status "Checking chain progress..."
        local round1
        round1=$(curl -s "${sharder_url}/v1/chain/get/stats" -m 5 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")
        sleep 5
        local round2
        round2=$(curl -s "${sharder_url}/v1/chain/get/stats" -m 5 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")
        local delta=$((round2 - round1))
        if [ "$delta" -gt 0 ]; then
            print_status "  Chain advancing: $round1 -> $round2 (+$delta rounds in 5s)"
        elif [ "$round1" -gt 0 ]; then
            print_error "  Chain STUCK at round $round1 (no progress in 5s)"
            all_ok=false
        fi
    fi

    # ---- 5. Blobber Health Check Freshness ----
    if [ -n "$sharder_url" ]; then
        print_status "Checking blobbers on-chain..."
        local blobbers_json
        blobbers_json=$(curl -s "${sharder_url}/v1/screst/${SC_ADDRESS}/getblobbers" 2>/dev/null)
        if [ -n "$blobbers_json" ]; then
            echo "$blobbers_json" | python3 -c "
import json, sys, time
data = json.load(sys.stdin)
now = int(time.time())
total = 0; healthy = 0; stale = 0; killed = 0; enterprise = 0; nonlocal = 0
stale_list = []
for b in data.get('Nodes', []):
    total += 1
    if b.get('is_killed', False):
        killed += 1
        continue
    if b.get('is_enterprise', False):
        enterprise += 1
    host = b.get('url','').replace('https://','').replace('http://','').split(':')[0].split('/')[0]
    if not (host.startswith('198.18.') or host == 'localhost'):
        nonlocal += 1
    lhc = b.get('last_health_check', 0)
    age = now - lhc
    url = b.get('url', '?')
    if age < 300:  # 5 minutes
        healthy += 1
    else:
        stale += 1
        stale_list.append(f'    {url}: stale ({age}s ago)')
print(f'  Total: {total}, Healthy: {healthy}, Stale: {stale}, Killed: {killed}, Enterprise: {enterprise}')
if nonlocal > 0:
    print(f'  WARNING: {nonlocal} non-local blobber(s)')
if stale_list:
    print('  Stale blobbers (health check > 5min):')
    for s in stale_list:
        print(s)
" 2>/dev/null
            # Count allocatable non-enterprise blobbers (fresh health check, any URL)
            # Note: blobbers may use public URLs (e.g. test.zus.network) not just 198.18.x
            local alloc_count
            alloc_count=$(echo "$blobbers_json" | python3 -c "
import json, sys, time
data = json.load(sys.stdin)
now = int(time.time())
count = 0
for b in data.get('Nodes', []):
    if b.get('is_killed') or b.get('is_shut_down') or b.get('is_enterprise'):
        continue
    lhc = b.get('last_health_check', 0)
    if now - lhc < 5400:
        count += 1
print(count)
" 2>/dev/null || echo "0")
            if [ "$alloc_count" -ge 3 ]; then
                print_status "  Allocatable non-enterprise blobbers: $alloc_count (OK)"
            else
                print_error "  Allocatable non-enterprise blobbers: $alloc_count (need 3+, tests will fail!)"
                all_ok=false
            fi
        else
            print_error "  Could not fetch blobber list"
            all_ok=false
        fi
    fi

    # ---- 6. Blobber Container Health ----
    print_status "Checking blobber containers..."
    local blobber_containers_ok=0
    for i in 1 2 3 4 5 6 7 8 9; do
        local cname="blobber-${i}"
        if docker ps --format '{{.Names}}' | grep -q "^${cname}$"; then
            local restarts=$(docker inspect "$cname" --format '{{.RestartCount}}' 2>/dev/null || echo "0")
            if [ "$restarts" -gt 2 ]; then
                print_warning "  $cname: restarts=$restarts"
            fi
            blobber_containers_ok=$((blobber_containers_ok + 1))
        fi
    done
    print_status "  Running blobber containers: $blobber_containers_ok"

    echo ""
    if $all_ok && [ "$warnings" -eq 0 ]; then
        print_status "Chain health: ALL OK"
    elif $all_ok; then
        print_warning "Chain health: OK with $warnings warning(s)"
    else
        print_error "Chain health: ISSUES FOUND (see above)"
    fi
}

# Build and start the crawler service.
# The crawler watches allocations on-chain and indexes file metadata into elasticsearch.
# Prerequisites: elasticsearch running, chain live, testnet0 Docker network exists.
build_and_start_crawler() {
    print_header "Building and Starting Crawler"

    local CRAWLER_DIR="${BASE_DIR}/crawler"

    if [ ! -d "$CRAWLER_DIR" ]; then
        print_status "Cloning crawler repo..."
        cd "${BASE_DIR}"
        git clone https://github.com/0chain/crawler.git 2>/dev/null || {
            print_warning "Failed to clone crawler repo — skipping"
            return 0
        }
    fi

    if [ ! -f "$CRAWLER_DIR/docker.local/Dockerfile" ]; then
        print_warning "Crawler Dockerfile not found at $CRAWLER_DIR/docker.local/Dockerfile — skipping"
        return 0
    fi

    # Checkout correct gosdk branch for crawler and inject local gosdk
    checkout_gosdk_for_dependent "crawler"
    inject_local_gosdk "$CRAWLER_DIR" "${CRAWLER_DIR}/docker.local/Dockerfile"

    # Build the crawler Docker image
    print_status "Building crawler Docker image..."
    cd "$CRAWLER_DIR"
    docker compose -f docker.local/docker-compose.yml build 2>&1 | tail -5 || {
        cleanup_injected_gosdk "$CRAWLER_DIR"
        print_warning "Crawler Docker build failed"
        return 0
    }
    cleanup_injected_gosdk "$CRAWLER_DIR"
    print_status "Crawler image built successfully"

    # Ensure config directory and file exist
    local CRAWLER_CONFIG_DIR="${CRAWLER_DIR}/docker.local/config"
    local CRAWLER_CONFIG="${CRAWLER_CONFIG_DIR}/crawler.yaml"
    mkdir -p "$CRAWLER_CONFIG_DIR"

    local ZCN_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local client_id=$(jq -r '.client_id' "$ZCN_WALLET" 2>/dev/null)
    local private_key=$(jq -r '.keys[0].private_key' "$ZCN_WALLET" 2>/dev/null)
    local public_key=$(jq -r '.keys[0].public_key' "$ZCN_WALLET" 2>/dev/null)
    local chain_id="0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
    local genesis_block_id="ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4"

    local WO="--wallet ${ZCN_SC_OWNER_WALLET} --configDir ${ZCN_CONFIG_DIR} --config ${ZCN_CONFIG_FILE}"

    # Check if existing config has a valid allocation on-chain; if not, recreate config
    # Use targeted regex: look specifically under the 'allocations:' section (not wallet keys)
    local need_new_config=1
    if [ -f "$CRAWLER_CONFIG" ]; then
        local existing_alloc=$(python3 -c "
import re
txt = open('${CRAWLER_CONFIG}').read()
m = re.search(r'allocations:\s*\n\s*-\s+([0-9a-f]{64})', txt)
print(m.group(1) if m else '')
" 2>/dev/null || true)
        if [ -n "$existing_alloc" ]; then
            # Check allocation exists on-chain AND owner matches current wallet
            local alloc_json=$(curl -s "http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation?allocation=${existing_alloc}" 2>/dev/null)
            local alloc_owner=$(echo "$alloc_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('owner_id',''))" 2>/dev/null || true)
            if [ -n "$alloc_owner" ] && [ "$alloc_owner" = "$client_id" ]; then
                need_new_config=0
                print_status "Crawler config valid — allocation ${existing_alloc:0:12}... exists on-chain (owner matches)"
            elif [ -n "$alloc_owner" ]; then
                print_status "Crawler allocation owner mismatch (alloc: ${alloc_owner:0:16}..., wallet: ${client_id:0:16}...) — recreating"
            else
                print_status "Crawler config has stale allocation ${existing_alloc:0:12}... — recreating"
            fi
        fi
    fi

    if [ "$need_new_config" = "1" ]; then
        # Fund the wallet via faucet (need ~100 ZCN to lock for allocation; pour 100 ZCN x 2 = 200 ZCN)
        print_status "Funding crawler wallet via faucet..."
        ${ZWALLET} faucet --methodName pour --tokens 100 --silent $WO 2>/dev/null || true
        sleep 2

        print_status "Creating crawler allocation on-chain (data=10, parity=2 = all 12 regular blobbers)..."
        local alloc_output
        alloc_output=$(${ZBOX} newallocation --size 10737418240 --lock 100 --data 10 --parity 2 \
            --silent $WO 2>&1) || true
        local crawler_alloc_id=$(echo "$alloc_output" | grep -oE '[0-9a-f]{64}' | head -1)

        if [ -z "$crawler_alloc_id" ]; then
            print_warning "Could not create crawler allocation — starting crawler with empty allocations"
            crawler_alloc_id=""
        else
            print_status "Crawler allocation created: ${crawler_alloc_id}"
        fi

        # Write alloc entry (empty list if no allocation)
        local alloc_entry=""
        if [ -n "$crawler_alloc_id" ]; then
            alloc_entry="  - ${crawler_alloc_id}"
        fi

        cat > "$CRAWLER_CONFIG" << CRAWLEREOF
version: 1.0

logging:
  level: "debug"
  console: true

host: http://198.18.0.100:9091
port: 9081

server_chain:
  id: "${chain_id}"
  owner: "${client_id}"
  genesis_block:
    id: "${genesis_block_id}"
  signature_scheme: "bls0chain"

block_worker: http://198.18.0.100:9091
wallet_client_id: ${client_id}
wallet_private_keys:
  - ${private_key}
wallet_public_keys:
  - ${public_key}

allocations:
${alloc_entry}

kafka:
  enabled: true
  host: "kafka:9092"
  username: "admin"
  password: "admin-secret"
  eventsTopic: "monitor"
  eventsGroupId: "events-consumer"
  eventsRetryTopic: "monitor-retry"
  triggerRound: 1
CRAWLEREOF
        print_status "Generated crawler config at $CRAWLER_CONFIG"
    fi

    # Fix docker-compose: crawler reads config from /usr/src/app/docker.local/config.
    # Patch both the volume mount AND the --config_path command arg.
    sed -i 's|- ./config:/crawler/config|- ./config:/usr/src/app/docker.local/config|g' \
        "$CRAWLER_DIR/docker.local/docker-compose.yml" 2>/dev/null || true
    sed -i 's|--config_path=/crawler/config|--config_path=/usr/src/app/docker.local/config|g' \
        "$CRAWLER_DIR/docker.local/docker-compose.yml" 2>/dev/null || true

    # Stop existing crawler if running (to pick up new image)
    if docker ps --format "{{.Names}}" | grep -q "^crawler$"; then
        print_status "Stopping existing crawler container..."
        docker stop crawler 2>/dev/null || true
        docker rm crawler 2>/dev/null || true
    fi

    # Start crawler via docker compose
    print_status "Starting crawler container..."
    cd "$CRAWLER_DIR"
    docker compose -f docker.local/docker-compose.yml up -d 2>&1 || {
        print_warning "Failed to start crawler container"
        return 0
    }
    sleep 3

    # Health check
    if docker ps --format "{{.Names}}" | grep -q "^crawler$"; then
        local status=$(docker inspect crawler --format "{{.State.Status}}" 2>/dev/null)
        print_status "Crawler container is ${status}"
    else
        print_warning "Crawler container is not running"
    fi
}

# Build and start zs3server (S3-compatible gateway backed by Zus blockchain storage).
# zs3server runs as a native Go binary (not Docker), using the minio gateway zcn command.
# Prerequisites: gosdk dependencies installed, zboxcli available, chain live with blobbers.
build_and_start_zs3server() {
    print_header "Building and Starting zs3server"

    local ZS3_DIR="${BASE_DIR}/zs3server"

    if [ ! -d "$ZS3_DIR" ]; then
        print_status "Cloning zs3server repo..."
        cd "${BASE_DIR}"
        git clone https://github.com/0chain/zs3server.git 2>/dev/null || {
            print_warning "Failed to clone zs3server repo — skipping"
            return 0
        }
    fi

    # Checkout correct gosdk branch for zs3server (enterprise-blobber branch)
    # zs3server depends on gosdk enterprise-blobber branch, not the default lfb branch.
    # Without this, go mod tidy during build picks up the wrong gosdk code.
    checkout_gosdk_for_dependent "zs3server"

    # Check if the minio binary needs to be built
    if [ ! -f "$ZS3_DIR/minio" ]; then
        print_status "Building zs3server (minio binary)..."
        cd "$ZS3_DIR"

        # Build using golang:1.22.5 Docker (zs3server requires go 1.22, Makefile uses golang:1.20 which fails)
        print_status "Building with Docker golang:1.22.5..."
        docker run --rm -v "${ZS3_DIR}:/minio" -w /minio golang:1.22.5 \
            go build -buildvcs=false -o minio . 2>&1 | tail -10 || true

        # Verify binary was created; fallback to native go build if not
        if [ ! -f "$ZS3_DIR/minio" ]; then
            print_status "Docker build failed, trying native go build..."
            CGO_ENABLED=1 go build -buildvcs=false -o minio . 2>&1 | tail -5 || {
                print_warning "zs3server build failed — skipping"
                return 0
            }
        fi
        [ -f "$ZS3_DIR/minio" ] && print_status "zs3server binary built successfully" || { print_warning "zs3server binary not created"; return 0; }
    else
        print_status "zs3server binary already exists at $ZS3_DIR/minio"
    fi

    # Ensure zcnconfig directory and config exist
    local ZS3_CONFIG_DIR="${ZS3_DIR}/zcnconfig"
    mkdir -p "$ZS3_CONFIG_DIR"

    if [ ! -f "${ZS3_CONFIG_DIR}/config.yaml" ]; then
        print_status "Generating zs3server config..."
        cat > "${ZS3_CONFIG_DIR}/config.yaml" << ZS3CONFIGEOF
---
block_worker: http://localhost:9091
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 10
confirmation_chain_length: 3
max_txn_query: 10
query_sleep_time: 5

miners:
  - http://localhost:7071
  - http://localhost:7072
  - http://localhost:7073
  - http://localhost:7074

sharders:
  - http://localhost:7171
  - http://localhost:7172
ZS3CONFIGEOF
        print_status "Generated zs3server config"
    fi

    # Copy wallet if not present
    if [ ! -f "${ZS3_CONFIG_DIR}/wallet.json" ]; then
        local ZCN_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
        if [ -f "$ZCN_WALLET" ]; then
            cp "$ZCN_WALLET" "${ZS3_CONFIG_DIR}/wallet.json"
            print_status "Copied wallet to zs3server config"
        else
            print_warning "No wallet found to copy to zs3server — will need manual setup"
        fi
    fi

    # Create allocation for zs3server (delegates to start_zs3server which handles this)
    # start_zs3server creates allocation, extends it, and starts the process
    start_zs3server
}

# Refresh crawler allocation — creates a new allocation for the crawler to index
# and updates the crawler config. Call after blobbers are staked and chain is live.
# The crawler watches allocations on-chain and indexes file metadata into elasticsearch.
# Without a valid allocation, the crawler has nothing to index.
refresh_crawler_allocation() {
    print_header "Refreshing Crawler Allocation"

    local CRAWLER_DIR="${BASE_DIR}/crawler"
    local CRAWLER_CONFIG="${CRAWLER_DIR}/docker.local/config/crawler.yaml"

    if [ ! -d "$CRAWLER_DIR" ]; then
        print_warning "Crawler repo not found at $CRAWLER_DIR — skipping"
        return 0
    fi

    if [ ! -f "$CRAWLER_CONFIG" ]; then
        print_warning "Crawler config not found at $CRAWLER_CONFIG — skipping"
        return 0
    fi

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent"

    # Check if existing allocation is still valid
    local existing_alloc
    existing_alloc=$(grep -A1 "^allocations:" "$CRAWLER_CONFIG" 2>/dev/null | grep "^  - " | head -1 | sed "s/.*- //" | tr -d " " || true)

    if [ -n "$existing_alloc" ] && [ "$existing_alloc" != "null" ]; then
        local alloc_stdout alloc_stderr
        alloc_stdout=$($ZBOX getallocation --allocation "$existing_alloc" $W 2>/dev/null || echo "")
        alloc_stderr=$($ZBOX getallocation --allocation "$existing_alloc" $W 2>&1 >/dev/null || echo "")
        if echo "$alloc_stdout" | grep -q "^allocation:" \
           && ! echo "$alloc_stderr" | grep -qi "expired\|invalid_parameters\|not found\|alloc_not_found"; then
            # Check if allocation covers all 9 regular blobbers (data+parity shards >= 9)
            local alloc_shards
            alloc_shards=$(echo "$alloc_stdout" | grep -E "data_shards|parity_shards" | \
                grep -oE "[0-9]+" | paste -sd+ | python3 -c "import sys; print(sum(int(x) for x in sys.stdin.read().split('+') if x.strip()))" 2>/dev/null || echo "0")
            if [ "${alloc_shards:-0}" -ge 12 ]; then
                print_status "Existing crawler allocation is valid and covers ${alloc_shards} blobbers: ${existing_alloc:0:16}..."
                return 0
            fi
            print_status "Existing allocation only has ${alloc_shards} blobbers (need 12), recreating..."
        else
            print_status "Existing allocation ${existing_alloc:0:16}... is invalid/expired, creating new one"
        fi
    fi

    # Create a new allocation for the crawler (10 data + 2 parity = 12 blobbers = all regular blobbers)
    # Enterprise blobbers are excluded from regular allocations by the SC.
    print_status "Creating new crawler allocation (data=10, parity=2, covers all 12 regular blobbers)..."
    local alloc_output
    alloc_output=$($ZBOX newallocation \
        --data 10 --parity 2 --size 10737418240 --lock 100 \
        $W 2>&1) || true

    # Extract allocation ID from output — look for "ID:" or "Allocation:" marker first,
    # fall back to last 64-hex string (avoids picking up tx/block hashes that appear earlier)
    local new_alloc
    new_alloc=$(echo "$alloc_output" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1 || true)
    if [ -z "$new_alloc" ]; then
        new_alloc=$(echo "$alloc_output" | grep -oE "[a-f0-9]{64}" | tail -1 || true)
    fi

    if [ -z "$new_alloc" ]; then
        print_warning "Failed to create crawler allocation"
        print_warning "Output: $(echo "$alloc_output" | tail -3)"
        return 0
    fi

    print_status "New crawler allocation: ${new_alloc}"

    # Update crawler config — replace the ENTIRE allocations list (not append) to avoid stale entries
    python3 - <<PYEOF
import re
config_file = "${CRAWLER_CONFIG}"
new_alloc = "${new_alloc}"
try:
    with open(config_file) as f:
        content = f.read()
    new_block = "allocations:\n  - {}\n".format(new_alloc)
    if re.search(r'^allocations:', content, re.MULTILINE):
        # Replace entire allocations block (key + all indented list items).
        # Pattern: 'allocations:' followed by any number of lines that start with whitespace.
        # Use [\s\S] approach: match from 'allocations:' through all consecutive indented lines.
        content = re.sub(
            r'^allocations:[ \t]*\n(?:[ \t]+[^\n]*\n?)*',
            new_block,
            content,
            flags=re.MULTILINE
        )
    else:
        content = content.rstrip() + "\n\n" + new_block
    with open(config_file, 'w') as f:
        f.write(content)
    print("Updated crawler config with allocation: " + new_alloc[:16] + "...")
except Exception as e:
    print("ERROR updating crawler config: " + str(e))
PYEOF

    print_status "Updated crawler config with new allocation"

    # Restart crawler container if running
    if docker ps --format "{{.Names}}" | grep -q "^crawler$"; then
        print_status "Restarting crawler container..."
        docker restart crawler 2>/dev/null || true
        print_status "Crawler restarted with new allocation"
    else
        print_warning "Crawler container not running — start it manually"
    fi
}

# Perpetual background monitor that watches the crawler's allocation and renews it before
# it expires. Runs as a detached background loop, logging to /tmp/crawler_monitor.log.
# The loop checks every 30 minutes; if the allocation is expired/invalid, it creates a new
# one, updates the crawler config, and restarts the crawler container automatically.
start_crawler_monitor() {
    print_header "Starting Crawler Allocation Monitor"

    local CRAWLER_DIR="${BASE_DIR}/crawler"
    local CRAWLER_CONFIG="${CRAWLER_DIR}/docker.local/config/crawler.yaml"
    local MONITOR_LOG="/tmp/crawler_monitor.log"
    local MONITOR_PID_FILE="/tmp/crawler_monitor.pid"

    if [ ! -d "$CRAWLER_DIR" ]; then
        print_warning "Crawler repo not found at $CRAWLER_DIR — skipping monitor"
        return 0
    fi

    # Kill any existing monitor
    if [ -f "$MONITOR_PID_FILE" ]; then
        local old_pid
        old_pid=$(cat "$MONITOR_PID_FILE" 2>/dev/null || true)
        if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
            print_status "Stopping existing monitor (pid $old_pid)"
            kill "$old_pid" 2>/dev/null || true
        fi
        rm -f "$MONITOR_PID_FILE"
    fi

    local ZBOX_BIN="${BASE_DIR}/zboxcli/zbox"
    local ZWALLET_BIN="${BASE_DIR}/zwalletcli/zwallet"
    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"

    local CHECK_INTERVAL=3600  # 1 hour (allocations last 30 days with --lock 10)
    local SHARDER_URL="${SHARDER_URL:-http://198.18.0.81:7171}"
    local STORAGE_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

    # Run perpetual monitor loop in background subshell
    # NOTE: check happens FIRST on startup (no sleep before first check), then every CHECK_INTERVAL.
    (
        echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Crawler allocation monitor started (interval=${CHECK_INTERVAL}s)" >> "$MONITOR_LOG"
        while true; do
            echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Checking crawler allocation..." >> "$MONITOR_LOG"

            existing_alloc=$(grep -A1 "^allocations:" "$CRAWLER_CONFIG" 2>/dev/null \
                | grep "^  - " | head -1 | sed "s/.*- //" | tr -d " " || true)

            if [ -n "$existing_alloc" ] && [ "$existing_alloc" != "null" ]; then
                # Check expiry via SC REST API (reliable — no SDK stderr false-positives)
                # Renew if expiry is within 7 days (604800s)
                local expiry_check
                expiry_check=$(curl -sf "${SHARDER_URL}/v1/screst/${STORAGE_SC}/allocation?allocation=${existing_alloc}" \
                    2>/dev/null | python3 -c '
import json,sys,time
try:
    d=json.load(sys.stdin)
    exp=d.get("expiration_date",0)
    remaining=exp-time.time()
    print("ok" if remaining > 604800 else "renew")
except: print("renew")
' 2>/dev/null || echo "renew")
                if [ "$expiry_check" = "ok" ]; then
                    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Allocation ${existing_alloc:0:16}... healthy (>7 days remaining)" >> "$MONITOR_LOG"
                    sleep "$CHECK_INTERVAL"
                    continue
                fi
                echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Allocation ${existing_alloc:0:16}... expiring soon — renewing" >> "$MONITOR_LOG"
                sed -i.bak "/^  - ${existing_alloc}/d" "$CRAWLER_CONFIG" 2>/dev/null || true
                rm -f "${CRAWLER_CONFIG}.bak"
            else
                echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] No valid allocation in config — creating new one" >> "$MONITOR_LOG"
            fi

            # Create new allocation covering all 9 regular blobbers (--lock 25 gives ~30 days with time_unit=720h)
            alloc_output=$("$ZBOX_BIN" newallocation \
                --data 7 --parity 2 --size 10737418240 --lock 25 \
                $W 2>&1) || true
            # Extract allocation ID — prefer "ID:" marker, fall back to last 64-hex string
            new_alloc=$(echo "$alloc_output" | grep -oP '(?i)(?:ID:|allocation[^:]*:)\s*\K[a-f0-9]{64}' | head -1 || true)
            if [ -z "$new_alloc" ]; then
                new_alloc=$(echo "$alloc_output" | grep -oE "[a-f0-9]{64}" | tail -1 || true)
            fi

            if [ -z "$new_alloc" ]; then
                echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Failed to create allocation: $alloc_output" >> "$MONITOR_LOG"
                sleep "$CHECK_INTERVAL"
                continue
            fi

            echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] New allocation: $new_alloc" >> "$MONITOR_LOG"

            # Update crawler config — replace ENTIRE allocations list (not append) to avoid stale entries
            python3 - <<PYEOF 2>>"$MONITOR_LOG"
import re
config_file = "${CRAWLER_CONFIG}"
new_alloc = "${new_alloc}"
try:
    with open(config_file) as f:
        content = f.read()
    new_block = "allocations:\n  - {}\n".format(new_alloc)
    if re.search(r'^allocations:', content, re.MULTILINE):
        content = re.sub(r'^allocations:(\s*\n(?:[ \t]+[^\n]*\n)*)', new_block, content, flags=re.MULTILINE)
    else:
        content = content.rstrip() + "\n\n" + new_block
    with open(config_file, 'w') as f:
        f.write(content)
    print("Updated crawler config with allocation: " + new_alloc[:16] + "...")
except Exception as e:
    print("ERROR updating crawler config: " + str(e))
PYEOF

            # Restart crawler to pick up new allocation
            if docker ps --format "{{.Names}}" | grep -q "^crawler$"; then
                docker restart crawler >> "$MONITOR_LOG" 2>&1
                echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Crawler restarted with new allocation" >> "$MONITOR_LOG"
            fi
            sleep "$CHECK_INTERVAL"
        done
    ) &
    local monitor_pid=$!
    echo "$monitor_pid" > "$MONITOR_PID_FILE"
    print_status "Crawler monitor started (pid $monitor_pid, interval=${CHECK_INTERVAL}s), log: $MONITOR_LOG"
    return 0
}

# Verify all services including Docker network connectivity.
# Tests access services via Docker IPs (198.18.0.x), not localhost.
# This function checks both localhost and Docker network access.
verify_services() {
    print_header "Verifying Services"

    local all_ok=true
    local critical_fail=false

    # Helper to check a service on both localhost and Docker network
    check_service() {
        local name="$1"
        local localhost_url="$2"
        local docker_url="$3"      # optional: Docker network URL (how tests reach it)
        local critical="${4:-false}" # if true, failure marks critical

        local localhost_ok=false
        local docker_ok=false

        if curl -s -o /dev/null -w '' "$localhost_url" -m 3 2>/dev/null; then
            localhost_ok=true
        fi
        if [ -n "$docker_url" ]; then
            if curl -s -o /dev/null -w '' "$docker_url" -m 3 2>/dev/null; then
                docker_ok=true
            fi
        else
            docker_ok=$localhost_ok  # no Docker URL, same as localhost
        fi

        if $localhost_ok && ($docker_ok || [ -z "$docker_url" ]); then
            print_status "$name: OK"
        elif $localhost_ok && ! $docker_ok; then
            print_warning "$name: localhost OK but Docker network UNREACHABLE ($docker_url)"
            all_ok=false
        else
            if [ "$critical" = "true" ]; then
                print_error "$name: FAILED (critical)"
                critical_fail=true
            else
                print_warning "$name: Not responding"
            fi
            all_ok=false
        fi
    }

    # ---- Critical Services (tests fail without these) ----
    print_status "Critical services:"
    check_service "  0dns (9091)" "http://127.0.0.1:9091/network" "http://198.18.0.100:9091/network" true

    # Check zauth via Docker IP (how API tests reach it)
    check_service "  zauth (8080)" "http://127.0.0.1:8080/" "http://198.18.0.200:8080/" false
    check_service "  zvault (8090)" "http://127.0.0.1:8090/" "http://198.18.0.210:8090/" false
    check_service "  0box (9081)" "http://127.0.0.1:9081/" "http://198.18.0.220:9081/" false

    # ---- Optional Services ----
    print_status "Optional services:"
    check_service "  Elasticsearch (9200)" "http://localhost:9200/_cluster/health" "" false
    check_service "  zs3server (9100)" "http://localhost:9100/minio/health/live" "" false
    check_service "  Gotenberg/render (3010)" "http://localhost:3010/health" "" false
    # Crawler check (Docker container)
    if docker ps --format "{{.Names}}" | grep -q "^crawler$"; then
        print_status "  Crawler: running (container)"
    else
        print_warning "  Crawler: not running"
    fi

    # ---- Kafka (check container + broker connectivity) ----
    print_status "Kafka:"
    if docker ps --format '{{.Names}}' | grep -q "kafka"; then
        local kafka_status=$(docker inspect kafka --format '{{.State.Status}}' 2>/dev/null)
        local kafka_restarts=$(docker inspect kafka --format '{{.RestartCount}}' 2>/dev/null || echo "0")
        print_status "  Container: $kafka_status (restarts: $kafka_restarts)"
        # Try to list topics to verify broker is actually working
        local topics
        topics=$(docker exec kafka kafka-topics.sh --list --bootstrap-server localhost:9092 2>/dev/null | head -5)
        if [ -n "$topics" ]; then
            local topic_count=$(echo "$topics" | wc -l | tr -d ' ')
            print_status "  Broker: OK ($topic_count topics)"
        else
            print_warning "  Broker: NOT responding (SASL issue?)"
        fi
    else
        print_warning "  Kafka: Not running (graph/aggregate API tests will skip)"
    fi

    # ---- Kafka -> 0box Event Pipeline Verification ----
    if docker ps --format '{{.Names}}' | grep -q "kafka" && docker ps --format '{{.Names}}' | grep -q "0box"; then
        print_status "Kafka -> 0box pipeline:"
        local sharder_round
        sharder_round=$(curl -s "http://127.0.0.1:7171/v1/chain/get/stats" -m 5 2>/dev/null | \
            python3 -c "import json,sys; print(json.load(sys.stdin).get('current_round',0))" 2>/dev/null || echo "0")

        local obox_round
        obox_round=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT COALESCE(MAX(round),0) FROM snapshots;" 2>/dev/null | tr -d ' \n' || echo "0")

        if [ "$sharder_round" -gt 0 ] && [ "$obox_round" -gt 0 ]; then
            local pipeline_lag=$((sharder_round - obox_round))
            if [ "$pipeline_lag" -lt 100 ]; then
                print_status "  Pipeline: OK (chain round: $sharder_round, 0box processed: $obox_round, lag: $pipeline_lag)"
            elif [ "$pipeline_lag" -lt 1000 ]; then
                print_warning "  Pipeline: LAGGING (chain round: $sharder_round, 0box processed: $obox_round, lag: $pipeline_lag)"
            else
                print_error "  Pipeline: STALLED (chain round: $sharder_round, 0box processed: $obox_round, lag: $pipeline_lag)"
                print_error "  Check: sharder trigger_round, 0box triggerRound, Kafka connectivity"
                all_ok=false
            fi
        else
            print_warning "  Could not determine pipeline status (sharder: $sharder_round, 0box: $obox_round)"
        fi

        # Check 0box trigger_round vs chain round
        local trigger_round
        trigger_round=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT COALESCE(MIN(round),0) FROM snapshots;" 2>/dev/null | tr -d ' \n' || echo "0")
        if [ "$trigger_round" -gt 0 ]; then
            print_status "  0box first snapshot at round: $trigger_round"
        fi
    fi

    # ---- Firebase Authentication ----
    print_status "Firebase auth:"
    local FIREBASE_API_KEY="${FIREBASE_TEST_API_KEY:-}"
    local FIREBASE_EMAIL="${FIREBASE_TEST_EMAIL:-test_system_test@0chain.net}"
    local FIREBASE_PASSWORD="${FIREBASE_TEST_PASSWORD:-}"
    local firebase_response
    firebase_response=$(curl -s "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=${FIREBASE_API_KEY}" \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"${FIREBASE_EMAIL}\",\"password\":\"${FIREBASE_PASSWORD}\",\"returnSecureToken\":true}" \
        -m 10 2>/dev/null || true)
    if echo "$firebase_response" | grep -q '"idToken"'; then
        local firebase_uid=$(echo "$firebase_response" | python3 -c "import sys,json; print(json.load(sys.stdin).get('localId',''))" 2>/dev/null || true)
        print_status "  Firebase auth: OK (uid: ${firebase_uid:0:16}...)"
    elif echo "$firebase_response" | grep -q '"error"'; then
        local firebase_err=$(echo "$firebase_response" | python3 -c "import sys,json; print(json.load(sys.stdin).get('error',{}).get('message',''))" 2>/dev/null || true)
        print_warning "  Firebase auth: FAILED ($firebase_err)"
        print_warning "  0box API tests requiring authentication will fail"
    else
        print_warning "  Firebase auth: Could not reach Firebase API"
    fi

    # ---- Web Apps (Vult, Bolt, Blimp, Explorer, Chimney) ----
    print_status "Web apps:"
    local _prefix="${APP_DOMAIN_PREFIX:-test}"
    local -A _web_app_domains=(
        [vult]="${_prefix}.vult.network"
        [bolt]="${_prefix}.bolt.holdings"
        [blimp]="${_prefix}.blimp.software"
        [explorer]="${_prefix}.atlus.cloud"
        [chimney]="${_prefix}.chimney.software"
    )
    local web_apps_ok=0 web_apps_total=0
    for app in vult bolt blimp explorer chimney; do
        local domain="${_web_app_domains[$app]}"
        web_apps_total=$((web_apps_total + 1))
        local http_code
        http_code=$(curl -sk -o /dev/null -w '%{http_code}' "https://${domain}/" -m 5 2>/dev/null || echo "000")
        if [ "$http_code" = "200" ] || [ "$http_code" = "301" ] || [ "$http_code" = "302" ]; then
            print_status "  ${app} (${domain}): OK ($http_code)"
            web_apps_ok=$((web_apps_ok + 1))
        else
            print_warning "  ${app} (${domain}): NOT responding ($http_code)"
            all_ok=false
        fi
    done
    if [ "$web_apps_ok" -eq "$web_apps_total" ]; then
        print_status "  All $web_apps_total web apps responding"
    else
        print_warning "  $web_apps_ok/$web_apps_total web apps responding"
    fi

    # ---- Blobber Direct Connectivity (from host) ----
    print_status "Blobber connectivity:"
    local blobbers_reachable=0
    local blobbers_total=0
    # Regular blobbers 1-12: port = 505N for 1-9, 50610 for 10, 50611 for 11, 50612 for 12
    for i in $(seq 1 12); do
        local port
        case $i in 10) port=50610;; 11) port=50611;; 12) port=50612;; *) port="505${i}";; esac
        blobbers_total=$((blobbers_total + 1))
        if curl -s -o /dev/null -w '' "http://localhost:${port}/" -m 2 2>/dev/null; then
            blobbers_reachable=$((blobbers_reachable + 1))
        fi
    done
    # Enterprise blobbers 1-3: port = 507${N}
    for i in 1 2 3; do
        local port="507${i}"
        blobbers_total=$((blobbers_total + 1))
        if curl -s -o /dev/null -w '' "http://localhost:${port}/" -m 2 2>/dev/null; then
            blobbers_reachable=$((blobbers_reachable + 1))
        fi
    done
    print_status "  Reachable blobbers (direct): $blobbers_reachable/$blobbers_total"

    # ---- Blobber On-Chain Status ----
    local active_blobbers=$($ZBOX ls-blobbers --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null \
        | jq 'length' 2>/dev/null || echo "0")
    print_status "  On-chain active blobbers: $active_blobbers"

    local zero_read_price=$($ZBOX ls-blobbers --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null \
        | jq '[.[] | select(.terms.read_price == 0)] | length' 2>/dev/null || echo "0")
    print_status "  Blobbers with read_price=0: $zero_read_price"

    # ---- Delegate Wallet Verification (CRITICAL) ----
    # delegate_wallet is set at first blobber registration and CANNOT be changed.
    # If it doesn't match blobber_owner_wallet.json, bl-update tests will ALL fail.
    print_status "Delegate wallet verification:"
    verify_blobber_delegate_wallets

    # ---- Enterprise Blobber Count ----
    # NOTE: Chain may not return is_enterprise flag; also match by URL pattern
    local enterprise_count=$($ZBOX ls-blobbers --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null \
        | jq '[.[] | select(.is_enterprise == true or (.url | test("198\\.18\\.0\\.20[1-5]|:507[1-5]|eblobber")))] | length' 2>/dev/null || echo "0")
    print_status "  Enterprise blobbers: $enterprise_count"
    if [ "$enterprise_count" -lt 3 ]; then
        print_warning "  Need at least 3 enterprise blobbers for tokenomics tests!"
    fi

    # ---- Miner/Sharder Health Transactions ----
    print_status "Miner/Sharder health transactions:"
    local miner_sc_addr="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
    local sharder_url="http://127.0.0.1:7172"
    local miner_list
    miner_list=$(curl -s "${sharder_url}/v1/screst/${miner_sc_addr}/getMinerList" -m 5 2>/dev/null || echo "")
    if [ -n "$miner_list" ] && echo "$miner_list" | python3 -c "import sys,json; json.load(sys.stdin)" &>/dev/null; then
        local now_ts
        now_ts=$(date +%s)
        local stale_miners=0
        local total_miners
        total_miners=$(echo "$miner_list" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('Nodes',[])))" 2>/dev/null || echo "0")
        stale_miners=$(echo "$miner_list" | python3 -c "
import sys,json
d=json.load(sys.stdin)
now=$now_ts
stale=0
for n in d.get('Nodes',[]):
    lhc=n.get('simple_miner',n).get('last_health_check',0)
    if now-lhc > 7200:
        stale+=1
print(stale)
" 2>/dev/null || echo "0")
        if [ "$stale_miners" -eq 0 ]; then
            print_status "  Miners: $total_miners healthy (all recent health txns)"
        else
            print_warning "  Miners: $stale_miners/$total_miners have STALE health transactions (>2h ago)"
            print_warning "  Miners may need funding: bash scripts/deploy_local.sh fund"
        fi
    else
        print_warning "  Miners: Could not query miner list"
    fi

    local sharder_list
    sharder_list=$(curl -s "${sharder_url}/v1/screst/${miner_sc_addr}/getSharderList" -m 5 2>/dev/null || echo "")
    if [ -n "$sharder_list" ] && echo "$sharder_list" | python3 -c "import sys,json; json.load(sys.stdin)" &>/dev/null; then
        local now_ts
        now_ts=$(date +%s)
        local stale_sharders=0
        local total_sharders
        total_sharders=$(echo "$sharder_list" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('Nodes',[])))" 2>/dev/null || echo "0")
        stale_sharders=$(echo "$sharder_list" | python3 -c "
import sys,json
d=json.load(sys.stdin)
now=$now_ts
stale=0
for n in d.get('Nodes',[]):
    lhc=n.get('simple_miner',n).get('last_health_check',0)
    if now-lhc > 7200:
        stale+=1
print(stale)
" 2>/dev/null || echo "0")
        if [ "$stale_sharders" -eq 0 ]; then
            print_status "  Sharders: $total_sharders healthy (all recent health txns)"
        else
            print_warning "  Sharders: $stale_sharders/$total_sharders have STALE health transactions (>2h ago)"
        fi
    else
        print_warning "  Sharders: Could not query sharder list"
    fi

    # ---- Rewards Pipeline (provider_rewards table + snapshot totals) ----
    if docker ps --format '{{.Names}}' | grep -q "postgres-0box"; then
        print_status "Rewards pipeline (0box):"
        local rewards_count
        rewards_count=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT COUNT(*) FROM provider_rewards;" 2>/dev/null | tr -d ' \n' || echo "0")
        if [ "${rewards_count:-0}" -gt 0 ]; then
            print_status "  provider_rewards: $rewards_count rows (rewards will appear in charts)"
        else
            print_warning "  provider_rewards: 0 rows — reward charts will show all zeros!"
            print_warning "  Run backfill SQL or wait for next reward event after 0box restart"
            all_ok=false
        fi

        local snapshot_miner_rewards
        snapshot_miner_rewards=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT COALESCE(miner_total_rewards,0) FROM snapshots ORDER BY round DESC LIMIT 1;" \
            2>/dev/null | tr -d ' \n' || echo "0")
        if [ "${snapshot_miner_rewards:-0}" -gt 0 ]; then
            print_status "  Latest snapshot miner_total_rewards: $snapshot_miner_rewards (non-zero, charts OK)"
        else
            print_warning "  Latest snapshot miner_total_rewards: 0 (reward charts may still be empty)"
            print_warning "  Wait 1-2 rounds for provider_rewards to accumulate or run backfill SQL"
        fi
    fi

    # ---- Crawler QoS (publishing to Kafka) ----
    if docker ps --format '{{.Names}}' | grep -q "^crawler$"; then
        print_status "Crawler QoS:"
        local crawler_publishes
        crawler_publishes=$(docker logs crawler --tail=100 2>&1 | grep -c "Publishing to kafka" || true)
        local crawler_errors
        crawler_errors=$(docker logs crawler --tail=100 2>&1 | grep -cE "invalid_signature|error.*blobber|failed to upload|consensus_not_met" || true)
        if [ "$crawler_publishes" -gt 0 ] && [ "$crawler_errors" -eq 0 ]; then
            print_status "  Crawler: GOOD QoS ($crawler_publishes publishes, 0 errors in last 100 lines)"
        elif [ "$crawler_publishes" -gt 0 ]; then
            print_warning "  Crawler: Publishing ($crawler_publishes), but $crawler_errors errors detected"
        else
            print_warning "  Crawler: No recent Kafka publishes ($crawler_errors errors in last 100 lines)"
            print_warning "  Check: docker logs crawler --tail=50"
            all_ok=false
        fi
    fi

    echo ""
    if $critical_fail; then
        print_error "CRITICAL services are down - tests WILL fail!"
    elif $all_ok; then
        print_status "All services are running!"
    else
        print_warning "Some services may need attention (see above)"
    fi
}

# Verify that on-chain blobber delegate_wallet matches the expected wallet.
# CRITICAL: delegate_wallet is immutable after first registration. If it doesn't
# match blobber_owner_wallet.json, all bl-update/bl-config tests will fail.
verify_blobber_delegate_wallets() {
    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local expected_delegate=$(jq -r '.client_id' "$SC_OWNER_WALLET" 2>/dev/null)

    if [ -z "$expected_delegate" ]; then
        print_warning "  Could not determine expected delegate wallet"
        return 0
    fi

    local blobber_json=$($ZBOX ls-blobbers --json --silent \
        --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE 2>/dev/null || echo "[]")

    local total=$(echo "$blobber_json" | jq 'length' 2>/dev/null || echo "0")
    local matching=$(echo "$blobber_json" | jq --arg dw "$expected_delegate" \
        '[.[] | select(.delegate_wallet == $dw or .stake_pool_settings.delegate_wallet == $dw)] | length' 2>/dev/null || echo "0")
    local mismatched=$((total - matching))

    if [ "$mismatched" -gt 0 ]; then
        print_error "  DELEGATE WALLET MISMATCH: $mismatched/$total blobbers have wrong delegate_wallet!"
        print_error "  Expected: ${expected_delegate:0:16}..."
        print_error "  This means bl-update, bl-config, stake pool tests will FAIL for those blobbers."
        print_error "  Fix: Redeploy blobbers from scratch with correct delegate_wallet in config."
        # Show which blobbers are mismatched
        echo "$blobber_json" | jq -r --arg dw "$expected_delegate" \
            '.[] | select(.delegate_wallet != $dw and (.stake_pool_settings.delegate_wallet // "") != $dw) | "    Mismatched: \(.id[0:16])... delegate=\(.delegate_wallet[0:16] // .stake_pool_settings.delegate_wallet[0:16])... enterprise=\(.is_enterprise)"' 2>/dev/null || true
    else
        print_status "  All $total blobbers have correct delegate_wallet (${expected_delegate:0:16}...)"
    fi
}

# ========================================================================
# TEST SETUP FUNCTIONS
# These set up wallets and configuration needed for system tests to pass.
# ========================================================================

# Setup test wallets: copies the chain's SC owner wallet and blobber delegate
# wallet to all test suite config directories.
#
# Key insight: The test suites reference these special wallets:
#   - sc_owner_wallet.json     → used for sc-update-config, mn-update-config
#   - blobber_owner_wallet.json → used for bl-update (must match blobber delegate_wallet)
#   - zcnsc_owner_wallet.json  → same as sc_owner for ZCNSC operations
#   - staking_wallet.json      → general staking operations
#
# On a fresh chain, the SC owner is defined in 0chain sc.yaml and matches
# the wallet at ~/.zcn/local.json. The blobber delegate wallet is the same
# wallet (set in blobber config). We copy this wallet to all test dirs.
setup_test_wallets() {
    print_header "Setting Up Test Wallets"

    # SC owner wallet: use owner.json (genesis SC owner) if it exists, else local.json
    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_SC_OWNER_WALLET}"
    local DELEGATE_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local SYSTEM_TEST_DIR="${BASE_DIR}/system_test"

    if [ ! -f "$SC_OWNER_WALLET" ]; then
        print_warning "SC owner wallet not found at $SC_OWNER_WALLET, falling back to $DELEGATE_WALLET"
        SC_OWNER_WALLET="$DELEGATE_WALLET"
    fi

    if [ ! -f "$DELEGATE_WALLET" ]; then
        print_error "Delegate wallet not found at $DELEGATE_WALLET"
        return 1
    fi

    local sc_owner_id=$(jq -r '.client_id' "$SC_OWNER_WALLET")
    local delegate_id=$(jq -r '.client_id' "$DELEGATE_WALLET")
    print_status "SC owner wallet (${ZCN_SC_OWNER_WALLET}): ${sc_owner_id:0:16}..."
    print_status "Delegate wallet (${ZCN_WALLET_FILE}): ${delegate_id:0:16}..."

    # IMPORTANT: Verify the on-chain SC owner before copying wallets.
    local onchain_sc_owner=""
    onchain_sc_owner=$(curl -s http://127.0.0.1:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('fields',{}).get('owner_id',''))" 2>/dev/null || true)

    if [ -n "$onchain_sc_owner" ] && [ "$onchain_sc_owner" != "$sc_owner_id" ]; then
        # Auto-correct: find the correct wallet matching the on-chain SC owner.
        if [ "$onchain_sc_owner" = "$delegate_id" ]; then
            print_status "ON-CHAIN SC owner matches local.json — using local.json as sc_owner_wallet"
            SC_OWNER_WALLET="$DELEGATE_WALLET"
            sc_owner_id="$delegate_id"
        else
            # Search the wallet pool for the on-chain SC owner
            print_warning "ON-CHAIN SC owner ($onchain_sc_owner) differs from SC owner wallet ($sc_owner_id) — searching wallet pool"
            local wallets_pool="${SYSTEM_TEST_DIR}/tests/cli_tests/config/wallets/wallets.json"
            if [ -f "$wallets_pool" ]; then
                local extracted_wallet
                extracted_wallet=$(python3 -c "
import sys, json
wallets = json.load(open('$wallets_pool'))
target = '$onchain_sc_owner'
w = next((w for w in wallets if w.get('client_id') == target), None)
if w:
    print(json.dumps(w, indent=2))
" 2>/dev/null || true)
                if [ -n "$extracted_wallet" ]; then
                    local tmp_wallet
                    tmp_wallet=$(mktemp /tmp/sc_owner_wallet_XXXXXX.json)
                    echo "$extracted_wallet" > "$tmp_wallet"
                    SC_OWNER_WALLET="$tmp_wallet"
                    sc_owner_id="$onchain_sc_owner"
                    print_status "Found on-chain SC owner in wallet pool — using it as sc_owner_wallet"
                else
                    print_warning "ON-CHAIN SC owner not found in wallet pool — sc_owner_wallet may be wrong"
                fi
            else
                print_warning "ON-CHAIN SC owner ($onchain_sc_owner) differs but wallet pool not found — sc_owner_wallet may be wrong"
            fi
        fi
    fi

    # Test suite config directories
    local TEST_DIRS=(
        "${SYSTEM_TEST_DIR}/tests/cli_tests/config/wallets"
        "${SYSTEM_TEST_DIR}/tests/tokenomics_tests/config/wallets"
    )
    local API_CONFIG="${SYSTEM_TEST_DIR}/tests/api_tests/config"
    local SDK_CONFIG="${SYSTEM_TEST_DIR}/tests/sdk_tests/config"

    for dir in "${TEST_DIRS[@]}"; do
        mkdir -p "$dir"

        # SC owner wallet: only overwrite if the existing one doesn't match on-chain owner
        local existing_sc_id=""
        if [ -f "${dir}/sc_owner_wallet.json" ]; then
            existing_sc_id=$(jq -r '.client_id' "${dir}/sc_owner_wallet.json" 2>/dev/null || true)
        fi

        if [ -n "$onchain_sc_owner" ] && [ "$existing_sc_id" = "$onchain_sc_owner" ]; then
            print_status "SC owner wallet in ${dir##*/} already matches on-chain owner, keeping it"
        else
            cp "$SC_OWNER_WALLET" "${dir}/sc_owner_wallet.json"
            print_status "Copied sc_owner to ${dir##*/}"
        fi

        # Blobber owner wallet (for bl-update - must match blobber delegate_wallet on-chain)
        # Auto-detect: query on-chain to find the actual delegate_wallet used at registration.
        local blobber_owner_src="$DELEGATE_WALLET"
        local onchain_blobber_delegate=""
        onchain_blobber_delegate=$(curl -s "http://127.0.0.1:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers" 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); nodes=d.get('Nodes') or d.get('nodes') or []; b=[n for n in nodes if not n.get('is_enterprise',False)]; print((b[0].get('stake_pool_settings',{}).get('delegate_wallet','') or b[0].get('delegate_wallet','')) if b else '')" 2>/dev/null || true)
        if [ -n "$onchain_blobber_delegate" ] && [ "$onchain_blobber_delegate" != "$delegate_id" ]; then
            if [ "$onchain_blobber_delegate" = "$sc_owner_id" ]; then
                blobber_owner_src="$SC_OWNER_WALLET"
                print_status "On-chain blobber delegate_wallet matches SC owner — using SC owner wallet as blobber_owner"
            else
                print_warning "On-chain blobber delegate ($onchain_blobber_delegate) unknown — falling back to local.json"
            fi
        fi
        cp "$blobber_owner_src" "${dir}/blobber_owner_wallet.json"
        print_status "Copied blobber_owner to ${dir##*/}"

        # ZCNSC owner wallet (same as SC owner = owner.json)
        cp "$SC_OWNER_WALLET" "${dir}/zcnsc_owner_wallet.json"
        print_status "Copied zcnsc_owner to ${dir##*/}"

        # zbox_team wallet (used by tokenomics tests for enterprise blobber auth tickets)
        cp "$SC_OWNER_WALLET" "${dir}/zbox_team_wallet.json"
        print_status "Copied zbox_team to ${dir##*/}"

        # Miner/Sharder delegate wallets (must match on-chain delegate_wallet)
        # Auto-detect: check the actual on-chain delegate_wallet for miners.
        local miner_node_delegate_src="$DELEGATE_WALLET"
        local onchain_miner_delegate=""
        onchain_miner_delegate=$(curl -s "http://127.0.0.1:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList" 2>/dev/null \
            | python3 -c "import sys,json; nodes=json.load(sys.stdin).get('Nodes',[]); print(nodes[0].get('stake_pool',{}).get('settings',{}).get('delegate_wallet','') if nodes else '')" 2>/dev/null || true)
        if [ -n "$onchain_miner_delegate" ] && [ "$onchain_miner_delegate" != "$delegate_id" ]; then
            if [ "$onchain_miner_delegate" = "$sc_owner_id" ]; then
                miner_node_delegate_src="$SC_OWNER_WALLET"
                print_status "On-chain miner delegate matches SC owner — using SC owner wallet for miner delegates"
            else
                print_warning "On-chain miner delegate ($onchain_miner_delegate) unknown — falling back to local.json"
            fi
        fi
        for node_wallet in miner01_node_delegate miner02_node_delegate miner03_node_delegate \
                           sharder01_node_delegate sharder02_node_delegate; do
            cp "$miner_node_delegate_src" "${dir}/${node_wallet}_wallet.json"
        done
        print_status "Copied miner/sharder delegate wallets to ${dir##*/}"

        # MinerSC owner wallet (may differ from StorageSC owner on servers with historical chain state)
        local miner_sc_owner_src="${ZCN_CONFIG_DIR}/miner_sc_owner.json"
        if [ -f "$miner_sc_owner_src" ]; then
            cp "$miner_sc_owner_src" "${dir}/miner_sc_owner_wallet.json"
            print_status "Copied miner_sc_owner to ${dir##*/}"
        else
            # Fall back to SC owner wallet if no dedicated miner SC owner
            cp "$SC_OWNER_WALLET" "${dir}/miner_sc_owner_wallet.json"
            print_status "Copied sc_owner as miner_sc_owner (no dedicated miner_sc_owner.json) to ${dir##*/}"
        fi
    done

    # API tests use a flat config dir (no wallets/ subdirectory)
    if [ -d "$API_CONFIG" ]; then
        local existing_api_id=""
        if [ -f "${API_CONFIG}/sc_owner_wallet.json" ]; then
            existing_api_id=$(jq -r '.client_id' "${API_CONFIG}/sc_owner_wallet.json" 2>/dev/null || true)
        fi
        if [ -n "$onchain_sc_owner" ] && [ "$existing_api_id" = "$onchain_sc_owner" ]; then
            print_status "SC owner wallet in api_tests already matches on-chain owner, keeping it"
        else
            cp "$SC_OWNER_WALLET" "${API_CONFIG}/sc_owner_wallet.json"
            print_status "Copied sc_owner to api_tests"
        fi
    fi

    # SDK tests
    if [ -d "$SDK_CONFIG" ]; then
        local existing_sdk_id=""
        if [ -f "${SDK_CONFIG}/sc_owner_wallet.json" ]; then
            existing_sdk_id=$(jq -r '.client_id' "${SDK_CONFIG}/sc_owner_wallet.json" 2>/dev/null || true)
        fi
        if [ -n "$onchain_sc_owner" ] && [ "$existing_sdk_id" = "$onchain_sc_owner" ]; then
            print_status "SC owner wallet in sdk_tests already matches on-chain owner, keeping it"
        else
            cp "$SC_OWNER_WALLET" "${SDK_CONFIG}/sc_owner_wallet.json"
            print_status "Copied sc_owner to sdk_tests"
        fi
    fi

    # Copy CLI binaries to test directories
    local CLI_TEST_DIR="${SYSTEM_TEST_DIR}/tests/cli_tests"
    if [ -f "$ZWALLET" ] && [ -d "$CLI_TEST_DIR" ]; then
        cp "$ZWALLET" "${CLI_TEST_DIR}/zwallet"
        print_status "Copied zwallet to cli_tests"
    fi
    if [ -f "$ZBOX" ] && [ -d "$CLI_TEST_DIR" ]; then
        cp "$ZBOX" "${CLI_TEST_DIR}/zbox"
        print_status "Copied zbox to cli_tests"
    fi

    # Also copy to tokenomics tests
    local TOK_TEST_DIR="${SYSTEM_TEST_DIR}/tests/tokenomics_tests"
    if [ -f "$ZWALLET" ] && [ -d "$TOK_TEST_DIR" ]; then
        cp "$ZWALLET" "${TOK_TEST_DIR}/zwallet"
        cp "$ZBOX" "${TOK_TEST_DIR}/zbox"
        print_status "Copied CLI tools to tokenomics_tests"
    fi

    print_status "Test wallets setup complete!"
}

# Configure test suite config files with correct service URLs and Firebase credentials.
# Ensures api_tests_config.yaml, tokenomics_tests_config.yaml, etc. have the right
# block_worker, 0box URL, and Firebase auth settings for the local deployment.
configure_test_configs() {
    print_header "Configuring Test Suite Config Files"

    local SYSTEM_TEST_DIR="${BASE_DIR}/system_test"

    # Firebase credentials (loaded from .secrets.env via environment variables)
    local FIREBASE_API_KEY="${FIREBASE_TEST_API_KEY:-}"
    local FIREBASE_EMAIL="${FIREBASE_TEST_EMAIL:-test_system_test@0chain.net}"
    local FIREBASE_PASSWORD="${FIREBASE_TEST_PASSWORD:-}"
    local FIREBASE_EMAIL_R="${FIREBASE_TEST_EMAIL_R:-test_referred_user@0chain.net}"
    local FIREBASE_PASSWORD_R="${FIREBASE_TEST_PASSWORD_R:-}"
    if [ -z "$FIREBASE_API_KEY" ] || [ -z "$FIREBASE_PASSWORD" ]; then
        print_warning "Firebase credentials not set — check .secrets.env (FIREBASE_TEST_API_KEY, FIREBASE_TEST_PASSWORD)"
    fi

    # Ensure DNS proxy is running on port 9099 so gosdk can find sharders even when
    # the magic block has 0 sharders (DKG deadlock). The proxy injects sharder IPs
    # into the /network response transparently.
    start_dns_proxy 2>/dev/null || true

    # Use proxy if running, otherwise fall back to direct 0dns
    local BLOCK_WORKER="http://127.0.0.1:9099"
    if ! curl -sk --max-time 2 http://127.0.0.1:9099/network >/dev/null 2>&1; then
        BLOCK_WORKER="http://198.18.0.100:9091"
        print_warning "DNS proxy not available, using direct 0dns: ${BLOCK_WORKER}"
    fi
    local ZBOX_URL="http://localhost:9081"
    local ZAUTH_URL="http://localhost:8080"
    local ZVAULT_URL="http://localhost:8090"
    local ZS3_URL="http://localhost:9100"
    local CHIMNEY_URL="http://127.0.0.1:9091"

    # ---- API tests config ----
    local API_CONFIG="${SYSTEM_TEST_DIR}/tests/api_tests/config/api_tests_config.yaml"
    if [ -f "$API_CONFIG" ]; then
        print_status "Updating api_tests_config.yaml..."

        # Ensure Firebase credentials are present
        if grep -q "firebase_api_key:" "$API_CONFIG"; then
            sed -i.bak "s|firebase_api_key:.*|firebase_api_key: \"${FIREBASE_API_KEY}\"|" "$API_CONFIG"
        else
            echo "firebase_api_key: \"${FIREBASE_API_KEY}\"" >> "$API_CONFIG"
        fi

        if grep -q "firebase_email:" "$API_CONFIG"; then
            sed -i.bak "s|firebase_email:.*|firebase_email: \"${FIREBASE_EMAIL}\"|" "$API_CONFIG"
        else
            echo "firebase_email: \"${FIREBASE_EMAIL}\"" >> "$API_CONFIG"
        fi

        if grep -q "firebase_password:" "$API_CONFIG"; then
            sed -i.bak "s|firebase_password:.*|firebase_password: \"${FIREBASE_PASSWORD}\"|" "$API_CONFIG"
        else
            echo "firebase_password: \"${FIREBASE_PASSWORD}\"" >> "$API_CONFIG"
        fi

        # Firebase credentials for referred user (_R identity, used in referral tests)
        if grep -q "firebase_email_r:" "$API_CONFIG"; then
            sed -i.bak "s|firebase_email_r:.*|firebase_email_r: \"${FIREBASE_EMAIL_R}\"|" "$API_CONFIG"
        else
            echo "firebase_email_r: \"${FIREBASE_EMAIL_R}\"" >> "$API_CONFIG"
        fi

        if grep -q "firebase_password_r:" "$API_CONFIG"; then
            sed -i.bak "s|firebase_password_r:.*|firebase_password_r: \"${FIREBASE_PASSWORD_R}\"|" "$API_CONFIG"
        else
            echo "firebase_password_r: \"${FIREBASE_PASSWORD_R}\"" >> "$API_CONFIG"
        fi

        # Ensure service URLs are correct for local deployment
        sed -i.bak "s|block_worker:.*|block_worker: ${BLOCK_WORKER}|" "$API_CONFIG"
        sed -i.bak "s|0box_url:.*|0box_url: ${ZBOX_URL}|" "$API_CONFIG"
        sed -i.bak "s|zauth_url:.*|zauth_url: ${ZAUTH_URL}|" "$API_CONFIG"
        sed -i.bak "s|zvault_url:.*|zvault_url: ${ZVAULT_URL}|" "$API_CONFIG"
        sed -i.bak "s|zs3_server_url:.*|zs3_server_url: ${ZS3_URL}|" "$API_CONFIG"
        sed -i.bak "s|chimney_test_network:.*|chimney_test_network: ${CHIMNEY_URL}|" "$API_CONFIG"

        rm -f "${API_CONFIG}.bak"
        print_status "api_tests_config.yaml updated with Firebase auth and service URLs"
    else
        print_warning "api_tests_config.yaml not found at $API_CONFIG"
    fi

    # ---- Tokenomics tests config ----
    local TOK_CONFIG="${SYSTEM_TEST_DIR}/tests/tokenomics_tests/config/tokenomics_tests_config.yaml"
    if [ -f "$TOK_CONFIG" ]; then
        print_status "Updating tokenomics_tests_config.yaml..."
        sed -i.bak "s|block_worker:.*|block_worker: ${BLOCK_WORKER}|" "$TOK_CONFIG"
        # Fix any external URLs to local
        sed -i.bak "s|0box_url:.*|0box_url: ${ZBOX_URL}|" "$TOK_CONFIG"
        sed -i.bak "s|zvault_url:.*|zvault_url: ${ZVAULT_URL}|" "$TOK_CONFIG"
        sed -i.bak "s|zauth_url:.*|zauth_url: ${ZAUTH_URL}|" "$TOK_CONFIG"
        rm -f "${TOK_CONFIG}.bak"
        print_status "tokenomics_tests_config.yaml updated"
    fi

    # ---- SDK tests config ----
    local SDK_CONFIG="${SYSTEM_TEST_DIR}/tests/sdk_tests/config/sdk_tests_config.yaml"
    mkdir -p "$(dirname "$SDK_CONFIG")"
    if [ ! -f "$SDK_CONFIG" ]; then
        print_status "Creating sdk_tests_config.yaml from scratch..."
        cat > "$SDK_CONFIG" <<SDKCFG
block_worker: ${BLOCK_WORKER}
default_test_case_timeout: 45s
signature_scheme: bls0chain
chain_id: 0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe
max_txn_query: 5
query_sleep_time: 5
min_submit: 10
min_confirmation: 10
0box_url: ${ZBOX_URL}
zauth_url: ${ZAUTH_URL}
zvault_url: ${ZVAULT_URL}
SDKCFG
        print_status "sdk_tests_config.yaml created"
    else
        print_status "Updating sdk_tests_config.yaml..."
        sed -i.bak "s|block_worker:.*|block_worker: ${BLOCK_WORKER}|" "$SDK_CONFIG"
        sed -i.bak "s|0box_url:.*|0box_url: ${ZBOX_URL}|" "$SDK_CONFIG"
        sed -i.bak "s|zauth_url:.*|zauth_url: ${ZAUTH_URL}|" "$SDK_CONFIG"
        sed -i.bak "s|zvault_url:.*|zvault_url: ${ZVAULT_URL}|" "$SDK_CONFIG"
        rm -f "${SDK_CONFIG}.bak"
        print_status "sdk_tests_config.yaml updated"
    fi

    # ---- Update zbox_config.yaml files for all suites ----
    # These control SDK transaction confirmation timeouts
    local ZBOX_CONFIGS=(
        "${SYSTEM_TEST_DIR}/tests/cli_tests/config/zbox_config.yaml"
        "${SYSTEM_TEST_DIR}/tests/tokenomics_tests/config/zbox_config.yaml"
    )
    for ZBOX_CFG in "${ZBOX_CONFIGS[@]}"; do
        if [ -f "$ZBOX_CFG" ]; then
            local suite_name=$(echo "$ZBOX_CFG" | grep -oP 'tests/\K[^/]+')
            print_status "Updating zbox_config.yaml for ${suite_name}..."

            # Ensure block_worker points to local
            sed -i.bak "s|block_worker:.*|block_worker: ${BLOCK_WORKER}|" "$ZBOX_CFG"

            # Ensure max_txn_query and query_sleep_time are set (for SDK confirmation reliability)
            if grep -q "max_txn_query:" "$ZBOX_CFG"; then
                sed -i.bak "s|max_txn_query:.*|max_txn_query: 10|" "$ZBOX_CFG"
            else
                # Insert after min_submit if present, otherwise append
                if grep -q "min_submit:" "$ZBOX_CFG"; then
                    sed -i.bak "/min_submit:.*/a max_txn_query: 10" "$ZBOX_CFG"
                else
                    echo "max_txn_query: 10" >> "$ZBOX_CFG"
                fi
            fi

            if grep -q "query_sleep_time:" "$ZBOX_CFG"; then
                sed -i.bak "s|query_sleep_time:.*|query_sleep_time: 5|" "$ZBOX_CFG"
            else
                if grep -q "max_txn_query:" "$ZBOX_CFG"; then
                    sed -i.bak "/max_txn_query:.*/a query_sleep_time: 5" "$ZBOX_CFG"
                else
                    echo "query_sleep_time: 5" >> "$ZBOX_CFG"
                fi
            fi

            # Ensure min_confirmation and min_submit are reasonable
            if grep -q "min_confirmation:" "$ZBOX_CFG"; then
                sed -i.bak "s|min_confirmation:.*|min_confirmation: 10|" "$ZBOX_CFG"
            fi
            if grep -q "min_submit:" "$ZBOX_CFG"; then
                sed -i.bak "s|min_submit:.*|min_submit: 100|" "$ZBOX_CFG"
            fi

            rm -f "${ZBOX_CFG}.bak"
            print_status "zbox_config.yaml for ${suite_name} updated"
        fi
    done

    # ---- Update config.yaml files (block_worker + bridge) for CLI and tokenomics ----
    local SUITE_CONFIGS=(
        "${SYSTEM_TEST_DIR}/tests/cli_tests/config/config.yaml"
        "${SYSTEM_TEST_DIR}/tests/tokenomics_tests/config/config.yaml"
    )
    for SUITE_CFG in "${SUITE_CONFIGS[@]}"; do
        if [ -f "$SUITE_CFG" ]; then
            local suite_name=$(echo "$SUITE_CFG" | grep -oP 'tests/\K[^/]+')
            print_status "Updating config.yaml for ${suite_name}..."
            sed -i.bak "s|block_worker:.*|block_worker: ${BLOCK_WORKER}|" "$SUITE_CFG"
            rm -f "${SUITE_CFG}.bak"
            print_status "config.yaml for ${suite_name} updated"
        fi
    done

    print_status "Test config files updated!"
}

# Clean stale build artifacts in test directories.
# Go module operations and rsync can accidentally create nested directory copies:
#   tests/tokenomics_tests/internal/tests/tokenomics_tests/utils/
#   tests/tokenomics_tests/utils/tests/tokenomics_tests/utils/
#   tests/sdk_tests/internal/...
# These stale copies contain old versions of source files. The Go compiler may
# pick them up instead of the correct files, causing hard-to-debug failures
# (e.g., using old code that lacks recent bug fixes).
#
# The legitimate internal/ directory is at the repo root (system_test/internal/).
# Test subdirectories should NEVER have their own internal/ or nested tests/ dirs.
clean_service_databases() {
    print_header "Cleaning Service Databases (0box, zvault, zauth)"

    # After a chain redeploy, 0box/zvault/zauth databases contain stale data
    # from the old chain (users, allocations, blobbers, events, etc.).
    # This causes "record not found" errors and stale state in web apps.
    # Truncate all tables except migration tracking (goose_db_version).
    #
    # IMPORTANT: After calling this function, you MUST call seed_0box_providers()
    # to repopulate the blobbers table. Without blobbers, 0box Kafka consumer
    # fails to insert allocations (FK constraint on allocation_blobber_terms)
    # and write_markers (FK constraint on blobbers), causing 0box data loss.

    # ---- 0box ----
    if docker ps --format '{{.Names}}' | grep -q "postgres-0box"; then
        print_status "Truncating 0box database tables..."
        local tables
        tables=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "SELECT string_agg('\"' || tablename || '\"', ', ')
             FROM pg_tables
             WHERE schemaname = 'public'
               AND tablename NOT IN ('goose_db_version');" 2>/dev/null | tr -d ' \n')
        if [ -n "$tables" ]; then
            docker exec postgres-0box psql -U zbox_user -d zbox -c \
                "TRUNCATE TABLE ${tables} CASCADE;" 2>/dev/null
            print_status "0box database truncated"
        fi
        # WARNING: After truncating 0box DB, the Kafka pipeline is broken —
        # 0box's lastProcessedRound resets to 0 but Kafka is at round N>>1.
        # The caller (clean_service_databases) should only be used for non-clean
        # deploys where the chain is reused. For fresh redeploy, do NOT call
        # clean_service_databases — let 0box keep its data from round 1.
    else
        print_warning "postgres-0box not running, skipping"
    fi

    # ---- zvault ----
    if docker ps --format '{{.Names}}' | grep -q "zvault-postgreszv"; then
        print_status "Truncating zvault database tables..."
        local vtables
        vtables=$(docker exec zvault-postgreszv-1 psql -U zvault_user -d zvault -t -c \
            "SELECT string_agg('\"' || tablename || '\"', ', ')
             FROM pg_tables
             WHERE schemaname = 'public'
               AND tablename NOT IN ('goose_db_version');" 2>/dev/null | tr -d ' \n')
        if [ -n "$vtables" ]; then
            docker exec zvault-postgreszv-1 psql -U zvault_user -d zvault -c \
                "TRUNCATE TABLE ${vtables} CASCADE;" 2>/dev/null
            print_status "zvault database truncated"
        fi
    else
        print_warning "zvault postgres not running, skipping"
    fi

    # ---- zauth ----
    if docker ps --format '{{.Names}}' | grep -q "zauth-postgres"; then
        print_status "Truncating zauth database tables..."
        local ztables
        ztables=$(docker exec zauth-postgres-1 psql -U zauth_user -d zauth -t -c \
            "SELECT string_agg('\"' || tablename || '\"', ', ')
             FROM pg_tables
             WHERE schemaname = 'public'
               AND tablename NOT IN ('goose_db_version');" 2>/dev/null | tr -d ' \n')
        if [ -n "$ztables" ]; then
            docker exec zauth-postgres-1 psql -U zauth_user -d zauth -c \
                "TRUNCATE TABLE ${ztables} CASCADE;" 2>/dev/null
            print_status "zauth database truncated"
        fi
    else
        print_warning "zauth postgres not running, skipping"
    fi

    print_status "Service databases cleaned"
}

cleanup_stale_test_artifacts() {
    print_header "Cleaning Stale Test Artifacts"

    local SYSTEM_TEST_DIR="${BASE_DIR}/system_test"
    local cleaned=0

    # Remove stale internal/ directories inside test subdirectories.
    # The only legitimate internal/ is at repo root: system_test/internal/
    # Search at ANY depth to catch dirs like tests/tokenomics_tests/utils/internal/
    for test_dir in "${SYSTEM_TEST_DIR}/tests/cli_tests" \
                    "${SYSTEM_TEST_DIR}/tests/api_tests" \
                    "${SYSTEM_TEST_DIR}/tests/sdk_tests" \
                    "${SYSTEM_TEST_DIR}/tests/tokenomics_tests"; do
        if [ -d "${test_dir}" ]; then
            # Remove ALL internal/ directories at any depth within test dirs
            while IFS= read -r -d '' stale_dir; do
                print_warning "Removing stale directory: ${stale_dir}"
                rm -rf "$stale_dir"
                cleaned=$((cleaned + 1))
            done < <(find "${test_dir}" -type d -name "internal" -print0 2>/dev/null)

            # Remove nested tests/ directories (e.g., tests/tokenomics_tests/utils/tests/)
            while IFS= read -r -d '' nested_dir; do
                print_warning "Removing stale nested directory: ${nested_dir}"
                rm -rf "$nested_dir"
                cleaned=$((cleaned + 1))
            done < <(find "${test_dir}" -mindepth 2 -type d -name "tests" -print0 2>/dev/null)

            # Remove .bak files left by sed operations
            find "${test_dir}" -name "*.bak" -type f -delete 2>/dev/null || true
        fi
    done

    # Clean stale per-test wallet and allocation files from CLI test config.
    # Wallet files (Test*_wallet.json) from previous runs cache old wallets that
    # may have depleted balances. Removing them forces createWallet() to re-fund
    # each wallet fresh from the faucet for the next run.
    local CLI_CONFIG="${SYSTEM_TEST_DIR}/tests/cli_tests/config"
    if [ -d "$CLI_CONFIG" ]; then
        local wallet_count
        wallet_count=$(find "$CLI_CONFIG" -name "Test*_wallet.json" | wc -l 2>/dev/null || echo 0)
        local alloc_count
        alloc_count=$(find "$CLI_CONFIG" -name "*_allocation.txt" | wc -l 2>/dev/null || echo 0)
        if [ "$wallet_count" -gt 0 ] || [ "$alloc_count" -gt 0 ]; then
            find "$CLI_CONFIG" -name "Test*_wallet.json" -delete 2>/dev/null || true
            find "$CLI_CONFIG" -name "*_allocation.txt" -delete 2>/dev/null || true
            print_status "Cleaned $wallet_count stale test wallet files and $alloc_count allocation files from CLI config"
            cleaned=$((cleaned + 1))
        fi
    fi

    # Clean Go build cache to ensure stale compiled objects are not reused
    if command -v go &>/dev/null; then
        print_status "Cleaning Go build cache..."
        go clean -cache 2>/dev/null || true
    fi

    # Clean stale 0box owner records that cause OTP signup failures.
    # Test runs create owner records (test_owner_1, referred_user, etc.) that
    # persist across runs. If a new test run uses a different Firebase UID,
    # the signup fails with "duplicate key value violates unique constraint owner_username_key".
    if docker ps --format '{{.Names}}' | grep -q "postgres-0box"; then
        print_status "Cleaning stale 0box owner records..."
        # Backup first, then delete test records
        # Matches: username LIKE 'test_%', username = 'referred_user',
        # AND user_id LIKE 'devnet_%' with a test email (created by OTP endpoint testing in dev mode).
        # IMPORTANT: only match devnet_ records with test/fake emails — NOT real user accounts
        # that happen to use devnet_ prefix (e.g. phone OTP login in dev mode with real email).
        docker exec postgres-0box psql -U zbox_user -d zbox -c \
            "CREATE TABLE IF NOT EXISTS owner_backup_cleanup AS SELECT * FROM owner WHERE false;" 2>/dev/null || true
        docker exec postgres-0box psql -U zbox_user -d zbox -c \
            "INSERT INTO owner_backup_cleanup SELECT * FROM owner WHERE username LIKE 'test_%' OR username = 'referred_user' OR (user_id LIKE 'devnet_%' AND (email IS NULL OR email LIKE '%@test.%' OR email LIKE 'test%@%')) ON CONFLICT DO NOTHING;" 2>/dev/null || true
        # Delete wallets belonging to test owners FIRST (FK dependency)
        docker exec postgres-0box psql -U zbox_user -d zbox -c \
            "DELETE FROM wallet WHERE owner_id IN (SELECT id FROM owner WHERE username LIKE 'test_%' OR username = 'referred_user' OR (user_id LIKE 'devnet_%' AND (email IS NULL OR email LIKE '%@test.%' OR email LIKE 'test%@%')));" 2>/dev/null || true
        # Also delete wallets with known test client_ids (orphaned from previous cleanups)
        docker exec postgres-0box psql -U zbox_user -d zbox -c \
            "DELETE FROM wallet WHERE name LIKE 'test_%';" 2>/dev/null || true
        local deleted=$(docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
            "DELETE FROM owner WHERE username LIKE 'test_%' OR username = 'referred_user' OR (user_id LIKE 'devnet_%' AND (email IS NULL OR email LIKE '%@test.%' OR email LIKE 'test%@%'));" 2>/dev/null | tr -d ' ')
        if [ -n "$deleted" ] && [ "$deleted" != "DELETE 0" ]; then
            print_status "Cleaned stale 0box owner records: $deleted"
            cleaned=$((cleaned + 1))
        fi
    fi

    if [ $cleaned -gt 0 ]; then
        print_status "Cleaned $cleaned stale artifacts"
    else
        print_status "No stale artifacts found"
    fi
}

# Reset test state: cancel all active allocations and unstake all delegate pools.
# Run this BEFORE a test suite to ensure clean starting state. Tests create allocations
# and stake tokens which persist across runs, causing later tests to fail due to
# exhausted delegate pools or insufficient balance.
#
# Usage: bash deploy_local.sh reset-test-state
reset_test_state() {
    print_header "Resetting Test State (Cancel Allocations + Unstake Pools)"

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent"
    local SHARDER_URL="http://127.0.0.1:7171"
    local STORAGE_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
    local MINER_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"

    # ---- 1. Cancel all active allocations ----
    print_status "Fetching active allocations..."
    local sc_owner_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}" 2>/dev/null)

    # Get allocations for SC owner wallet
    local alloc_json
    alloc_json=$($ZBOX listallocations --json $W 2>/dev/null || echo "[]")
    local alloc_ids
    alloc_ids=$(echo "$alloc_json" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    if isinstance(data, list):
        for a in data:
            aid = a.get('id', '')
            expired = a.get('expired', False)
            finalized = a.get('finalized', False)
            canceled = a.get('canceled', False)
            if aid and not expired and not finalized and not canceled:
                print(aid)
except:
    pass
" 2>/dev/null || true)

    if [ -z "$alloc_ids" ]; then
        print_status "  No active allocations found for SC owner"
    else
        local cancel_count=0
        while IFS= read -r alloc_id; do
            [ -z "$alloc_id" ] && continue
            print_status "  Cancelling allocation: ${alloc_id:0:16}..."
            $ZBOX alloc-cancel --allocation "$alloc_id" $W 2>/dev/null || true
            cancel_count=$((cancel_count + 1))
            sleep 2
        done <<< "$alloc_ids"
        print_status "  Cancelled $cancel_count allocation(s)"
    fi

    # Also cancel allocations from test wallets (wallets.json)
    local WALLETS_FILE="${BASE_DIR}/system_test/tests/api_tests/config/wallets.json"
    if [ -f "$WALLETS_FILE" ]; then
        local wallet_count=$(python3 -c "import json; print(len(json.load(open('$WALLETS_FILE'))))" 2>/dev/null || echo 0)
        print_status "Checking allocations for $wallet_count test wallets..."
        # Only check first 10 wallets (most likely to have allocations)
        local check_count=0
        for wallet_mnemonics in $(python3 -c "
import json
wallets = json.load(open('$WALLETS_FILE'))
seen = set()
for w in wallets[:10]:
    m = w.get('mnemonics', '')
    if m and m not in seen:
        seen.add(m)
        print(m.replace(' ', '_SPACE_'))
" 2>/dev/null); do
            wallet_mnemonics=$(echo "$wallet_mnemonics" | sed 's/_SPACE_/ /g')
            local tw_allocs
            tw_allocs=$($ZBOX listallocations --json \
                --wallet-mnemonics "$wallet_mnemonics" \
                --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || echo "[]")
            local tw_ids
            tw_ids=$(echo "$tw_allocs" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    if isinstance(data, list):
        for a in data:
            aid = a.get('id', '')
            if aid and not a.get('expired') and not a.get('finalized') and not a.get('canceled'):
                print(aid)
except:
    pass
" 2>/dev/null || true)
            if [ -n "$tw_ids" ]; then
                while IFS= read -r alloc_id; do
                    [ -z "$alloc_id" ] && continue
                    print_status "  Cancelling test wallet allocation: ${alloc_id:0:16}..."
                    $ZBOX alloc-cancel --allocation "$alloc_id" $W 2>/dev/null || true
                    check_count=$((check_count + 1))
                    sleep 2
                done <<< "$tw_ids"
            fi
        done
        if [ "$check_count" -gt 0 ]; then
            print_status "  Cancelled $check_count test wallet allocation(s)"
        fi
    fi

    # ---- 2. Unstake test-wallet pools from blobbers ----
    # Only unstake pools from test wallets, NOT from the infrastructure wallet (local.json).
    # The infrastructure stake keeps blobbers active in the SC; removing it causes blobbers
    # to drop out of allocation candidate lists.
    print_status "Unstaking test-wallet pools from blobbers (preserving infrastructure stake)..."
    local infra_wallet_id
    infra_wallet_id=$(jq -r '.client_id' "${ZCN_CONFIG_DIR}/local.json" 2>/dev/null || echo "")

    local blobbers_json
    blobbers_json=$(curl -s "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getblobbers" 2>/dev/null)

    if [ -n "$blobbers_json" ]; then
        local blobber_ids
        blobber_ids=$(echo "$blobbers_json" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for b in data.get('Nodes', []):
    if not b.get('is_killed') and not b.get('is_shutdown'):
        print(b['id'])
" 2>/dev/null || true)

        local unstake_count=0
        while IFS= read -r bid; do
            [ -z "$bid" ] && continue
            local pool_json
            pool_json=$(curl -s "${SHARDER_URL}/v1/screst/${STORAGE_SC}/getStakePoolStat?provider_id=${bid}&provider_type=3" 2>/dev/null || true)
            if [ -z "$pool_json" ]; then continue; fi

            # Extract test-wallet delegate IDs (exclude infrastructure wallet)
            local pool_ids
            pool_ids=$(echo "$pool_json" | python3 -c "
import json, sys
infra = '$infra_wallet_id'
try:
    data = json.load(sys.stdin)
    pools = data.get('pools', {})
    if isinstance(pools, dict):
        for delegate_id, pool_info in pools.items():
            if delegate_id != infra:
                print(delegate_id)
    elif isinstance(pools, list):
        for p in pools:
            did = p.get('delegate_id', '')
            if did and did != infra:
                print(did)
except:
    pass
" 2>/dev/null || true)

            while IFS= read -r pool_id; do
                [ -z "$pool_id" ] && continue
                print_status "  Unstaking from blobber ${bid:0:16}... pool=${pool_id:0:16}..."
                $ZBOX sp-unlock --blobber_id "$bid" --pool_id "$pool_id" $W 2>/dev/null || true
                unstake_count=$((unstake_count + 1))
                sleep 1
            done <<< "$pool_ids"
        done <<< "$blobber_ids"
        print_status "  Unstaked $unstake_count blobber pool(s)"
    fi

    # NOTE: Miner/sharder unstaking intentionally omitted.
    # Unstaking miners/sharders wipes the infrastructure stakes needed for atlus network
    # score, rewards charts, and miner/sharder visibility. These stakes must persist
    # across test runs. Tests create their own test wallets and use max_delegates=200,
    # so pool exhaustion is not a concern.

    print_status "Test state reset complete!"
}

# Seed challenge data: create a warm-up allocation and upload 2MB of data so the
# challenge protocol starts generating challenges. Without active allocations with
# data, challenge tests fail because no challenges are ever created.
seed_challenge_data() {
    print_header "Seeding Challenge Data for Regular + Enterprise Blobbers"

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent"

    # --- Regular blobbers: 4+2=6 shards (works with 9 blobbers, exceeds 4+2 requirement) ---
    print_status "Creating regular blobber warm-up allocation (4 data + 2 parity = 6 blobbers)..."
    local alloc_output
    alloc_output=$($ZBOX newallocation --size 2147483648 --data 4 --parity 2 --lock 5 $W 2>&1) || {
        print_warning "Failed to create regular warm-up allocation: $alloc_output"
        return 1
    }
    local alloc_id=$(echo "$alloc_output" | grep -oP '[a-f0-9]{64}' | head -1)
    if [ -z "$alloc_id" ]; then
        print_warning "Could not extract allocation ID from regular allocation output"
        return 1
    fi
    print_status "  Regular allocation: ${alloc_id:0:16}..."

    dd if=/dev/urandom of=/tmp/warmup_challenge_regular.bin bs=1M count=10 2>/dev/null
    print_status "Uploading 10MB to regular blobber allocation..."
    $ZBOX upload --allocation "$alloc_id" --remotepath /warmup_data.bin \
        --localpath /tmp/warmup_challenge_regular.bin $W 2>&1 || {
        print_warning "Failed to upload to regular allocation"
        rm -f /tmp/warmup_challenge_regular.bin
        return 1
    }
    rm -f /tmp/warmup_challenge_regular.bin
    print_status "  Regular blobber challenge data seeded (10MB across 6 blobbers)."

    # --- Enterprise blobbers: 5 eblobbers (4 data + 1 parity), 5MB upload ---
    print_status "Creating enterprise blobber warm-up allocation (4 data + 1 parity = 5 eblobbers)..."
    local ealloc_output
    ealloc_output=$($ZBOX newallocation --size 2147483648 --data 4 --parity 1 \
        --enterprise --lock 5 $W 2>&1) || {
        print_warning "Failed to create enterprise warm-up allocation: $ealloc_output"
        print_warning "(Enterprise blobbers may not be available — skipping)"
        return 0
    }
    local ealloc_id=$(echo "$ealloc_output" | grep -oP '[a-f0-9]{64}' | head -1)
    if [ -z "$ealloc_id" ]; then
        print_warning "Could not extract allocation ID from enterprise allocation output"
        return 0
    fi
    print_status "  Enterprise allocation: ${ealloc_id:0:16}..."

    dd if=/dev/urandom of=/tmp/warmup_challenge_enterprise.bin bs=1M count=5 2>/dev/null
    print_status "Uploading 5MB to enterprise blobber allocation..."
    $ZBOX upload --allocation "$ealloc_id" --remotepath /warmup_data.bin \
        --localpath /tmp/warmup_challenge_enterprise.bin $W 2>&1 || {
        print_warning "Failed to upload to enterprise allocation"
        rm -f /tmp/warmup_challenge_enterprise.bin
        return 0
    }
    rm -f /tmp/warmup_challenge_enterprise.bin
    print_status "  Enterprise blobber challenge data seeded (5MB across 5 eblobbers)."

    print_status "Challenge data seeded! Challenges will start generating within a few blocks."
}

# Generate challenge_allocations.txt and challenge_blobbers.txt for TestProtocolChallenge.
# Creates 6 allocations with specific setups (uploads, deletions, add/replace blobbers, cancel).
# These files are read by tests/cli_tests/0_challenge_protocol_test.go.
generate_challenge_protocol_files() {
    print_header "Generating Challenge Protocol Test Data Files"

    local W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent"
    local BLOBBER_OWNER_W="--wallet blobber_owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent"
    local CLI_TEST_DIR="$BASE_DIR/system_test/tests/cli_tests"
    local ALLOC_FILE="$CLI_TEST_DIR/challenge_allocations.txt"
    local BLOBBER_FILE="$CLI_TEST_DIR/challenge_blobbers.txt"

    rm -f "$ALLOC_FILE" "$BLOBBER_FILE"

    # Get all available non-enterprise blobber IDs
    local all_blobber_ids
    all_blobber_ids=$($ZBOX ls-blobbers --json $W 2>&1 | grep '^\[' | python3 -c "
import sys, json
d = json.loads(sys.stdin.read())
blobbers = [b['id'] for b in d if not b.get('is_enterprise', False) and not b.get('is_killed', False) and not b.get('is_shutdown', False)]
print('\n'.join(blobbers))
" 2>/dev/null)

    if [ -z "$all_blobber_ids" ]; then
        print_warning "Could not get blobber list — skipping challenge protocol file generation"
        return 0
    fi

    _create_alloc() {
        local size="${1:-1073741824}" data="${2:-3}" parity="${3:-1}" lock="${4:-50}"
        local out
        out=$($ZBOX newallocation --size "$size" --data "$data" --parity "$parity" \
            --lock "$lock" $W 2>&1) || return 1
        echo "$out" | grep -oP '[a-f0-9]{64}' | head -1
    }

    _upload() {
        local alloc_id="$1" remotepath="$2" mb="${3:-5}"
        local tmp="/tmp/chal_$$.bin"
        dd if=/dev/urandom of="$tmp" bs=1M count="$mb" 2>/dev/null
        $ZBOX upload --allocation "$alloc_id" --remotepath "$remotepath" --localpath "$tmp" $W 2>&1
        local rc=$?; rm -f "$tmp"; return $rc
    }

    _alloc_blobbers() {
        local alloc_id="$1"
        $ZBOX getallocation --allocation "$alloc_id" $W 2>&1 | \
            grep -oP 'blobber_id:\s+\K[a-f0-9]{64}' | sort -u
    }

    _spare_blobber() {
        local used_blobbers="$1"
        while IFS= read -r bid; do
            [ -n "$bid" ] && ! echo "$used_blobbers" | grep -q "$bid" && echo "$bid" && return 0
        done <<< "$all_blobber_ids"
        return 1
    }

    # Allocation 1: upload files → should get challenges
    print_status "Allocation 1: upload (should get challenges)..."
    local a1; a1=$(_create_alloc) || { print_warning "alloc 1 failed"; return 0; }
    _upload "$a1" /challenge_data.bin 5 || print_warning "upload to alloc 1 failed"
    echo "$a1" >> "$ALLOC_FILE"
    print_status "  ${a1:0:16}..."

    # Allocation 2: upload then delete → challenge count should plateau
    print_status "Allocation 2: upload then delete..."
    local a2; a2=$(_create_alloc) || { print_warning "alloc 2 failed"; echo "" >> "$ALLOC_FILE"; return 0; }
    _upload "$a2" /delete_me.bin 5 || print_warning "upload to alloc 2 failed"
    $ZBOX delete --allocation "$a2" --remotepath /delete_me.bin $W 2>&1 || print_warning "delete from alloc 2 failed"
    echo "$a2" >> "$ALLOC_FILE"
    print_status "  ${a2:0:16}..."

    # Allocation 3: empty → no challenges
    print_status "Allocation 3: empty (no challenges)..."
    local a3; a3=$(_create_alloc) || { print_warning "alloc 3 failed"; echo "" >> "$ALLOC_FILE"; return 0; }
    echo "$a3" >> "$ALLOC_FILE"
    print_status "  ${a3:0:16}..."

    # Allocation 4: upload then add a blobber → added blobber should get challenges
    print_status "Allocation 4: upload then add blobber..."
    local a4; a4=$(_create_alloc) || { print_warning "alloc 4 failed"; echo "" >> "$ALLOC_FILE"; echo "" >> "$BLOBBER_FILE"; return 0; }
    _upload "$a4" /add_blobber_data.bin 5 || print_warning "upload to alloc 4 failed"
    local used4; used4=$(_alloc_blobbers "$a4")
    local added4; added4=$(_spare_blobber "$used4")
    if [ -n "$added4" ]; then
        $ZBOX updateallocation --allocation "$a4" --add_blobber "$added4" --lock 10 $W 2>&1 || print_warning "add-blobber (updateallocation) alloc 4 failed"
        sleep 10
        # Verify blobber was actually added before writing to file
        local verify4; verify4=$(_alloc_blobbers "$a4")
        if echo "$verify4" | grep -q "$added4"; then
            echo "$a4" >> "$ALLOC_FILE"
            echo "$added4" >> "$BLOBBER_FILE"
            print_status "  ${a4:0:16}..., added blobber ${added4:0:16}... (verified)"
        else
            print_warning "add-blobber verification failed — blobber not in allocation, writing empty entry"
            echo "$a4" >> "$ALLOC_FILE"
            echo "" >> "$BLOBBER_FILE"
        fi
    else
        print_warning "No spare blobber for add-blobber test"
        echo "$a4" >> "$ALLOC_FILE"
        echo "" >> "$BLOBBER_FILE"
    fi

    # Allocation 5: upload then replace blobber → replaced blobber should stop getting challenges
    print_status "Allocation 5: upload then replace blobber..."
    local a5; a5=$(_create_alloc) || { print_warning "alloc 5 failed"; echo "" >> "$ALLOC_FILE"; echo "" >> "$BLOBBER_FILE"; echo "" >> "$BLOBBER_FILE"; return 0; }
    _upload "$a5" /replace_blobber_data.bin 5 || print_warning "upload to alloc 5 failed"
    local used5; used5=$(_alloc_blobbers "$a5")
    local old5; old5=$(echo "$used5" | head -1)
    local new5; new5=$(_spare_blobber "$used5")
    if [ -n "$old5" ] && [ -n "$new5" ]; then
        $ZBOX updateallocation --allocation "$a5" --add_blobber "$new5" \
            --remove_blobber "$old5" --lock 10 $W 2>&1 || print_warning "replace-blobber (updateallocation) alloc 5 failed"
        sleep 10
        # Verify replace-blobber: new5 should be in allocation, old5 should not
        local verify5; verify5=$(_alloc_blobbers "$a5")
        if echo "$verify5" | grep -q "$new5"; then
            echo "$a5" >> "$ALLOC_FILE"
            echo "$new5" >> "$BLOBBER_FILE"   # line 1: added blobber
            echo "$old5" >> "$BLOBBER_FILE"   # line 2: replaced blobber
            print_status "  ${a5:0:16}..., replaced ${old5:0:16} → ${new5:0:16} (verified)"
        else
            print_warning "replace-blobber verification failed — new blobber not in allocation, writing empty entries"
            echo "$a5" >> "$ALLOC_FILE"
            echo "" >> "$BLOBBER_FILE"; echo "" >> "$BLOBBER_FILE"
        fi
    else
        print_warning "No blobbers for replace-blobber test"
        echo "$a5" >> "$ALLOC_FILE"
        echo "" >> "$BLOBBER_FILE"; echo "" >> "$BLOBBER_FILE"
    fi

    # Allocation 6: upload then cancel → challenge count should stop increasing
    print_status "Allocation 6: upload then cancel..."
    local a6; a6=$(_create_alloc) || { print_warning "alloc 6 failed"; echo "" >> "$ALLOC_FILE"; return 0; }
    _upload "$a6" /cancel_alloc_data.bin 5 || print_warning "upload to alloc 6 failed"
    sleep 5
    $ZBOX alloc-cancel --allocation "$a6" $W 2>&1 || print_warning "alloc-cancel alloc 6 failed"
    echo "$a6" >> "$ALLOC_FILE"
    print_status "  ${a6:0:16}... (canceled)"

    print_status "Challenge protocol files generated:"
    print_status "  $ALLOC_FILE: $(wc -l < "$ALLOC_FILE" 2>/dev/null || echo 0) lines"
    print_status "  $BLOBBER_FILE: $(wc -l < "$BLOBBER_FILE" 2>/dev/null || echo 0) lines"
    print_status "Wait at least 5 minutes for challenges to accumulate before running TestProtocolChallenge."
}

# Fund test wallets: ensures SC owner, blobber owner, and staking wallets
# have sufficient balance for running tests.
fund_test_wallets() {
    print_header "Funding Test Wallets"

    local SC_OWNER_WALLET="${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    local sc_owner_id=$(jq -r '.client_id' "$SC_OWNER_WALLET")

    # Fund SC owner wallet (needs ~10 ZCN for config updates during tests)
    print_status "Funding SC owner wallet..."
    for i in $(seq 1 10); do
        $ZWALLET faucet --methodName pour --input "{}" --tokens 1 \
            --wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
        sleep 1
    done

    # Fund staking wallet
    local STAKING_WALLET="${BASE_DIR}/system_test/tests/cli_tests/config/wallets/staking_wallet.json"
    if [ -f "$STAKING_WALLET" ]; then
        local staking_id=$(jq -r '.client_id' "$STAKING_WALLET")
        print_status "Funding staking wallet: ${staking_id:0:16}..."
        # Register wallet first
        $ZWALLET create-wallet --silent --wallet wallets/staking_wallet.json \
            --configDir "${BASE_DIR}/system_test/tests/cli_tests/config" \
            --config zbox_config.yaml 2>/dev/null || true

        for i in $(seq 1 5); do
            $ZWALLET faucet --methodName pour --input "{}" --tokens 1 \
                --wallet wallets/staking_wallet.json \
                --configDir "${BASE_DIR}/system_test/tests/cli_tests/config" \
                --config zbox_config.yaml --silent 2>/dev/null || true
            sleep 1
        done
    fi

    print_status "Test wallet funding complete!"
}

# Checkout gosdk to the appropriate branch for a dependent repo.
# Reads gosdk_branch from deploy_config.yaml for the given repo,
# or falls back to the gosdk.branch in config, or master.
# Usage: checkout_gosdk_for_dependent <repo> [explicit_gosdk_branch]
checkout_gosdk_for_dependent() {
    local dependent_repo="$1"
    local explicit_branch="${2:-}"
    local gosdk_dir="${BASE_DIR}/gosdk"

    if [ ! -d "$gosdk_dir" ]; then
        print_warning "gosdk repo not found at $gosdk_dir - skipping gosdk checkout"
        return 0
    fi

    # Determine which gosdk branch to use:
    # 1. Explicit parameter (from CLI)
    # 2. gosdk_branch field for the dependent repo in config
    # 3. gosdk.branch in config
    # 4. master
    local gosdk_branch="$explicit_branch"
    if [ -z "$gosdk_branch" ]; then
        # Try to read gosdk_branch from the dependent repo's config section
        local in_repo=false
        while IFS= read -r line; do
            if echo "$line" | grep -qP "^\s+${dependent_repo}:"; then
                in_repo=true
                continue
            fi
            if $in_repo; then
                if echo "$line" | grep -qP '^\s+gosdk_branch:'; then
                    gosdk_branch=$(echo "$line" | awk -F': ' '{print $2}' | sed 's/#.*//' | tr -d ' "'"'"'')
                    break
                fi
                if echo "$line" | grep -qP '^\s+\w+:$'; then
                    break
                fi
            fi
        done < "$CONFIG_FILE"
    fi

    # Fallback to gosdk.branch in config
    if [ -z "$gosdk_branch" ]; then
        gosdk_branch=$(get_repo_branch "gosdk" "master")
    fi

    print_status "Checking out gosdk → ${gosdk_branch} (for ${dependent_repo})..."
    cd "$gosdk_dir"
    git fetch origin 2>/dev/null || true
    git stash 2>/dev/null || true
    if git rev-parse --verify "origin/${gosdk_branch}" >/dev/null 2>&1; then
        git checkout "$gosdk_branch" 2>/dev/null || git checkout -b "$gosdk_branch" "origin/${gosdk_branch}" 2>/dev/null || true
        git pull origin "$gosdk_branch" 2>/dev/null || true
    elif git rev-parse --verify "$gosdk_branch" >/dev/null 2>&1; then
        git checkout "$gosdk_branch" 2>/dev/null || true
    else
        print_warning "gosdk branch '${gosdk_branch}' not found, staying on current branch"
    fi
    local gosdk_hash=$(git rev-parse --short HEAD 2>/dev/null)
    print_status "gosdk on ${gosdk_branch} (${gosdk_hash})"
    cd "$SCRIPT_DIR"
}

# Inject local gosdk into a repo's build context so Docker/native builds use it.
# For Docker builds: copies gosdk dir, enables replace directive, adds COPY to Dockerfile.
# For native builds: enables replace directive in go.mod.
# Usage: inject_local_gosdk <repo_path> [dockerfile_paths...]
# Call cleanup_injected_gosdk <repo_path> after the build to restore original state.
inject_local_gosdk() {
    local target_repo="$1"
    shift
    local dockerfile_paths=("$@")
    local gosdk_dir="${BASE_DIR}/gosdk"

    if [ ! -d "$gosdk_dir" ]; then
        print_warning "gosdk not found at $gosdk_dir — skipping injection"
        return 1
    fi

    local gosdk_branch_name
    gosdk_branch_name=$(git -C "$gosdk_dir" branch --show-current 2>/dev/null || echo "unknown")
    print_status "Injecting local gosdk (${gosdk_branch_name}) into ${target_repo} build context..."

    # Copy gosdk into the repo
    rm -rf "${target_repo}/gosdk"
    cp -r "$gosdk_dir" "${target_repo}/gosdk"
    # Remove .git from copied gosdk to reduce Docker context size
    rm -rf "${target_repo}/gosdk/.git"

    # Enable replace directive in go.mod (try common patterns)
    if [ -f "${target_repo}/go.mod" ]; then
        # Pattern 1: commented-out replace pointing to ../gosdk
        sed -i 's|^// *replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ./gosdk|' "${target_repo}/go.mod"
        sed -i 's|^//replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ./gosdk|' "${target_repo}/go.mod"
        # Pattern 2: active replace pointing to ../gosdk (fix path for Docker context)
        sed -i 's|^replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ./gosdk|' "${target_repo}/go.mod"
        # Pattern 3: no replace at all — append one
        if ! grep -q 'replace github.com/0chain/gosdk =>' "${target_repo}/go.mod"; then
            echo 'replace github.com/0chain/gosdk => ./gosdk' >> "${target_repo}/go.mod"
        fi
        # Run go mod tidy on the HOST so go.mod/go.sum are correct BEFORE Docker copies them.
        # This avoids the Dockerfile ordering problem where tidy needs full source but runs
        # before COPY . . — by tidying on the host, the files are already reconciled.
        print_status "Running go mod tidy on ${target_repo} (host-side)..."
        (cd "${target_repo}" && go mod tidy 2>&1) || print_warning "go mod tidy failed for ${target_repo} (build may still work)"
    fi

    # Enable COPY ./gosdk in Dockerfiles with correct destination and placement.
    # The gosdk must be at $SRC_DIR/gosdk (where go.mod's replace directive resolves ./gosdk)
    # and the COPY must come BEFORE `go mod download` so the module is available at resolve time.
    for df in "${dockerfile_paths[@]}"; do
        if [ -f "$df" ]; then
            # Extract SRC_DIR from Dockerfile: prefer ENV SRC_DIR=, fall back to WORKDIR
            local src_dir
            src_dir=$(grep -oP '(?<=ENV SRC_DIR=)\S+' "$df" | head -1)
            if [ -z "$src_dir" ]; then
                src_dir=$(grep -oP '(?<=WORKDIR )\S+' "$df" | tail -1)
            fi
            [ -z "$src_dir" ] && src_dir="/app"

            # Remove any existing COPY ./gosdk lines (commented or uncommented) — clean slate
            sed -i '/^#\{0,1\} *COPY \.\/gosdk/d' "$df"

            # Add COPY ./gosdk BEFORE go mod download so the replace directive resolves.
            # go.mod/go.sum are already correct from host-side go mod tidy above.
            # Clean up stale tidy lines from previous inject runs.
            sed -i '/^RUN cd .* && go mod tidy$/d' "$df"
            sed -i '/^RUN go mod tidy$/d' "$df"
            if grep -q 'go mod download' "$df"; then
                sed -i "/RUN.*go mod download/i COPY ./gosdk ${src_dir}/gosdk" "$df"
            else
                sed -i "/COPY.*go\.mod/a COPY ./gosdk ${src_dir}/gosdk" "$df"
            fi
        fi
    done

    print_status "Local gosdk injected (${gosdk_branch_name})"
}

# Clean up injected gosdk from a repo's build context, restoring original files.
cleanup_injected_gosdk() {
    local target_repo="$1"
    rm -rf "${target_repo}/gosdk"
    git -C "$target_repo" checkout go.mod 2>/dev/null || true
    # Restore any modified Dockerfiles
    git -C "$target_repo" checkout -- '*.Dockerfile' 'Dockerfile' 'docker.local/' 2>/dev/null || true
}

# Swap image for a service without full redeployment
# Usage: ./deploy_local.sh swap-image <repo> [branch] [--gosdk-branch <branch>]
# Example: ./deploy_local.sh swap-image 0chain fix/my-branch
#          ./deploy_local.sh swap-image blobber  (uses branch from config)
#          ./deploy_local.sh swap-image eblobber staging --gosdk-branch enterprise-blobber
swap_image() {
    local repo="$1"
    local branch=""
    local gosdk_branch_override=""
    local apply_config=1  # Always apply config fixes after swap

    # $2 might be a branch name OR a flag like --gosdk-branch
    if [ -n "${2:-}" ] && [[ ! "${2:-}" == --* ]]; then
        branch="$2"
        shift 2 2>/dev/null || shift $# 2>/dev/null || true
    else
        shift 1 2>/dev/null || true
    fi

    # Parse remaining flags (e.g., --gosdk-branch)
    while [ $# -gt 0 ]; do
        case "$1" in
            --gosdk-branch)
                gosdk_branch_override="${2:-}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    if [ -z "$repo" ]; then
        print_error "Usage: $0 swap-image <repo> [branch] [--gosdk-branch <branch>]"
        print_error "Available repos: 0chain, blobber, eblobber, 0box, zauth-server, zvault, zs3server, web-apps, crawler, 0dns, zboxcli, zwalletcli, gosdk, rclone_zus"
        return 1
    fi

    # Get branch from config if not provided
    if [ -z "$branch" ]; then
        branch=$(get_repo_branch "$repo")
    fi

    local repo_path="${BASE_DIR}/${repo}"
    local config_path
    config_path=$(get_repo_branch_path "$repo")
    if [ -n "$config_path" ]; then
        repo_path="${BASE_DIR}/${config_path}"
    fi

    # Repo URL map for re-cloning if .git is missing
    declare -A SWAP_REPO_URLS=(
        ["0chain"]="https://github.com/0chain/0chain.git"
        ["blobber"]="https://github.com/0chain/blobber.git"
        ["eblobber"]="https://github.com/0chain/eblobber.git"
        ["gosdk"]="https://github.com/0chain/gosdk.git"
        ["0box"]="https://github.com/0chain/0box.git"
        ["0dns"]="https://github.com/0chain/0dns.git"
        ["zauth-server"]="https://github.com/0chain/zauth-server.git"
        ["zvault"]="https://github.com/0chain/zvault.git"
        ["zs3server"]="https://github.com/0chain/zs3server.git"
        ["crawler"]="https://github.com/0chain/crawler.git"
        ["web-apps"]="https://github.com/0chain/web-apps.git"
        ["zboxcli"]="https://github.com/0chain/zboxcli.git"
        ["zwalletcli"]="https://github.com/0chain/zwalletcli.git"
        ["rclone_zus"]="https://github.com/0chain/rclone.git"
    )

    print_header "Swapping Image: ${repo} (branch: ${branch})"

    # Step 1: Checkout and pull the branch.
    # If the directory has no .git (e.g., rsync-copied without git history), re-clone it.
    if [ ! -d "$repo_path/.git" ]; then
        local clone_url="${SWAP_REPO_URLS[$repo]:-}"
        if [ -z "$clone_url" ]; then
            print_error "No .git at $repo_path and no clone URL known for: $repo"
            return 1
        fi
        if [ -n "${GITHUB_TOKEN:-}" ]; then
            clone_url="${clone_url/https:\/\//https:\/\/${GITHUB_TOKEN}@}"
        fi
        print_status "No .git found at $repo_path — re-cloning from GitHub (branch: $branch)..."
        rm -rf "$repo_path"
        GIT_TERMINAL_PROMPT=0 git -c credential.helper= clone --branch "$branch" --depth 1 "$clone_url" "$repo_path" 2>&1 | tail -5
        if [ ! -d "$repo_path/.git" ]; then
            print_error "Clone failed for $repo"
            return 1
        fi
        print_status "Cloned $repo at branch $branch"
    else
        # Normal case: git repo already present, just checkout + pull
        print_status "Checking out ${branch}..."
        cd "$repo_path"
        git fetch origin 2>/dev/null || true
        git stash 2>/dev/null || true
        if git rev-parse --verify "origin/${branch}" >/dev/null 2>&1; then
            git checkout "$branch" 2>/dev/null || git checkout -b "$branch" "origin/${branch}" 2>/dev/null || true
            git fetch origin "$branch" 2>/dev/null || true
            git reset --hard "origin/${branch}" 2>/dev/null || git pull origin "$branch" 2>/dev/null || true
        elif git rev-parse --verify "$branch" >/dev/null 2>&1; then
            git checkout "$branch" 2>/dev/null || true
        else
            print_error "Branch '${branch}' not found"
            return 1
        fi
    fi
    local short_hash=$(git -C "$repo_path" rev-parse --short HEAD 2>/dev/null)
    print_status "On ${branch} (${short_hash})"

    # Step 2: Rebuild the Docker image
    # Auto-detect gosdk branch if not explicitly provided via --gosdk-branch.
    # This ensures a prior `swap-image gosdk <branch>` is honored by all dependents.
    if [ -z "$gosdk_branch_override" ] && [ -d "${BASE_DIR}/gosdk/.git" ]; then
        gosdk_branch_override=$(git -C "${BASE_DIR}/gosdk" branch --show-current 2>/dev/null)
    fi
    print_status "Rebuilding Docker image..."
    case "$repo" in
        gosdk)
            # gosdk is a library — check out the branch and rebuild all CLI dependents
            print_status "Checking out gosdk branch and rebuilding CLI dependents..."
            cd "$repo_path"
            git fetch origin 2>/dev/null || true
            git checkout "$branch" 2>/dev/null || git checkout -b "$branch" "origin/${branch}" 2>/dev/null || true
            git pull origin "$branch" 2>/dev/null || true
            local short_hash_sdk=$(git rev-parse --short HEAD 2>/dev/null)
            print_status "gosdk now at ${branch} (${short_hash_sdk})"
            # Rebuild zboxcli with local gosdk replace
            for _cli_repo in zboxcli zwalletcli; do
                local _cli_dir="${BASE_DIR}/${_cli_repo}"
                local _bin_name="${_cli_repo/cli/}"
                if [ ! -d "$_cli_dir" ]; then
                    print_warning "${_cli_repo} not found at ${_cli_dir} — skipping"
                    continue
                fi
                print_status "Rebuilding ${_cli_repo} with local gosdk..."
                cd "$_cli_dir"
                # Enable replace directive for local gosdk
                if [ -f "go.mod" ]; then
                    sed -i 's|^// *replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ../gosdk|' go.mod
                    sed -i 's|^//replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ../gosdk|' go.mod
                    if ! grep -q 'replace github.com/0chain/gosdk =>' go.mod; then
                        echo 'replace github.com/0chain/gosdk => ../gosdk' >> go.mod
                    fi
                fi
                go build -o "$_bin_name" . 2>&1 | tail -5 || print_warning "${_cli_repo} build failed"
                cp "$_bin_name" "${BASE_DIR}/system_test/tests/cli_tests/${_bin_name}" 2>/dev/null || true
                cp "$_bin_name" "${BASE_DIR}/system_test/tests/tokenomics_tests/${_bin_name}" 2>/dev/null || true
                git checkout go.mod 2>/dev/null || true
            done
            print_status "gosdk swap complete — zboxcli and zwalletcli rebuilt"

            # Rebuild WASM and restart web-apps so browser uses the new gosdk
            local gosdk_dir="${BASE_DIR}/gosdk"
            local web_apps_dir="${BASE_DIR}/web-apps"
            if [ -d "$gosdk_dir/wasmsdk" ] && [ -d "$web_apps_dir" ]; then
                print_status "Rebuilding zcn.wasm from gosdk ${branch}..."
                cd "$gosdk_dir"
                # Patch wasmsdk/wallet.go (privateKey guard) if needed
                if grep -q 'mnemonic == "" && !isSplit {' wasmsdk/wallet.go 2>/dev/null; then
                    sed -i 's/if mnemonic == "" \&\& !isSplit {/if mnemonic == "" \&\& !isSplit \&\& privateKey == "" {/' wasmsdk/wallet.go
                    print_status "Patched wasmsdk/wallet.go (privateKey guard)"
                fi
                # Build WASM: Docker first (matches CI), fall back to native
                local wasm_ok=false
                docker run --rm -v "$gosdk_dir":/gosdk -w /gosdk golang:1.22.5 sh -c \
                    "git config --global --add safe.directory /gosdk; make wasm-build" 2>&1 && wasm_ok=true
                if ! $wasm_ok; then
                    print_status "Docker WASM build failed, trying native..."
                    CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-s -w" -buildvcs=false -o zcn.wasm ./wasmsdk 2>&1 && wasm_ok=true
                fi
                if $wasm_ok && [ -f "$gosdk_dir/zcn.wasm" ]; then
                    print_status "zcn.wasm built ($(du -h "$gosdk_dir/zcn.wasm" | cut -f1))"
                    # Copy to all web-app public dirs
                    for app in blimp vult bolt shared chimney explorer; do
                        local pub_dir="${web_apps_dir}/packages/${app}/public"
                        if [ -d "$pub_dir" ]; then
                            cp "$gosdk_dir/zcn.wasm" "$pub_dir/zcn.wasm"
                        fi
                    done
                    # Copy to SDK src dirs
                    for pkg_dir in "${web_apps_dir}/packages/nft-core-js/src/wasm" \
                                   "${web_apps_dir}/packages/zus-sdk/src/wasm"; do
                        [ -d "$(dirname "$pkg_dir")" ] && mkdir -p "$pkg_dir" && cp "$gosdk_dir/zcn.wasm" "$pkg_dir/"
                    done
                    print_status "zcn.wasm copied to all web-app packages"
                    # Restart PM2 processes (no full yarn rebuild needed — WASM is a static asset)
                    if command -v pm2 &>/dev/null; then
                        for _app in vult bolt blimp explorer chimney; do
                            pm2 restart "$_app" --update-env 2>/dev/null && \
                                print_status "Restarted PM2: $_app" || true
                        done
                    fi
                else
                    print_warning "WASM build failed — web-apps not updated"
                fi
            else
                print_status "gosdk/wasmsdk or web-apps not found — skipping WASM rebuild"
            fi

            cd "$SCRIPT_DIR"
            return 0
            ;;
        0chain)
            cd "$repo_path"
            # Force-restore Dockerfiles to committed version in case of local modifications
            git checkout -- docker.local/build.miner/Dockerfile docker.local/build.sharder/Dockerfile 2>/dev/null || true
            print_status "Miner Dockerfile vendor line: $(grep 'go mod vendor' docker.local/build.miner/Dockerfile)"
            # Ensure base images exist (needed after docker system prune)
            if ! docker image inspect zchain_build_base > /dev/null 2>&1; then
                print_status "Building base images..."
                docker.local/bin/build.base.sh || { print_error "Failed to build base images"; return 1; }
            fi
            print_status "Building miner image..."
            docker.local/bin/build.miners.sh || { print_error "Failed to build miner image"; return 1; }
            print_status "Building sharder image..."
            docker.local/bin/build.sharders.sh || { print_error "Failed to build sharder image"; return 1; }
            ;;
        blobber)
            checkout_gosdk_for_dependent "blobber" "$gosdk_branch_override"
            cd "$repo_path"
            inject_local_gosdk "$repo_path" \
                "${repo_path}/docker.local/blobber.Dockerfile" \
                "${repo_path}/docker.local/validatorDockerfile"
            print_status "Running go mod tidy..."
            go mod tidy 2>&1 | tail -5 || true
            if ! docker image inspect blobber_base > /dev/null 2>&1; then
                print_status "Building blobber base image..."
                docker.local/bin/build.base.sh 2>&1 || { print_error "Blobber base build failed"; return 1; }
            fi
            print_status "Building blobber image..."
            docker.local/bin/build.blobber.sh 2>&1 || { print_error "Blobber build failed"; return 1; }
            print_status "Building validator image..."
            docker.local/bin/build.validator.sh 2>&1 || { print_error "Validator build failed"; return 1; }
            cleanup_injected_gosdk "$repo_path"
            ;;
        0box)
            checkout_gosdk_for_dependent "0box" "$gosdk_branch_override"
            fix_0box_config
            cd "${repo_path}"
            inject_local_gosdk "$repo_path" "${repo_path}/docker.local/Dockerfile"
            local LOCAL_IMAGE="0chaindev/0box:local-build"
            print_status "Building 0box image as ${LOCAL_IMAGE}..."
            ./docker.local/bin/build.base.sh 2>&1 || { print_error "Failed to build zbox_base"; return 1; }
            ./docker.local/bin/build.zbox.sh 2>&1 || { print_error "0box docker build failed"; return 1; }
            docker tag zbox "${LOCAL_IMAGE}"
            cleanup_injected_gosdk "$repo_path"
            local box_compose="${repo_path}/docker.local/docker-compose.yml"
            if [ -f "$box_compose" ]; then
                sed -i "s|image: 0chaindev/0box:.*|image: ${LOCAL_IMAGE}|g" "$box_compose"
                print_status "Updated docker-compose.yml to use ${LOCAL_IMAGE}"
            fi
            ;;
        zauth-server)
            # Checkout correct gosdk branch for zauth-server and inject local gosdk
            checkout_gosdk_for_dependent "zauth-server" "$gosdk_branch_override"
            inject_local_gosdk "$repo_path" "${repo_path}/docker.local/Dockerfile"
            if [ "$apply_config" = "1" ]; then
                patch_zauth_faucet
            fi
            cd "${repo_path}/docker.local"
            docker build -f Dockerfile -t zauthserver ../ 2>&1 || print_error "zauthserver image build failed"
            cleanup_injected_gosdk "$repo_path"
            docker compose -p zauth up -d --force-recreate zauthserver
            ;;
        zvault)
            cd "${repo_path}/docker.local"
            docker compose build 2>/dev/null || true
            ;;
        zs3server)
            checkout_gosdk_for_dependent "zs3server" "$gosdk_branch_override"
            cd "$repo_path"
            inject_local_gosdk "$repo_path" "${repo_path}/Dockerfile"
            # Detect required Go version from go.work or go.mod
            local zs3_required_go
            zs3_required_go=$(grep '^go ' "${repo_path}/go.work" 2>/dev/null | awk '{print $2}' | head -1)
            [ -z "$zs3_required_go" ] && zs3_required_go=$(grep '^go ' "${repo_path}/go.mod" 2>/dev/null | awk '{print $2}' | head -1)
            # zs3server Dockerfile uses zbox_base — build it with the required Go version
            # (0box base.Dockerfile may use an older Go version, so patch it if needed)
            local zbox_base_src="${BASE_DIR}/0box/docker.local/base.Dockerfile"
            if [ ! -f "$zbox_base_src" ]; then
                print_error "Cannot find 0box/docker.local/base.Dockerfile to build zbox_base"
                return 1
            fi
            if [ -n "$zs3_required_go" ]; then
                print_status "Building zbox_base with go ${zs3_required_go} for zs3server..."
                local tmp_zbox_df
                tmp_zbox_df=$(mktemp /tmp/zbox_base.XXXXXX.Dockerfile)
                # Map go 1.X minor to canonical Alpine version (go 1.22 → alpine3.19, go 1.23+ → alpine3.20)
                local zs3_req_minor; zs3_req_minor=$(echo "$zs3_required_go" | cut -d. -f2)
                local zbox_target_alpine
                if [ "${zs3_req_minor:-0}" -le 21 ]; then zbox_target_alpine="alpine3.18"
                elif [ "${zs3_req_minor:-0}" -eq 22 ]; then zbox_target_alpine="alpine3.19"
                else zbox_target_alpine="alpine3.20"
                fi
                sed "s|FROM golang:[0-9][0-9.]*-alpine[0-9.]*|FROM golang:${zs3_required_go}-${zbox_target_alpine}|g" \
                    "$zbox_base_src" > "$tmp_zbox_df"
                DOCKER_BUILDKIT=0 docker build -t zbox_base -f "$tmp_zbox_df" "${BASE_DIR}/0box" 2>&1
                local zbox_build_rc=$?
                rm -f "$tmp_zbox_df"
                [ $zbox_build_rc -ne 0 ] && { print_error "Failed to build zbox_base"; return 1; }
            else
                if ! docker image inspect zbox_base > /dev/null 2>&1; then
                    print_status "Building zbox_base (required by zs3server Dockerfile)..."
                    DOCKER_BUILDKIT=0 docker build -t zbox_base -f "$zbox_base_src" "${BASE_DIR}/0box" 2>&1 || { print_error "Failed to build zbox_base"; return 1; }
                fi
            fi
            # Build zs3server image from repo root Dockerfile
            local ZS3_IMAGE="0chaindev/blimp-minioserver:local-build"
            print_status "Building zs3server image as ${ZS3_IMAGE}..."
            DOCKER_BUILDKIT=0 docker build -t "${ZS3_IMAGE}" . 2>&1 || { print_error "zs3server docker build failed"; return 1; }
            cleanup_injected_gosdk "$repo_path"
            local zs3_compose="${repo_path}/environment/docker-compose.yaml"
            if [ -f "$zs3_compose" ]; then
                sed -i "s|image: 0chaindev/blimp-minioserver:.*|image: ${ZS3_IMAGE}|g" "$zs3_compose"
                print_status "Updated docker-compose.yaml to use ${ZS3_IMAGE}"
            fi
            ;;
        eblobber)
            checkout_gosdk_for_dependent "eblobber" "$gosdk_branch_override"
            cd "$repo_path"
            inject_local_gosdk "$repo_path" \
                "${repo_path}/docker.local/blobber.Dockerfile" \
                "${repo_path}/docker.local/validator.Dockerfile"
            # Detect required Go version from go.mod AND injected gosdk's go.mod; patch base.Dockerfile
            local eblobber_required_go gosdk_required_go
            eblobber_required_go=$(grep '^go ' "${repo_path}/go.mod" 2>/dev/null | awk '{print $2}' | head -1)
            gosdk_required_go=$(grep '^go ' "${repo_path}/gosdk/go.mod" 2>/dev/null | awk '{print $2}' | head -1)
            # Use whichever requires the higher Go version
            if [ -n "$gosdk_required_go" ]; then
                local _eb_minor _gs_minor
                _eb_minor=$(echo "${eblobber_required_go:-0}" | cut -d. -f2)
                _gs_minor=$(echo "$gosdk_required_go" | cut -d. -f2)
                if [ "$_gs_minor" -gt "${_eb_minor:-0}" ] 2>/dev/null; then
                    eblobber_required_go="$gosdk_required_go"
                elif [ "$_gs_minor" -eq "${_eb_minor:-0}" ] 2>/dev/null; then
                    local _eb_patch _gs_patch
                    _eb_patch=$(echo "${eblobber_required_go:-0}" | cut -d. -f3)
                    _gs_patch=$(echo "$gosdk_required_go" | cut -d. -f3)
                    if [ "${_gs_patch:-0}" -gt "${_eb_patch:-0}" ] 2>/dev/null; then
                        eblobber_required_go="$gosdk_required_go"
                    fi
                fi
            fi
            local base_dockerfile="${repo_path}/docker.local/base.Dockerfile"
            if [ -n "$eblobber_required_go" ] && [ -f "$base_dockerfile" ]; then
                local base_go
                base_go=$(grep -oP '(?<=FROM golang:)[0-9]+\.[0-9]+' "$base_dockerfile" | head -1)
                local req_minor cur_minor
                req_minor=$(echo "$eblobber_required_go" | cut -d. -f2)
                cur_minor=$(echo "$base_go" | cut -d. -f2)
                if [ -n "$req_minor" ] && [ -n "$cur_minor" ] && [ "$req_minor" -gt "$cur_minor" ] 2>/dev/null; then
                    local target_alpine
                    if [ "$req_minor" -le 21 ]; then target_alpine="alpine3.18"
                    elif [ "$req_minor" -eq 22 ]; then target_alpine="alpine3.19"
                    else target_alpine="alpine3.20"
                    fi
                    print_status "go.mod requires go ${eblobber_required_go} — updating base.Dockerfile to golang:${eblobber_required_go}-${target_alpine}..."
                    sed -i "s|FROM golang:[0-9][0-9.]*-alpine[0-9.]*|FROM golang:${eblobber_required_go}-${target_alpine}|" "$base_dockerfile"
                    if ! grep -q 'CGO_CFLAGS' "$base_dockerfile"; then
                        sed -i '/^FROM golang:/a ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE=1"' "$base_dockerfile"
                    fi
                fi
            fi
            print_status "Running go mod tidy..."
            go mod tidy 2>&1 | tail -5 || true
            print_status "Building eblobber_base image..."
            DOCKER_IMAGE_BASE=eblobber_base docker.local/bin/build.base.sh 2>&1 || {
                print_error "Failed to build eblobber_base image"
                return 1
            }
            print_status "Building eblobber Docker image..."
            DOCKER_IMAGE_BASE=eblobber_base DOCKER_IMAGE_BLOBBER="-t eblobber" docker.local/bin/build.blobber.sh 2>&1 || {
                print_error "Failed to build eblobber Docker image"
                return 1
            }
            cleanup_injected_gosdk "$repo_path"
            ;;
        web-apps)
            # Step A: Build zcn.wasm from gosdk
            print_status "Building zcn.wasm from gosdk for web-apps..."
            checkout_gosdk_for_dependent "web-apps" "$gosdk_branch_override"
            local gosdk_dir="${BASE_DIR}/gosdk"
            if [ ! -d "$gosdk_dir/wasmsdk" ]; then
                print_error "gosdk/wasmsdk directory not found at $gosdk_dir - cannot build WASM"
                return 1
            fi
            # Build WASM using Docker (matches CI: golang:1.22.5)
            print_status "Building WASM with Docker (golang:1.22.5)..."
            docker run --rm -v "$gosdk_dir":/gosdk -w /gosdk golang:1.22.5 sh -c \
                "git config --global --add safe.directory /gosdk; make wasm-build" || {
                print_error "Docker WASM build failed, trying native build..."
                (cd "$gosdk_dir" && CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-s -w" -buildvcs=false -o ./zcn.wasm ./wasmsdk) || {
                    print_error "Native WASM build also failed"
                    return 1
                }
            }
            if [ ! -f "$gosdk_dir/zcn.wasm" ]; then
                print_error "zcn.wasm not found after build"
                return 1
            fi
            print_status "zcn.wasm built successfully ($(du -h "$gosdk_dir/zcn.wasm" | cut -f1))"

            # Step B: Copy zcn.wasm into all web-app packages
            cd "$repo_path"
            local wasm_copied=0
            for pkg_dir in packages/*/public; do
                if [ -d "$pkg_dir" ]; then
                    cp "$gosdk_dir/zcn.wasm" "$pkg_dir/zcn.wasm"
                    print_status "Copied zcn.wasm to $pkg_dir/"
                    wasm_copied=$((wasm_copied + 1))
                fi
            done
            if [ "$wasm_copied" -eq 0 ]; then
                print_warning "No packages/*/public directories found - copying to repo root"
                cp "$gosdk_dir/zcn.wasm" "$repo_path/zcn.wasm"
            fi

            # Step C: Also copy matching wasm_exec.js from Go 1.22.5
            local wasm_exec_src="$(go env GOROOT 2>/dev/null)/misc/wasm/wasm_exec.js"
            if [ -f "$wasm_exec_src" ]; then
                for pkg_dir in packages/*/public; do
                    if [ -d "$pkg_dir" ] && [ -f "$pkg_dir/wasm_exec.js" ]; then
                        cp "$wasm_exec_src" "$pkg_dir/wasm_exec.js"
                    fi
                done
                print_status "Updated wasm_exec.js from local Go installation"
            fi

            # Step D: Fix .env before build — git .env has production values (mainnet Firebase,
            # production Alchemy API keys, mainnet.zus.network domain) that break test deployments.
            # Instead of patching individual fields via sed (fragile — misses new production values),
            # rewrite all .env files from scratch using setup_web_app_env_files() which uses the
            # complete env template with values from .secrets.env.
            if [ "$apply_config" = "1" ]; then
                setup_web_app_env_files "$repo_path"
            else
                print_status "Skipping .env rewrite (pass --with-config to apply)"
            fi

            # Step D1b: Add fetch timeout to getZusBlogs — blog.zus.network may be unreachable
            # from test servers, causing SSG to hang indefinitely during next build.
            local _org_schemes="${repo_path}/packages/shared/src/lib/constants/orgSchemes.js"
            if [ -f "$_org_schemes" ] && ! grep -q "AbortSignal.timeout" "$_org_schemes"; then
                sed -i.tmp \
                    's/const res = await fetch(GRAPH_ENDPOINT, {/const res = await fetch(GRAPH_ENDPOINT, { signal: AbortSignal.timeout(5000),/' \
                    "$_org_schemes"
                rm -f "${_org_schemes}.tmp"
                print_status "Added 5s fetch timeout to getZusBlogs (prevents SSG hang)"
            fi

            # Step D1c: Replace deprecated blog.zus.network with zus.network/blog
            if [ -f "$_org_schemes" ] && grep -q "blog\.zus\.network" "$_org_schemes"; then
                sed -i.tmp 's|blog\.zus\.network/graphql|zus.network/blog/graphql|g' "$_org_schemes"
                rm -f "${_org_schemes}.tmp"
                print_status "Updated blog URL to zus.network/blog/graphql"
            fi

            # Step D2: Apply WASM wallet fix — git wasm/index.js has a guard that returns early
            # when mnemonic is empty AND wallet is not split. This skips setWallet() so the
            # gosdk clientId is never set → "Client id is required" on all SDK calls (faucet, etc.).
            # Fix: also allow when privateKey is available (keys-only wallets after resetGoWasm).
            local _wasm_index="${repo_path}/packages/shared/src/lib/wasm/index.js"
            if [ -f "$_wasm_index" ]; then
                sed -i.tmp \
                    's/if (!mnemonic && !wallet?.is_split) {/if (!mnemonic \&\& !wallet?.is_split \&\& !privateKey) {/' \
                    "$_wasm_index"
                rm -f "${_wasm_index}.tmp"
                print_status "Applied WASM wallet fix (privateKey guard) in wasm/index.js"
            fi

            # Step D3: Apply user.js patches (user_id in OTP body + mock token dispatch for dev)
            local _user_js="${repo_path}/packages/shared/src/store/user/actions/user.js"
            if [ -f "$_user_js" ]; then
                python3 - "$_user_js" << 'PYEOF'
import sys
path = sys.argv[1]
with open(path) as f:
    content = f.read()
changed = False

# Fix A: user_id in OTP body (needed for 0box IsDevelopmentNoAuth() bypass)
if "devUserId" not in content:
    old_a = "  if (userName) body.append('username', userName)\n"
    new_a = old_a + (
        "\n"
        "  // Dev mode: 0box IsDevelopmentNoAuth() requires user_id for OTP verify endpoints.\n"
        "  // In production: 0box derives user ID from Firebase token; this field is ignored.\n"
        "  const devUserId = firebaseTokens?.uid ||\n"
        "    ('devnet_' + (phoneNumber || email || '').toLowerCase().replace(/[^a-z0-9]/g, ''))\n"
        "  body.append('user_id', devUserId)\n"
    )
    if old_a in content:
        content = content.replace(old_a, new_a)
        changed = True
        print("Patched user.js: added user_id to OTP verify body")

# Fix B: mock token dispatch when 0box returns no customToken
if "mock-token-" not in content:
    old_b = (
        "    dispatch({ type: types.VERIFY_OTP_SUCCESS, payload: firebaseTokens })\n"
        "    return defaultResponse\n"
        "  } catch (e) {"
    )
    new_b = (
        "    // Dev mode: 0box returns no customToken when IsDevelopmentNoAuth() is true.\n"
        "    // Synthesize mock tokens so subsequent API calls have valid X-App-User-ID + X-App-ID-TOKEN.\n"
        "    const syntheticFirebaseData = {\n"
        "      uid: devUserId,\n"
        "      accessToken: 'mock-token-' + devUserId,\n"
        "      refreshToken: 'mock-token-' + devUserId,\n"
        "      expirationTime: Date.now() + 3600000,\n"
        "    }\n"
        "    dispatch({ type: types.VERIFY_OTP_SUCCESS, payload: syntheticFirebaseData })\n"
        "    return { data: syntheticFirebaseData }\n"
        "  } catch (e) {"
    )
    if old_b in content:
        content = content.replace(old_b, new_b, 1)
        changed = True
        print("Patched user.js: mock token dispatch for dev OTP flow")

if changed:
    with open(path, 'w') as f:
        f.write(content)
else:
    print("user.js: already patched or targets not found")
PYEOF
                print_status "Applied user.js dev patches"
            fi

            # Step D4: Patch WASM loader to use local /zcn.wasm (not CDN)
            patch_wasm_loader_local "$repo_path"

            # Step E: Build web-apps using yarn workspaces (same as build_web_apps())
            cd "$repo_path"
            if ! command -v yarn &>/dev/null; then
                npm install -g yarn 2>/dev/null || true
            fi
            print_status "Installing web-app dependencies with yarn..."
            yarn install 2>/dev/null || true
            print_status "Building shared package..."
            yarn workspace shared build 2>/dev/null || true
            for _app in vult bolt blimp explorer chimney; do
                print_status "Building ${_app}..."
                yarn workspace "$_app" build 2>/dev/null || true
            done
            print_status "Web apps built"
            # Step F: Restart PM2 processes so they pick up the new build artifacts.
            if command -v pm2 &>/dev/null; then
                for _app in vult bolt blimp explorer chimney; do
                    pm2 restart "$_app" --update-env 2>/dev/null && \
                        print_status "Restarted PM2 process: $_app" || true
                done
            fi
            ;;
        zboxcli|zwalletcli)
            # CLI tools: native Go build — enable replace directive for local gosdk
            checkout_gosdk_for_dependent "$repo" "$gosdk_branch_override"
            cd "$repo_path"
            # Enable go.mod replace to use local gosdk (../gosdk for native builds)
            if [ -f "go.mod" ]; then
                sed -i 's|^// *replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ../gosdk|' go.mod
                sed -i 's|^//replace github.com/0chain/gosdk => \.\./gosdk|replace github.com/0chain/gosdk => ../gosdk|' go.mod
                if ! grep -q 'replace github.com/0chain/gosdk =>' go.mod; then
                    echo 'replace github.com/0chain/gosdk => ../gosdk' >> go.mod
                fi
            fi
            local bin_name
            bin_name=$(basename "$repo_path" | sed 's/cli$//')
            print_status "Building ${repo} binary..."
            go build -o "${BASE_DIR}/system_test/tests/cli_tests/${bin_name}" . 2>&1 | tail -5 || true
            cp "${BASE_DIR}/system_test/tests/cli_tests/${bin_name}" "${BASE_DIR}/system_test/tests/tokenomics_tests/${bin_name}" 2>/dev/null || true
            # Restore go.mod
            git checkout go.mod 2>/dev/null || true
            print_status "Binary built and copied to test directories"
            cd "$SCRIPT_DIR"
            return 0
            ;;
        crawler)
            checkout_gosdk_for_dependent "crawler" "$gosdk_branch_override"
            cd "$repo_path"
            inject_local_gosdk "$repo_path" "${repo_path}/docker.local/Dockerfile"
            print_status "Building crawler Docker image..."
            docker compose -f docker.local/docker-compose.yml build 2>&1 | tail -10 || {
                print_error "Crawler Docker build failed"
                return 1
            }
            cleanup_injected_gosdk "$repo_path"
            ;;
        0dns)
            cd "$repo_path/docker.local"
            print_status "Building 0dns Docker image..."
            docker compose build 2>&1 | tail -10 || {
                print_error "0dns Docker build failed"
                return 1
            }
            ;;
        rclone_zus)
            # rclone is a Go binary — rebuild from source
            checkout_gosdk_for_dependent "rclone_zus" "$gosdk_branch_override"
            cd "$repo_path"
            print_status "Building rclone binary..."
            go build -o rclone . 2>&1 | tail -5 || { print_error "rclone build failed"; return 1; }
            print_status "rclone binary built"
            cd "$SCRIPT_DIR"
            return 0
            ;;
        *)
            print_error "Don't know how to build image for: $repo"
            return 1
            ;;
    esac

    # Step 3: Restart containers with new image
    print_status "Restarting containers..."
    case "$repo" in
        0chain)
            print_status "Stopping miners and sharders..."
            docker stop miner-1 miner-2 miner-3 miner-4 2>/dev/null || true
            docker stop sharder-1 sharder-2 2>/dev/null || true
            sleep 2
            # CRITICAL: Re-enable Kafka before restarting sharders.
            # git pull resets 0chain.yaml to defaults (kafka.enabled: false).
            # Must patch BEFORE force-recreate so sharders start with Kafka on.
            if [ "$apply_config" = "1" ]; then
                configure_kafka_in_configs
            else
                print_warning "Skipping Kafka config (pass --with-config if Kafka stops working after swap)"
            fi
            # Recreate with new image
            for i in 1 2; do
                cd "${repo_path}/docker.local/build.sharder"
                SHARDER=$i docker compose -p sharder$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
            done
            for i in 1 2 3 4; do
                cd "${repo_path}/docker.local/build.miner"
                MINER=$i docker compose -p miner$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
            done
            ;;
        blobber)
            # Fix config before restart so block_worker points to local chain (not dev.0chain.net)
            if [ "$apply_config" = "1" ]; then
                fix_blobber_config
            fi
            for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
                docker rm -f "blobber-$i" "validator-$i" 2>/dev/null || true
            done
            sleep 2
            # Blobbers 1-6: use specific compose if available, else generic
            for i in 1 2 3 4 5 6; do
                cd "${repo_path}/docker.local"
                if [ -f "b0docker-compose-${i}.yml" ]; then
                    docker compose -p blobber$i -f b0docker-compose-${i}.yml up -d --force-recreate 2>/dev/null || true
                else
                    BLOBBER=$i docker compose -p blobber$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
                fi
            done
            # Blobbers 7-12: use specific compose if available, else generic (10-12 have specific files to avoid invalid ports)
            for i in 7 8 9 10 11 12; do
                cd "${repo_path}/docker.local"
                if [ -f "b0docker-compose-${i}.yml" ]; then
                    docker compose -p blobber$i -f b0docker-compose-${i}.yml up -d --force-recreate 2>/dev/null || true
                else
                    BLOBBER=$i docker compose -p blobber$i -f b0docker-compose.yml up -d --force-recreate 2>/dev/null || true
                fi
            done
            ;;
        0box)
            # Pin Redis image before restart (redis:alpine v8+ causes SIGSEGV)
            local box_compose="${repo_path}/docker.local/docker-compose.yml"
            if grep -q 'redis:alpine' "$box_compose" 2>/dev/null; then
                sed -i 's|"redis:alpine"|"redis:7.4.3-alpine"|g; s|image: redis:alpine|image: redis:7.4.3-alpine|g' "$box_compose"
            fi
            # Fix ES: 0box compose includes elasticsearch as a dependency, but ES runs
            # standalone on testnet0. Remove depends_on/links for ES and point env var to IP.
            local es_ip
            es_ip=$(docker inspect elasticsearch --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null || echo "198.18.14.5")
            [ -z "$es_ip" ] && es_ip="198.18.14.5"
            if [ -f "$box_compose" ]; then
                sed -i "s|ELASTIC_HOST=elasticsearch|ELASTIC_HOST=${es_ip}|" "$box_compose"
                # Remove elasticsearch from depends_on (multi-line block)
                python3 -c "
import re, sys
with open('$box_compose') as f: t = f.read()
# Remove elasticsearch dependency lines
t = re.sub(r'\n\s+elasticsearch:\s*\n\s+condition:\s+service_healthy', '', t)
# Remove elasticsearch link
t = re.sub(r'\n\s+- elasticsearch:elasticsearch', '', t)
with open('$box_compose','w') as f: f.write(t)
" 2>/dev/null || true
            fi
            docker rm -f 0box 0box-redis postgres-0box 2>/dev/null || true
            cd "${repo_path}/docker.local"
            docker compose up -d --no-deps postgres redis 0box 2>/dev/null || true
            ;;
        zauth-server)
            docker stop zauth 2>/dev/null || true
            cd "${repo_path}/docker.local"
            docker compose -p zauth up -d --force-recreate 2>/dev/null || true
            ;;
        zvault)
            docker stop zvault 2>/dev/null || true
            cd "${repo_path}/docker.local"
            docker compose -p zvault up -d --force-recreate 2>/dev/null || true
            ;;
        zs3server)
            docker rm -f minioserver environment-logsearchapi-1 environment-postgres-db-1 postgres-db minioclient 2>/dev/null || true
            cd "${repo_path}/environment"
            local zs3_compose="${repo_path}/environment/docker-compose.yaml"
            if [ "$apply_config" = "1" ]; then
            # Patch docker-compose: remove conflicting port bindings and fix zcn config mount
            ZS3_COMPOSE="$zs3_compose" python3 << 'ZS3PYEOF'
import yaml, os
compose_path = os.environ['ZS3_COMPOSE']
with open(compose_path) as f:
    data = yaml.safe_load(f)
svcs = data.get('services', {})
# Remove port bindings from internal services (conflicts with postgres-0box:5432, zauth:8080, web-apps:3001)
for svc in ['postgres-db', 'logsearchapi', 'minioclient']:
    if svc in svcs and 'ports' in svcs[svc]:
        del svcs[svc]['ports']
# Fix minioserver: mount /root/.zcn-zs3 instead of ~/.zcn (needs internal 0dns IP, not localhost)
if 'minioserver' in svcs:
    vols = svcs['minioserver'].get('volumes', [])
    new_vols = [v for v in vols if '/.zcn' not in str(v)]
    new_vols.append('/root/.zcn-zs3:/root/.zcn')
    svcs['minioserver']['volumes'] = new_vols
    # Connect minioserver to testnet0 so it can reach 0dns at 198.18.0.100:9091
    nets = svcs['minioserver'].get('networks', {})
    if isinstance(nets, list):
        if 'testnet0' not in nets:
            nets.append('testnet0')
    elif isinstance(nets, dict):
        if 'testnet0' not in nets:
            nets['testnet0'] = {}
    else:
        nets = ['testnet0']
    svcs['minioserver']['networks'] = nets
# Declare testnet0 as external (created by the chain docker-compose)
if 'networks' not in data:
    data['networks'] = {}
if 'testnet0' not in data['networks']:
    data['networks']['testnet0'] = {'external': True}
with open(compose_path, 'w') as f:
    yaml.dump(data, f, default_flow_style=False, allow_unicode=True)
print('Patched docker-compose.yaml')
ZS3PYEOF
            # Create zs3server-specific zcn config with internal 0dns IP (containers can't use localhost:9091)
            mkdir -p /root/.zcn-zs3
            # config.yaml is the standard zcn SDK config (not local.yaml which is used by CLI tools)
            sed 's|block_worker: http://localhost:9091|block_worker: http://198.18.0.100:9091|g' \
                "${ZCN_CONFIG_DIR}/config.yaml" > /root/.zcn-zs3/config.yaml 2>/dev/null || \
            sed 's|block_worker: http://localhost:9091|block_worker: http://198.18.0.100:9091|g' \
                "${ZCN_CONFIG_DIR}/${ZCN_CONFIG_FILE}" > /root/.zcn-zs3/config.yaml 2>/dev/null || true
            cp -f "${ZCN_CONFIG_DIR}"/*.json /root/.zcn-zs3/ 2>/dev/null || true
            # Copy allocation.txt (required by zs3server to identify which Zus allocation to use)
            [ -f "${ZCN_CONFIG_DIR}/allocation.txt" ] && cp -f "${ZCN_CONFIG_DIR}/allocation.txt" /root/.zcn-zs3/ 2>/dev/null || true
            fi  # end --with-config for zs3server
            docker compose up -d --force-recreate 2>/dev/null || true
            ;;
        eblobber)
            # Fix config before restart so block_worker points to local chain (not dev.0chain.net)
            if [ "$apply_config" = "1" ]; then
                fix_blobber_config
            fi
            local EBLOBBER_DOCKER_DIR="${repo_path}/docker.local"
            local eb_count="${ENTERPRISE_BLOBBER_COUNT:-5}"
            for i in $(seq 1 "$eb_count"); do
                docker stop "eblobber-${i}" 2>/dev/null || true
            done
            sleep 2
            cd "$EBLOBBER_DOCKER_DIR"
            for i in $(seq 1 "$eb_count"); do
                if [ -f "eb0docker-compose.yml" ]; then
                    BLOBBER=$i docker compose -p "eblobber${i}" -f "eb0docker-compose.yml" up -d --force-recreate eblobber 2>/dev/null || true
                fi
            done
            ;;
        web-apps)
            # Restart PM2 web-app processes with the new build output
            start_web_apps "$repo_path"
            ;;
        crawler)
            docker stop crawler 2>/dev/null || true
            docker rm -f crawler 2>/dev/null || true
            cd "${repo_path}/docker.local"
            # Fix volume mount if needed (crawler reads from /usr/src/app/docker.local/config)
            local crawler_compose="${repo_path}/docker.local/docker-compose.yml"
            if [ -f "$crawler_compose" ] && grep -q './config:/crawler/config' "$crawler_compose"; then
                sed -i 's|./config:/crawler/config|./config:/usr/src/app/docker.local/config|g' "$crawler_compose"
            fi
            docker compose up -d --force-recreate 2>/dev/null || true
            ;;
        0dns)
            docker stop 0dns 2>/dev/null || true
            docker rm -f 0dns 2>/dev/null || true
            if [ "$apply_config" = "1" ]; then
                # Copy magic block and fix config
                local DNS_MB="${repo_path}/docker.local/config/b0magicBlock.json"
                local CHAIN_MB_FULL="${BASE_DIR}/0chain/docker.local/config/b0magicBlock_4_miners_2_sharders.json"
                local CHAIN_MB="${BASE_DIR}/0chain/docker.local/config/b0magicBlock.json"
                if [ -f "$CHAIN_MB_FULL" ]; then
                    cp "$CHAIN_MB_FULL" "$DNS_MB"
                elif [ -f "$CHAIN_MB" ]; then
                    cp "$CHAIN_MB" "$DNS_MB"
                fi
                local DNS_CONFIG="${repo_path}/docker.local/config/0dns.yaml"
                if [ -f "$DNS_CONFIG" ]; then
                    sed -i.bak 's/use_localhost: true/use_localhost: false/' "$DNS_CONFIG" 2>/dev/null || true
                    rm -f "${DNS_CONFIG}.bak"
                fi
                local DNS_COMPOSE="${repo_path}/docker.local/docker-compose.yml"
                if [ -f "$DNS_COMPOSE" ] && ! grep -q "198.18.0.100" "$DNS_COMPOSE"; then
                    sed -i.bak 's/198\.18\.0\.[0-9]*/198.18.0.100/g' "$DNS_COMPOSE"
                    rm -f "${DNS_COMPOSE}.bak"
                fi
            fi
            cd "${repo_path}/docker.local"
            docker compose -p 0dns up -d --force-recreate 2>/dev/null || true
            ;;
    esac

    cd "$SCRIPT_DIR"

    # After swapping, ensure test configs point to local chain (git pull may
    # have overwritten them with dev.zus.network URLs)
    if [ -d "${BASE_DIR}/system_test/tests" ]; then
        configure_test_configs 2>/dev/null || true
    fi

    print_status "Image swap complete for ${repo}!"
    print_status "Branch: ${branch} (${short_hash})"
}

# ============================================================
# Monitoring, View Change, and Chaos Testing
# ============================================================

# Start cAdvisor container monitoring (port 8080)
start_cadvisor() {
    print_header "Starting cAdvisor Container Monitoring"

    # Stop existing cadvisor container if running
    docker stop cadvisor 2>/dev/null || true
    docker rm cadvisor 2>/dev/null || true

    # Use port 8085 (8080 conflicts with zauth)
    # Use v0.49.1+ for cgroup v2 support (google/cadvisor:latest is too old)
    docker run \
        --volume=/:/rootfs:ro \
        --volume=/var/run:/var/run:rw \
        --volume=/sys:/sys:ro \
        --volume=/var/lib/docker/:/var/lib/docker:ro \
        --volume=/dev/disk/:/dev/disk:ro \
        --publish=8085:8080 \
        --detach=true \
        --restart=unless-stopped \
        --name=cadvisor \
        gcr.io/cadvisor/cadvisor:v0.49.1

    print_status "cAdvisor running at http://localhost:8085"
}

# Start pgAdmin for sharder postgres databases
# Provides web UI at port 5434 (http://localhost:5434) for inspecting sharder event DBs.
# Nginx routes: /pgadmin-sharder/
start_pgadmin_sharders() {
    print_header "Starting pgAdmin for Sharder Databases"

    # Stop existing pgadmin-sharder container if running
    docker stop pgadmin-sharder 2>/dev/null || true
    docker rm pgadmin-sharder 2>/dev/null || true

    # Create server config for auto-connecting to sharder postgres containers
    local PGADMIN_DIR="/tmp/pgadmin-sharder"
    mkdir -p "$PGADMIN_DIR"

    cat > "$PGADMIN_DIR/servers.json" << 'PGEOF'
{
    "Servers": {
        "1": {
            "Name": "Sharder-1 EventDB",
            "Group": "Sharders",
            "Host": "sharder-postgres-1",
            "Port": 5432,
            "MaintenanceDB": "events_db",
            "Username": "zchain_user",
            "SSLMode": "prefer",
            "PassFile": "/pgpass"
        },
        "2": {
            "Name": "Sharder-2 EventDB",
            "Group": "Sharders",
            "Host": "sharder-postgres-2",
            "Port": 5432,
            "MaintenanceDB": "events_db",
            "Username": "zchain_user",
            "SSLMode": "prefer",
            "PassFile": "/pgpass"
        }
    }
}
PGEOF

    # Create .pgpass file for auto-login (hostname:port:database:username:password)
    cat > "$PGADMIN_DIR/pgpass" << 'PASSEOF'
sharder-postgres-1:5432:events_db:zchain_user:zchian
sharder-postgres-2:5432:events_db:zchain_user:zchian
sharder-postgres-1:5432:*:zchain_user:zchian
sharder-postgres-2:5432:*:zchain_user:zchian
PASSEOF
    chmod 600 "$PGADMIN_DIR/pgpass"
    chown 5050:5050 "$PGADMIN_DIR/pgpass" "$PGADMIN_DIR/servers.json"

    # Run pgAdmin container connected to sharder1 network so it can reach sharder postgres containers
    docker run \
        --detach \
        --restart=unless-stopped \
        --name=pgadmin-sharder \
        --publish=5434:80 \
        --network=sharder1_default \
        -e PGADMIN_DEFAULT_EMAIL=admin@zus.network \
        -e PGADMIN_DEFAULT_PASSWORD="${PGADMIN_DEFAULT_PASSWORD:-admin}" \
        -e PGADMIN_CONFIG_SERVER_MODE=False \
        -e PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False \
        -e SCRIPT_NAME=/pgadmin-sharder \
        -v "$PGADMIN_DIR/servers.json:/pgadmin4/servers.json:ro" \
        -v "$PGADMIN_DIR/pgpass:/pgpass" \
        dpage/pgadmin4:latest 2>/dev/null || {
        print_warning "pgadmin-sharder failed to start"
        return 0
    }

    # Connect to sharder2 and testnet0 networks
    docker network connect sharder2_default pgadmin-sharder 2>/dev/null || true
    docker network connect testnet0 pgadmin-sharder 2>/dev/null || true

    print_status "pgAdmin for sharders running at http://localhost:5434"
    print_status "  Auto-login enabled (no password prompt)"
    print_status "  Pre-configured servers: Sharder-1 EventDB, Sharder-2 EventDB"
}

# Start pgAdmin for 0box database
start_pgadmin_0box() {
    print_header "Starting pgAdmin for 0box Database"

    # Stop existing pgadmin-0box container if running
    docker stop pgadmin-0box 2>/dev/null || true
    docker rm pgadmin-0box 2>/dev/null || true

    local PGADMIN_DIR="/tmp/pgadmin-0box"
    mkdir -p "$PGADMIN_DIR"

    cat > "$PGADMIN_DIR/servers.json" << 'PGEOF'
{
    "Servers": {
        "1": {
            "Name": "0box DB",
            "Group": "Services",
            "Host": "postgres-0box",
            "Port": 5432,
            "MaintenanceDB": "zbox",
            "Username": "zbox_user",
            "SSLMode": "prefer",
            "PassFile": "/pgpass"
        }
    }
}
PGEOF

    cat > "$PGADMIN_DIR/pgpass" << 'PASSEOF'
postgres-0box:5432:zbox:zbox_user:zbox_server
postgres-0box:5432:*:zbox_user:zbox_server
PASSEOF
    chmod 600 "$PGADMIN_DIR/pgpass"
    chown 5050:5050 "$PGADMIN_DIR/pgpass" "$PGADMIN_DIR/servers.json"

    docker run \
        --detach \
        --restart=unless-stopped \
        --name=pgadmin-0box \
        --publish=5435:80 \
        --network=testnet0 \
        -e PGADMIN_DEFAULT_EMAIL=admin@zus.network \
        -e PGADMIN_DEFAULT_PASSWORD="${PGADMIN_DEFAULT_PASSWORD:-admin}" \
        -e PGADMIN_CONFIG_SERVER_MODE=False \
        -e PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False \
        -e SCRIPT_NAME=/pgadmin-0box \
        -v "$PGADMIN_DIR/servers.json:/pgadmin4/servers.json:ro" \
        -v "$PGADMIN_DIR/pgpass:/pgpass" \
        dpage/pgadmin4:latest 2>/dev/null || {
        print_warning "pgadmin-0box failed to start"
        return 0
    }

    # Connect to 0box network (postgres-0box is likely on the 0box compose network)
    docker network connect testnet0 pgadmin-0box 2>/dev/null || true
    # Try common 0box network names
    for net in 0box_default dockerlocal_0box; do
        docker network connect "$net" pgadmin-0box 2>/dev/null || true
    done

    print_status "pgAdmin for 0box running at http://localhost:5435"
    print_status "  Credentials: zbox_user / zbox_server (database: zbox)"
    print_status "  Auto-login enabled (no password prompt)"
}

# Start pgAdmin for blobber postgres databases (replaces the docker-compose pgadmin4_container)
# Provides web UI at port 5050 for inspecting all blobber databases.
# Nginx routes: /pgadmin/
start_pgadmin_blobbers() {
    print_header "Starting pgAdmin for Blobber Databases"

    # Stop existing blobber pgadmin containers (both our new one and the docker-compose one)
    docker stop pgadmin-blobber 2>/dev/null || true
    docker rm pgadmin-blobber 2>/dev/null || true
    docker stop pgadmin4_container 2>/dev/null || true
    docker rm pgadmin4_container 2>/dev/null || true

    local PGADMIN_DIR="/tmp/pgadmin-blobber"
    mkdir -p "$PGADMIN_DIR"

    # Generate servers.json for all 9 regular blobbers (always, regardless of whether containers are running)
    local servers_json='{\n    "Servers": {\n'
    local idx=1
    for i in $(seq 1 9); do
        local pg_name="postgres-blob-${i}"
        [ $idx -gt 1 ] && servers_json="${servers_json},\n"
        servers_json="${servers_json}        \"${idx}\": {\n"
        servers_json="${servers_json}            \"Name\": \"Blobber-${i} DB\",\n"
        servers_json="${servers_json}            \"Group\": \"Blobbers\",\n"
        servers_json="${servers_json}            \"Host\": \"${pg_name}\",\n"
        servers_json="${servers_json}            \"Port\": 5432,\n"
        servers_json="${servers_json}            \"MaintenanceDB\": \"blobber_meta\",\n"
        servers_json="${servers_json}            \"Username\": \"blobber_user\",\n"
        servers_json="${servers_json}            \"SSLMode\": \"prefer\",\n"
        servers_json="${servers_json}            \"PassFile\": \"/pgpass\"\n"
        servers_json="${servers_json}        }"
        idx=$((idx + 1))
    done
    servers_json="${servers_json}\n    }\n}"
    echo -e "$servers_json" > "$PGADMIN_DIR/servers.json"

    # Create pgpass file for all 9 blobber postgres containers
    local pgpass=""
    for i in $(seq 1 9); do
        pgpass="${pgpass}postgres-blob-${i}:5432:blobber_meta:blobber_user:blobber\n"
        pgpass="${pgpass}postgres-blob-${i}:5432:*:blobber_user:blobber\n"
    done
    echo -e "$pgpass" > "$PGADMIN_DIR/pgpass"
    chmod 600 "$PGADMIN_DIR/pgpass"
    chown 5050:5050 "$PGADMIN_DIR/pgpass" "$PGADMIN_DIR/servers.json"

    docker run \
        --detach \
        --restart=unless-stopped \
        --name=pgadmin-blobber \
        --publish=5050:80 \
        --network=testnet0 \
        -e PGADMIN_DEFAULT_EMAIL=admin@zus.network \
        -e PGADMIN_DEFAULT_PASSWORD="${PGADMIN_DEFAULT_PASSWORD:-admin}" \
        -e PGADMIN_CONFIG_SERVER_MODE=False \
        -e PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False \
        -e SCRIPT_NAME=/pgadmin \
        -v "$PGADMIN_DIR/servers.json:/pgadmin4/servers.json:ro" \
        -v "$PGADMIN_DIR/pgpass:/pgpass" \
        dpage/pgadmin4:latest 2>/dev/null || {
        print_warning "pgadmin-blobber failed to start"
        return 0
    }

    # Connect to blobber networks so pgadmin can reach all blobber postgres containers
    for net in $(docker network ls --format '{{.Name}}' | grep -E 'blobber[0-9]+_default'); do
        docker network connect "$net" pgadmin-blobber 2>/dev/null || true
    done

    print_status "pgAdmin for blobbers running at http://localhost:5050"
    print_status "  Credentials: blobber_user / blobber (database: blobber_meta)"
    print_status "  Auto-login enabled (no password prompt)"
}

# Start pgAdmin for zauth postgres database
# Provides web UI at port 8081 for inspecting zauth database.
# Nginx routes: /pgadmin-zauth/
start_pgadmin_zauth() {
    print_header "Starting pgAdmin for Zauth Database"

    # Stop existing zauth pgadmin (both our new one and docker-compose one)
    docker stop pgadmin-zauth 2>/dev/null || true
    docker rm pgadmin-zauth 2>/dev/null || true
    # Stop docker-compose pgadmin if running on port 8081
    local old_pgadmin=$(docker ps --format '{{.Names}}' | grep -i 'zauth.*pgadmin' | head -1)
    if [ -n "$old_pgadmin" ]; then
        docker stop "$old_pgadmin" 2>/dev/null || true
        docker rm "$old_pgadmin" 2>/dev/null || true
    fi

    local PGADMIN_DIR="/tmp/pgadmin-zauth"
    mkdir -p "$PGADMIN_DIR"

    cat > "$PGADMIN_DIR/servers.json" << 'PGEOF'
{
    "Servers": {
        "1": {
            "Name": "Zauth DB",
            "Group": "Services",
            "Host": "postgres",
            "Port": 5432,
            "MaintenanceDB": "zauth",
            "Username": "zauth_user",
            "SSLMode": "prefer",
            "PassFile": "/pgpass"
        }
    }
}
PGEOF

    cat > "$PGADMIN_DIR/pgpass" << 'PASSEOF'
postgres:5432:zauth:zauth_user:zauth_pass
postgres:5432:*:zauth_user:zauth_pass
PASSEOF
    chmod 600 "$PGADMIN_DIR/pgpass"
    chown 5050:5050 "$PGADMIN_DIR/pgpass" "$PGADMIN_DIR/servers.json"

    docker run \
        --detach \
        --restart=unless-stopped \
        --name=pgadmin-zauth \
        --publish=8081:80 \
        -e PGADMIN_DEFAULT_EMAIL=admin@zus.network \
        -e PGADMIN_DEFAULT_PASSWORD="${PGADMIN_DEFAULT_PASSWORD:-admin}" \
        -e PGADMIN_CONFIG_SERVER_MODE=False \
        -e PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False \
        -e SCRIPT_NAME=/pgadmin-zauth \
        -v "$PGADMIN_DIR/servers.json:/pgadmin4/servers.json:ro" \
        -v "$PGADMIN_DIR/pgpass:/pgpass" \
        dpage/pgadmin4:latest 2>/dev/null || {
        print_warning "pgadmin-zauth failed to start"
        return 0
    }

    # Connect to zauth's docker-compose network so pgadmin can reach its postgres
    docker network connect zauth_default pgadmin-zauth 2>/dev/null || true
    for net in $(docker network ls --format '{{.Name}}' | grep -i 'zauth'); do
        docker network connect "$net" pgadmin-zauth 2>/dev/null || true
    done

    print_status "pgAdmin for zauth running at http://localhost:8081"
    print_status "  Credentials: zauth_user / zauth_pass (database: zauth)"
    print_status "  Auto-login enabled (no password prompt)"
}

# Start pgAdmin for zvault postgres database
# Provides web UI at port 8083 for inspecting zvault database.
# Nginx routes: /pgadmin-zvault/
start_pgadmin_zvault() {
    print_header "Starting pgAdmin for Zvault Database"

    # Stop existing zvault pgadmin (both our new one and docker-compose one)
    docker stop pgadmin-zvault 2>/dev/null || true
    docker rm pgadmin-zvault 2>/dev/null || true
    local old_pgadmin=$(docker ps --format '{{.Names}}' | grep -i 'zvault.*pgadmin' | head -1)
    if [ -n "$old_pgadmin" ]; then
        docker stop "$old_pgadmin" 2>/dev/null || true
        docker rm "$old_pgadmin" 2>/dev/null || true
    fi

    local PGADMIN_DIR="/tmp/pgadmin-zvault"
    mkdir -p "$PGADMIN_DIR"

    cat > "$PGADMIN_DIR/servers.json" << 'PGEOF'
{
    "Servers": {
        "1": {
            "Name": "Zvault DB",
            "Group": "Services",
            "Host": "postgreszv",
            "Port": 5432,
            "MaintenanceDB": "zvault",
            "Username": "zvault_user",
            "SSLMode": "prefer",
            "PassFile": "/pgpass"
        }
    }
}
PGEOF

    cat > "$PGADMIN_DIR/pgpass" << 'PASSEOF'
postgreszv:5432:zvault:zvault_user:zvault_pass
postgreszv:5432:*:zvault_user:zvault_pass
PASSEOF
    chmod 600 "$PGADMIN_DIR/pgpass"
    chown 5050:5050 "$PGADMIN_DIR/pgpass" "$PGADMIN_DIR/servers.json"

    docker run \
        --detach \
        --restart=unless-stopped \
        --name=pgadmin-zvault \
        --publish=8083:80 \
        -e PGADMIN_DEFAULT_EMAIL=admin@zus.network \
        -e PGADMIN_DEFAULT_PASSWORD="${PGADMIN_DEFAULT_PASSWORD:-admin}" \
        -e PGADMIN_CONFIG_SERVER_MODE=False \
        -e PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False \
        -e SCRIPT_NAME=/pgadmin-zvault \
        -v "$PGADMIN_DIR/servers.json:/pgadmin4/servers.json:ro" \
        -v "$PGADMIN_DIR/pgpass:/pgpass" \
        dpage/pgadmin4:latest 2>/dev/null || {
        print_warning "pgadmin-zvault failed to start"
        return 0
    }

    # Connect to zvault's docker-compose network so pgadmin can reach postgreszv
    docker network connect zvault_default pgadmin-zvault 2>/dev/null || true
    for net in $(docker network ls --format '{{.Name}}' | grep -i 'zvault'); do
        docker network connect "$net" pgadmin-zvault 2>/dev/null || true
    done

    print_status "pgAdmin for zvault running at http://localhost:8083"
    print_status "  Credentials: zvault_user / zvault_pass (database: zvault)"
    print_status "  Auto-login enabled (no password prompt)"
}

# Stop cAdvisor
stop_cadvisor() {
    print_status "Stopping cAdvisor..."
    docker stop cadvisor 2>/dev/null || true
    docker rm cadvisor 2>/dev/null || true
    print_status "cAdvisor stopped"
}

# Get miner and sharder IDs for vc.sh (the ones to add/remove during view change testing)
# Queries the chain directly for actual on-chain node IDs (key files may not match).
# Uses the last miner and last sharder (keeping others for consensus).
# Start View Change loop test (vc.sh) in background
start_vc_test() {
    print_header "Starting View Change Loop Test"

    local VC_SCRIPT="${BASE_DIR}/0chain/docker.local/bin/vc.sh"
    if [ ! -f "$VC_SCRIPT" ]; then
        print_error "vc.sh not found at $VC_SCRIPT"
        return 1
    fi

    # Find zwallet binary - prefer built zwalletcli, fall back to system_test
    local ZWALLET_DIR="${BASE_DIR}/zwalletcli"
    if [ ! -f "${ZWALLET_DIR}/zwallet" ]; then
        ZWALLET_DIR="${BASE_DIR}/system_test/tests/cli_tests"
    fi
    if [ ! -f "${ZWALLET_DIR}/zwallet" ]; then
        print_error "zwallet binary not found in ${BASE_DIR}/zwalletcli or ${BASE_DIR}/system_test/tests/cli_tests"
        return 1
    fi
    print_status "Using zwallet from: $ZWALLET_DIR"

    # Create adapted copy of vc.sh with correct paths
    local VC_ADAPTED="/tmp/vc_adapted.sh"
    cp "$VC_SCRIPT" "$VC_ADAPTED"

    # Patch zwallet path and wallet/config (miner/sharder IDs are randomly selected per iteration by vc.sh itself)
    sed -i "s|^ZWALLET_DIR=.*|ZWALLET_DIR=\"${ZWALLET_DIR}\"|" "$VC_ADAPTED"
    sed -i "s|^ZWALLET_PATH=.*|ZWALLET_PATH=\"\$ZWALLET_DIR/zwallet\"|" "$VC_ADAPTED"
    sed -i "s|^WALLET=.*|WALLET=\"local.json\"|" "$VC_ADAPTED"
    sed -i "s|^CONFIG=.*|CONFIG=\"local.yaml\"|" "$VC_ADAPTED"
    # Wallet file is in ~/.zcn, not in ZWALLET_DIR
    sed -i 's|cat "\$ZWALLET_DIR/\$WALLET"|cat "$HOME/.zcn/$WALLET"|g' "$VC_ADAPTED"
    chmod +x "$VC_ADAPTED"

    # Kill any existing vc test
    pkill -f "vc_adapted.sh" 2>/dev/null || true
    sleep 1

    # Clear old log and launch in background
    local LOG_FILE="/tmp/vc.log"
    echo "" > "$LOG_FILE"
    nohup bash "$VC_ADAPTED" > "$LOG_FILE" 2>&1 &
    local PID=$!
    echo "$PID" > /tmp/vc_test.pid

    print_status "View Change test started (PID: $PID)"
    print_status "Log file: $LOG_FILE"
    print_status "Stop with: kill $PID (or: $0 stop-vc)"
}

# Stop View Change test
stop_vc_test() {
    print_status "Stopping View Change test..."
    if [ -f /tmp/vc_test.pid ]; then
        local PID=$(cat /tmp/vc_test.pid)
        kill "$PID" 2>/dev/null || true
        rm -f /tmp/vc_test.pid
        print_status "View Change test stopped (PID: $PID)"
    else
        pkill -f "vc_adapted.sh" 2>/dev/null || true
        print_status "View Change test stopped"
    fi
}

# Get active sharders for chaos.sh
get_active_sharders() {
    local sharders=()
    for i in 1 2; do
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^sharder-${i}$"; then
            sharders+=("sharder-${i}")
        fi
    done
    echo "${sharders[@]}"
}

# Start Chaos Light test (chaos_light.sh) in background
start_chaos_test() {
    print_header "Starting Chaos Light Test"

    local CHAOS_SCRIPT="${BASE_DIR}/0chain/docker.local/bin/chaos_light.sh"
    if [ ! -f "$CHAOS_SCRIPT" ]; then
        print_error "chaos_light.sh not found at $CHAOS_SCRIPT"
        return 1
    fi

    # Get active sharders
    local active_sharders=$(get_active_sharders)
    local sharder_count=$(echo "$active_sharders" | wc -w)

    # Create adapted copy
    local CHAOS_ADAPTED="/tmp/chaos_adapted.sh"
    cp "$CHAOS_SCRIPT" "$CHAOS_ADAPTED"

    # Patch SHARDERS array to match active sharders
    if [ "$sharder_count" -eq 1 ]; then
        # Only one sharder - update array and set MIN_SHARDERS_RUNNING=1
        sed -i 's|^SHARDERS=.*|SHARDERS=("'"$(echo $active_sharders)"'")|' "$CHAOS_ADAPTED"
        sed -i 's|^MIN_SHARDERS_RUNNING=.*|MIN_SHARDERS_RUNNING=1|' "$CHAOS_ADAPTED"
    fi

    # Force ANSI colors even when output is redirected to a file (nohup)
    # The original script disables colors when [ -t 1 ] is false.
    # Replace the terminal check so colors are always enabled (for HTML rendering).
    sed -i 's|if \[ -t 1 \]; then|if true; then|' "$CHAOS_ADAPTED"

    chmod +x "$CHAOS_ADAPTED"

    # Kill any existing chaos test
    pkill -f "chaos_adapted.sh" 2>/dev/null || true
    sleep 1

    # Clear old log and launch in background
    local LOG_FILE="/tmp/chaos.log"
    echo "" > "$LOG_FILE"
    nohup bash "$CHAOS_ADAPTED" > "$LOG_FILE" 2>&1 &
    local PID=$!
    echo "$PID" > /tmp/chaos_test.pid

    print_status "Chaos light test started (PID: $PID)"
    print_status "Active sharders: $active_sharders"
    print_status "Log file: $LOG_FILE"
    print_status "Stop with: kill $PID (or: $0 stop-chaos)"
}

# Stop Chaos test
stop_chaos_test() {
    print_status "Stopping Chaos test..."
    if [ -f /tmp/chaos_test.pid ]; then
        local PID=$(cat /tmp/chaos_test.pid)
        kill "$PID" 2>/dev/null || true
        rm -f /tmp/chaos_test.pid
        print_status "Chaos test stopped (PID: $PID)"
    else
        pkill -f "chaos_adapted.sh" 2>/dev/null || true
        print_status "Chaos test stopped"
    fi
}

# Start DKG monitor (monitor.sh) in background
start_dkg_monitor() {
    print_header "Starting DKG Monitor"

    local MONITOR_SCRIPT="${BASE_DIR}/0chain/docker.local/bin/monitor.sh"
    if [ ! -f "$MONITOR_SCRIPT" ]; then
        print_error "monitor.sh not found at $MONITOR_SCRIPT"
        return 1
    fi

    # Kill any existing monitor
    pkill -f "monitor.sh" 2>/dev/null || true
    sleep 1

    # Clear old log and launch in background (60s interval for frequent updates)
    local LOG_FILE="/tmp/monitor.log"
    echo "" > "$LOG_FILE"
    nohup bash "$MONITOR_SCRIPT" 60 > "$LOG_FILE" 2>&1 &
    local PID=$!
    echo "$PID" > /tmp/dkg_monitor.pid

    print_status "DKG monitor started (PID: $PID, interval: 60s)"
    print_status "Log file: $LOG_FILE"
}

# Stop DKG monitor
stop_dkg_monitor() {
    print_status "Stopping DKG monitor..."
    if [ -f /tmp/dkg_monitor.pid ]; then
        local PID=$(cat /tmp/dkg_monitor.pid)
        kill "$PID" 2>/dev/null || true
        rm -f /tmp/dkg_monitor.pid
        print_status "DKG monitor stopped (PID: $PID)"
    else
        pkill -f "monitor.sh" 2>/dev/null || true
        print_status "DKG monitor stopped"
    fi
}

# Start all monitoring and testing tools
start_monitoring() {
    print_header "Starting Monitoring & Resilience Testing"

    start_cadvisor
    echo ""

    # VC and Chaos tests are NOT started automatically — they destabilize DKG and cause
    # 0-sharder magic blocks. Run manually only: bash scripts/deploy_local.sh start-vc
    # start_vc_test
    # echo ""

    # start_chaos_test
    # echo ""

    start_dkg_monitor
    echo ""

    print_header "All Monitoring Started"
    echo "Services:"
    echo "  - cAdvisor:     http://localhost:8085 (container metrics)"
    echo "  - DKG Monitor:  /tmp/monitor.log (chain stats & DKG status)"
    echo "  NOTE: VC/Chaos tests disabled (cause DKG deadlock). Use start-vc/start-chaos commands manually."
    echo ""
    echo "Stop all: $0 stop-monitor"
}

# Stop all monitoring and testing tools
stop_monitoring() {
    print_header "Stopping All Monitoring & Testing"
    stop_cadvisor
    stop_vc_test
    stop_chaos_test
    stop_dkg_monitor
    print_status "All monitoring stopped"
}

# Show monitoring status
monitor_status() {
    print_header "Monitoring Status"

    # cAdvisor
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^cadvisor$"; then
        echo -e "  ${GREEN}[RUNNING]${NC} cAdvisor (http://localhost:8085)"
    else
        echo -e "  ${RED}[STOPPED]${NC} cAdvisor"
    fi

    # VC Test
    if [ -f /tmp/vc_test.pid ] && kill -0 "$(cat /tmp/vc_test.pid)" 2>/dev/null; then
        local vc_pid=$(cat /tmp/vc_test.pid)
        local vc_last=$(tail -1 /tmp/vc.log 2>/dev/null || echo "no output")
        echo -e "  ${GREEN}[RUNNING]${NC} View Change Test (PID: $vc_pid)"
        echo -e "           Last: $vc_last"
    else
        echo -e "  ${RED}[STOPPED]${NC} View Change Test"
    fi

    # Chaos Test
    if [ -f /tmp/chaos_test.pid ] && kill -0 "$(cat /tmp/chaos_test.pid)" 2>/dev/null; then
        local chaos_pid=$(cat /tmp/chaos_test.pid)
        local chaos_last=$(tail -1 /tmp/chaos.log 2>/dev/null || echo "no output")
        echo -e "  ${GREEN}[RUNNING]${NC} Chaos Light Test (PID: $chaos_pid)"
        echo -e "           Last: $chaos_last"
    else
        echo -e "  ${RED}[STOPPED]${NC} Chaos Light Test"
    fi

    # DKG Monitor
    if [ -f /tmp/dkg_monitor.pid ] && kill -0 "$(cat /tmp/dkg_monitor.pid)" 2>/dev/null; then
        local mon_pid=$(cat /tmp/dkg_monitor.pid)
        echo -e "  ${GREEN}[RUNNING]${NC} DKG Monitor (PID: $mon_pid)"
    else
        echo -e "  ${RED}[STOPPED]${NC} DKG Monitor"
    fi

    echo ""
}

# Display usage
usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  remote-deploy    Deploy to a remote server from your LOCAL machine (rsync + bootstrap + deploy)"
    echo "                   Example: $0 remote-deploy --server 1.2.3.4 --pass PWD --domain test2.zus.network redeploy"
    echo "                   Flags:   --server IP  --pass PASS  --user USER  --domain DOMAIN  --secrets /path/.secrets.env"
    echo "  all              Full deployment (default, runs ON the server)"
    echo "  redeploy         Clean all data then full deployment (clean + all)"
    echo "  bootstrap        Fresh server setup: install Docker, Go, tools, clone all repos"
    echo "  clone-repos      Clone/update all repositories without other setup"
    echo "  clean            Clean ALL service data (preserves configs/keys/dirs)"
    echo "  checkout         Checkout repo branches from deploy_config.yaml"
    echo "  loopback         Setup loopback aliases only"
    echo "  config           Setup ZCN config only"
    echo "  clean-chain      Clean all chain data (rocksdb, postgres, redis, logs)"
    echo "  start-chain      Clean data + start miners, sharders, 0dns"
    echo "  chain            Initialize chain configuration only (hardforks, SC config)"
    echo "  services         Start supporting services only (0box, zauth, zvault, etc.)"
    echo "  blobbers         Start blobbers, fund, stake, and configure"
    echo "  fix-validators   Fix validator config (block_worker, delegate_wallet, service_charge)"
    echo "  test-setup       Setup test wallets, fund them, copy CLI tools, generate challenge files"
  echo "  challenge-files  Generate challenge_allocations.txt + challenge_blobbers.txt for TestProtocolChallenge"
    echo "  swap-image REPO [BRANCH]  Rebuild image and restart containers (no full redeploy)"
    echo "  test             Run system tests (API, CLI, tokenomics) with auto-retry (~1h, long tests skipped)"
    echo "  test --long-tests  Run ONLY long-running tests (graph, challenge rewards, etc.)"
    echo "  reset-nonces     Reset all wallet nonces to 0"
    echo "  nginx            Setup nginx reverse proxy with SSL (requires domain in config)"
    echo "  verify           Verify chain health + cleanup stale blobbers + verify services"
    echo "  verify-all       Comprehensive verification: chain, services, CORS, Vult, Atlus, blobbers"
    echo "                   Options: --tests [suites...] --fix --section <name>"
    echo "  ensure-config    Ensure all SC configs are correct (idempotent, safe to re-run)"
    echo "  fix-split-dkg    Fix split-DKG: disable view_change + restart all miners simultaneously"
    echo "  kafka-config     Configure Kafka settings in sharder and 0box YAML configs"
    echo "  finalize-alloc   Fund blobbers + restart to trigger FinalizeWorker (cleans expired allocations)"
  echo "  cleanup-blobbers Kill stale/unreachable blobbers from previous deployments"
    echo "  cleanup-tests    Remove stale build artifacts from test directories"
    echo "  fund             Check and top up all provider balances (one-shot)"
    echo "  fund-daemon      Start background funding daemon (checks every 5 min)"
    echo "  fund-daemon-stop Stop background funding daemon"
    echo "  results-daemon   Start background results page regeneration (every 10 min)"
    echo "  results-daemon-stop Stop background results daemon"
    echo "  monitor          Start all monitoring (cAdvisor + VC test + chaos test)"
    echo "  stop-monitor     Stop all monitoring and testing"
    echo "  monitor-status   Show monitoring status"
    echo "  start-vc         Start view change loop test"
    echo "  stop-vc          Stop view change loop test"
    echo "  start-chaos      Start chaos light resilience test"
    echo "  stop-chaos       Stop chaos light test"
    echo "  start-dkg-monitor  Start DKG monitoring (monitor.sh)"
    echo "  stop-dkg-monitor   Stop DKG monitoring"
    echo "  web-apps         Build web-apps (Vult, Bolt, Blimp, etc.)"
    echo "  rclone-zus       Build rclone with Zus backend"
    echo "  zs3-tools        Setup mc/warp/hosts.yaml for zs3 tests (also renews allocation if expired)"
    echo "  renew-zs3        Check and renew ZS3 server allocation if expired, restart zs3server"
    echo "  crawler          Build, start crawler and refresh its allocation"
    echo "  zs3server        Build and start zs3server (S3 gateway)"
    echo "  start-cadvisor   Start cAdvisor container monitoring"
    echo "  stop-cadvisor    Stop cAdvisor"
    echo "  pgadmin-blobber  Start pgAdmin for blobber databases"
    echo "  pgadmin-sharder  Start pgAdmin for sharder event databases"
    echo "  pgadmin-0box     Start pgAdmin for 0box database"
    echo "  pgadmin-zauth    Start pgAdmin for zauth database"
    echo "  pgadmin-zvault   Start pgAdmin for zvault database"
    echo "  smoke            Run infrastructure smoke tests (quick verification)"
    echo "  help             Show this message"
    echo ""
    echo "Options:"
    echo "  --branch REPO=BRANCH   Override branch for a repo (can be repeated)"
    echo "  --domain DOMAIN        Set nginx domain (e.g. test2.zus.network); also sets app domain prefix"
    echo "  --server IP            Remote server IP (for remote-deploy)"
    echo "  --pass PASSWORD        SSH password (for remote-deploy)"
    echo "  --user USERNAME        SSH username (default: root, for remote-deploy)"
    echo "  --secrets PATH         Path to .secrets.env (default: scripts/.secrets.env)"
    echo ""
    echo "Configuration:"
    echo "  Config:  ${CONFIG_FILE}"
    echo "  Wallet:  ${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    echo "  Secrets: scripts/.secrets.env (GITHUB_TOKEN, Firebase, Kafka, etc.)"
    echo ""
    echo "Examples:"
    echo "  # FROM LOCAL MACHINE — deploy to a new server:"
    echo "  $0 remote-deploy --server 1.2.3.4 --pass 'mypass' --domain test2.zus.network redeploy"
    echo "  $0 remote-deploy --server 1.2.3.4 --pass 'mypass' --domain test2.zus.network --secrets ~/my.secrets.env redeploy"
    echo ""
    echo "  # ON THE SERVER — run directly:"
    echo "  $0                                                  # Full deployment using config branches"
    echo "  $0 bootstrap --domain test1.zus.network            # Fresh server: install everything + deploy"
    echo "  $0 --domain test1.zus.network redeploy             # Redeploy with custom domain"
    echo "  $0 checkout                                        # Just checkout branches"
    echo "  $0 --branch 0chain=fix/my-branch all               # Deploy with custom 0chain branch"
    echo "  $0 --branch 0chain=staging --branch blobber=staging all"
    echo "  $0 swap-image 0chain fix/my-branch                 # Swap chain image to a branch"
    echo "  $0 swap-image blobber                              # Swap blobber image (uses config branch)"
    echo "  $0 test                                            # Run all system tests with retry"
    echo "  $0 test --retries 3                                # Run tests with 3 retries"
}

# Clear all monitoring logs, test results, and service data for fresh deploy
clear_all_logs() {
    print_header "Clearing All Monitoring Logs & Test Data"

    # Clear monitoring logs
    for logfile in /tmp/vc.log /tmp/chaos.log /tmp/auto_fund_daemon.log /tmp/funding_daemon.log /tmp/monitor.log \
                   /tmp/deploy_local.log /tmp/smoke_test.log \
                   /tmp/test_sdk.log /tmp/test_api.log /tmp/test_cli.log \
                   /tmp/test_tokenomics.log /tmp/test_zs3.log /tmp/test_mc.log \
                   /tmp/test_rclone.log /tmp/test_cypress.log; do
        echo "" > "$logfile" 2>/dev/null || true
    done

    # Clear HTML rendered logs
    if [ -d /var/log/0chain ]; then
        for htmlfile in /var/log/0chain/*.html /var/log/0chain/*.log /var/log/0chain/*.txt; do
            [ -f "$htmlfile" ] && echo "" > "$htmlfile" 2>/dev/null || true
        done
    fi

    # Stop funding daemon (uses PID file)
    if [ -f /tmp/funding_daemon.pid ]; then
        local fpid=$(cat /tmp/funding_daemon.pid)
        kill "$fpid" 2>/dev/null || true
        rm -f /tmp/funding_daemon.pid
    fi
    pkill -f "funding_daemon" 2>/dev/null || true

    # Kill existing monitoring processes
    pkill -f "vc_adapted.sh" 2>/dev/null || true
    pkill -f "chaos_adapted.sh" 2>/dev/null || true
    pkill -f "monitor.sh" 2>/dev/null || true

    # Clean up PID files
    rm -f /tmp/dkg_monitor.pid /tmp/funding_daemon.pid 2>/dev/null || true

    print_status "All scripts stopped and logs cleared"
}

# test_swap_image — Swap each critical service to its configured branch, then verify health.
# This tests the swap-image pipeline end-to-end: checkout, build, restart, health check.
# Services are swapped sequentially. On failure, log a warning and continue (non-blocking).
test_swap_image() {
    print_header "Swap-Image Tests"
    local SWAP_START
    SWAP_START=$(date +%s)
    local swap_pass=0
    local swap_fail=0
    local swap_total=0

    # Health check helper: curl URL up to N times with 5s interval
    _swap_health_check() {
        local label="$1" url="$2" max_attempts="${3:-12}" expect="${4:-200}"
        for i in $(seq 1 "$max_attempts"); do
            local code
            code=$(curl -sk -o /dev/null -w '%{http_code}' "$url" 2>/dev/null) || code="000"
            if [ "$code" = "$expect" ]; then
                return 0
            fi
            sleep 5
        done
        return 1
    }

    # Docker container health check helper
    _swap_container_check() {
        local name="$1" max_attempts="${2:-12}"
        for i in $(seq 1 "$max_attempts"); do
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -qw "$name"; then
                local status
                status=$(docker inspect -f '{{.State.Status}}' "$name" 2>/dev/null)
                if [ "$status" = "running" ]; then
                    return 0
                fi
            fi
            sleep 5
        done
        return 1
    }

    # Test a single swap-image operation
    _test_swap() {
        local service="$1"
        local check_type="$2"  # "url" or "container"
        local check_target="$3"
        local expect="${4:-200}"
        local max_wait="${5:-12}"

        swap_total=$((swap_total + 1))
        print_status "Testing swap-image: ${service}..."
        local t_start
        t_start=$(date +%s)

        # Perform the swap (use configured branch)
        if ! swap_image "$service" 2>&1 | tail -20; then
            print_error "  FAIL: swap_image ${service} returned error"
            swap_fail=$((swap_fail + 1))
            return 1
        fi

        # Verify health after swap
        local ok=false
        if [ "$check_type" = "url" ]; then
            if _swap_health_check "$service" "$check_target" "$max_wait" "$expect"; then
                ok=true
            fi
        elif [ "$check_type" = "container" ]; then
            if _swap_container_check "$check_target" "$max_wait"; then
                ok=true
            fi
        fi

        local t_end
        t_end=$(date +%s)
        local elapsed=$((t_end - t_start))

        if $ok; then
            print_status "  PASS: ${service} swap OK (${elapsed}s)"
            swap_pass=$((swap_pass + 1))
        else
            print_error "  FAIL: ${service} not healthy after swap (${elapsed}s)"
            swap_fail=$((swap_fail + 1))
        fi
    }

    # Test each critical service. Order: services first, then chain/blobbers last (most disruptive).
    # zauth, zvault, 0box — lightweight Docker services
    _test_swap "zauth-server"  "url"       "http://localhost:8080/healthz"         200 12
    _test_swap "zvault"        "container" "zvault"                                200 12
    _test_swap "0box"          "url"       "http://localhost:9081/v2/health"       200 18

    # 0dns — chain DNS service
    _test_swap "0dns"          "url"       "http://198.18.0.100:9091/dns/network"  200 12

    # blobber — rebuilds all 12 blobbers + validators (most time-consuming)
    _test_swap "blobber"       "container" "blobber-1"                             200 24

    # Wait for blobbers to re-register after blobber swap
    print_status "Waiting for blobbers to re-register after swap..."
    wait_for_blobbers_ready 4 300 || print_warning "Blobbers slow to re-register after swap"

    # Enterprise blobber
    _test_swap "eblobber"      "container" "eblobber-1"                            200 24

    # CLI tools (no container — just binary rebuild)
    swap_total=$((swap_total + 1))
    print_status "Testing swap-image: zboxcli..."
    if swap_image "zboxcli" 2>&1 | tail -5; then
        if [ -f "${BASE_DIR}/system_test/tests/cli_tests/zbox" ]; then
            print_status "  PASS: zboxcli binary rebuilt"
            swap_pass=$((swap_pass + 1))
        else
            print_error "  FAIL: zboxcli binary not found after swap"
            swap_fail=$((swap_fail + 1))
        fi
    else
        print_error "  FAIL: zboxcli swap returned error"
        swap_fail=$((swap_fail + 1))
    fi

    swap_total=$((swap_total + 1))
    print_status "Testing swap-image: zwalletcli..."
    if swap_image "zwalletcli" 2>&1 | tail -5; then
        if [ -f "${BASE_DIR}/system_test/tests/cli_tests/zwallet" ]; then
            print_status "  PASS: zwalletcli binary rebuilt"
            swap_pass=$((swap_pass + 1))
        else
            print_error "  FAIL: zwalletcli binary not found after swap"
            swap_fail=$((swap_fail + 1))
        fi
    else
        print_error "  FAIL: zwalletcli swap returned error"
        swap_fail=$((swap_fail + 1))
    fi

    local SWAP_END
    SWAP_END=$(date +%s)
    local SWAP_ELAPSED=$((SWAP_END - SWAP_START))

    print_header "Swap-Image Test Results"
    echo "  Passed: ${swap_pass}/${swap_total}"
    echo "  Failed: ${swap_fail}/${swap_total}"
    echo "  Time:   ${SWAP_ELAPSED}s"
    echo ""

    if [ "$swap_fail" -gt 0 ]; then
        print_warning "${swap_fail} swap-image test(s) failed — check logs above"
    else
        print_status "All swap-image tests PASSED!"
    fi
}

# Main deployment
main() {
    print_header "0Chain Complete Local Deployment"

    check_prerequisites "${1:-}"

    # Fix git safe.directory for all repos (prevents 'dubious ownership' errors in Docker builds).
    # Must run early — before any git or Docker build operations.
    for _sd in 0chain blobber 0box zauth-server zvault gosdk eblobber zs3server crawler web-apps; do
        [ -d "${BASE_DIR}/${_sd}/.git" ] && git config --global --add safe.directory "${BASE_DIR}/${_sd}" 2>/dev/null || true
    done

    # Phase 0: Clear old logs and monitoring data for fresh start
    clear_all_logs

    # Phase 1: Checkout branches from config (with any --branch overrides)
    checkout_branches

    # Phase 2: Kafka + supporting services — start before chain.
    # Kafka retains all messages from offset 0, so 0box can start after the chain
    # and still consume all events. 0dns starts with the chain (Phase 3), and
    # 0box requires 0dns to initialise — so 0box starts in Phase 3b.
    start_kafka
    configure_kafka_in_configs
    start_elasticsearch
    start_zauth
    start_zvault
    start_gotenberg

    # Phase 2b: Verify Kafka broker is ready BEFORE chain starts.
    if [ -f "${SCRIPT_DIR}/verify_kafka_pipeline.sh" ]; then
        bash "${SCRIPT_DIR}/verify_kafka_pipeline.sh" pre-chain \
            || { print_error "Kafka pre-chain verification failed — aborting deployment"; return 1; }
    else
        print_warning "verify_kafka_pipeline.sh not found — skipping pre-chain verification"
    fi

    # Phase 3: Chain — start miners, sharders, 0dns; configure hardforks + SC config
    setup_zcn_config
    reset_wallet_nonces
    start_chain
    if ! wait_for_chain; then
        print_error "Chain failed to start — cannot continue deployment"
        return 1
    fi
    init_chain_config

    # Phase 3b: Start 0box AFTER chain + 0dns are up (0box panics if 0dns unreachable on init).
    # Kafka retains all events from block 0 so 0box will consume the full history.
    start_0box
    fund_0box
    register_0box_render_user

    # Phase 3c: Verify events are flowing from chain → Kafka → 0box BEFORE setting up blobbers.
    # Wait 15s for 0box to start consuming Kafka events before checking.
    if [ -f "${SCRIPT_DIR}/verify_kafka_pipeline.sh" ]; then
        sleep 15
        bash "${SCRIPT_DIR}/verify_kafka_pipeline.sh" post-chain \
            || print_warning "Kafka post-chain verification failed — 0box may not have chain data (check logs)"
    fi

    # Phase 4a: Networking — loopback (macOS) or nginx (Linux) for path routing
    # Done after chain so 0dns is available; containers already communicate via Docker network
    setup_loopback
    setup_nginx || print_warning "Nginx setup had issues (non-critical, services accessible via Docker IPs)"

    # Phase 4: Funding daemon — install cron+watchdog (survives SSH disconnects)
    # Kill ALL stale daemon instances first to prevent accumulation across deploys.
    pkill -f 'auto_fund_daemon' 2>/dev/null && print_status "Killed stale auto_fund_daemon instances" || true
    sleep 1
    setup_auto_funding || print_warning "Fund daemon setup failed (non-critical, run fund-daemon manually)"
    # Also start immediately (cron won't fire for up to 5 min)
    if [ -f /usr/local/bin/auto_fund_daemon.sh ]; then
        nohup bash /usr/local/bin/auto_fund_daemon.sh 120 5 20 >> /tmp/auto_fund_daemon.log 2>&1 &
        disown
        print_status "Auto-fund daemon started (PID: $!)"
    fi
    setup_block_pruning
    # Phase 5: Monitoring — cAdvisor, pgAdmin
    start_cadvisor || print_warning "cAdvisor failed to start (non-critical)"
    start_pgadmin_blobbers || print_warning "pgAdmin for blobbers failed to start (non-critical)"
    start_pgadmin_sharders || print_warning "pgAdmin for sharders failed to start (non-critical)"
    start_pgadmin_0box || print_warning "pgAdmin for 0box failed to start (non-critical)"
    start_pgadmin_zauth || print_warning "pgAdmin for zauth failed to start (non-critical)"
    start_pgadmin_zvault || print_warning "pgAdmin for zvault failed to start (non-critical)"
    # VC test, chaos light, and DKG monitor will start after blobbers are ready (Phase 8b)

    # Phase 6: Blobbers — fix config (delegate_wallet), build, register, fund, stake
    # CRITICAL: fix_blobber_config and fix_validator_config MUST run BEFORE first blobber start
    # because delegate_wallet is set at registration and cannot be changed after.
    # 0box already running → receives TagAddOrOverwriteBlobber events as blobbers register.
    fix_blobber_config
    fix_validator_config
    build_and_create_blobbers
    ensure_blobber_hdd_tablespace
    # Wait for blobber wallets to appear in logs before funding
    print_status "Waiting 30s for blobber wallets to initialize..."
    sleep 30
    fund_blobbers_and_validators
    # Restart any blobbers that failed to register (insufficient balance at first start)
    restart_failed_blobbers
    # Wait for restarted blobbers to register
    print_status "Waiting 30s for blobbers to register after restart..."
    sleep 30
    stake_and_configure_blobbers

    # NOTE: 0box is populated via Kafka (chain events) — do NOT seed manually.
    # 0box started before the chain (Phase 2) so it receives all events from block 0.

    # Phase 8: Enterprise blobbers, zs3, web-apps, rclone
    configure_0box_public_key_in_blobbers
    build_and_deploy_enterprise_blobbers
    stake_enterprise_blobbers

    # Phase 8a: Ensure all SC configs are correct (idempotent reset)
    ensure_chain_config

    build_and_start_zs3server
    build_web_apps || print_warning "web-apps build failed (non-critical)"
    build_rclone_zus || print_warning "rclone-zus build failed (non-critical)"

    # Phase 8b: VC, chaos, and DKG monitor are disabled — not needed for test runs.
    # Enable manually if needed: bash scripts/deploy_local.sh start-vc / start-chaos / start-dkg-monitor
    # Funding daemon keeps provider wallets topped up during long test runs.
    start_funding_daemon || print_warning "Funding daemon failed to start (non-critical)"

    # Phase 9: Update nginx with full service routes, logs, and configs
    setup_nginx || print_warning "Nginx setup had issues (non-critical)"

    # Phase 9b: Clean stale service data — ONLY for non-clean deploys (e.g. `all` without prior `clean`).
    # On fresh redeploy, chain data is clean and 0box has been receiving events from round 1 since
    # Phase 3b. Truncating now would destroy all accumulated data and break the Kafka pipeline
    # (0box resets lastProcessedRound to 0 but Kafka is at round N>>1, causing permanent skip).
    if [ "${1:-}" != "redeploy" ]; then
        # clean_service_databases truncates blobbers/allocations/write_markers.
        # seed_0box_providers repopulates from sharder events_db.
        clean_service_databases
        seed_0box_providers || print_warning "0box provider seeding had issues (non-critical)"
    else
        print_status "Skipping clean_service_databases (fresh redeploy — 0box has events from round 1)"
    fi

    # Phase 10: Test setup — wallets, configs, funding, tools
    cleanup_stale_test_artifacts
    setup_test_wallets
    configure_test_configs
    fund_test_wallets
    setup_zs3_test_tools || print_warning "zs3 test tools setup failed (non-critical)"

    # Phase 10b: Crawler — build, start, refresh allocation, and start perpetual monitor
    build_and_start_crawler || print_warning "Crawler build/start failed (non-critical)"
    refresh_crawler_allocation || print_warning "Crawler allocation refresh failed (non-critical)"
    start_crawler_monitor || print_warning "Crawler monitor start failed (non-critical)"

    # Phase 11: Verify everything
    verify_services

    # Phase 11b: Wait for blobbers to be ready (health-checking and answering alloc_blobbers)
    # This prevents tests from failing with "not enough blobbers" when blobbers just started.
    stop_chaos_test || true
    wait_for_blobbers_ready 4 600 || print_warning "Blobbers may not be fully ready — tests may fail with 'not enough blobbers'"

    # Phase 11c: Full environment verification
    print_header "Running Full Environment Verification"
    if [ -f "${SCRIPT_DIR}/verify_all.sh" ]; then
        bash "${SCRIPT_DIR}/verify_all.sh" || print_warning "verify_all.sh reported issues — check output above before running tests"
    else
        print_warning "verify_all.sh not found, skipping full verification"
    fi

    # Phase 11d: Swap-image tests — skipped (already tested, takes too long in deploy pipeline)
    # test_swap_image

    # Phase 11e: Smoke tests
    print_header "Running Smoke Tests"
    local SMOKE_LOG="/tmp/smoke_test.log"
    echo "" > "$SMOKE_LOG"
    cd "${BASE_DIR}/system_test"
    if go test -run "TestSmokeInfrastructure|TestSmokeServices" -v ./tests/sdk_tests/ -timeout 5m 2>&1 | tee "$SMOKE_LOG"; then
        print_status "Smoke tests PASSED"
    else
        print_warning "Smoke tests FAILED (chain may need more time to stabilize)"
    fi
    /usr/local/bin/refresh-0chain-logs.sh 2>/dev/null || true

    # Phase 12: Start system tests sequentially to avoid nonce conflicts (chaos stays off)
    if [ -f "${SCRIPT_DIR}/run_tests.sh" ]; then
        print_status "Starting system tests..."
        bash "${SCRIPT_DIR}/run_tests.sh"
    else
        print_warning "run_tests.sh not found, skipping test execution"
    fi

    local DOMAIN_DISPLAY="${NGINX_DOMAIN:-}"
    if [ -z "$DOMAIN_DISPLAY" ] && [ -f "$CONFIG_FILE" ]; then
        DOMAIN_DISPLAY=$(grep "^  domain:" "$CONFIG_FILE" 2>/dev/null | awk -F': ' '{print $2}' | tr -d '"' | xargs)
    fi

    print_header "Deployment Complete!"
    if [ -n "$DOMAIN_DISPLAY" ]; then
        echo ""
        echo "Dashboard: http://${DOMAIN_DISPLAY}/"
        echo ""
        echo "Chain:          /miner01..04/  /sharder01..02/  (each: /log /config)"
        echo "Blobbers:       /blobber01..12/  /eblobber01..05/  (each: /log)"
        echo "Services:       /0dns/  /0box/  /zauth/  /zvault/  /zs3/  (each: /log /config)"
        echo "Admin:          /pgadmin/  /pgadmin-sharder/  /pgadmin-0box/  /pgadmin-zauth/  /pgadmin-zvault/  /cadvisor/"
        echo "Web Apps:       /vult/  /bolt/  /blimp/  /explorer/  /chimney/"
        echo "Monitoring:     /vc  /chaos  /monitor  /deploy"
        echo "Tests:          /test/api  /test/cli  /test/sdk  /test/tokenomics"
        echo "All Logs:       /logs/"
    fi
    echo ""
    echo "Owner wallet: ${ZCN_CONFIG_DIR}/${ZCN_WALLET_FILE}"
    echo "Config file:  ${ZCN_CONFIG_DIR}/${ZCN_CONFIG_FILE}"
}

# Only run command parsing and execution when executed directly (not sourced).
# This allows: source deploy_local.sh && start_zauth  (only loads functions)
# vs:           ./deploy_local.sh all                  (runs full deploy)
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script is being sourced — just load functions, don't execute
    return 0 2>/dev/null || true
fi

# Parse --branch flags first (before command parsing)
parse_branch_overrides "$@"

# Strip known flags to get the command and positional args
COMMAND=""
EXTRA_ARGS=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        --branch|--domain|--server|--pass|--user|--secrets|--chain-image)
            shift 2 ;;  # Skip flag and its value (already parsed by parse_branch_overrides)
        *)
            if [ -z "$COMMAND" ]; then
                COMMAND="$1"
            else
                EXTRA_ARGS+=("$1")
            fi
            shift
            ;;
    esac
done

# Parse command
case "${COMMAND:-all}" in
    all)
        main
        ;;
    remote-deploy)
        remote_deploy
        ;;
    bootstrap)
        # Fresh server setup: install Docker, Go, tools, clone repos
        # Usage: bash deploy_local.sh bootstrap [--domain test.zus.network]
        if [ -n "${EXTRA_ARGS[0]}" ] && [ "${EXTRA_ARGS[0]}" = "--domain" ] && [ -n "${EXTRA_ARGS[1]}" ]; then
            export NGINX_DOMAIN="${EXTRA_ARGS[1]}"
        fi
        bootstrap_server
        ;;
    clone-repos)
        clone_repos
        ;;
    redeploy)
        print_header "Clean Redeploy (clean + all)"
        kill_stale_processes
        if [ -f "${SCRIPT_DIR}/clean.sh" ]; then
            bash "${SCRIPT_DIR}/clean.sh" --force
        else
            print_error "clean.sh not found at ${SCRIPT_DIR}/clean.sh"
            exit 1
        fi
        main "redeploy"
        ;;
    clean)
        if [ -f "${SCRIPT_DIR}/clean.sh" ]; then
            bash "${SCRIPT_DIR}/clean.sh" "${EXTRA_ARGS[@]}"
        else
            print_error "clean.sh not found at ${SCRIPT_DIR}/clean.sh"
            exit 1
        fi
        ;;
    checkout)
        checkout_branches
        ;;
    loopback)
        setup_loopback
        ;;
    config)
        setup_zcn_config
        ;;
    clean-chain)
        clean_chain_data
        ;;
    start-chain)
        start_chain
        ;;
    chain)
        reset_wallet_nonces
        wait_for_chain
        init_chain_config
        ;;
    fix-validators)
        fix_validator_config
        ;;
    vc-add-sharders)
        vc_add_sharders
        ;;
    kafka-config)
        configure_kafka_in_configs
        ;;
    sync-kafka)
        # Simpler Kafka sync: move 0box triggerRound and Kafka consumer offset to current chain round.
        # Use this when 0box is lagging behind the chain — skips the backlog without restarting sharders.
        print_header "Syncing Kafka → 0box to Current Chain Round"

        # Get current chain round — prefer sharder-1 (most up to date)
        current_round=$(curl -s http://198.18.0.81:7171/v1/chain/get/stats -m 5 2>/dev/null \
            | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("current_round", d.get("latest_finalized_round", 0)))' 2>/dev/null)
        # Fallback to sharder-2
        if [ -z "$current_round" ] || [ "$current_round" -eq 0 ]; then
            current_round=$(curl -s http://198.18.0.82:7172/v1/chain/get/stats -m 5 2>/dev/null \
                | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("current_round", d.get("latest_finalized_round", 0)))' 2>/dev/null)
        fi
        current_round=${current_round:-0}
        if [ "$current_round" -eq 0 ]; then
            print_error "Could not get current chain round — aborting"
            exit 1
        fi
        print_status "Current chain round: $current_round"

        # Step 1: Stop 0box (required before resetting Kafka consumer group)
        print_status "Stopping 0box..."
        docker stop 0box 2>/dev/null || docker stop 0box-0box-1 2>/dev/null || true

        # Step 2: Reset Kafka consumer group offset to latest (skip backlog)
        # Write SASL config inside the Kafka container, then run the reset
        print_status "Resetting Kafka consumer group 'events-consumer' to latest..."
        docker exec kafka bash -c '
cat > /tmp/sasl.props << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";
EOF
/opt/kafka/bin/kafka-consumer-groups.sh \
    --bootstrap-server 198.19.0.99:9092 \
    --command-config /tmp/sasl.props \
    --group events-consumer \
    --topic events \
    --reset-offsets --to-latest --execute' 2>/dev/null \
            && print_status "Kafka consumer group reset to latest" \
            || print_warning "Kafka offset reset failed (non-critical — triggerRound will re-seek)"

        # Step 3: Set 0box triggerRound to current chain round in config
        # Update on the host volume path directly (container is stopped at this point)
        print_status "Setting 0box triggerRound to $current_round..."
        local box_cfg="${BASE_DIR}/0box/docker.local/config/0box.yaml"
        if [ -f "$box_cfg" ]; then
            sed -i "s/triggerRound: [0-9]*/triggerRound: ${current_round}/" "$box_cfg" \
                && print_status "Updated 0box.yaml triggerRound to $current_round"
        else
            print_warning "0box config not found at $box_cfg"
        fi

        # Step 4: Insert a sentinel block row so 0box knows it's already at current_round
        # blocks PK is (id, round); unique on (hash, round) — insert a sync-point row
        docker exec postgres-0box psql -U zbox_user -d zbox \
            -c "INSERT INTO blocks (round, hash, finality_duration) VALUES (${current_round}, 'sync-point-${current_round}', 0) ON CONFLICT DO NOTHING;" 2>/dev/null \
            && print_status "Inserted sync-point block at round $current_round" \
            || print_warning "Could not insert blocks sync-point (round may already exist)"

        # Step 5: Update snapshots table max row to current round
        docker exec postgres-0box psql -U zbox_user -d zbox \
            -c "UPDATE snapshots SET round = ${current_round} WHERE round = (SELECT MAX(round) FROM snapshots);" 2>/dev/null \
            && print_status "Updated snapshots to round $current_round" \
            || print_warning "Could not update snapshots"

        # Step 6: Restart 0box
        print_status "Starting 0box..."
        docker start 0box 2>/dev/null || docker start 0box-0box-1 2>/dev/null || true
        sleep 5

        print_status "Kafka sync complete — 0box will now process from round $current_round onwards"
        ;;

    reset-0box)
        reset_0box_data
        ;;

    fund-0box)
        fund_0box
        ;;

    fix-kafka)
        # Full Kafka pipeline fix: start kafka + configure + restart sharders + seed 0box + restart 0box
        print_header "Fixing Kafka Pipeline"

        # Step 1: Start Kafka if not running, then configure YAML files
        if ! docker ps --format '{{.Names}}' | grep -q "^kafka$"; then
            print_status "Kafka container not running — starting it..."
            start_kafka
        else
            print_status "Kafka already running"
        fi
        configure_kafka_in_configs

        # Step 2: Advance is_published marker in sharder postgres to skip stale rounds
        # After sharder restart, the RocksDB ring buffer is empty but getLastPublishedRound()
        # still returns the old round. This causes "could not find events in round X" panic.
        print_status "Advancing is_published marker in sharder databases..."
        for pg_host in sharder-postgres-1 sharder-postgres-2; do
            # Find the docker network that has this container
            pg_container=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -m1 "$pg_host" || true)
            if [ -n "$pg_container" ]; then
                docker exec -e PGPASSWORD=zchian "$pg_container" psql \
                    -U zchain_user -d events_db \
                    -c "UPDATE events SET is_published = true WHERE block_number = (SELECT MAX(block_number) FROM events);" 2>/dev/null \
                    && print_status "Advanced is_published in $pg_host" \
                    || print_error "Failed to update is_published in $pg_host"
            else
                print_error "Container $pg_host not found"
            fi
        done

        # Step 3: Restart sharders to pick up Kafka config
        print_status "Restarting sharders to pick up Kafka config..."
        for i in 1 2; do
            cd "${BASE_DIR}/0chain/docker.local/build.sharder"
            if SHARDER=$i docker compose -p sharder$i -f b0docker-compose.yml up -d --force-recreate 2>&1 | tail -1; then
                print_status "Restarted sharder-$i"
            else
                print_error "Failed to restart sharder-$i"
            fi
        done
        print_status "Waiting 15s for sharders to sync..."
        sleep 15

        # Step 4: Seed 0box snapshots table if empty (prevents 0box deadlock)
        snap_count=$(docker exec -e PGPASSWORD=${ZBOX_DB_PASS:-zbox_server} \
            ${ZBOX_PG_CONTAINER:-postgres-0box} psql \
            -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} -t \
            -c "SELECT count(*) FROM snapshots;" 2>/dev/null | tr -d ' ') || snap_count=0
        if [ "${snap_count:-0}" -eq 0 ]; then
            print_status "Snapshots table empty — seeding with current round..."
            lfr=$(curl -s http://198.18.0.82:7172/v1/chain/get/stats -m 5 2>/dev/null \
                | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("latest_finalized_round", d.get("current_round",0)))' 2>/dev/null)
            lfr=${lfr:-1}
            seed_round=$((lfr - 1))
            epoch=$(date +%s)
            docker exec -e PGPASSWORD=${ZBOX_DB_PASS:-zbox_server} \
                ${ZBOX_PG_CONTAINER:-postgres-0box} psql \
                -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} \
                -c "INSERT INTO snapshots (round, created_at) VALUES ($seed_round, $epoch) ON CONFLICT DO NOTHING;" 2>/dev/null
            print_status "Seeded snapshots at round $seed_round"
        else
            # Update existing snapshot to current round if it's stale
            lfr=$(curl -s http://198.18.0.82:7172/v1/chain/get/stats -m 5 2>/dev/null \
                | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("latest_finalized_round", d.get("current_round",0)))' 2>/dev/null)
            lfr=${lfr:-1}
            seed_round=$((lfr - 1))
            epoch=$(date +%s)
            docker exec -e PGPASSWORD=${ZBOX_DB_PASS:-zbox_server} \
                ${ZBOX_PG_CONTAINER:-postgres-0box} psql \
                -U ${ZBOX_DB_USER:-zbox_user} -d ${ZBOX_DB_NAME:-zbox} \
                -c "UPDATE snapshots SET round = $seed_round, created_at = $epoch WHERE round = (SELECT MAX(round) FROM snapshots);" 2>/dev/null
            print_status "Updated snapshots to round $seed_round"
        fi

        # Step 5: Fix 0box Kafka triggerRound — set to 1 so sharder replays full history from Kafka
        # (after sharder restart, Kafka has all events from round 1; 0box re-processes them)
        docker exec 0box-0box-1 sed -i 's/triggerRound: [0-9]*/triggerRound: 1/' /0box/config/0box.yaml 2>/dev/null || \
        docker exec 0box sed -i 's/triggerRound: [0-9]*/triggerRound: 1/' /0box/config/0box.yaml 2>/dev/null || true

        # Step 6: Restart 0box
        print_status "Restarting 0box..."
        docker restart 0box-0box-1 2>/dev/null || docker restart 0box 2>/dev/null
        sleep 5

        # Step 7: Seed 0box provider tables from sharder events_db (for Explorer)
        print_status "Seeding 0box provider tables from events_db..."
        seed_0box_providers

        print_status "Kafka pipeline fix complete. Verify at /test/results or Explorer."
        ;;
    fix-snapshots)
        # Patch 0box snapshot aggregates with real data from events_db.
        # Fixes unique_addresses=0, total_challenges=0, total_rewards=0 on Explorer.
        fix_snapshot_aggregates
        ;;
    fix-0box-data)
        # Seed 0box historical data from events_db (challenges, provider_rewards, blobber stats).
        # Use when 0box was restarted and missed historical Kafka events.
        seed_0box_historical_data
        ;;
    crawler)
        # Rebuild crawler config (creates new allocation if stale) and restart container.
        build_and_start_crawler
        ;;
    services)
        start_elasticsearch
        start_kafka
        configure_kafka_in_configs
        start_zauth
        start_zvault
        start_gotenberg
        start_0box
        fund_0box
        register_0box_render_user
        ;;
    blobbers)
        reset_wallet_nonces
        fix_blobber_config
        fix_validator_config
        build_and_create_blobbers
        ensure_blobber_hdd_tablespace
        # Wait for blobber wallets to appear in logs before funding
        print_status "Waiting 30s for blobber wallets to initialize..."
        sleep 30
        fund_blobbers_and_validators
        restart_failed_blobbers
        # Wait for restarted blobbers to register
        print_status "Waiting 30s for blobbers to register after restart..."
        sleep 30
        stake_and_configure_blobbers
        build_and_deploy_enterprise_blobbers
        stake_enterprise_blobbers
        refresh_crawler_allocation || print_warning "Crawler allocation refresh failed (non-critical)"
        # Deploy ZS3 server + test tools (mc, warp, rclone) so ZS3/MC tests can run
        build_and_start_zs3server || print_warning "ZS3 server deploy failed (non-critical)"
        setup_zs3_test_tools || print_warning "ZS3 test tools setup failed (non-critical)"
        build_rclone_zus || print_warning "rclone-zus build failed (non-critical)"
        ;;
    update-blobber-prices)
        # Update write_price=0.001, read_price=0 for all on-chain blobbers WITHOUT restarting them.
        # Use when blobbers have incorrect prices after deploy (faster than 'blobbers' command).
        # Uses owner.json (sc_owner wallet) to avoid nonce conflicts with view_change_loop (local.json).
        # NOTE: min_write_price=0.001 ZCN is the chain-enforced minimum in the current 0chain code.
        print_header "Updating Blobber Prices (write=0.001, read=0)"
        # Also fix the blobber config YAML so future restarts use the correct price
        _ubp_blobber_yaml="${BASE_DIR}/blobber/config/0chain_blobber.yaml"
        if [ -f "$_ubp_blobber_yaml" ] && grep -q 'write_price:' "$_ubp_blobber_yaml"; then
            sed -i 's/write_price:.*/write_price: 0.001/' "$_ubp_blobber_yaml"
            sed -i 's/read_price:.*/read_price: 0.00/' "$_ubp_blobber_yaml"
            print_status "Updated blobber config YAML: write_price=0.001, read_price=0"
        fi
        _ubp_W="--wallet $ZCN_SC_OWNER_WALLET --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
        _ubp_blobber_json=$($ZBOX ls-blobbers --json --silent $_ubp_W 2>/dev/null || echo "[]")
        _ubp_blobber_ids=$(echo "$_ubp_blobber_json" | jq -r '.[].id' 2>/dev/null || true)
        _ubp_count=0
        for blobber_id in $_ubp_blobber_ids; do
            _ubp_count=$((_ubp_count + 1))
            print_status "  Updating blobber $_ubp_count: ${blobber_id:0:16}..."
            $ZBOX bl-update \
                --blobber_id "$blobber_id" \
                --write_price 0.001 \
                --read_price 0 \
                --service_charge 0.3 \
                --num_delegates 100 \
                --storage_version 1 \
                $_ubp_W --silent 2>&1 || print_warning "  bl-update failed for ${blobber_id:0:16}"
            sleep 1
        done
        print_status "Updated $_ubp_count blobbers. Verifying..."
        curl -s 'http://198.18.0.82:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers' \
            | python3 -c "import json,sys; d=json.load(sys.stdin); [print(b['url'].split('/')[-2], 'wp:', b['terms']['write_price']) for b in d.get('Nodes',[])]" 2>/dev/null || true
        ;;
    regenerate-blobber-keys)
        # Regenerate BLS keys for blobbers and validators.
        # Use when blobbers are killed on-chain (killed IDs cannot re-register).
        regenerate_blobber_keys --regular-only "${EXTRA_ARGS[@]}"
        ;;
    test-setup)
        cleanup_stale_blobbers || true
        clean_service_databases
        seed_0box_providers || print_warning "0box provider seeding had issues (non-critical)"
        cleanup_stale_test_artifacts
        setup_test_wallets
        configure_test_configs
        fund_test_wallets
        setup_zs3_test_tools || true
        seed_challenge_data || true
        generate_challenge_protocol_files || print_warning "Challenge protocol file generation failed (non-critical)"
        ;;
    challenge-files)
        generate_challenge_protocol_files
        ;;
    swap-image)
        swap_image "${EXTRA_ARGS[@]}"
        ;;
    test)
        # If not already inside tmux, re-launch inside a tmux session so the run
        # survives SSH disconnects. Pass _IN_TMUX=1 to avoid infinite re-launch.
        if [ -z "${TMUX}" ] && [ -z "${_IN_TMUX}" ] && command -v tmux >/dev/null 2>&1; then
            SESSION="tests"
            LOG="/root/test_run_$(date +%Y%m%d_%H%M%S).log"
            print_status "Re-launching inside tmux session '${SESSION}' (log: ${LOG})"
            # Kill any existing 'tests' session so we start clean
            tmux kill-session -t "$SESSION" 2>/dev/null || true
            tmux new-session -d -s "$SESSION" \
                "_IN_TMUX=1 bash $(realpath "$0") ${COMMAND} ${EXTRA_ARGS[*]} 2>&1 | tee ${LOG}"
            echo "Attach with: tmux attach -t ${SESSION}"
            echo "Follow log:  tail -f ${LOG}"
            exit 0
        fi

        # Kill any stale test processes before starting
        stale_pids=$(pgrep -f "go test.*tests/" 2>/dev/null || true)
        if [ -n "$stale_pids" ]; then
            print_status "Killing stale test processes..."
            echo "$stale_pids" | xargs kill -9 2>/dev/null || true
            sleep 2
        fi
        stale_pids=$(pgrep -f "run_tests.sh" 2>/dev/null || true)
        if [ -n "$stale_pids" ]; then
            echo "$stale_pids" | xargs kill -9 2>/dev/null || true
        fi

        # Reset test state: unstake pools and clean stale artifacts before tests
        reset_test_state || true
        cleanup_stale_test_artifacts || true
        fund_test_wallets || true
        seed_challenge_data || true

        # Ensure zs3server is running with a valid (non-expired) allocation before tests
        renew_zs3_allocation || print_warning "zs3server allocation renewal failed (zs3 tests may skip)"
        setup_zs3_test_tools || print_warning "zs3 test tools setup failed (non-critical)"

        # Ensure Gotenberg (render server) is running
        if ! docker ps --format '{{.Names}}' | grep -q "^gotenberg$"; then
            print_status "Gotenberg not running — starting it now..."
            start_gotenberg || print_warning "Gotenberg startup failed (render/PDF tests will skip)"
        fi

        # Kill stale results generator
        if [ -f /tmp/results_gen.pid ]; then
            kill "$(cat /tmp/results_gen.pid)" 2>/dev/null || true
            rm -f /tmp/results_gen.pid
        fi

        # Verify environment health before running tests
        print_header "Environment Verification (pre-test)"
        if [ -f "${SCRIPT_DIR}/verify_all.sh" ]; then
            bash "${SCRIPT_DIR}/verify_all.sh" || print_warning "verify_all.sh reported issues — tests may fail; investigate before proceeding"
        else
            print_warning "verify_all.sh not found, skipping pre-test verification"
        fi

        # Run system tests using run_tests.sh
        if [ -f "${SCRIPT_DIR}/run_tests.sh" ]; then
            bash "${SCRIPT_DIR}/run_tests.sh" "${EXTRA_ARGS[@]}"
        else
            print_error "run_tests.sh not found at ${SCRIPT_DIR}/run_tests.sh"
            exit 1
        fi
        ;;
    rerun-failed)
        # Re-run only tests that failed in the most recent run (no state reset needed).
        # Adds a new runN.txt file per suite; existing results are preserved.
        # Usage: bash deploy_local.sh rerun-failed [api|cli|...] [--retries N]
        if [ -f "${SCRIPT_DIR}/run_tests.sh" ]; then
            bash "${SCRIPT_DIR}/run_tests.sh" --rerun-failed "${EXTRA_ARGS[@]}"
        else
            print_error "run_tests.sh not found"; exit 1
        fi
        ;;
    full-retest)
        # Clean all previous run files and run the full test suite from scratch.
        # Use this for a definitive clean-slate pass/fail result.
        # Usage: bash deploy_local.sh full-retest [api|cli|...] [--retries N]
        stale_pids=$(pgrep -f "go test.*tests/" 2>/dev/null || true)
        [ -n "$stale_pids" ] && echo "$stale_pids" | xargs kill -9 2>/dev/null || true
        reset_test_state || true
        cleanup_stale_test_artifacts || true
        fund_test_wallets || true
        seed_challenge_data || true
        if [ -f "${SCRIPT_DIR}/run_tests.sh" ]; then
            bash "${SCRIPT_DIR}/run_tests.sh" --full-retest "${EXTRA_ARGS[@]}"
        else
            print_error "run_tests.sh not found"; exit 1
        fi
        ;;
    reset-nonces)
        reset_wallet_nonces
        ;;
    nginx)
        setup_nginx
        ;;
    finalize-alloc)
        # Trigger blobbers to finalize expired allocations immediately.
        # Blobbers are the ONLY entities authorized to call FinalizeAllocation on behalf of
        # allocations (aside from the allocation owner). The FinalizeWorker runs at startup
        # and then every finalize_allocations_interval (168h/weekly, or every 10 min if
        # >100 pending). This command funds blobbers (so they have ZCN for finalization tx
        # fees) then restarts each blobber, causing FinalizeWorker to run immediately.
        print_header "Finalizing Expired Allocations"
        print_status "Step 1: Funding blobbers (needed to pay finalization transaction fees)..."
        check_and_fund_providers
        print_status "Step 2: Restarting blobbers to trigger immediate FinalizeWorker run..."
        local blobber_compose="${BASE_DIR}/blobber/docker.local/b0docker-compose.yml"
        for i in $(seq 1 "${NUM_BLOBBERS:-9}"); do
            print_status "  Restarting blobber$i..."
            BLOBBER=$i docker compose -p "blobber${i}" -f "$blobber_compose" \
                restart blobber 2>&1 | tail -1 || true
            sleep 3
        done
        print_status "All blobbers restarted. FinalizeWorker will process expired allocations within ~1-2 minutes."
        print_status "Check logs: docker logs blobber1-blobber-1 2>&1 | grep -i finali"
        ;;
    cleanup-blobbers)
        cleanup_stale_blobbers
        ;;
    cleanup-tests)
        cleanup_stale_test_artifacts
        ;;
    clean-dbs)
        clean_service_databases
        ;;
    refresh-crawler)
        refresh_crawler_allocation
        docker restart crawler 2>/dev/null && print_status "Crawler restarted with new allocation" || true
        ;;
    fund)
        check_and_fund_providers "${EXTRA_ARGS[@]}"
        ;;
    fund-daemon)
        start_funding_daemon "${EXTRA_ARGS[@]}"
        ;;
    fund-daemon-stop)
        stop_funding_daemon
        ;;
    results-daemon)
        start_results_daemon "${EXTRA_ARGS[@]}"
        ;;
    results-daemon-stop)
        stop_results_daemon
        ;;
    verify)
        verify_chain_health || true
        cleanup_stale_blobbers || true
        verify_services
        ;;
    verify-all)
        # Comprehensive verification: chain, services, CORS, Vult, Atlus, blobbers.
        # Optionally run test suites: bash scripts/deploy_local.sh verify-all --tests [suites...]
        # Optionally auto-fix known issues: bash scripts/deploy_local.sh verify-all --fix
        bash "${SCRIPT_DIR}/verify_all.sh" "${EXTRA_ARGS[@]:-}"
        ;;
    monitor)
        start_monitoring
        ;;
    stop-monitor)
        stop_monitoring
        ;;
    monitor-status)
        monitor_status
        ;;
    start-vc)
        start_vc_test
        ;;
    stop-vc)
        stop_vc_test
        ;;
    start-chaos)
        start_chaos_test
        ;;
    stop-chaos)
        stop_chaos_test
        ;;
    start-dkg-monitor)
        start_dkg_monitor
        ;;
    stop-dkg-monitor)
        stop_dkg_monitor
        ;;
    start-cadvisor)
        start_cadvisor
        ;;
    web-apps)
        build_web_apps
        # Run web app verification after build
        _verify_script="${SCRIPT_DIR}/verify_all.sh"
        if [ -f "$_verify_script" ]; then
            print_header "Verifying Web Apps"
            bash "$_verify_script" --section webapps 2>&1 || true
            bash "$_verify_script" --section vult 2>&1 || true
            bash "$_verify_script" --section blimp 2>&1 || true
            bash "$_verify_script" --section bolt 2>&1 || true
            bash "$_verify_script" --section atlus 2>&1 || true
        fi
        ;;
    rclone-zus)
        build_rclone_zus
        ;;
    zs3-tools)
        renew_zs3_allocation
        setup_zs3_test_tools
        ;;
    renew-zs3)
        # Check and renew ZS3 server allocation if expired, then restart zs3server
        renew_zs3_allocation
        ;;
    zs3-server)
        # Build and start the ZS3 server (minio gateway backed by 0chain allocations).
        # Uses enterprise allocation if eblobbers are registered, otherwise regular.
        build_and_start_zs3server
        setup_zs3_test_tools
        ;;
    regen-keys)
        regenerate_blobber_keys "${EXTRA_ARGS[@]}"
        ;;
    reset-test-state)
        reset_test_state
        ;;
    crawler)
        build_and_start_crawler
        refresh_crawler_allocation
        start_crawler_monitor
        ;;
    crawler-monitor)
        # Start (or restart) the perpetual crawler allocation monitor only
        start_crawler_monitor
        ;;
    zs3server)
        build_and_start_zs3server
        ;;
    stop-cadvisor)
        stop_cadvisor
        ;;
    pgadmin-blobber)
        start_pgadmin_blobbers
        ;;
    pgadmin-sharder)
        start_pgadmin_sharders
        ;;
    pgadmin-0box)
        start_pgadmin_0box
        ;;
    pgadmin-zauth)
        start_pgadmin_zauth
        ;;
    pgadmin-zvault)
        start_pgadmin_zvault
        ;;
    clear-logs)
        clear_all_logs
        ;;
    smoke)
        print_header "Running Infrastructure Smoke Tests"
        SMOKE_LOG="/tmp/smoke_test.log"
        echo "" > "$SMOKE_LOG"
        print_status "Output: $SMOKE_LOG"
        print_status "Web: https://${DOMAIN:-localhost}/smoke/html"
        cd "${BASE_DIR}/system_test"
        go test -run "TestSmokeInfrastructure|TestSmokeServices" -v ./tests/sdk_tests/ -timeout 5m 2>&1 | tee "$SMOKE_LOG"
        EXIT_CODE=${PIPESTATUS[0]}
        if [ "$EXIT_CODE" -eq 0 ]; then
            print_status "Smoke tests PASSED"
        else
            print_error "Smoke tests FAILED (exit code $EXIT_CODE)"
        fi
        # Trigger HTML conversion immediately
        /usr/local/bin/refresh-0chain-logs.sh 2>/dev/null || true
        ;;
    fix-blobbers)
        # Apply updated blobber config (service_charge, write_price, etc.) and
        # rebuild the blobber binary using local gosdk (with nonce fix).
        # Also restarts enterprise blobbers to apply config changes.
        # Usage: bash scripts/deploy_local.sh fix-blobbers
        fix_blobber_config
        rebuild_blobbers_with_local_gosdk
        # Restart enterprise blobbers to apply config changes (service_charge, write_price)
        print_header "Restarting Enterprise Blobbers to Apply Config Changes"
        for i in $(seq 1 5); do
            eb="eblobber-${i}"
            if docker ps -a -q --filter "name=^${eb}$" 2>/dev/null | grep -q .; then
                print_status "Restarting $eb..."
                docker stop "$eb" >/dev/null 2>&1 || true
                sleep 2
                docker start "$eb" >/dev/null 2>&1 \
                    && print_status "  Restarted $eb" \
                    || print_warning "  Failed to restart $eb"
            else
                print_warning "  Container $eb not found, skipping"
            fi
        done
        # Run bl-update on all enterprise blobbers to set correct on-chain settings.
        # Restarting alone won't update service_charge/write_price — bl-update is required.
        print_header "Updating Enterprise Blobber On-Chain Settings via bl-update"
        # Ensure ZCN config has the correct block_worker before calling zbox bl-update
        setup_zcn_config
        print_status "Waiting 30s for enterprise blobbers to register after restart..."
        sleep 30
        W="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
        # Use sharder API (more reliable than zbox ls-blobbers --json in deploy context)
        enterprise_ids=$(curl -sf --max-time 15 \
            "http://127.0.0.1:7172/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers" \
            2>/dev/null \
            | python3 -c "import sys,json; data=json.load(sys.stdin); blobbers=data.get('Nodes',data.get('nodes',[])); print('\n'.join(b['id'] for b in blobbers if b.get('is_enterprise') or 'eblobber' in b.get('url','')))" \
            2>/dev/null || true)
        if [ -n "$enterprise_ids" ]; then
            for eblobber_id in $enterprise_ids; do
                print_status "  bl-update enterprise blobber ${eblobber_id:0:16}..."
                $ZBOX bl-update \
                    --blobber_id "$eblobber_id" \
                    --read_price 0 \
                    --write_price 0.001 \
                    --service_charge 0.3 \
                    --storage_version 1 \
                    --num_delegates 100 \
                    $W --silent 2>/dev/null \
                    && print_status "    Done" \
                    || print_warning "    bl-update failed (may retry on next health check)"
                sleep 2
                # Also mark not_available=true so they aren't selected for regular allocations
                $ZBOX bl-update \
                    --blobber_id "$eblobber_id" \
                    --not_available=true \
                    --wallet owner.json --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE --silent 2>/dev/null || true
                sleep 1
            done
        else
            print_warning "No enterprise blobbers found on chain — skipping bl-update"
        fi
        ;;
    fund-blobbers)
        fund_blobbers_and_validators
        ;;
    stake-blobbers)
        stake_and_configure_blobbers
        stake_enterprise_blobbers || true
        ;;
    ensure-config|reset-config)
        ensure_chain_config
        seed_0box_providers
        fix_snapshot_aggregates
        ;;
    fix-split-dkg)
        # Fix split-DKG: miners restarted at different times load different DKG key states
        # from disk, causing VRF share verification failures and the chain getting stuck.
        # Root cause: view_change=true + individual miner restarts → diverged DKG secrets.
        # Fix: disable view_change on-chain + restart ALL miners simultaneously so they
        # re-sync their DKG state from the same finalized block.
        print_header "Fixing Split-DKG (Chain Stuck)"
        local WO="--wallet $ZCN_WALLET_FILE --configDir $ZCN_CONFIG_DIR --config $ZCN_CONFIG_FILE"
        print_status "Step 1: Disabling view_change on-chain to prevent future split-DKG..."
        $ZWALLET global-update-config --keys 'server_chain.view_change' --values 'false' $WO \
            --silent 2>&1 | tail -3 || print_warning "view_change update failed (may already be false)"
        print_status "Step 2: Restarting all miners simultaneously to re-sync DKG state..."
        local miners_to_restart
        mapfile -t miners_to_restart < <(docker ps --format '{{.Names}}' | grep '^miner-[0-9]' | sort)
        if [ ${#miners_to_restart[@]} -eq 0 ]; then
            print_error "No running miner containers found"
        else
            print_status "Restarting: ${miners_to_restart[*]}"
            docker restart "${miners_to_restart[@]}" 2>&1
            print_status "Waiting 30s for miners to sync..."
            sleep 30
            # Verify chain is moving
            local round_before round_after
            round_before=$(docker logs "${miners_to_restart[0]}" --tail 5 2>&1 | grep -oP '"current round": \K[0-9]+' | tail -1 || echo "?")
            sleep 15
            round_after=$(docker logs "${miners_to_restart[0]}" --tail 5 2>&1 | grep -oP '"current round": \K[0-9]+' | tail -1 || echo "?")
            if [ "$round_before" != "$round_after" ] && [ "$round_after" != "?" ]; then
                print_status "Chain is advancing: round $round_before → $round_after"
            else
                print_warning "Chain may still be stuck at round $round_after — check miner logs"
            fi
        fi
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        print_error "Unknown command: ${COMMAND}"
        usage
        exit 1
        ;;
esac
