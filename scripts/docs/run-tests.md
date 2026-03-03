# Running Tests — Approach & Reference

This document describes how to approach running, debugging, and fixing the 0Chain system tests. It is intended for both human developers and AI assistants working on this codebase.

---

## Golden Rules

1. **NEVER run broad test suites on the remote server** (e.g. `go test ./... -v`). Always run specific tests by name.
2. **Identify failures from existing logs first**, then fix and retest individual tests.
3. **Test locally first** (`go vet`, compilation check) before syncing to remote.
4. **Never add `t.Skip()` to mask a real failure** — only skip when the test genuinely cannot run (missing infrastructure, removed feature, external dependency).

---

## Test Suites Overview

| Suite | Directory | Timeout | Key Dependencies |
|-------|-----------|---------|------------------|
| API | `tests/api_tests/` | 30m | 0box, zauth, zvault, blobbers, sharders, Firebase tokens |
| CLI | `tests/cli_tests/` | 60m | `zbox`/`zwallet` Linux binaries, blobbers, chain |
| SDK | `tests/sdk_tests/` | 30m | gosdk, blobbers, chain |
| Tokenomics | `tests/tokenomics_tests/` | 60m | blobbers, enterprise blobbers, chain, SC owner wallet |

---

## Step-by-Step Workflow

### 1. Compile & Vet Locally

Before syncing anything to remote, always verify the code compiles:

```bash
# From repo root
go vet ./tests/api_tests/
go vet ./tests/cli_tests/
go vet ./tests/sdk_tests/
go vet ./tests/tokenomics_tests/
```

If `go vet` fails, fix the issue before proceeding. Common issues:
- Unused imports (remove them)
- Wrong function signatures (check the client library)
- Missing struct fields (check `internal/api/model/`)

### 2. Sync to Remote

```bash
# Sync code (EXCLUDE Mac binaries and .git)
sshpass -p '<server-password>' rsync -avz \
  --exclude 'zbox' --exclude 'zwallet' --exclude 'mc' --exclude '.git' \
  /Users/saswatabasu/Code/system_test/ root@<server-ip>:/root/Code/system_test/

# CRITICAL: Restore Linux CLI binaries (rsync overwrites them with Mac versions)
sshpass -p '<server-password>' ssh root@<server-ip> \
  "cp /root/Code/zboxcli/zbox /root/Code/system_test/tests/cli_tests/zbox && \
   cp /root/Code/zwalletcli/zwallet /root/Code/system_test/tests/cli_tests/zwallet"
```

### 3. Kill Stale Test Processes

Always kill old test processes before starting new ones:

```bash
sshpass -p '<server-password>' ssh root@<server-ip> \
  "pkill -f 'go test' || true; pkill -f 'api_tests.test' || true; pkill -f 'cli_tests.test' || true"
```

### 4. Run Targeted Tests

**Always run specific tests by name, never the full suite:**

```bash
# API: Run specific test functions
ssh root@<server-ip> "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin && \
  cd /root/Code/system_test && \
  go test -run 'TestSpecificName' -v -timeout 30m ./tests/api_tests/ 2>&1 | tee /tmp/api_test.log"

# CLI: Run specific test functions
ssh root@<server-ip> "export PATH=\$PATH:/usr/local/go/bin:/root/go/bin && \
  cd /root/Code/system_test && \
  go test -run 'TestSpecificName' -v -timeout 60m ./tests/cli_tests/ 2>&1 | tee /tmp/cli_test.log"

# Multiple test patterns with pipe
go test -run 'TestFoo|TestBar|TestBaz' -v -timeout 30m ./tests/api_tests/
```

### 5. Pre-Test Setup (Before Full Suite Runs)

Before running a full test suite, the test environment must be in a clean, funded, and properly configured state. Follow these steps in order:

#### a. Reset — Cancel Allocations & Unstake Providers

```bash
# Cancel all existing allocations (prevents stale allocation interference)
# List and cancel via zbox CLI:
zbox listallocations --configDir /root/.zcn --wallet local.json --json | \
  jq -r '.[].id' | while read id; do
    zbox alloc-cancel --allocation "$id" --configDir /root/.zcn --wallet local.json || true
  done

# Unstake from providers to free up delegate pools:
# Miners
for id in $(zwallet ls-miners --configDir /root/.zcn --wallet local.json --json | jq -r '.[].id'); do
  zwallet mn-unlock --id "$id" --configDir /root/.zcn --wallet local.json || true
done
# Sharders
for id in $(zwallet ls-sharders --configDir /root/.zcn --wallet local.json --json | jq -r '.[].id'); do
  zwallet mn-unlock --id "$id" --configDir /root/.zcn --wallet local.json || true
done
# Blobbers
for id in $(zbox ls-blobbers --configDir /root/.zcn --wallet local.json --json | jq -r '.[].id'); do
  zbox sp-unlock --blobber_id "$id" --configDir /root/.zcn --wallet local.json || true
done
```

