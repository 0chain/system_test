# 0Chain Local Deployment & System Test Guide

Complete guide to deploying a local 0Chain network, updating service images, accessing web apps in the browser, and running the full system test suite.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Step-by-Step Deployment](#step-by-step-deployment)
- [Updating Service Images](#updating-service-images)
- [Web Apps (Browser Access)](#web-apps-browser-access)
- [Running System Tests](#running-system-tests)
- [Service Architecture](#service-architecture)
- [Scripts Reference](#scripts-reference)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Developer Workflow](#developer-workflow)
- [Monitoring, Chaos & View Change Testing](#monitoring-chaos--view-change-testing)
- [Kill/Shutdown Tests (Destructive)](#killshutdown-tests-destructive)

---

## Prerequisites

### Required Software

- **Docker** + Docker Compose v2
- **Go** 1.22+ (for building CLI tools and running tests)
- **jq** (JSON parsing for wallet operations)
- **curl** (health checks and API calls)
- **Node.js** + npm (for building web-apps)
- **python3** (used by some helper scripts)
- **git** (repo management)

### Required Repositories

All repos must be cloned under a common base directory (default: `~/Code/`):

```
~/Code/
  0chain/           # Core blockchain (miners, sharders)
  0dns/             # DNS/service discovery
  blobber/          # Storage providers (blobbers + validators)
  eblobber/         # Enterprise blobber (separate repo)
  gosdk/            # Go SDK (dependency for WASM, eblobber, zs3server)
  0box/             # File listing/query service
  zauth-server/     # JWT authentication service
  zvault/           # Encrypted storage vault
  zs3server/        # MinIO S3 gateway
  crawler/          # Blockchain crawler
  zusCloudNative/   # Datalake (Blimp backend)
  web-apps/         # Frontend web applications (Vult, Bolt, Blimp, etc.)
  zboxcli/          # zbox CLI tool
  zwalletcli/       # zwallet CLI tool
  system_test/      # This repo (test suites + deploy scripts)
```

### macOS Note

The deployment uses Docker networking with custom IP ranges (198.18.0.x). On macOS, loopback aliases are created automatically by the deploy script (`ifconfig lo0 alias ...`). This requires `sudo`.

---

## Quick Start

```bash
cd ~/Code/system_test/scripts

# Full deployment: checkout branches, start chain, configure everything, deploy all services
./deploy_local.sh all

# Verify everything is healthy
./deploy_local.sh verify

# Run all system tests
./deploy_local.sh test
```

To wipe everything and start fresh (e.g., after config changes that need a new genesis):

```bash
# Clean + full deploy in one command
./deploy_local.sh redeploy
```

This takes 15-30 minutes for a full deployment from scratch.

---

## Step-by-Step Deployment

If you prefer more control, deploy each component individually:

### 1. Configure Branches

Edit `deploy_config.yaml` to set which branch each service should use:

```yaml
repositories:
  0chain:
    branch: staging      # Core blockchain
  blobber:
    branch: staging      # Storage providers
  0box:
    branch: staging      # Query service
  web-apps:
    branch: master       # Frontend apps
    gosdk_branch: staging  # gosdk branch for WASM build
```

Or override at the command line:

```bash
./deploy_local.sh --branch 0chain=fix/my-feature --branch blobber=staging all
```

### 2. Checkout Repositories

```bash
./deploy_local.sh checkout
```

Clones missing repos and checks out the configured branch for each.

### 3. Setup Network (macOS Loopback)

```bash
./deploy_local.sh loopback
```

Creates IP aliases on `lo0` for Docker containers (198.18.0.71-102, etc.).

### 4. Start the Blockchain

```bash
./deploy_local.sh start-chain
```

Starts 4 miners, 2 sharders, and 0dns (service discovery). Waits for the chain to produce blocks.

### 5. Initialize Chain Configuration

```bash
./deploy_local.sh chain
```

Applies:
- Hardfork rounds (all set to 0 for immediate activation)
- SC config: `min_alloc_size=1024`, `min_write_price=0`, `max_block_cost=100000`
- View change settings for fast round production

### 6. Start Supporting Services

```bash
./deploy_local.sh services
```

Starts (in order):
1. **Elasticsearch** - Event indexing for 0box
2. **Kafka** - Event broker (sharder -> 0box pipeline) with SASL auth
3. **zauth-server** - JWT authentication (port 8080)
4. **zvault** - Encrypted storage vault (port 8090)
5. **0box** - File listing and queries (port 9081)
6. **zusCloudNative** - Datalake/Blimp backend (port 8088)

### 7. Deploy Blobbers

```bash
./deploy_local.sh blobbers
```

This:
1. Builds the blobber Docker image
2. Creates blobber containers 1-12 (6 regular + 6 more)
3. Funds each blobber wallet via faucet
4. Registers and stakes blobbers on-chain
5. Configures read/write prices
6. Optionally deploys enterprise blobbers (eblobber 1-3)

### 8. Setup Test Environment

```bash
./deploy_local.sh test-setup
```

Copies wallets, CLI binaries, and config files to all test directories:
- `sc_owner_wallet.json` -> all test suites
- `blobber_owner_wallet.json` -> all test suites
- `zbox`/`zwallet` binaries -> CLI test directory
- Updates test config YAML files with correct service URLs

### 9. Setup Nginx (Optional)

If you want services accessible via a domain name:

```bash
# Set domain in deploy_config.yaml:
#   nginx:
#     domain: "test.zus.network"
#     enable_ssl: true

./deploy_local.sh nginx
```

### 10. Verify Deployment

```bash
./deploy_local.sh verify
```

Checks:
- All containers are running and healthy
- Miners and sharders are producing blocks
- Blobbers are registered on-chain
- All services respond to health checks
- SC configuration is correct

---

## Updating Service Images

### swap-image: Hot-Swap Without Full Redeploy

Rebuild and restart a single service without tearing down the network:

```bash
# Syntax
./deploy_local.sh swap-image <REPO> [BRANCH] [--gosdk-branch BRANCH]

# Examples
./deploy_local.sh swap-image 0chain fix/my-branch        # Switch chain to a branch
./deploy_local.sh swap-image blobber                       # Rebuild from config branch
./deploy_local.sh swap-image 0box staging                  # Rebuild 0box from staging
./deploy_local.sh swap-image zboxcli staging               # Rebuild CLI binary
./deploy_local.sh swap-image web-apps master --gosdk-branch staging  # Rebuild with WASM
```

### What swap-image Does

1. Checks out the specified branch (or uses `deploy_config.yaml` default)
2. Rebuilds the Docker image from source
3. Stops the affected containers
4. Recreates containers with the new image
5. **Chain data and on-chain state are preserved**

### Supported Repos

| Repo | What It Rebuilds | Containers Restarted |
|------|-----------------|---------------------|
| `0chain` | Miner + sharder images | miner-1..4, sharder-1..2 |
| `blobber` | Blobber + validator images | blobber-1..12, validator-1..12 |
| `eblobber` | Enterprise blobber image | eblobber-1..3 |
| `0dns` | 0dns image | 0dns |
| `0box` | 0box image | 0box |
| `zauth-server` | zauth image | zauth |
| `zvault` | zvault image | zvault |
| `zs3server` | MinIO S3 gateway image | minioserver |
| `crawler` | Crawler image | crawler |
| `zusCloudNative` | Datalake image | zuscloudnative |
| `web-apps` | Frontend apps (builds WASM from gosdk first) | web-apps containers |
| `zboxcli` | zbox binary (no Docker) | N/A - copies binary to test dirs |
| `zwalletcli` | zwallet binary (no Docker) | N/A - copies binary to test dirs |

### gosdk-Dependent Builds

Some services need a specific gosdk branch checked out before building:

| Service | Config Key | Default gosdk Branch | Why |
|---------|-----------|---------------------|-----|
| `eblobber` | `gosdk_branch` | staging | Enterprise blobber uses gosdk as a Go dependency |
| `zs3server` | `gosdk_branch` | enterprise-blobber | S3 gateway uses enterprise gosdk |
| `web-apps` | `gosdk_branch` | staging | Needs `zcn.wasm` built from gosdk |

Override at runtime:

```bash
./deploy_local.sh swap-image eblobber staging --gosdk-branch enterprise-blobber
./deploy_local.sh swap-image web-apps master --gosdk-branch fix/my-sdk-branch
```

---

## Web Apps (Browser Access)

### Overview

The `web-apps` repo contains multiple Next.js frontend applications:

| App | Description | Package |
|-----|------------|---------|
| **Vult** | Personal cloud storage (like Google Drive) | `packages/vult` |
| **Bolt** | Token wallet and transactions | `packages/bolt` |
| **Blimp** | Enterprise file storage (S3-compatible) | `packages/blimp` |
| **Chalk** | Document collaboration | `packages/chalk` |
| **Explorer** | Blockchain explorer | `packages/explorer` |
| **Chimney** | Developer tools | `packages/chimney` |

### How Web Apps Use the Network

```
Browser
  ├── Loads web app (Next.js)
  ├── Loads zcn.wasm (Go SDK compiled to WebAssembly)
  ├── wasm_exec.js (Go WASM runtime)
  └── zcn.js (JavaScript bridge)
        │
        ├── window.__zcn_wasm__.sdk.init(chainID, blockWorker, ...)
        │     └── Connects to 0dns (block_worker URL) to discover network
        │
        ├── window.__zcn_wasm__.sdk.setWallet(...)
        │     └── Wallet operations via zauth-server
        │
        ├── File operations (upload/download/share)
        │     └── Directly communicates with blobbers
        │
        └── Allocation management
              └── Transactions go through miners → sharders
```

### Building Web Apps with WASM

The web apps require `zcn.wasm` - the Go SDK compiled to WebAssembly. The build process:

1. **Checkout gosdk** to the configured branch
2. **Build WASM**: `make wasm-build` in gosdk (uses Docker `golang:1.22.5`)
3. **Copy `zcn.wasm`** to each app's `packages/*/public/` directory
4. **Build the web app** (Next.js build)

This is all automated by `swap-image`:

```bash
# Build web-apps with WASM from staging gosdk
./deploy_local.sh swap-image web-apps master --gosdk-branch staging
```

### Manual WASM Build (if needed)

```bash
cd ~/Code/gosdk
git checkout staging

# Option A: Docker build (recommended - matches CI exactly)
docker run --rm -v "$PWD":/gosdk -w /gosdk golang:1.22.5 sh -c \
  "git config --global --add safe.directory /gosdk; make wasm-build"

# Option B: Native build (requires Go 1.22+)
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-s -w" -buildvcs=false -o ./zcn.wasm ./wasmsdk

# Copy to web-apps
for pkg in ~/Code/web-apps/packages/*/public; do
  cp zcn.wasm "$pkg/zcn.wasm"
done
```

### Accessing Web Apps in the Browser

#### Option A: Local Development Server

```bash
cd ~/Code/web-apps

# Install dependencies
npm install

# Start Vult in dev mode (typically port 3000)
cd packages/vult
npm run dev
```

Then open `http://localhost:3000` in your browser.

#### Option B: Docker Deployment

If web-apps has a Docker setup:

```bash
./deploy_local.sh swap-image web-apps
# Access via the configured port (check docker-compose.yml in web-apps repo)
```

#### Option C: Via Nginx Reverse Proxy

If nginx is configured with a domain:

```bash
# In deploy_config.yaml:
#   nginx:
#     domain: "test.zus.network"
./deploy_local.sh nginx
```

Then access `https://test.zus.network/` in your browser.

### Configuring Web Apps for Local Network

The web app needs to know your local network's `block_worker` URL. Set this in the app's `.env` file:

```bash
# In packages/vult/.env (or whichever app you're running)
NEXT_PUBLIC_BLOCK_WORKER=http://localhost:9091
```

Or if using Docker networking:

```bash
NEXT_PUBLIC_BLOCK_WORKER=http://198.18.0.100:9091
```

### Key Files in Web Apps

| File | Purpose |
|------|---------|
| `packages/*/public/zcn.wasm` | Go SDK compiled to WebAssembly (~29 MB) |
| `packages/*/public/wasm_exec.js` | Go WASM runtime (from Go 1.22.5) |
| `packages/*/public/zcn.js` | JavaScript bridge for WASM SDK calls |
| `packages/*/.env` | Environment config (block_worker URL, etc.) |
| `packages/shared/` | Shared components used by all apps |

---

## rclone-zus (CLI File Transfer)

### Overview

**rclone-zus** is a custom [rclone](https://rclone.org/) backend for Zus decentralized storage. It allows using familiar rclone commands (`copy`, `sync`, `ls`, `move`) to manage files on the Zus network.

### Building

```bash
# Build rclone with Zus backend
./deploy_local.sh rclone-zus

# Or build manually
cd ~/Code/rclone_zus
CGO_ENABLED=1 go build -tags bn256 -o rclone rclone.go
```

The build requires CGO (for BLS cryptography) and produces a single `rclone` binary.

### Configuration

The deploy script automatically creates `~/.config/rclone/rclone.conf` pointing to the local chain:

```ini
[automation]
type = zus
config = ~/.zcn/local.yaml
wallet = ~/.zcn/local.json
allocation = <allocation_id>
```

### Integration Tests

rclone-zus has integration tests at `backend/zus/zus_test.go` that use rclone's standard `fstests.Run` framework. These require a running chain with an allocation configured in the `automation` remote.

```bash
cd ~/Code/rclone_zus
go test -tags bn256 -v ./backend/zus/
```

### Repository

- Source: `~/Code/rclone_zus`
- Branch: configured in `deploy_config.yaml` under `rclone_zus.branch`

---

## zs3/mc/warp Test Tools

The CLI test suite includes S3 gateway tests (`zs3server_tests/`) and MinIO client tests (`mc_tests/`) that require additional binaries.

### Setup

```bash
# Automatic setup (downloads mc, installs warp, creates configs)
./deploy_local.sh zs3-tools

# Also included in test-setup
./deploy_local.sh test-setup
```

### Required Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `mc` | `tests/cli_tests/mc` | MinIO Client for S3 operations |
| `warp` | `tests/cli_tests/warp` | MinIO benchmark tool |
| `minio` | `tests/cli_tests/minio` | MinIO server (from zs3server repo) |

### Configuration Files

| File | Location | Contents |
|------|----------|----------|
| `hosts.yaml` | `tests/cli_tests/zs3server_tests/` | S3 endpoint + credentials |
| `hosts.yaml` | `tests/cli_tests/mc_tests/` | S3 endpoint + credentials |
| `allocation.yaml` | `tests/cli_tests/zs3server_tests/` | data/parity shards, S3 keys |

Tests skip gracefully if the required binaries or configs are missing.

---

## Running System Tests

### Test Suites

| Suite | Directory | What It Tests |
|-------|-----------|--------------|
| **API tests** | `tests/api_tests/` | REST API endpoints, smart contracts, blobber operations |
| **CLI tests** | `tests/cli_tests/` | `zbox` and `zwallet` command-line tools |
| **SDK tests** | `tests/sdk_tests/` | Go SDK integration |
| **Tokenomics tests** | `tests/tokenomics_tests/` | Financial operations, staking, rewards, enterprise blobbers |

### Quick Test Run

```bash
# Run all test suites with auto-retry
./deploy_local.sh test

# Or use run_tests.sh directly for more control
./run_tests.sh --retries 3 api cli tokenomics
```

### Run Specific Suites

```bash
./run_tests.sh api                      # API tests only
./run_tests.sh cli tokenomics           # CLI + tokenomics
./run_tests.sh --filter TestCreate cli  # Only tests matching pattern
```

### Run Individual Tests

```bash
# API test
cd ~/Code/system_test/tests/api_tests
go test -v -run "^TestCreateAllocation$" -timeout 30m .

# CLI test
cd ~/Code/system_test/tests/cli_tests
go test -v -run "^TestUpload$" -timeout 45m .

# Tokenomics test
cd ~/Code/system_test/tests/tokenomics_tests
go test -v -run "^TestBlobberChallengeReward$" -timeout 60m .

# SDK test
cd ~/Code/system_test/tests/sdk_tests
go test -v -run "^TestCreateWallet$" -timeout 30m .
```

### Smoke Tests (Fast Subset)

```bash
SMOKE_TEST_MODE=true go test -v -timeout 30m .
```

### run_tests.sh Options

| Flag | Default | Description |
|------|---------|-------------|
| `--retries N` | 2 | Max retries for failed tests |
| `--timeout DURATION` | 45m | Per-suite timeout |
| `--test-timeout DURATION` | (none) | Per-test timeout override |
| `--filter PATTERN` | (none) | Only run tests matching pattern |

### Test Results

Results are saved to `test_results/`:

```
test_results/
  api_run1.txt         # First run output
  api_run2.txt         # Retry output (failed tests only)
  cli_run1.txt
  tokenomics_run1.txt
  summary.txt          # Final pass/fail summary
```

### Test Prerequisites

Before running tests, ensure:

1. Chain is running and producing blocks: `./deploy_local.sh verify`
2. Test wallets are set up: `./deploy_local.sh test-setup`
3. For CLI tests: `zbox` and `zwallet` binaries exist in `tests/cli_tests/`
4. For enterprise tokenomics tests: enterprise blobbers are deployed
5. For 0box/zauth/zvault API tests: these services are running

---

## Service Architecture

### Network Topology

```
                    ┌─────────────┐
                    │   Browser   │
                    │  (web-apps) │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    nginx    │ (optional reverse proxy)
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐     ┌─────▼─────┐     ┌────▼────┐
    │  0dns   │     │   0box    │     │  zauth  │
    │  :9091  │     │   :9081   │     │  :8080  │
    └────┬────┘     └─────┬─────┘     └─────────┘
         │                │
    ┌────▼────────────────┼──────────────────┐
    │              0Chain Network             │
    │  ┌───────┐ ┌───────┐ ┌───────┐ ┌─────┐│
    │  │Miner 1│ │Miner 2│ │Miner 3│ │  M4 ││
    │  │ :7071 │ │ :7072 │ │ :7073 │ │:7074││
    │  └───────┘ └───────┘ └───────┘ └─────┘│
    │  ┌─────────┐  ┌─────────┐             │
    │  │Sharder 1│  │Sharder 2│             │
    │  │  :7171  │  │  :7172  │             │
    │  └────┬────┘  └─────────┘             │
    └───────┼───────────────────────────────┘
            │
     ┌──────▼──────┐
     │    Kafka    │ (event pipeline)
     │ :9092 SASL  │
     └──────┬──────┘
            │
    ┌───────▼─────────────────────────────────┐
    │           Storage Layer                  │
    │  ┌────────┐ ┌────────┐     ┌────────┐  │
    │  │Blob  7 │ │Blob  8 │ ... │Blob 12 │  │
    │  │ :5057  │ │ :5058  │     │ :5062  │  │
    │  └────────┘ └────────┘     └────────┘  │
    │  ┌────────┐ ┌────────┐     ┌────────┐  │
    │  │Valid  7│ │Valid  8│ ... │Valid 12│  │
    │  │ :5067  │ │ :5068  │     │ :5072  │  │
    │  └────────┘ └────────┘     └────────┘  │
    │  ┌─────────┐ ┌─────────┐ ┌─────────┐  │
    │  │EBlob  1 │ │EBlob  2 │ │EBlob  3 │  │
    │  │  :5071  │ │  :5072  │ │  :5073  │  │
    │  └─────────┘ └─────────┘ └─────────┘  │
    └─────────────────────────────────────────┘
```

### Port Reference

#### Blockchain

| Service | Container IP | Port | Purpose |
|---------|-------------|------|---------|
| Miner 1-4 | 198.18.0.71-74 | 7071-7074 | Block production, transaction processing |
| Sharder 1-2 | 198.18.0.81-82 | 7171-7172 | Block storage, REST API, event database |
| 0dns | 198.18.0.100 | 9091 | Service discovery (block_worker) |

#### Storage

| Service | Container IP | REST Port | Purpose |
|---------|-------------|-----------|---------|
| Blobber 7-12 | 198.18.0.97-102 | 5057-5062 | File storage |
| Validator 7-12 | 198.18.0.97-102 | 5067-5072 | Challenge validation |
| Enterprise Blobber 1-3 | 198.18.0.201-203 | 5071-5073 | Enterprise file storage |

#### Supporting Services

| Service | Port | Purpose |
|---------|------|---------|
| Elasticsearch | 9200 | Event indexing for 0box |
| Kafka | 9092 | Event broker (SASL_PLAINTEXT auth) |
| 0box | 9081 | File listing, queries, aggregates |
| zauth-server | 8080 | JWT authentication, split-key wallets |
| zvault | 8090 | Encrypted storage vault |
| zs3server (MinIO) | 9000 | S3-compatible gateway |
| zusCloudNative | 8088 | Datalake / Blimp backend |
| cAdvisor | 8080 | Container monitoring (optional) |

### Event Pipeline

```
Sharder → Kafka (198.19.0.99:9092, SASL) → 0box
```

The sharder publishes blockchain events to Kafka. 0box consumes and processes them into aggregates and snapshots. Both hardcode `SASL.Enable=true`, so Kafka must use SASL_PLAINTEXT authentication (credentials: `admin`/`admin-secret`).

---

## Scripts Reference

| Script | Purpose |
|--------|---------|
| `deploy_local.sh` | Main deployment orchestration |
| `run_tests.sh` | Test runner with parallel execution and auto-retry |
| `deploy_config.yaml` | Branch and service configuration |
| `deploy_nginx.sh` | Standalone nginx reverse proxy setup |
| `deploy_kafka.sh` | Standalone Kafka deployment with SASL auth |
| `init_chain.sh` | Chain initialization helpers |
| `fund_blobbers.sh` | Blobber funding helpers |
| `setup_loopback.sh` | macOS loopback alias setup |

### deploy_local.sh Commands

```bash
# === Deployment ===
./deploy_local.sh all                  # Full deployment (default)
./deploy_local.sh redeploy             # Clean all data then full deployment (clean + all)
./deploy_local.sh clean                # Clean all data + docker system prune (see below)
./deploy_local.sh clean-chain          # Clean chain data only (rocksdb, postgres, redis, logs)

# === Individual Phases ===
./deploy_local.sh checkout             # Checkout repo branches from config
./deploy_local.sh loopback             # Setup loopback aliases (macOS)
./deploy_local.sh config               # Setup ZCN config only
./deploy_local.sh start-chain          # Start miners, sharders, 0dns
./deploy_local.sh chain                # Initialize chain config (hardforks, SC config)
./deploy_local.sh services             # Start 0box, zauth, zvault, Kafka, etc.
./deploy_local.sh blobbers             # Start, fund, stake, configure blobbers
./deploy_local.sh fix-validators       # Fix validator config
./deploy_local.sh test-setup           # Copy wallets and CLI tools to test dirs

# === Image Management ===
./deploy_local.sh swap-image REPO [BRANCH] [--gosdk-branch BRANCH]

# === Testing ===
./deploy_local.sh test [ARGS]          # Run system tests
./deploy_local.sh smoke                # Quick infrastructure verification
./deploy_local.sh verify               # Health check all services

# === Maintenance ===
./deploy_local.sh reset-nonces         # Reset wallet nonces
./deploy_local.sh kafka-config         # Configure Kafka in sharder/0box configs
./deploy_local.sh cleanup-blobbers     # Kill stale blobbers from previous deploys
./deploy_local.sh cleanup-tests        # Clean stale Go build cache
./deploy_local.sh fund                 # Top up all provider balances
./deploy_local.sh fund-daemon          # Start background funding daemon (5 min interval)
./deploy_local.sh fund-daemon-stop     # Stop background funding daemon

# === Monitoring & Resilience ===
./deploy_local.sh monitor              # Start all monitoring (cAdvisor + VC + chaos)
./deploy_local.sh stop-monitor         # Stop all monitoring
./deploy_local.sh monitor-status       # Show monitoring status
./deploy_local.sh start-vc             # Start view change loop test
./deploy_local.sh stop-vc              # Stop view change loop test
./deploy_local.sh start-chaos          # Start chaos light resilience test
./deploy_local.sh stop-chaos           # Stop chaos light test
./deploy_local.sh start-dkg-monitor    # Start DKG monitoring
./deploy_local.sh stop-dkg-monitor     # Stop DKG monitoring
./deploy_local.sh start-cadvisor       # Start cAdvisor container monitoring
./deploy_local.sh stop-cadvisor        # Stop cAdvisor

# === Infrastructure ===
./deploy_local.sh nginx                # Setup nginx reverse proxy
./deploy_local.sh web-apps             # Build web-apps (Vult, Bolt, Blimp, etc.)
./deploy_local.sh rclone-zus           # Build rclone with Zus backend
./deploy_local.sh zs3-tools            # Setup mc/warp/hosts.yaml for zs3 tests

# === Database Admin ===
./deploy_local.sh pgadmin-blobber      # Start pgAdmin for blobber databases
./deploy_local.sh pgadmin-sharder      # Start pgAdmin for sharder event databases
./deploy_local.sh pgadmin-0box         # Start pgAdmin for 0box database
./deploy_local.sh pgadmin-zauth        # Start pgAdmin for zauth database
./deploy_local.sh pgadmin-zvault       # Start pgAdmin for zvault database

./deploy_local.sh help                 # Show help
```

### redeploy: Clean + Full Deploy in One Step

```bash
./deploy_local.sh redeploy
```

Equivalent to running `clean` followed by `all`. This is the recommended way to do a full fresh deployment when you want to start from scratch. It:
1. Runs `clean.sh --force` (stops containers, removes data, runs `docker system prune --all --volumes -f`)
2. Runs the full `all` deployment (checkout, build images, start chain, configure, deploy services)

### Image Auto-Build

After `clean` or `docker system prune`, all Docker images are removed. The deploy script automatically rebuilds images when they're missing. Each service function checks for its required images before starting:

| Service | Images Checked | Build Method |
|---------|---------------|--------------|
| Chain (miners/sharders) | `zchain_build_base`, `miner`, `sharder` | `docker.local/bin/build.base.sh` + `docker.local/bin/build.miners-integration-tests.sh` |
| Blobbers/Validators | `blobber_base`, `blobber`, `validator` | `docker.local/bin/build.base.sh` + `docker.local/bin/build.blobber-integration-tests.sh` |
| zauth-server | `zauthserver` | `docker build -f Dockerfile -t zauthserver` |
| zvault | `zvault` | `docker build -f Dockerfile -t zvault` |
| Enterprise blobber | `eblobber` | `docker.local/bin/build.sh` |

This means you can safely run `docker system prune` and then run any individual command (e.g., `./deploy_local.sh blobbers`) — the required images will be rebuilt automatically.

### Sourcing the Script (Advanced)

The deploy script supports being sourced to load its functions without executing any commands. This is useful for calling individual functions directly:

```bash
# Source the script (loads functions only, does NOT run deployment)
source scripts/deploy_local.sh

# Now call individual functions
start_zauth
start_zvault
fund_blobbers_and_validators
```

This works because the script has a `BASH_SOURCE` guard that detects when it's being sourced vs executed directly.

### Branch Override

```bash
# Override at command line (highest priority)
./deploy_local.sh --branch 0chain=fix/my-feature all

# Multiple overrides
./deploy_local.sh --branch 0chain=staging --branch blobber=fix/write-marker all
```

Branch resolution order:
1. `--branch` CLI flag (highest priority)
2. `deploy_config.yaml` branch field
3. Fallback: `master`

---

## Configuration

### deploy_config.yaml Structure

```yaml
# Base directory for all repositories
base_dir: ~/Code

# Per-repo branch configuration
repositories:
  0chain:
    branch: staging        # Git branch to checkout
    path: 0chain           # Directory name under base_dir
    build_required: true   # Whether to build Docker images
    containers:            # Docker containers using this image
      - miner-1
      - sharder-1

  web-apps:
    branch: master
    gosdk_branch: staging  # gosdk branch for WASM build

  eblobber:
    branch: staging
    gosdk_branch: staging  # gosdk branch for Go dependency

  zs3server:
    branch: feat/enterprise-timings
    gosdk_branch: enterprise-blobber  # gosdk branch for Go dependency

# Chain configuration
chain_config:
  global_config:
    max_block_cost: 100000   # Reduce transaction fees for testing

# Blobber settings
blobber_config:
  read_price: 0.00           # Required for free allocation tests
  write_price: 0.10
  funding_tokens: 10         # ZCN per blobber
  stake_tokens: 1

# Enterprise blobbers
enterprise_blobbers:
  enabled: true
  count: 3

# Kafka (required for event pipeline)
kafka:
  host: "198.19.0.99:9092"
  username: "admin"
  password: "admin-secret"

# Firebase (for 0box API auth tests)
firebase:
  api_key: "..."
  email: "test_system_test@0chain.net"
  password: "..."

# Nginx reverse proxy (optional)
nginx:
  domain: ""                 # Set to enable (e.g., "test.zus.network")
  enable_ssl: true
```

### Key Configuration Files

| File | Purpose |
|------|---------|
| `~/.zcn/local.yaml` | SDK config (block_worker: `http://localhost:9091`) |
| `~/.zcn/local.json` | SC owner wallet (private key) |
| `tests/*/config/zbox_config.yaml` | Test SDK config (Docker IPs) |
| `tests/*/config/wallets/sc_owner_wallet.json` | SC owner wallet for tests |
| `tests/*/config/wallets/blobber_owner_wallet.json` | Blobber owner wallet |
| `deploy_config.yaml` | Branches and deployment settings |

### Wallet Configuration

The SC owner wallet must match across:
1. On-chain SC owner (set at genesis in 0chain's `sc.yaml`)
2. `~/.zcn/local.json` (for CLI admin operations)
3. `tests/*/config/wallets/sc_owner_wallet.json` (for test operations)

Verify on-chain owner:
```bash
curl -s localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config | jq .fields.owner_id
```

---

## Common Workflows

### Deploy Fresh and Run Tests

```bash
./deploy_local.sh all
./deploy_local.sh verify
./deploy_local.sh test
```

### Test a Feature Branch (Chain Only)

```bash
./deploy_local.sh swap-image 0chain fix/my-feature
./deploy_local.sh test
```

### Test a Blobber Fix

```bash
./deploy_local.sh swap-image blobber fix/write-marker
./run_tests.sh --filter TestUpload cli
```

### Update Web Apps and View in Browser

```bash
# Build with WASM from a specific gosdk branch
./deploy_local.sh swap-image web-apps master --gosdk-branch staging

# Start the development server
cd ~/Code/web-apps/packages/vult
echo "NEXT_PUBLIC_BLOCK_WORKER=http://localhost:9091" > .env.local
npm run dev

# Open http://localhost:3000 in your browser
```

### Re-Run Only Failed Tests

```bash
./run_tests.sh --filter "TestReplaceBlobber|TestCreateAllocation" --retries 3 api
```

### Monitor Provider Balances

```bash
# One-time check and top-up
./deploy_local.sh fund

# Background daemon (checks every 5 minutes)
./deploy_local.sh fund-daemon
```

---

## Troubleshooting

### Chain Stuck After Restart

If miners' LFB is ahead of sharders, the chain deadlocks. Fix by resetting miners' RocksDB config:

```bash
docker stop miner-1 miner-2 miner-3 miner-4
for i in 1 2 3 4; do
  dir=~/Code/0chain/docker.local/miner$i/data/rocksdb
  mv $dir/config $dir/config_backup_$(date +%Y%m%d)
done
docker start miner-1 miner-2 miner-3 miner-4
```

### Blobbers Not Registering

Check blobber balance - they need ZCN to pay registration fees:

```bash
./deploy_local.sh blobbers  # Re-funds and re-stakes
```

### SC Owner Wallet Mismatch

If admin operations fail with "unauthorized access":

```bash
# Check on-chain owner
curl -s localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config | jq .fields.owner_id

# Compare with local wallet
cat ~/.zcn/local.json | jq .client_id
```

### Tests Failing with "Unexpected End of JSON"

Chain may need higher `max_block_cost`:

```bash
zwallet global-update-config --keys 'server_chain.block.max_block_cost' --values '200000' \
  --wallet local.json --configDir ~/.zcn --config local.yaml
```

### WASM Build Fails

If `make wasm-build` fails in Docker:

```bash
# Check Go version
go version  # Should be 1.22+

# Try native build
cd ~/Code/gosdk
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -ldflags="-s -w" -buildvcs=false -o ./zcn.wasm ./wasmsdk

# Verify output
ls -lh zcn.wasm  # Should be ~29 MB
```

### Web App Can't Connect to Network

1. Verify 0dns is running: `curl http://localhost:9091/dns/network`
2. Check `.env.local` has correct `NEXT_PUBLIC_BLOCK_WORKER`
3. Check browser console for WASM initialization errors
4. Verify `zcn.wasm` is accessible: open `http://localhost:3000/zcn.wasm` in browser

### Kafka Event Pipeline Stalled

If 0box isn't processing events:

```bash
# Check Kafka is running
docker logs kafka 2>&1 | tail -20

# Check sharder is pushing events
docker logs sharder-2 2>&1 | grep -i kafka | tail -10

# Check 0box is consuming
docker logs 0box 2>&1 | grep -i kafka | tail -10
```

Both sharder and 0box hardcode `SASL.Enable=true`. Ensure Kafka JAAS config matches credentials in sharder/0box YAML.

### Enterprise Blobber Tests Skipping

Enterprise tokenomics tests skip if no enterprise blobbers are found on-chain:

```bash
# Check if enterprise blobbers are registered
curl -s 'localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers' | jq '.[] | select(.is_enterprise==true) | .id'
```

If none found, deploy them:

```bash
./deploy_local.sh blobbers  # Includes enterprise blobber deployment
```

---

## Developer Workflow

This section documents the end-to-end cycle developers follow when working with 0Chain: clean deploy, code changes, image swaps, and test runs.

### Workflow A: Fresh Deployment from Scratch

Use this when you need a completely clean environment (new genesis, all data wiped).

```bash
cd ~/Code/system_test/scripts

# Option 1: Single command (recommended)
./deploy_local.sh redeploy

# Option 2: Step-by-step (more control)
./deploy_local.sh clean
./deploy_local.sh all

# Verify and test
./deploy_local.sh verify
./deploy_local.sh test
```

**What `clean` does:**
- Stops all Docker containers (miners, sharders, blobbers, services)
- Kills background processes (funding_daemon, vc_test, chaos_test)
- Removes all data directories: RocksDB (`config/`, `state/`, `dkg/`, `dkgkey/`, `mb/`), PostgreSQL data, blobber file storage
- Removes tablespace directories (`ssd/`, `hdd/`) for sharders and blobbers
- Runs `docker system prune --all --volumes -f` to remove all unused images, networks, volumes, and build cache
- Preserves: keys, config templates, source code
- After `clean`, the next `all` will rebuild all Docker images from source (chain, blobber, zauth, zvault, etc.)

**What `all` does (in order):**
1. `checkout` - Clone missing repos, checkout configured branches
2. `loopback` - Setup macOS loopback aliases (198.18.0.x IPs)
3. `config` - Generate `~/.zcn/local.yaml` and `~/.zcn/local.json`
4. `start-chain` - Build and start miners/sharders/0dns, wait for blocks
5. `chain` - Apply hardforks, SC config (`min_alloc_size`, `max_block_cost`, etc.)
6. `services` - Start Elasticsearch, Kafka, zauth, zvault, 0box, zusCloudNative
7. `blobbers` - Build and start blobbers/validators, fund, register, stake
8. `test-setup` - Copy wallets and CLI binaries to test directories
9. `monitor` - Start VC test, chaos test, cAdvisor

### Workflow B: Hot-Swap a Service Branch (No Redeploy)

Use this when you're developing on a service and want to test your branch changes without a full redeploy. Chain state and on-chain data are preserved.

```bash
# Step 1: Swap the image for the service you changed
./deploy_local.sh swap-image <REPO> [BRANCH]

# Examples:
./deploy_local.sh swap-image 0chain fix/dkg-broadcast   # Test a chain fix
./deploy_local.sh swap-image blobber fix/write-marker    # Test a blobber fix
./deploy_local.sh swap-image 0box staging                # Rebuild 0box
./deploy_local.sh swap-image zboxcli fix/cli-output      # Rebuild CLI binary

# Step 2: Verify the swap worked
./deploy_local.sh verify

# Step 3: Run relevant tests (targeted, not full suite)
./run_tests.sh --filter "TestUpload|TestDownload" cli
```

**What `swap-image` does:**
1. Checks out the specified branch (or the branch from `deploy_config.yaml`)
2. Rebuilds the Docker image from source
3. Stops the affected containers
4. Recreates containers with the new image
5. Chain data, on-chain registrations, and wallet state are all preserved

### Workflow C: Full Deploy with Custom Branches

Use this when you want to deploy a specific combination of branches from scratch.

```bash
# Deploy with custom branches for multiple repos
./deploy_local.sh clean
./deploy_local.sh --branch 0chain=fix/my-chain-fix --branch blobber=fix/my-blobber-fix all
./deploy_local.sh verify
./deploy_local.sh test
```

### Workflow D: Iterative Test-Fix-Retest Cycle

Use this during active development when fixing test failures.

```bash
# Step 1: Run only the specific failing tests (don't run full suites)
cd ~/Code/system_test/tests/cli_tests
go test -v -run "^TestUpload$" -timeout 30m .

# Step 2: Make code changes (in system_test repo or the service repo)
# Edit your test code or service code...

# Step 3a: If you changed TEST CODE only, just re-run
go test -v -run "^TestUpload$" -timeout 30m .

# Step 3b: If you changed a SERVICE, swap the image first
cd ~/Code/system_test/scripts
./deploy_local.sh swap-image blobber fix/my-fix
cd ~/Code/system_test/tests/cli_tests
go test -v -run "^TestUpload$" -timeout 30m .

# Step 3c: If you changed CLI binaries (zbox/zwallet), rebuild and copy
cd ~/Code/zboxcli && make build
cd ~/Code/system_test/scripts
./deploy_local.sh swap-image zboxcli  # Copies binary to test dirs
cd ~/Code/system_test/tests/cli_tests
go test -v -run "^TestUpload$" -timeout 30m .
```

### Workflow E: Remote Server Testing

When running tests on a remote server, sync code changes and run specific tests.

```bash
# Sync test code changes to remote (exclude platform-specific binaries)
rsync -avz --exclude 'zbox' --exclude 'zwallet' --exclude '.git' \
  ~/Code/system_test/tests/cli_tests/ root@REMOTE:/root/Code/system_test/tests/cli_tests/

# On remote: restore Linux binaries (rsync from Mac would overwrite with Mach-O)
ssh root@REMOTE "cp /root/Code/zboxcli/zbox /root/Code/system_test/tests/cli_tests/zbox && \
                  cp /root/Code/zwalletcli/zwallet /root/Code/system_test/tests/cli_tests/zwallet"

# On remote: run specific failing tests
ssh root@REMOTE "cd /root/Code/system_test/tests/cli_tests && \
  go test -v -run '^TestUpload$' -timeout 30m ."
```

### Quick Reference: Which Workflow to Use

| Scenario | Workflow |
|----------|----------|
| First time setup | A (Fresh Deploy) |
| Testing a feature branch | B (Hot-Swap) or C (Full Deploy with `--branch`) |
| After chain config change | A (need fresh genesis) |
| After test code change | D (just re-run tests) |
| After service code change | B (swap-image) then D (re-run tests) |
| Debugging intermittent failure | D (re-run specific test with `-count 3`) |
| Full CI-like test run | `./deploy_local.sh test` or `./run_tests.sh` |

---

## Monitoring, Chaos & View Change Testing

The deploy script includes built-in support for resilience testing and monitoring. These run as background processes and can be started/stopped independently.

### Overview

| Component | What It Does | Log File |
|-----------|-------------|----------|
| **View Change (VC) Test** | Continuously kills and restarts miners/sharders to test view change recovery | `/tmp/view_change_loop.log` |
| **Chaos Test** | Randomly stops/starts sharders to test resilience | `/tmp/chaos_test.log` |
| **cAdvisor** | Docker container resource monitoring (CPU, memory, network) | Web UI at `http://localhost:8080` |

### Start All Monitoring

```bash
# Start everything (cAdvisor + VC + chaos)
./deploy_local.sh monitor

# Check status
./deploy_local.sh monitor-status

# Stop everything
./deploy_local.sh stop-monitor
```

### Start Individual Components

```bash
# View Change test only
./deploy_local.sh start-vc
./deploy_local.sh stop-vc

# Chaos test only
./deploy_local.sh start-chaos
./deploy_local.sh stop-chaos

# cAdvisor only
./deploy_local.sh start-cadvisor
./deploy_local.sh stop-cadvisor
```

### Monitoring During Full Deployment

When you run `./deploy_local.sh all`, monitoring (VC + chaos + cAdvisor) is started automatically at the end of deployment. The `clean` command stops all monitoring processes.

### View Change (VC) Test Details

The VC test uses `0chain/docker.local/bin/vc.sh` which:
1. Picks a miner and sharder by ID
2. Kills one miner (docker stop), waits for recovery
3. Verifies the chain continues producing blocks
4. Restarts the miner, verifies it rejoins the network
5. Repeats for sharder
6. Loops continuously

The deploy script adapts `vc.sh` to use the correct wallet path, config, and provider IDs from the current deployment.

### Chaos Test Details

The chaos test uses `0chain/docker.local/bin/chaos.sh` which:
1. Randomly selects a sharder from the active set
2. Stops it (docker stop)
3. Waits a configurable interval
4. Restarts it
5. Ensures at least `MIN_SHARDERS_RUNNING` sharders remain active
6. Loops continuously

The deploy script adapts the script for the current number of active sharders.

### Running Tests During Monitoring

You can run system tests while VC/chaos tests are active. This validates that tests pass even under adverse conditions (provider restarts). However, some tests may fail transiently due to timing, so it's recommended to:

1. Run tests **without** monitoring first to establish a passing baseline
2. Then enable monitoring and re-run to validate resilience

---

## Kill/Shutdown Tests (Destructive)

The system test suite includes tests that **permanently kill or shutdown** on-chain service providers. These tests are **disabled by default** because they are destructive - killed providers cannot be restored without a fresh deployment.

### Running Kill/Shutdown Tests

```bash
# Enable kill tests via environment variable
cd ~/Code/system_test/tests/cli_tests
ENABLE_KILL_TESTS=1 go test -v -run "^TestKillBlobber$" -timeout 30m .
ENABLE_KILL_TESTS=1 go test -v -run "^TestKillSharder$" -timeout 30m .
ENABLE_KILL_TESTS=1 go test -v -run "^TestKillMiner$" -timeout 30m .

# Run ALL kill tests at once (destructive!)
ENABLE_KILL_TESTS=1 go test -v -run "^TestKill" -timeout 60m .
```

### Available Kill/Shutdown Tests

#### TestKillBlobber (`tests/cli_tests/zboxcli_kill_blobber_test.go`)

| Subtest | What It Does |
|---------|-------------|
| `killed blobber is not available for allocations` | Kills a blobber, verifies ~90% stake slashing, confirms killed blobber can't be used in new allocations |
| `kill blobber by non-smartcontract owner should fail` | Authorization check - only SC owner can kill |
| `shutdowned blobber is not available for allocations` | Shuts down a blobber (softer than kill), verifies ~45% stake slashing |
| `shutdown blobber by non-smartcontract owner should fail` | Authorization check for shutdown |

**Requirements:**
- Minimum 6 active blobbers (4 for other tests + 2 to kill/shutdown)
- SC owner wallet (`sc_owner_wallet.json`)
- Test wallet with 10+ ZCN

**CLI Commands Used:**
- `./zbox kill-blobber --id <blobber_id> --wallet sc_owner_wallet.json`
- `./zbox shutdown-blobber --id <blobber_id> --wallet sc_owner_wallet.json`

#### TestKillSharder (`tests/cli_tests/zzwalletcli_kill_sharder_test.go`)

| Subtest | What It Does |
|---------|-------------|
| `kill sharder by non-smartcontract owner should fail` | Authorization check |
| `Killed sharder does not receive rewards` | Kills a sharder, verifies TotalReward stops increasing |

**CLI Command:** `./zwallet sh-kill --id <sharder_id> --wallet sc_owner_wallet.json`

#### TestKillMiner (`tests/cli_tests/zzzwalletcli_kill_miner_test.go`)

| Subtest | What It Does |
|---------|-------------|
| `kill miner by non-smartcontract owner should fail` | Authorization check |
| `Killed miner does not receive rewards` | Kills a miner, verifies TotalReward stops increasing |

**CLI Command:** `./zwallet mn-kill --id <miner_id> --wallet sc_owner_wallet.json`

### Missing Kill/Shutdown Coverage

| Provider | Kill | Shutdown | Notes |
|----------|------|----------|-------|
| Blobber | Yes | Yes | Full coverage including stake slashing verification |
| Sharder | Yes | No | No shutdown variant for sharders |
| Miner | Yes | No | No shutdown variant for miners |
| **Validator** | **No** | **No** | No kill/shutdown tests exist |
| **Authorizer** | **No** | **No** | No kill/shutdown tests exist |

### Important Notes

1. **Disabled by default**: Tests skip unless `ENABLE_KILL_TESTS=1` is set
2. **Destructive**: Killed providers are permanently removed from the network. After running these tests, you likely need a fresh deployment (`clean` + `all`)
3. **Run LAST**: File naming convention ensures these run after all other tests:
   - `zboxcli_kill_blobber_test.go` (normal alphabetical order)
   - `zzwalletcli_kill_sharder_test.go` (zz prefix - runs late)
   - `zzzwalletcli_kill_miner_test.go` (zzz prefix - runs last)
4. **SC owner required**: All kill/shutdown operations require the SC owner wallet
5. **Blobber kill vs shutdown**: Kill applies full `stakepool.kill_slash` (~90%), shutdown applies half (~45%)
