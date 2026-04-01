# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 🚨 MANDATORY PROCESS FOR ALL FIXES (4 PHASES — MUST FOLLOW)

**Every fix MUST go through these 4 phases. Report and ask for permission after EACH phase.**

1. **ROOT CAUSE ANALYSIS**: Diagnose the actual root cause. Understand WHY it broke, not just what broke. Check logs, trace the code path, identify the exact failure point. REPORT findings and ask permission to proceed.

2. **PERMANENT FIX**: Change the script, test code, or config that caused the issue. Never apply manual server-side patches. REPORT the exact code changes made and ask permission to proceed.

3. **APPLY AND TEST**: Rsync code to server, run the relevant deploy command, verify the fix works by checking logs and system state. REPORT test results and ask permission to proceed.

4. **COMMIT AND PUSH**: Only after successful testing, commit the changes with a clear message explaining the root cause and fix. REPORT the commit and ask permission to push.

---

## 🚨 ROOT CAUSE ANALYSIS: Why Recurrent Deployment Issues Happen

**READ THIS FIRST.** Every time blobbers fail to register, validators have no stake, or tokens are missing — the root cause is nearly always one of these:

### Root Cause 1: Fund script used stale `.txt.json` keys instead of `.txt` keys
- Blobber key files come in two forms: `b0bnode10_keys.txt` (authoritative, updated by build scripts) and `b0bnode10_keys.txt.json` (stale JSON copy, may have DIFFERENT keys).
- The old `fund_blobbers_and_validators()` function tried `.txt.json` BEFORE computing from `.txt`, so it funded the WRONG wallet.
- **Fix**: `fund_blobbers_and_validators()` now computes SHA3-256(pub_key from `.txt`) first, falling back to `.json` only if `.txt` doesn't exist.

### Root Cause 2: Circular dependency — blobbers need ZCN to register but daemon only funds REGISTERED blobbers
- `auto_fund_daemon.sh` funded blobbers by querying the chain REST API for registered blobbers.
- New blobbers need ZCN to register. They can't register without ZCN. The daemon never sees them. They never get funded. They never register.
- **Fix**: `auto_fund_daemon.sh` now has `fund_container_blobbers()` which checks RUNNING CONTAINERS and funds their wallets using the `.txt` key file — BEFORE they appear on chain.

### Root Cause 3: Daemon did not stake validators
- `auto_fund_daemon.sh` only funded wallets, never staked providers.
- After initial deployment, validators had 0 stake → no challenges generated → 0 token rewards.
- The manual workaround was `ensure-config` or `stake-blobbers`. This was NOT automated.
- **Fix**: `auto_fund_daemon.sh` now has `stake_unstaked_providers()` which stakes blobbers (10 ZCN) and validators (20 ZCN) that are registered but have 0 stake. This runs every 2-minute cycle.

### Root Cause 4: `blobbers` deploy case called `fund_blobbers_and_validators` with wrong `.txt.json` priority
- Same as Root Cause 1. The deploy script `blobbers` case funded wallets but used stale key files for blobbers 10-12 when their key files were regenerated.
- **Fix**: Already fixed in `fund_blobbers_and_validators()` — `.txt` file hash computation now comes before `.txt.json` fallback.

### The Rule: NEVER apply manual fixes. If something breaks, fix the script.
- **Wrong**: `ssh server zwallet send --to_client_id <id> --tokens 100` (one-off manual send)
- **Right**: Fix `fund_blobbers_and_validators()` or `auto_fund_daemon.sh` so it funds the right wallets automatically
- **Wrong**: `ssh server zbox sp-lock --validator_id <id>` (one-off manual stake)
- **Right**: Fix `stake_unstaked_providers()` in `auto_fund_daemon.sh` to stake automatically
- Every manual fix will break again on the next deployment. Script fixes are permanent.

---

## Critical Workflow Rule: Deploy Script / Test Code First — NEVER Manual Fixes