#### b. Clean — Reset 0box, Zauth, Zvault Databases

```bash
# Clean 0box test records (stale owners cause "duplicate key" errors)
docker exec postgres-0box psql -U zbox_user -d zbox -c \
  "DELETE FROM owner WHERE username LIKE 'test_%' OR username LIKE 'ref_%';"

# Clean zauth test records
docker exec postgres-zauth psql -U zbox_user -d zbox -c \
  "DELETE FROM peer_connections WHERE 1=1;" 2>/dev/null || true

# Clean zvault test records
docker exec postgreszv psql -U zbox_user -d zbox -c \
  "DELETE FROM split_wallets WHERE 1=1;" 2>/dev/null || true

# Or use the deploy script which does all cleanup:
bash scripts/deploy_local.sh services
```

#### c. Fund — Top Up Wallets and Providers

```bash
# Fund the SC owner wallet (used by all tests)
zwallet faucet --methodName pour --input "{Pay day}" --tokens 100 \
  --configDir /root/.zcn --wallet local.json

# Fund all blobbers (need balance for write marker redemption)
# The deploy script automates this:
bash scripts/deploy_local.sh fund

# Or start the funding daemon for continuous background funding:
bash scripts/deploy_local.sh fund-daemon
```

#### d. Setup — Create Test Allocation & Configure

```bash
# Stake providers (10 delegates max for tests — conserves pool capacity)
zwallet mn-update-config --keys num_delegates --values 10 \
  --configDir /root/.zcn --wallet local.json

# Ensure SC config is correct for tests
zwallet sc-update-config \
  --keys "min_alloc_size,min_write_price,max_block_cost,challenge_enabled" \
  --values "1024,0,100000,true" \
  --configDir /root/.zcn --wallet local.json

# Create a pre-loaded allocation (12 regular + 5 enterprise blobbers)
# with 10MB of data for challenge generation:
zbox newallocation --lock 5 --data 4 --parity 2 --size 1073741824 \
  --configDir /root/.zcn --wallet local.json

# Upload seed data to trigger challenges
zbox upload --allocation <alloc_id> --localpath /tmp/10mb.bin --remotepath /seed_data.bin \
  --configDir /root/.zcn --wallet local.json

# Or use the deploy script which handles everything:
bash scripts/deploy_local.sh test-setup
```

#### e. Verify — Check Everything is Healthy

```bash
bash scripts/deploy_local.sh verify
# This checks: chain health, blobber health, 0box, zauth, zvault,
# delegate wallets, SC config, and Kafka pipeline
```

**TL;DR**: For a complete pre-test setup, just run:
```bash
bash scripts/deploy_local.sh test-setup    # Reset + Clean + Fund + Setup
bash scripts/deploy_local.sh verify        # Verify everything is healthy
```

### 6. Run Full Suite (When Needed)

Use the deploy script's test runner for full suite runs with auto-retry:

```bash
bash scripts/deploy_local.sh test api              # API only
bash scripts/deploy_local.sh test cli              # CLI only
bash scripts/deploy_local.sh test --retries 2      # All suites, 2 retries
bash scripts/deploy_local.sh test --filter TestName # Filtered run
```

Or the standalone runner:

```bash
bash scripts/run_tests.sh api --timeout 45m
bash scripts/run_tests.sh cli --timeout 90m
```

### 7. Analyze Results

```bash
# Count PASS/FAIL/SKIP from a test log
grep -c "^--- PASS:" /tmp/test.log
grep -c "^--- FAIL:" /tmp/test.log
grep -c "^--- SKIP:" /tmp/test.log

# Extract just failures with context
grep -B2 "^--- FAIL:" /tmp/test.log

# Get detailed error for a specific test
grep -A20 "FAIL.*TestSpecificName" /tmp/test.log
```

---

## Common Failure Patterns & How to Fix

### Infrastructure Failures (NOT Code Bugs)

| Pattern | Symptom | Diagnosis | Fix |
|---------|---------|-----------|-----|
| Expired allocations | `MovedToChallenge=0`, challenge tests fail | `curl .../storage-config` → check `challenge_enabled` | Create new allocations, ensure challenges are enabled |
| Exhausted delegate pools | `max_delegates reached` | `zwallet mn-pool-info` shows all slots filled | `zwallet mn-update-config --keys num_delegates --values 200` |
| Stale 0box data | Empty blobber lists, `record not found` | Check `snapshots` table in 0box DB | `bash scripts/deploy_local.sh services` to reseed |
| Single sharder | `unexpected end of JSON input` | Sharder-1 down | Use only sharder-2 in configs |
| Firebase tokens expired | 0box auth tests fail with 401 | Check `firebaseTokenValid` in test logs | Refresh tokens via Firebase REST API |
| Junk blobbers | SDK init slow, wrong blobbers selected | `curl .../getblobbers` shows fake domains | Redeploy chain or filter in test code |

