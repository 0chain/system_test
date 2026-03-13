#!/bin/bash
# 0Chain Comprehensive Data Cleanup Script
# Removes ALL service data while preserving directory structure, configs, and keys.
#
# Usage:
#   ./clean.sh              # Interactive: prompts before destructive actions
#   ./clean.sh --force      # Non-interactive: skips prompts
#   ./clean.sh --dry-run    # Show what would be deleted without deleting
#
# Services cleaned:
#   - Miners (1-4): RocksDB data, state, DKG keys, magic block, config backups
#   - Sharders (1-2): RocksDB data, DKG, PostgreSQL data, ssd/hdd tablespace dirs, blocks
#   - Blobbers (1-12): files, data, logs, PostgreSQL data, ssd/hdd tablespace dirs
#   - Enterprise Blobbers (1-3): files, data, logs, PostgreSQL data, ssd/hdd tablespace dirs
#   - Validators (1-12): data, logs
#   - 0box: PostgreSQL, Redis, logs
#   - Kafka: broker data, logs
#   - zauth: PostgreSQL data
#   - zvault: PostgreSQL data
#   - zs3server: MinIO data
#   - Crawler: data, logs
#   - Elasticsearch: data
#   - 0dns: no persistent data (stateless)
#
# What is PRESERVED:
#   - Config files (*.yaml, *.yml, *.json configs)
#   - Key files (keys_config/, wallet files)
#   - Directory structure (recreated if needed)
#   - The scripts themselves

set -e

# Ensure common binary paths are in PATH (needed for tmux/non-login shells)
export PATH="$PATH:/usr/local/go/bin:/root/go/bin:/usr/local/bin"

# Set up automatic logging: all stdout/stderr goes to both terminal AND log file.
CLEAN_LOG="${CLEAN_LOG:-/tmp/clean.log}"
if [ -z "$_CLEAN_LOGGING_SET" ]; then
    export _CLEAN_LOGGING_SET=1
    exec > >(stdbuf -oL tee -a "$CLEAN_LOG") 2>&1
    echo "" > "$CLEAN_LOG"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="${HOME}/Code"

# Flags
FORCE=false
DRY_RUN=false

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

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --force|-f) FORCE=true; shift ;;
        --dry-run|-n) DRY_RUN=true; shift ;;
        --help|-h)
            echo "Usage: $0 [--force] [--dry-run]"
            echo ""
            echo "  --force    Skip confirmation prompts"
            echo "  --dry-run  Show what would be deleted without deleting"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Safe rm -rf wrapper
safe_rm() {
    local target="$1"
    if [ -e "$target" ] || ls "$target" 2>/dev/null | head -1 > /dev/null 2>&1; then
        if $DRY_RUN; then
            print_status "[DRY-RUN] Would remove: $target"
        else
            rm -rf $target
            print_status "Removed: $target"
        fi
    fi
}

# Recreate directory (preserves structure)
ensure_dir() {
    local dir="$1"
    if ! $DRY_RUN; then
        mkdir -p "$dir"
    fi
}

# Confirmation prompt
confirm() {
    if $FORCE || $DRY_RUN; then
        return 0
    fi
    echo -e "${RED}WARNING: This will delete ALL service data!${NC}"
    echo -e "${RED}Configs, keys, and directory structure will be preserved.${NC}"
    echo ""
    read -p "Are you sure you want to continue? (type 'yes' to confirm): " response
    if [ "$response" != "yes" ]; then
        echo "Aborted."
        exit 0
    fi
}