**ALL fixes MUST be made to either the test code or `scripts/deploy_local.sh` or `scripts/auto_fund_daemon.sh`.** Never make manual one-off changes directly on the remote server. This ensures:
- Changes are reproducible and survive redeployments
- The deploy script remains the single source of truth
- Fixes don't get lost when the environment is rebuilt

**EXAMPLES OF WHAT NOT TO DO (manual one-off):**
- Running `ssh ... zwallet mn-update-config --keys ...` directly on the server
- Editing config files on the server via ssh without updating deploy_local.sh
- Copying wallet files on the server manually without updating deploy_local.sh

**EXAMPLES OF WHAT TO DO (permanent code fixes):**
- If a blobber delegate_wallet mismatch causes tests to fail → fix `setup_test_wallets()` in deploy_local.sh
- If a chain config value needs to change → fix `init_chain_config()` or `ensure_chain_config()` in deploy_local.sh
- If a test assertion is wrong → fix the test file in `tests/`

Workflow: Edit test code or deploy script locally → rsync to remote → run the appropriate deploy script command on remote.

## Project Overview

0Chain System Tests - A black/grey box integration test suite that tests the 0Chain blockchain network functionality as an end user. Tests validate storage, wallets, allocations, and tokenomics features.

## Common Commands

### Running Tests

```bash
# Run all API tests
cd tests/api_tests && go test ./... -v

# Run all CLI tests (requires zbox, zwallet binaries in tests/cli_tests/)
cd tests/cli_tests && go test ./... -v

# Run SDK tests
cd tests/sdk_tests && go test ./... -v

# Run tokenomics tests
cd tests/tokenomics_tests && go test ./... -v

# Run a specific test by name
go test -run TestCreateWallet ./... -v

# Run tests excluding known broken features (CLI tests)
go test -run "^Test[^___]*$" ./... -v

# Enable debug logging
DEBUG=true go test ./... -v

# Run with custom config
CONFIG_PATH=/path/to/config.yaml go test ./... -v

# Run smoke tests only
SMOKE_TEST_MODE=true go test ./... -v
```

### Linting

```bash
golangci-lint run --timeout 5m
```

### Testing Approach & Debugging

See **[scripts/docs/run-tests.md](scripts/docs/run-tests.md)** for the full testing guide, including:
- Step-by-step workflow (compile locally → sync → run targeted tests)
- Common failure patterns and how to diagnose them
- When to skip vs when to fix
- Suite-specific notes and debugging checklist

## Architecture

### Test Suites

- **`tests/api_tests/`** - REST API endpoint and smart contract tests (47 files)
- **`tests/cli_tests/`** - CLI tool tests for zbox, zwallet, minio (80+ files)
- **`tests/sdk_tests/`** - Go SDK integration tests
- **`tests/tokenomics_tests/`** - Financial/tokenomics tests (16 files)

### Internal Libraries (`internal/`)

- **`internal/api/model/`** - Data models (Wallet, SdkWallet, Allocation, Blobber, etc.)
- **`internal/api/util/client/`** - HTTP clients (APIClient, SDKClient, ZboxClient, ZvaultClient, ZauthClient)
- **`internal/api/util/test/`** - `SystemTest` wrapper providing timeouts, logging, and test lifecycle
- **`internal/api/util/crypto/`** - BLS signatures and cryptographic operations
- **`internal/cli/`** - CLI-specific utilities and models

### Test Framework Pattern

Tests follow this structure:
```go
func TestMain(m *testing.M) {
    // Global setup: load config, create clients
}

func TestFeatureName(testSetup *testing.T) {
    t := test.NewSystemTest(testSetup)
    t.SetSmokeTests("SubtestName1", "SubtestName2")

    t.RunSequentially("SubtestName", func(t *test.SystemTest) {
        // Test code with t.Require(), t.Assert()
    })

    t.RunWithTimeout("SubtestWithTimeout", 5*time.Minute, func(t *test.SystemTest) {
        // Test with custom timeout
    })
}
```

### Configuration