### Code Failures (Fix in Test Code)

| Pattern | Symptom | Fix |
|---------|---------|-----|
| Assertion mismatch | Expected "X" got "Y" | Update assertion to match current API behavior |
| Nil pointer deref | `panic: runtime error` | Add nil checks before accessing response fields |
| Timing issues | Intermittent failures on token balance checks | Add polling loops with `wait.PoolImmediately()` or `time.Sleep()` |
| Flag ordering | CLI command fails with wrong flag values | Ensure `createParams()` sorts map keys |
| Nonce errors | `verify_nonce: nonce too low` | Ensure sequential operations, add delays between txns |

### When to Skip vs When to Fix

**SKIP is appropriate when:**
- Feature was removed from the chain/SDK (e.g., read pools, per-user payment)
- External dependency unavailable (Firebase, Chimney test network, Tenderly)
- Infrastructure configuration makes test impossible (e.g., `min_write_price=0` means "below minimum" test is impossible)
- Test requires specific hardware/environment not available

**SKIP is NOT appropriate when:**
- The test reveals a real bug in test code (fix the assertion)
- The API behavior changed (update the test to match)
- The test is flaky due to timing (add retries/polling)
- The test needs better error handling (add nil checks)

---

## Test Suite Specific Notes

### API Tests

- **Firebase-dependent tests** (`Test0BoxOwner`, `Test0BoxWallet`, `Test0BoxShareinfo`, `Test0BoxReferral`, `Test0BoxFreeStorage`): Require valid Firebase tokens. Tests have `firebaseTokenValid` guard.
- **Zauth/Zvault JWT tests**: Require zauth and zvault services running with matching JWT secrets.
- **Blobber endpoint tests** (`TestFileReferencePath`, `TestBlobberHashNode`, `TestBlobberObjectTree`): Create fresh allocations and talk directly to blobbers.
- **Challenge timing tests**: Each subtest waits 20 minutes. Only run when challenge pipeline is working.
- **ZS3 server tests**: Require zs3server running on configured port. Have health check guard.

### CLI Tests

- **Requires Linux binaries**: `zbox` and `zwallet` must be Linux binaries in `tests/cli_tests/`. Mac versions cause `exec format error`.
- **Sequential tests**: Many tests use `t.RunSequentially()` — cannot be parallelized.
- **Wallet funding**: Tests create fresh wallets and fund from faucet. If faucet is depleted, cascade failures.
- **Enterprise blobber tests**: Must filter enterprise blobbers from allocation blobber lists.
- **Miner/sharder stake tests**: Need `num_delegates` high enough (200) to not exhaust pools.

### Tokenomics Tests

- **SC owner wallet**: Tests use `sc_owner_wallet.json` / `blobber_owner_wallet.json` which must match the on-chain SC owner.
- **Enterprise blobber tests**: Require enterprise blobbers registered with correct delegate_wallet.
- **Token precision**: Use `InEpsilon` assertions (not `Equal`) for token amounts due to fee variations.
- **Long-running**: Some tests wait for challenge completion (10-30 minutes).

### SDK Tests

- **gosdk dependency**: Tests use the Go SDK directly. Ensure `go.mod` has the correct gosdk version.
- **Smoke test**: `smoke_test.go` creates wallet, allocation, uploads, downloads — good for infrastructure validation.

---

## Debugging Checklist

When a test fails, follow this order:

1. **Read the error message** — what does it actually say?
2. **Check if it's infrastructure** — is the service running? Is the chain healthy? Are blobbers responsive?
3. **Check if the API behavior changed** — curl the endpoint manually, compare expected vs actual response
4. **Check if the test code is wrong** — stale assertions, missing nil checks, wrong expected values
5. **Check timing** — does the test need more time for transactions to confirm?
6. **Check dependencies** — does the test depend on state from a previous test that may have failed?
7. **Add logging** — if unclear, add `t.Logf("response: %s", resp.String())` to see what's actually returned

---

## Quick Reference: Remote Server

```bash
# SSH
sshpass -p '<server-password>' ssh root@<server-ip>

# PATH (always set this)
export PATH=$PATH:/usr/local/go/bin:/root/go/bin

# Check chain health
curl http://198.18.0.81:7171/_chain_stats 2>/dev/null | jq .

# Check blobber health
for i in $(seq 1 15); do
  port=$((5050 + $i))
  echo -n "Blobber $i (port $port): "
  curl -s http://198.18.0.100:$port/_blobber_info | jq -r '.id[:8] + " " + .base_url' 2>/dev/null || echo "DOWN"
done

# Check SC config
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config" 2>/dev/null | jq .fields
```