# ============================================================
# STEP 1: Stop all containers
# ============================================================
stop_all_containers() {
    print_header "Stopping and Removing All Containers"

    if $DRY_RUN; then
        print_status "[DRY-RUN] Would stop and remove all 0chain containers"
        return
    fi

    # Stop monitoring daemons
    for pid_file in /tmp/funding_daemon.pid /tmp/vc_test.pid /tmp/chaos_test.pid; do
        if [ -f "$pid_file" ]; then
            kill "$(cat "$pid_file")" 2>/dev/null || true
            rm -f "$pid_file"
        fi
    done

    # Collect ALL container names to stop+remove
    # Using docker rm -f (force remove = stop + remove in one step, no stale containers left)
    local containers=()

    # Miners
    for i in 1 2 3 4; do containers+=("miner-$i"); done
    # Sharders + postgres
    for i in 1 2; do containers+=("sharder-$i" "sharder-postgres-$i"); done
    # 0dns
    containers+=("0dns")
    # Blobbers, validators, postgres (regular)
    for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
        containers+=("blobber-$i" "validator-$i" "postgres-blob-$i")
    done
    # Enterprise blobbers
    for i in 1 2 3; do
        containers+=("eblobber-$i" "postgres-eblob-$i")
    done
    # Services
    containers+=("0box" "0box-redis" "redis-0box" "redis" "0box-postgres" "postgres-0box")
    containers+=("0box-elasticsearch" "elasticsearch")
    containers+=("zauth" "zauth-postgres" "postgres")
    containers+=("zvault" "zvault-postgres" "postgreszv" "postgres-zvault")
    containers+=("pgadmin-zvault" "pgadmin" "pgadmin-sharder" "pgadmin4_container")
    containers+=("kafka" "minioserver" "crawler" "cadvisor" "zuscloudnative")

    # Force-remove all containers (stops + removes in one shot)
    for c in "${containers[@]}"; do
        docker rm -f "$c" 2>/dev/null || true
    done

    # Also use docker compose down for services that use named volumes
    local CHAIN_DIR="${BASE_DIR}/0chain/docker.local"
    for i in 1 2 3 4; do
        MINER=$i docker compose -p miner$i -f "${CHAIN_DIR}/build.miner/b0docker-compose.yml" down -v 2>/dev/null || true
    done
    for i in 1 2; do
        SHARDER=$i docker compose -p sharder$i -f "${CHAIN_DIR}/build.sharder/b0docker-compose.yml" down -v 2>/dev/null || true
    done

    local BLOBBER_DIR="${BASE_DIR}/blobber/docker.local"
    for i in 1 2 3 4 5 6; do
        if [ -f "${BLOBBER_DIR}/b0docker-compose-${i}.yml" ]; then
            docker compose -p blobber$i -f "${BLOBBER_DIR}/b0docker-compose-${i}.yml" down -v 2>/dev/null || true
        else
            BLOBBER=$i docker compose -p blobber$i -f "${BLOBBER_DIR}/b0docker-compose.yml" down -v 2>/dev/null || true
        fi
    done
    for i in 7 8 9 10 11 12 13 14 15; do
        BLOBBER=$i docker compose -p blobber$i -f "${BLOBBER_DIR}/b0docker-compose.yml" down -v 2>/dev/null || true
    done

    sleep 3
    # Verify no containers left
    local remaining=$(docker ps -q 2>/dev/null | wc -l)
    if [ "$remaining" -gt 0 ]; then
        print_warning "$remaining containers still running - force killing"
        docker kill $(docker ps -q) 2>/dev/null || true
        docker rm -f $(docker ps -aq) 2>/dev/null || true
    fi

    print_status "All containers stopped and removed"

    # Prune Docker: unused networks, dangling images, build cache.
    # Use timeout to prevent hanging (seen on Hetzner VPS with large layer stores).
    # PRESERVE base images (zchain_build_base, zchain_run_base, blobber_base, eblobber_base)
    # — these take 20+ min to rebuild and are needed by chain/blobber builds.
    # Tag them to prevent `docker system prune --all` from deleting them.
    for base_img in zchain_build_base zchain_run_base blobber_base eblobber_base; do
        if docker image inspect "$base_img" > /dev/null 2>&1; then
            docker tag "$base_img" "${base_img}:preserve" 2>/dev/null || true
        fi
    done
    print_status "Pruning Docker volumes..."
    timeout 120 docker volume prune -f 2>/dev/null || print_warning "Volume prune timed out (non-critical)"
    print_status "Pruning Docker images and build cache (preserving base images)..."
    timeout 300 docker system prune -f 2>/dev/null || print_warning "System prune timed out (non-critical)"
    # Remove dangling images but NOT all unused (--all would remove preserved base images)
    timeout 120 docker image prune -f 2>/dev/null || true
    # Truncate container logs >500MB (prevents disk-full on long-running deployments)
    for f in /var/lib/docker/containers/*/*-json.log; do
        local sz=$(stat -c%s "$f" 2>/dev/null || echo 0)
        if [ "$sz" -gt 524288000 ]; then
            truncate -s 0 "$f" 2>/dev/null
        fi
    done
    print_status "Docker prune complete"
}

# ============================================================
# STEP 2: Clean miner data
# ============================================================
clean_miners() {
    print_header "Cleaning Miner Data"

    local CHAIN_DIR="${BASE_DIR}/0chain/docker.local"

    for i in 1 2 3 4; do
        local miner_dir="${CHAIN_DIR}/miner${i}"
        if [ -d "$miner_dir" ]; then
            # RocksDB data (config/ contains LFB state, data/ contains chain data)
            safe_rm "${miner_dir}/data/rocksdb/config"
            safe_rm "${miner_dir}/data/rocksdb/data"
            safe_rm "${miner_dir}/data/rocksdb/state"
            safe_rm "${miner_dir}/data/rocksdb/log"

            # DKG data (distributed key generation state)
            safe_rm "${miner_dir}/data/rocksdb/dkg"
            safe_rm "${miner_dir}/data/rocksdb/dkgkey"
            safe_rm "${miner_dir}/data/rocksdb/mb"

            # Remove any config backup directories from previous LFB recovery
            for bak in "${miner_dir}"/data/rocksdb/config_bak_*; do
                safe_rm "$bak"
            done

            # Recreate directory structure
            ensure_dir "${miner_dir}/data/rocksdb"

            # Logs
            safe_rm "${miner_dir}/log/*"
            ensure_dir "${miner_dir}/log"

            print_status "Cleaned miner-$i"
        else
            print_warning "miner${i} directory not found at ${miner_dir}"
        fi
    done
}

# ============================================================
# STEP 3: Clean sharder data
# ============================================================
clean_sharders() {
    print_header "Cleaning Sharder Data"

    local CHAIN_DIR="${BASE_DIR}/0chain/docker.local"

    for i in 1 2; do
        local sharder_dir="${CHAIN_DIR}/sharder${i}"
        if [ -d "$sharder_dir" ]; then
            # RocksDB data
            safe_rm "${sharder_dir}/data/rocksdb/config"
            safe_rm "${sharder_dir}/data/rocksdb/data"
            safe_rm "${sharder_dir}/data/rocksdb/state"
            safe_rm "${sharder_dir}/data/rocksdb/log"

            # DKG data (distributed key generation state)
            safe_rm "${sharder_dir}/data/rocksdb/dkg"
            safe_rm "${sharder_dir}/data/rocksdb/mb"
            safe_rm "${sharder_dir}/data/rocksdb/txnsummary"
            safe_rm "${sharder_dir}/data/rocksdb/blocksummary"
            safe_rm "${sharder_dir}/data/rocksdb/blockevents"
            safe_rm "${sharder_dir}/data/rocksdb/roundsummary"
            safe_rm "${sharder_dir}/data/rocksdb/magicblockmap"

            ensure_dir "${sharder_dir}/data/rocksdb"

            # PostgreSQL data (event database)
            safe_rm "${sharder_dir}/data/postgresql"
            ensure_dir "${sharder_dir}/data/postgresql"

            # PostgreSQL tablespace directories (ssd/hdd)
            safe_rm "${sharder_dir}/data/ssd"
            ensure_dir "${sharder_dir}/data/ssd"
            safe_rm "${sharder_dir}/data/hdd"
            ensure_dir "${sharder_dir}/data/hdd"

            # Block data
            safe_rm "${sharder_dir}/data/blocks"
            ensure_dir "${sharder_dir}/data/blocks"

            # Logs
            safe_rm "${sharder_dir}/log/*"
            ensure_dir "${sharder_dir}/log"

            print_status "Cleaned sharder-$i"
        else
            print_warning "sharder${i} directory not found at ${sharder_dir}"
        fi
    done
}

# ============================================================
# STEP 4: Clean blobber data (regular 1-12)
# ============================================================
clean_blobbers() {
    print_header "Cleaning Blobber Data (Regular 1-12)"

    local BLOBBER_DIR="${BASE_DIR}/blobber/docker.local"

    for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
        local blob_dir="${BLOBBER_DIR}/blobber${i}"
        if [ -d "$blob_dir" ]; then
            # Files uploaded to blobber
            safe_rm "${blob_dir}/files"
            ensure_dir "${blob_dir}/files"

            # Blobber internal data (includes postgresql, postgresql2/hdd tablespace, tmp)
            safe_rm "${blob_dir}/data"
            ensure_dir "${blob_dir}/data/postgresql"
            ensure_dir "${blob_dir}/data/postgresql2"
            ensure_dir "${blob_dir}/data/tmp"

            # Tablespace directories (ssd/hdd) - may be mounted separately
            safe_rm "${blob_dir}/data/ssd"
            ensure_dir "${blob_dir}/data/ssd"
            safe_rm "${blob_dir}/data/hdd"
            ensure_dir "${blob_dir}/data/hdd"

            # Logs
            safe_rm "${blob_dir}/log"
            ensure_dir "${blob_dir}/log"

            print_status "Cleaned blobber-$i"
        fi

        # Validator data (shares directory structure with blobbers)
        local val_dir="${BLOBBER_DIR}/validator${i}"
        if [ -d "$val_dir" ]; then
            safe_rm "${val_dir}/data"
            ensure_dir "${val_dir}/data"
            safe_rm "${val_dir}/log"
            ensure_dir "${val_dir}/log"
            print_status "Cleaned validator-$i"
        fi
    done
}

# ============================================================
# STEP 5: Clean enterprise blobber data
# ============================================================
clean_enterprise_blobbers() {
    print_header "Cleaning Enterprise Blobber Data"

    local EBLOBBER_DIR="${BASE_DIR}/eblobber/docker.local"

    if [ ! -d "$EBLOBBER_DIR" ]; then
        print_warning "eblobber directory not found, skipping"
        return
    fi

    for i in 1 2 3; do
        local eb_dir="${EBLOBBER_DIR}/eblobber${i}"
        if [ -d "$eb_dir" ]; then
            safe_rm "${eb_dir}/files"
            ensure_dir "${eb_dir}/files"

            safe_rm "${eb_dir}/data"
            ensure_dir "${eb_dir}/data/postgresql"
            ensure_dir "${eb_dir}/data/postgresql2"
            ensure_dir "${eb_dir}/data/tmp"

            # Tablespace directories (ssd/hdd)
            safe_rm "${eb_dir}/data/ssd"
            ensure_dir "${eb_dir}/data/ssd"
            safe_rm "${eb_dir}/data/hdd"
            ensure_dir "${eb_dir}/data/hdd"

            safe_rm "${eb_dir}/log"
            ensure_dir "${eb_dir}/log"

            print_status "Cleaned eblobber-$i"
        fi
    done
}

# ============================================================
# STEP 6: Clean 0box data
# ============================================================
clean_0box() {
    print_header "Cleaning 0box Data"

    local BOX_DIR="${BASE_DIR}/0box/docker.local"

    if [ ! -d "$BOX_DIR" ]; then
        print_warning "0box directory not found, skipping"
        return
    fi

    # Docker compose volumes (postgresql, redis)
    if ! $DRY_RUN; then
        cd "$BOX_DIR"
        docker compose -p 0box down -v 2>/dev/null || true
    else
        print_status "[DRY-RUN] Would remove 0box docker compose volumes"
    fi

    # Data directories
    # NOTE: postgres bind-mount is ./0box/data/postgresql (NOT ./data which is empty)
    safe_rm "${BOX_DIR}/0box/data"
    ensure_dir "${BOX_DIR}/0box/data"
    # Legacy ./data directory (kept for backward compat in case some builds use it)
    safe_rm "${BOX_DIR}/data"
    ensure_dir "${BOX_DIR}/data"

    # 0box app log
    safe_rm "${BOX_DIR}/0box/log"
    ensure_dir "${BOX_DIR}/0box/log"
    safe_rm "${BOX_DIR}/log"
    ensure_dir "${BOX_DIR}/log"

    # Redis and elasticsearch data (bind-mounts, not removed by compose down -v)
    safe_rm "${BOX_DIR}/redis"
    ensure_dir "${BOX_DIR}/redis"
    safe_rm "${BOX_DIR}/elasticsearch"
    ensure_dir "${BOX_DIR}/elasticsearch"
    safe_rm "${BOX_DIR}/esdata"
    ensure_dir "${BOX_DIR}/esdata"

    print_status "Cleaned 0box"
}

# ============================================================
# STEP 7: Clean Kafka data
# ============================================================
clean_kafka() {
    print_header "Cleaning Kafka Data"

    local KAFKA_DATA="${BASE_DIR}/0chain/docker.local/kafka"
    local KAFKA_DIR="${BASE_DIR}/0chain/docker.local/build.kafka"

    safe_rm "${KAFKA_DATA}"
    ensure_dir "${KAFKA_DATA}"

    # Remove docker compose volumes
    if [ -d "$KAFKA_DIR" ] && ! $DRY_RUN; then
        cd "$KAFKA_DIR"
        docker compose down -v 2>/dev/null || true
    fi

    print_status "Cleaned Kafka"
}

# ============================================================
# STEP 8: Clean zauth data
# ============================================================
clean_zauth() {
    print_header "Cleaning zauth Data"

    local ZAUTH_DIR="${BASE_DIR}/zauth-server/docker.local"

    if [ ! -d "$ZAUTH_DIR" ]; then
        print_warning "zauth directory not found, skipping"
        return
    fi

    if ! $DRY_RUN; then
        cd "$ZAUTH_DIR"
        docker compose -p zauth down -v 2>/dev/null || true
    else
        print_status "[DRY-RUN] Would remove zauth docker compose volumes"
    fi

    safe_rm "${ZAUTH_DIR}/data"
    ensure_dir "${ZAUTH_DIR}/data"

    print_status "Cleaned zauth"
}

# ============================================================
# STEP 9: Clean zvault data
# ============================================================
clean_zvault() {
    print_header "Cleaning zvault Data"

    local ZVAULT_DIR="${BASE_DIR}/zvault/docker.local"

    if [ ! -d "$ZVAULT_DIR" ]; then
        print_warning "zvault directory not found, skipping"
        return
    fi

    if ! $DRY_RUN; then
        cd "$ZVAULT_DIR"
        docker compose -p zvault down -v 2>/dev/null || true
    else
        print_status "[DRY-RUN] Would remove zvault docker compose volumes"
    fi

    safe_rm "${ZVAULT_DIR}/data"
    ensure_dir "${ZVAULT_DIR}/data"

    print_status "Cleaned zvault"
}

# ============================================================
# STEP 10: Clean Elasticsearch data
# ============================================================
clean_elasticsearch() {
    print_header "Cleaning Elasticsearch Data"

    # Remove the container entirely to avoid name conflicts on redeploy
    if ! $DRY_RUN; then
        docker rm -f elasticsearch 2>/dev/null || true
        docker rm -f 0box-elasticsearch 2>/dev/null || true
    else
        print_status "[DRY-RUN] Would remove elasticsearch containers"
    fi

    # ES data is typically in docker volumes managed by 0box compose
    # Also check standalone data dir
    local ES_DATA="${BASE_DIR}/0box/docker.local/esdata"
    safe_rm "${ES_DATA}"
    ensure_dir "${ES_DATA}"

    print_status "Cleaned Elasticsearch"
}

# ============================================================
# STEP 11: Clean Crawler data
# ============================================================
clean_crawler() {
    print_header "Cleaning Crawler Data"

    local CRAWLER_DIR="${BASE_DIR}/crawler/docker.local"

    if [ ! -d "$CRAWLER_DIR" ]; then
        print_warning "crawler directory not found, skipping"
        return
    fi

    if ! $DRY_RUN; then
        cd "$CRAWLER_DIR"
        docker compose -p crawler down -v 2>/dev/null || true
    fi

    safe_rm "${CRAWLER_DIR}/data"
    ensure_dir "${CRAWLER_DIR}/data"

    safe_rm "${CRAWLER_DIR}/log"
    ensure_dir "${CRAWLER_DIR}/log"

    print_status "Cleaned Crawler"
}

# ============================================================
# STEP 12: Clean zs3server data
# ============================================================
clean_zs3server() {
    print_header "Cleaning zs3server (MinIO) Data"

    local ZS3_DIR="${BASE_DIR}/zs3server/environment"

    if [ ! -d "$ZS3_DIR" ]; then
        print_warning "zs3server directory not found, skipping"
        return
    fi

    if ! $DRY_RUN; then
        cd "$ZS3_DIR"
        docker compose down -v 2>/dev/null || true
    fi

    # MinIO data directory
    safe_rm "${ZS3_DIR}/data"
    ensure_dir "${ZS3_DIR}/data"

    print_status "Cleaned zs3server"
}

# ============================================================
# STEP 13: Clean wallet nonces (reset to 0 for fresh chain)
# ============================================================
clean_wallet_nonces() {
    print_header "Resetting Wallet Nonces"

    local ZCN_CONFIG_DIR="${HOME}/.zcn"
    local BLOBBER_KEYS_DIR="${BASE_DIR}/blobber/docker.local/keys_config"

    if $DRY_RUN; then
        print_status "[DRY-RUN] Would reset all wallet nonces to 0"
        return
    fi

    # Reset blobber/validator wallet nonces
    for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
        for prefix in b0bnode b0vnode; do
            local wallet_file="${BLOBBER_KEYS_DIR}/${prefix}${i}_keys.txt.json"
            if [ -f "$wallet_file" ]; then
                local current_nonce=$(jq '.nonce' "$wallet_file" 2>/dev/null)
                if [ -n "$current_nonce" ] && [ "$current_nonce" != "0" ] && [ "$current_nonce" != "null" ]; then
                    jq '.nonce = 0' "$wallet_file" > "${wallet_file}.tmp" && mv "${wallet_file}.tmp" "$wallet_file"
                fi
            fi
        done
    done

    # Reset owner wallet nonce
    for wallet in "${ZCN_CONFIG_DIR}/local.json" \
                  "${BASE_DIR}/system_test/tests/cli_tests/config/wallets/sc_owner_wallet.json" \
                  "${BASE_DIR}/system_test/tests/cli_tests/config/wallets/blobber_owner_wallet.json" \
                  "${BASE_DIR}/system_test/tests/tokenomics_tests/config/wallets/sc_owner_wallet.json" \
                  "${BASE_DIR}/system_test/tests/tokenomics_tests/config/wallets/blobber_owner_wallet.json"; do
        if [ -f "$wallet" ]; then
            local current_nonce=$(jq '.nonce' "$wallet" 2>/dev/null)
            if [ -n "$current_nonce" ] && [ "$current_nonce" != "0" ] && [ "$current_nonce" != "null" ]; then
                jq '.nonce = 0' "$wallet" > "${wallet}.tmp" && mv "${wallet}.tmp" "$wallet"
            fi
        fi
    done

    print_status "All wallet nonces reset to 0"
}

# ============================================================
# STEP 14: Clean temp/log files
# ============================================================
clean_temp_files() {
    print_header "Cleaning Temp and Log Files"

    # Test run logs
    safe_rm "/tmp/api_*.log"
    safe_rm "/tmp/cli_*.log"
    safe_rm "/tmp/token*.log"
    safe_rm "/tmp/test_*.log"
    safe_rm "/tmp/funding_daemon.log"
    safe_rm "/tmp/view_change_loop.log"
    safe_rm "/tmp/chaos_test.log"

    # PID files
    safe_rm "/tmp/funding_daemon.pid"
    safe_rm "/tmp/vc_test.pid"
    safe_rm "/tmp/chaos_test.pid"

    # Status files
    safe_rm "/tmp/*_status"

    print_status "Cleaned temp files"
}

# ============================================================
# MAIN
# ============================================================
print_header "0Chain Comprehensive Data Cleanup"

if $DRY_RUN; then
    print_warning "DRY RUN MODE - no data will be deleted"
    echo ""
fi

confirm

stop_all_containers
clean_miners
clean_sharders
clean_blobbers
clean_enterprise_blobbers
clean_0box
clean_kafka
clean_zauth
clean_zvault
clean_elasticsearch
clean_crawler
clean_zs3server
clean_wallet_nonces
clean_temp_files

cd "$SCRIPT_DIR"

print_header "Cleanup Complete!"
echo ""
echo "All service data has been cleared. Directory structure preserved."
echo ""
echo "Next steps:"
echo "  1. Deploy fresh: ./deploy_local.sh all"
echo "  2. Or start specific services: ./deploy_local.sh start-chain"
echo ""
echo "Preserved:"
echo "  - Config files (*.yaml)"
echo "  - Key files (keys_config/)"
echo "  - Docker images are pruned (will be rebuilt on next deploy)"