- **API tests**: `tests/api_tests/config/api_tests_config.yaml`
- **CLI tests**: `tests/cli_tests/config/zbox_config.yaml`, `config/nodes.yaml`
- **Pre-generated wallets**: `tests/*/config/wallets.json`

### Key Dependencies

- `github.com/0chain/gosdk` - Official 0Chain Go SDK
- `github.com/stretchr/testify` - Testing assertions
- `github.com/go-resty/resty/v2` - HTTP client
- `github.com/herumi/bls-go-binary` - BLS cryptography

## Remote Server: Stale go.mod Files

The remote server has remnants of the old repo structure where each test suite was a separate Go module. These stale `go.mod` files create module boundaries that prevent Go from resolving `internal/` packages:

- **`internal/go.mod`** - The `internal/` directory is a separate git clone of the entire repo. Its `go.mod` creates a module boundary that hides `internal/api/`, `internal/cli/`, etc. from the root module.
- **`tests/api_tests/go.mod`** and **`tests/cli_tests/go.mod`** - Old per-suite module files from when each test suite was an independent module.

**Fix**: Rename these to `.bak` so Go uses only the root `go.mod`:
```bash
mv internal/go.mod internal/go.mod.bak
mv internal/go.sum internal/go.sum.bak
mv tests/api_tests/go.mod tests/api_tests/go.mod.bak
mv tests/cli_tests/go.mod tests/cli_tests/go.mod.bak
# Same for go.sum files
```

**Symptom**: `no required module provides package github.com/0chain/system_test/internal/api/util/config`

## Deploy Script (`scripts/deploy_local.sh`)

The deploy script manages the full 0Chain test environment on a remote server. It handles chain deployment, blobbers, supporting services, web apps, and test execution.

### Quick Reference

```bash
# Full clean redeploy (wipe everything, start from scratch ~45 min)
bash scripts/deploy_local.sh redeploy

# Full deploy without cleaning (reuse existing chain data)
bash scripts/deploy_local.sh all

# Deploy with custom branch
bash scripts/deploy_local.sh --branch 0chain=fix/my-branch all
```

### Swap a Single Service Image

Rebuild and restart just one service without full redeploy. See **[scripts/docs/swap-image.md](scripts/docs/swap-image.md)** for full reference.

```bash
bash scripts/deploy_local.sh swap-image 0chain fix/my-branch           # miners + sharders
bash scripts/deploy_local.sh swap-image blobber feat/my-feature         # all blobbers + validators
bash scripts/deploy_local.sh swap-image eblobber staging --gosdk-branch enterprise-blobber  # enterprise blobbers
bash scripts/deploy_local.sh swap-image 0box transcoder-updated        # 0box service
bash scripts/deploy_local.sh swap-image web-apps player-fmp4           # all 5 web apps + WASM
bash scripts/deploy_local.sh swap-image web-apps master --gosdk-branch fix/sdk  # custom gosdk for WASM
bash scripts/deploy_local.sh swap-image zauth-server staging           # zauth
bash scripts/deploy_local.sh swap-image zvault staging                 # zvault
```

### Incremental Commands

```bash
bash scripts/deploy_local.sh web-apps      # Build & start web apps (see docs/web-apps-update-plan.md)
bash scripts/deploy_local.sh services      # Restart 0box, zauth, zvault
bash scripts/deploy_local.sh blobbers      # Restart blobbers, fund, stake, configure
bash scripts/deploy_local.sh chain         # Reconfigure SC settings and hardforks
bash scripts/deploy_local.sh test-setup    # Refund wallets, copy CLI tools
bash scripts/deploy_local.sh verify        # Health check all services
bash scripts/deploy_local.sh nginx         # Setup nginx reverse proxy with SSL
```

### Run Tests

```bash
bash scripts/deploy_local.sh test                  # All suites with auto-retry
bash scripts/deploy_local.sh test api              # API tests only
bash scripts/deploy_local.sh test cli              # CLI tests only
bash scripts/deploy_local.sh test tokenomics       # Tokenomics tests only
bash scripts/deploy_local.sh test sdk              # SDK tests only
bash scripts/deploy_local.sh test api cli          # API + CLI together
bash scripts/deploy_local.sh test --retries 3      # 3 retries per failing suite
bash scripts/deploy_local.sh test --filter TestName          # Run only matching tests
bash scripts/deploy_local.sh test cli --filter TestBlobber   # Specific test in one suite
bash scripts/deploy_local.sh smoke                 # Quick infrastructure smoke test
```

You can also call the test runner directly:

```bash
bash scripts/run_tests.sh                      # All suites
bash scripts/run_tests.sh api                  # API only
bash scripts/run_tests.sh cli --timeout 60m    # CLI with custom timeout
bash scripts/run_tests.sh --test-timeout 5m    # Custom per-test timeout
```

### Branch Selection

When no branch is specified, `swap-image` and all commands resolve branches in this order:

1. **CLI flag**: `--branch 0chain=fix/my-branch` (highest priority)
2. **`scripts/deploy_config.yaml`**: reads `repositories → <repo> → branch`
3. **Fallback**: `master`

Current defaults in `deploy_config.yaml`:

| Repo | Default Branch |
|---|---|
| 0chain | `fix/dkg-broadcast-fee` |
| blobber | `staging` |
| 0box | `staging` |
| eblobber | `staging` |
| gosdk | `staging` |
| web-apps | `master` |

Override examples:
```bash
bash scripts/deploy_local.sh swap-image 0box fix/my-feature       # branch as argument
bash scripts/deploy_local.sh --branch 0box=fix/my-feature all     # --branch flag
# Or edit scripts/deploy_config.yaml directly
```

### Other Commands

```bash
bash scripts/deploy_local.sh fund             # Top up all provider balances
bash scripts/deploy_local.sh fix-kafka        # Fix Kafka pipeline (config + seed + restart)
bash scripts/deploy_local.sh seed-explorer    # Seed 0box provider tables for Explorer
bash scripts/deploy_local.sh cleanup-blobbers # Kill stale blobbers from previous deploys
```

### rsync to Remote Server

```bash
# Sync code (exclude Mac-only binaries + server-specific config that has per-server domain)
sshpass -p 'MC36hG7d4puCr%' rsync -avz \
  --exclude 'zbox' --exclude 'zwallet' \
  --exclude 'mc' --exclude 'warp' --exclude 'rclone-zus' \
  --exclude '.git' \
  --exclude 'scripts/deploy_config.yaml' \
  /Users/saswatabasu/Code/system_test/ root@37.27.65.188:/root/Code/system_test/

# Restore Linux CLI binaries after rsync (zbox/zwallet + mc/warp via test-setup)
ssh root@37.27.65.188 "cp /root/Code/zboxcli/zbox /root/Code/system_test/tests/cli_tests/zbox && \
  cp /root/Code/zwalletcli/zwallet /root/Code/system_test/tests/cli_tests/zwallet"

# Re-run test-setup after rsync to ensure Linux mc/warp/rclone-zus are in place
ssh root@37.27.65.188 "cd /root/Code/system_test && bash scripts/deploy_local.sh test-setup"
```

## Known Infrastructure Issues

See **[scripts/docs/infrastructure-fixes.md](scripts/docs/infrastructure-fixes.md)** for details on:
- **0box Redis SIGSEGV**: `redis:alpine` (v8+) crashes; pinned to `redis:7.4.3-alpine`
- **zvault postgres connection**: Config ships with `host: localhost`; must be `postgreszv` (compose service name)
- **eblobber block_worker**: Config ships with `dev.0chain.net`; must be `http://198.18.0.100:9091`
- **eblobber port conflicts**: Must use `eb0docker-compose.yml` (not `b0docker-compose.yml`)

All fixes are applied automatically by `fix_blobber_config()` and `swap-image` restart logic.

## 0Chain Smart Contract REST API Endpoints

All SC REST endpoints are served by **sharders** (not miners). Base URL for local: `http://198.18.0.81:7171`

### Smart Contract Addresses

| Smart Contract | Address (suffix) | Full Address |
|---|---|---|
| Storage SC | `...d7` | `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7` |
| Miner SC | `...d9` | `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9` |
| Faucet SC | `...d3` | `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3` |
| Vesting SC | N/A | `2bba5b05949ea59c80aed3ac3474d7379d3be737e8eb5a968c52295e48333ead` |
| ZCN SC | `...e0` | `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0` |

### Key Diagnostic Endpoints (local URLs)

```bash
# Chain diagnostics
curl http://198.18.0.81:7171/_diagnostics          # Sharder-1 diagnostics
curl http://198.18.0.81:7171/_chain_stats           # Block finalization stats
curl http://198.18.0.81:7171/v1/config/get          # Sharder config (kafka, dbs, etc.)

# Latest finalized magic block (view change count)
curl http://198.18.0.81:7171/v1/block/get/latest_finalized_magic_block

# Miner SC: configs, hardforks, DKG phase
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork?name=apollo"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList"

# Storage SC: blobbers, allocations, challenges
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber?blobber_id=<ID>"
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges?blobber=<ID>"
```

## SC Config Updates via CLI

```bash
# Storage SC settings (from SC owner wallet)
zwallet sc-update-config --keys "min_alloc_size,min_write_price,challenge_enabled" --values "1024,0,true" --configDir /root/.zcn --wallet local.json

# Miner SC settings (max_delegates, etc.)
zwallet mn-update-config --keys max_delegates --values 200 --configDir /root/.zcn --wallet local.json

# Global chain settings (fees, block size)
zwallet global-update-config --keys server_chain.transaction.cost_fee_coeff --values 100000 --configDir /root/.zcn --wallet local.json

# View configs
curl "http://198.18.0.81:7171/v1/screst/...d7/storage-config"   # Storage SC
curl "http://198.18.0.81:7171/v1/screst/...d9/configs"           # Miner SC
```

### Important SC Config Notes

- **`min_read_price` does NOT exist** — SC returns "unknown key"
- **`min_write_price`**: defaults to `0.001`, override to `0` for tests
- **`min_alloc_size`**: defaults to `1048576`, override to `1024` for tests
- **`max_block_cost`**: set to `100000` to avoid "max block cost exceeded" errors

## Firebase / App Login Fix

See **[scripts/docs/firebase-login-fix.md](scripts/docs/firebase-login-fix.md)** for the full auth flow (Firebase ID Token → CSRF → JWT), Firebase Console setup steps, and troubleshooting.

## 0box Blobber Data and Free Allocations

The 0box `blobbers` table is populated by Kafka health check events (every 90 minutes). Until the first health check after 0box starts, the table is empty. This causes:
- Blimp "no provider match the range" error when creating allocations
- Empty blobber lists on Explorer and Bolt

The deploy script's `seed_0box_blobbers()` function bootstraps the table immediately by copying from the sharder's events_db. Key requirements for blobbers to appear in allocation queries:
- `not_available = false` (NULL is excluded by SQL WHERE)
- `brand_id` must match a row in `provider_brand` table (INNER JOIN)
- `last_health_check` must be within the last hour
- `blobber_type = 'HotMinus'` (default)
- `is_enterprise = false` for free/regular allocations

## Prerequisites for Running Tests

1. Go 1.22.0+
2. A running 0Chain network
3. For CLI tests: zbox, zwallet binaries copied to `tests/cli_tests/`
4. Configuration files pointing to the network
5. Pre-funded wallets on the test network

## CI/CD

The main pipeline (`.github/workflows/ci.yml`) can:
- Deploy a new 0Chain network with custom docker images
- Run tests against an existing network
- Generate HTML test reports

Manual triggers support:
- `test_file_filter` - Run specific test files
- `run_smoke_tests` - Run fast subset
- `existing_network` - Test against pre-deployed network
