# 0Chain Test Environment — Operations Guide

This document covers the five core operational areas for managing the 0Chain test environment at `test.zus.network` (server `<server-ip>`).

---

## Table of Contents

1. [Full Clean Deployment](#1-full-clean-deployment)
2. [Data-Only Clean Restart](#2-data-only-clean-restart)
3. [Fix Sharder → Kafka → 0box Pipeline](#3-fix-sharder--kafka--0box-pipeline)
4. [Fix Web App Issues](#4-fix-web-app-issues)
5. [Swap a Service Image](#5-swap-a-service-image)
6. [Run Test Suites and Monitor Results](#6-run-test-suites-and-monitor-results)

---

## 1. Full Clean Deployment

A full clean deployment wipes all chain data and rebuilds everything from scratch (~45 minutes). Use this when the chain is in an unrecoverable state, sharder RocksDB is corrupted, or you need a guaranteed-clean environment.

### Prerequisites

- SSH access: `sshpass -p '<server-password>' ssh root@<server-ip>`
- Go 1.22+ on the remote server
- All service source repos cloned under `/root/Code/`
- `scripts/deploy_config.yaml` configured with the correct branches

### Step 1 — Sync the deploy script to remote

Always make changes locally first, then rsync:

```bash
# From your local machine
sshpass -p '<server-password>' rsync -avz \
  --exclude 'zbox' --exclude 'zwallet' --exclude 'mc' --exclude '.git' \
  /Users/saswatabasu/Code/system_test/ root@<server-ip>:/root/Code/system_test/
```

### Step 2 — Run full clean redeploy

```bash
ssh root@<server-ip>
export PATH=$PATH:/usr/local/go/bin:/root/go/bin
cd /root/Code/system_test

# Full wipe + redeploy (takes ~45 min)
bash scripts/deploy_local.sh redeploy
```

What `redeploy` does internally, **in this order**:

1. Stops all Docker containers
2. Wipes chain data (miners, sharders, blobbers)
3. Builds fresh Docker images from configured branches
4. **Starts Kafka** (must be up before sharders start)
5. **Starts miners + sharders** (with Kafka enabled in config — sharders begin publishing events immediately, no `is_published` gap, no panic)
6. Waits for chain to reach a stable round
7. Starts and configures blobbers (12 regular + 5 enterprise)
8. Starts supporting services: zauth, zvault, validators, 0dns
9. **Starts 0box** (after sharders are producing events — 0box begins consuming the Kafka stream)
10. Configures SC settings (min_alloc_size, min_write_price, etc.)
11. Funds wallets, sets up delegate relationships
12. Builds and deploys web apps (Vult, Bolt, Blimp, Explorer, Player)
13. Configures nginx with SSL and reverse proxy rules

> **Why this order matters**: If sharders start before Kafka is ready, they panic trying to publish events to a broker that doesn't exist. If 0box starts before sharders, it connects to an empty stream and may deadlock waiting for round 1. Starting Kafka → sharders → 0box in sequence avoids all of this on a clean deploy.

### Step 3 — Post-deploy verification

```bash
bash scripts/deploy_local.sh verify
```

This checks:
- Chain round advancing
- Sharder API reachable
- Blobbers registered and staked
- 0box API responding
- Web apps serving HTTP 200

### Branch Configuration

Branches are controlled in `scripts/deploy_config.yaml`:

```yaml
repositories:
  0chain:
    branch: fix/dkg-broadcast-fee   # patched SC code
  blobber:
    branch: staging
  0box:
    branch: staging
  eblobber:
    branch: staging
  gosdk:
    branch: staging
  web-apps:
    branch: master
```

Override at runtime with `--branch`:

```bash
bash scripts/deploy_local.sh --branch 0chain=fix/my-branch redeploy
```

### Incremental Commands (partial redeploy)

If you don't need a full wipe, use targeted commands:

```bash
bash scripts/deploy_local.sh chain        # Reconfigure SC settings only
bash scripts/deploy_local.sh blobbers     # Restart blobbers, fund, stake
bash scripts/deploy_local.sh services     # Restart 0box, zauth, zvault
bash scripts/deploy_local.sh test-setup   # Refund wallets, copy CLI binaries
bash scripts/deploy_local.sh fund         # Top up all provider balances
```

### SC Owner & Delegate Wallet — Critical Notes

- SC Owner wallet lives at `/root/.zcn/local.json` on the remote server
- Client ID: `edb90b850f2e7e7cbd0a...`
- **All providers (blobbers, miners, sharders) must use this wallet as their `delegate_wallet`**
- Delegate wallet is set in the provider YAML config **before first start** — it cannot be changed after
- `tests/*/config/wallets/sc_owner_wallet.json` and `blobber_owner_wallet.json` are copies of `local.json`

### Required SC Config for Tests

After deployment, the following SC settings must be in place:

```bash
zwallet sc-update-config \
  --keys "min_alloc_size,min_write_price,challenge_enabled" \
  --values "1024,0.025,true" \
  --configDir /root/.zcn --wallet local.json

zwallet mn-update-config --keys max_delegates --values 200 \
  --configDir /root/.zcn --wallet local.json

zwallet global-update-config \
  --keys "server_chain.transaction.cost_fee_coeff" \
  --values "100000" --configDir /root/.zcn --wallet local.json
```

> **Note**: `min_read_price` does **not** exist as a valid SC config key — do not use it.

---

## 2. Data-Only Clean Restart

Use this when you want a fresh chain and clean service state **without rebuilding Docker images or re-cloning repos**. This is faster than a full `redeploy` (~10–15 min vs ~45 min) and is the right choice when:

- The chain is stuck or corrupted (e.g., sharder RocksDB is at round 0 after a crash)
- You want to clear accumulated test data (junk blobbers, old allocations, stale 0box records)
- You need to restart from a known-clean state without changing code branches

Docker images, networks, and container definitions are **preserved**. Only data directories, databases, logs, and test artifacts are wiped.

### What Gets Wiped

| Layer | What's Cleaned |
|-------|---------------|
| **Miners** | RocksDB (`config*`, `mb*`, `state*`, `dkg*`), Redis state + transactions, logs |
| **Sharders** | RocksDB (entire directory), PostgreSQL (`events_db`), block files, logs |
| **Blobbers** | PostgreSQL (`blobber_meta`) via `TRUNCATE`, stored files, logs |
| **Enterprise Blobbers** | PostgreSQL, stored files |
| **0box** | All tables except `goose_db_version` (schema migrations kept) |
| **zauth** | All tables except `goose_db_version` |
| **zvault** | All tables except `goose_db_version` |
| **Test logs** | `/tmp/test_*.log`, `/tmp/run_tests.log`, `/var/log/0chain/*.html` |
| **Kafka** | Topic data + offsets reset |

### Step 1 — Stop all containers

```bash
ssh root@<server-ip>
# Stop chain
for c in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2; do
    docker stop "$c" 2>/dev/null || true
done
# Stop blobbers (all 12 + 5 enterprise)
for i in $(seq 1 12); do docker stop "blobber-$i" "validator-$i" 2>/dev/null || true; done
for i in $(seq 1 5); do docker stop "eblobber-$i" 2>/dev/null || true; done
# Stop services
docker stop 0box-0box-1 zauth-zauthserver-1 zvault-zvault-1 2>/dev/null || true
```

### Step 2 — Wipe chain data (RocksDB + PostgreSQL + blocks)

```bash
DOCKER_LOCAL=/root/Code/0chain/docker.local

# --- Miners: wipe RocksDB state and Redis ---
for i in 1 2 3 4; do
    mdir="$DOCKER_LOCAL/miner${i}"
    rm -rf "$mdir"/log/*
    rm -rf "$mdir"/data/redis/state/*
    rm -rf "$mdir"/data/redis/transactions/*
    rm -rf "$mdir"/data/rocksdb/config*
    rm -rf "$mdir"/data/rocksdb/mb*
    rm -rf "$mdir"/data/rocksdb/state*
    rm -rf "$mdir"/data/rocksdb/dkg*
done

# --- Sharders: wipe RocksDB, PostgreSQL data dir, and blocks ---
for i in 1 2; do
    sdir="$DOCKER_LOCAL/sharder${i}"
    rm -rf "$sdir"/log/*
    rm -rf "$sdir"/data/rocksdb/*
    rm -rf "$sdir"/data/postgresql/*
    rm -rf "$sdir"/data/postgresql2/*
    rm -rf "$sdir"/data/blocks/*
done

# Re-initialize empty directories (PostgreSQL needs them to exist)
for i in 1 2; do
    sdir="$DOCKER_LOCAL/sharder${i}"
    mkdir -p "$sdir"/data/blocks
    mkdir -p "$sdir"/data/rocksdb
    mkdir -p "$sdir"/data/postgresql
    mkdir -p "$sdir"/data/postgresql2
done
```

> **Why wipe sharder PostgreSQL directories?** The PostgreSQL data directory (`/var/lib/postgresql/data`) is mounted from host. Deleting `postgresql/*` causes postgres to run `initdb` on next startup — a fresh empty database. This is intentional: schema migrations (goose) run automatically on sharder startup and rebuild the correct schema.

### Step 3 — Wipe blobber data

Regular blobbers store data in `/root/Code/blobber/docker.local/blobber{N}/`:

```bash
BLOBBER_LOCAL=/root/Code/blobber/docker.local
for i in $(seq 1 12); do
    rm -rf "$BLOBBER_LOCAL/blobber${i}/files/*"
    rm -rf "$BLOBBER_LOCAL/blobber${i}/data/*"
    rm -rf "$BLOBBER_LOCAL/blobber${i}/log/*"
done
```

Enterprise blobbers store data in `/root/Code/eblobber/docker.local/eblobber{N}/`:

```bash
EBLOBBER_LOCAL=/root/Code/eblobber/docker.local
for i in $(seq 1 5); do
    rm -rf "$EBLOBBER_LOCAL/eblobber${i}/files/*"
    rm -rf "$EBLOBBER_LOCAL/eblobber${i}/data/*"
done
```

> Blobber PostgreSQL (`blobber_meta`) is in the `data/postgresql/` subdirectory above. Wiping `data/*` wipes the DB. On next startup, blobber runs migrations to recreate the schema.

### Step 4 — Truncate service databases (0box, zauth, zvault)

The service postgres containers keep running between chain restarts. Their data dirs are NOT wiped. Instead, truncate all application tables while preserving schema:

```bash
# 0box
docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
  "SELECT string_agg('\"' || tablename || '\"', ', ')
   FROM pg_tables WHERE schemaname = 'public'
   AND tablename NOT IN ('goose_db_version');" | \
xargs -I{} docker exec postgres-0box psql -U zbox_user -d zbox -c "TRUNCATE TABLE {} CASCADE;"

# Or use the deploy script function:
bash scripts/deploy_local.sh services   # restarts and re-seeds 0box, zauth, zvault
```

The deploy script's `clean_service_databases()` handles this automatically with a `TRUNCATE … CASCADE` covering all tables in one call.

### Step 5 — Clear logs and test artifacts

```bash
# Test result logs
for f in /tmp/test_sdk.log /tmp/test_api.log /tmp/test_cli.log \
          /tmp/test_tokenomics.log /tmp/test_zs3.log /tmp/test_mc.log \
          /tmp/run_tests.log /tmp/deploy_local.log; do
    echo "" > "$f" 2>/dev/null || true
done

# Dashboard HTML
for f in /var/log/0chain/*.html /var/log/0chain/*.log; do
    [ -f "$f" ] && echo "" > "$f" || true
done
```

### Step 6 — Restart the chain

```bash
cd /root/Code/system_test
export PATH=$PATH:/usr/local/go/bin:/root/go/bin

# Restart chain only (no image rebuild)
bash scripts/deploy_local.sh chain
```

Or restart everything in order without wiping images:

```bash
bash scripts/deploy_local.sh all
```

`all` skips the image build step if images already exist, runs `clean_chain_data`, then starts services in the correct order: Kafka → sharders → chain → blobbers → 0box.

### Step 7 — Verify and re-seed

```bash
# Check chain health
curl http://198.18.0.81:7171/v1/chain/get/stats | jq '{round: .current_round, lfr: .latest_finalized_round}'

# Fund wallets and set SC config
bash scripts/deploy_local.sh test-setup
bash scripts/deploy_local.sh chain

# Full health check
bash scripts/deploy_local.sh verify
```

> `fix-kafka` is **not needed** after a clean restart. Because Kafka was started before sharders, sharders begin publishing from round 0 with no gap in `is_published`. 0box starts consuming from the live stream immediately. Use `fix-kafka` only when recovering a broken pipeline on an already-running chain (see [Section 3](#3-fix-sharder--kafka--0box-pipeline)).

### Using the Deploy Script (Automated)

The deploy script's `redeploy` command combines all of the above automatically. If you want to preserve Docker images and just wipe data, run the two internal functions directly:

```bash
# On the remote server, from the system_test directory:
source scripts/deploy_local.sh
clean_chain_data          # Wipes miner/sharder RocksDB, PostgreSQL, blocks
clean_service_databases   # Truncates 0box/zauth/zvault tables
clear_all_logs            # Clears /tmp/test_*.log and dashboard HTML
```

Then restart with:
```bash
bash scripts/deploy_local.sh all
```

---

## 3. Fix Sharder → Kafka → 0box Pipeline

**This section is for recovery only** — not needed during a clean deploy or data-only restart, where starting Kafka before sharders prevents these issues entirely.

Use `fix-kafka` when the pipeline breaks on an already-running chain: sharders restart unexpectedly, 0box falls behind, Explorer tables go empty, or charts stop updating.

The data pipeline flows: **Sharder → Kafka → 0box → Explorer/Web Apps**

```
Sharder (events_db) → Kafka (198.19.0.99:9092, SASL) → 0box (zbox DB) → Vult/Bolt/Blimp/Explorer
```

- Kafka credentials: `admin / admin-secret`
- Sharder writes block events to Kafka; 0box consumes them and populates its postgres DB

### Quick Fix — One Command

```bash
ssh root@<server-ip>
cd /root/Code/system_test
bash scripts/deploy_local.sh fix-kafka
```

This runs all recovery steps automatically:
1. Configures Kafka settings in sharder YAML files (if missing)
2. Advances the `is_published` marker to `MAX(block_number)` — fixes the round-gap PANIC caused by sharder restarting mid-stream
3. Seeds the `snapshots` table in 0box so it doesn't deadlock waiting for round 1 events that already passed
4. Restarts the affected sharder + 0box in the right order
5. Seeds provider tables immediately (bypasses the ~90-min Kafka health check delay)

### Known Issues and Manual Fixes

#### Issue 1 — entity.go Name Mismatch (Sharder PANIC crash loop)

**Symptom**: Sharder crashes every ~10 seconds with a PANIC.

**Cause**: `0chain/code/go/0chain.net/chaincore/chain/entity.go:970` calls `GetEntityMetadata("last_block_events")` but the entity is registered as `"block_events"`.

**Fix**:
```bash
ssh root@<server-ip>
sed -i 's/"last_block_events"/"block_events"/g' \
  /root/Code/0chain/code/go/0chain.net/chaincore/chain/entity.go

# Rebuild and restart sharder
bash /root/Code/system_test/scripts/deploy_local.sh swap-image 0chain fix/dkg-broadcast-fee
```

#### Issue 2 — `is_published` Round Gap After Restart

**Symptom**: Sharder panics with `"could not find events in round X"` in `publishUnPublishedEvents()`.

**Cause**: After restart, the RocksDB ring buffer is empty but `getLastPublishedRound()` returns an old round from PostgreSQL. Sharder tries to publish events for rounds it no longer has.

**Fix** (run on BOTH sharder postgres containers):
```bash
# Sharder 1
docker exec -e PGPASSWORD=zchian sharder1-postgres-1 \
  psql -U zchain_user -d events_db -c \
  "UPDATE events SET is_published = true WHERE block_number = (SELECT MAX(block_number) FROM events);"

# Sharder 2
docker exec -e PGPASSWORD=zchian sharder2-postgres-1 \
  psql -U zchain_user -d events_db -c \
  "UPDATE events SET is_published = true WHERE block_number = (SELECT MAX(block_number) FROM events);"
```

#### Issue 3 — 0box Snapshots Deadlock

**Symptom**: 0box starts but doesn't process any Kafka events. Explorer shows no data.

**Cause**: 0box's `snapshots` table is empty. It waits for round 1 events, but Kafka only has recent rounds.

**Fix**:
```bash
# Get the latest finalized round
LFR=$(curl -s http://198.18.0.81:7171/v1/chain/get/stats | jq -r '.latest_finalized_round')
SEED_ROUND=$((LFR - 1))

docker exec -e PGPASSWORD=zbox_server postgres-0box \
  psql -U zbox_user -d zbox -c \
  "INSERT INTO snapshots (round, ...) VALUES ($SEED_ROUND, ...) ON CONFLICT DO NOTHING;"
```

> The deploy script's `fix-kafka` command handles this automatically with the correct field values.

#### Issue 4 — Provider Tables Empty (Blobbers Not Showing in Allocation UI)

**Symptom**: Explorer shows no blobbers, miners, or validators. Creating allocations in Blimp fails with "no provider match the range".

**Cause**: Provider tables only populate from Kafka health check events, which only happen every ~90 minutes.

**Fix**:
```bash
bash scripts/deploy_local.sh seed-explorer
```

This copies provider data directly from the sharder's `events_db` to 0box's `zbox` database. Requirements for blobbers to appear in allocation queries:
- `not_available = false` (NULL rows are excluded)
- `brand_id` must reference a row in `provider_brand` table
- `last_health_check` must be within the last hour
- `blobber_type = 'HotMinus'` (default for regular blobbers)
- `is_enterprise = false` for free/regular allocations

#### Issue 5 — 0box Multi-Sharder Validation Deadlock

**Symptom**: 0box stops processing events even though Kafka is flowing. Logs show it waiting for matching round from all sharders.

**Cause**: 0box requires events from ALL sharders at the same round before processing. If one sharder is behind, the pipeline deadlocks.

**Fix**: Disable Kafka on the lagging sharder so 0box only validates against the healthy one:
```bash
# In sharder1's 0chain.yaml, set:
# kafka:
#   enabled: false
# Then restart sharder1
```

### Verification

```bash
# Is Kafka flowing? (check 0box logs)
docker logs 0box-0box-1 --tail 20 2>&1 | grep "last_processed_round"

# Is sharder pushing events?
docker logs sharder2-sharder-1 --tail 20 2>&1 | grep "Pushed event to kafka"

# Are snapshots advancing?
docker exec -e PGPASSWORD=zbox_server postgres-0box \
  psql -U zbox_user -d zbox -c \
  "SELECT round FROM snapshots ORDER BY round DESC LIMIT 3;"

# Are provider tables populated?
docker exec -e PGPASSWORD=zbox_server postgres-0box \
  psql -U zbox_user -d zbox -c \
  "SELECT COUNT(*) FROM blobbers; SELECT COUNT(*) FROM miners; SELECT COUNT(*) FROM sharders;"
```

### Database Credentials

| Service | Container | User | Password | DB |
|---------|-----------|------|----------|----|
| Sharder 1 | `sharder1-postgres-1` | `zchain_user` | `zchian` | `events_db` |
| Sharder 2 | `sharder2-postgres-1` | `zchain_user` | `zchian` | `events_db` |
| 0box | `postgres-0box` | `zbox_user` | `zbox_server` | `zbox` |
| zauth | `zauth-postgres-1` | `zauth_user` | (default) | `zauth` |
| zvault | `zvault-postgreszv-1` | `zvault_user` | (default) | `zvault` |

---

## 4. Fix Web App Issues

The web apps (Vult, Bolt, Blimp, Explorer, Player) run behind nginx as a reverse proxy. Each app is a Next.js build running on a local port with a basePath, served under a subdomain.

### Architecture

```
https://test.zus.network/          → nginx → web-app on localhost:PORT/basePath
https://test.zus.network/test/     → nginx → local HTML page (test results)
```

Web app port assignments (see `scripts/deploy_local.sh`):
- Vult: `localhost:3000/vult`
- Bolt: `localhost:3001/bolt`
- Blimp: `localhost:3002/blimp`
- Explorer: `localhost:3003/explorer`
- Player: `localhost:3004/player`

### Common Issue: 502 Bad Gateway

**Cause**: The Next.js process for that web app has crashed or hasn't started.

**Fix**:
```bash
# Check which process is down
ps aux | grep 'next\|node' | grep -v grep

# Rebuild and restart all web apps
bash scripts/deploy_local.sh web-apps

# Or swap just one web app
bash scripts/deploy_local.sh swap-image web-apps master
```

### Common Issue: Login Fails (Firebase)

The authentication chain is: **Firebase ID Token → 0box CSRF Token → 0box JWT Token**

```
User → Firebase Auth (email/password) → Firebase ID Token
     → 0box /v2/csrftoken (with X-App-ID-TOKEN header) → CSRF Token
     → 0box CreateJwtToken() → JWT Token
```

**Check 1 — Firebase service account key**:
```bash
ls -la /root/Code/0box/docker.local/config/0box_firebase_key.json
# Must exist and be non-empty
```

If missing or expired:
1. Go to [Firebase Console](https://console.firebase.google.com) → Project `box-dev-ce8bf`
2. Project Settings → Service Accounts → Generate New Private Key
3. Save as `0box_firebase_key.json` and copy to the server
4. Restart 0box: `docker compose -p 0box restart`

**Check 2 — Authorized domains in Firebase Console**:
- Firebase Console → Authentication → Settings → Authorized domains
- Must include: `test.zus.network`, `localhost`, `box-dev-ce8bf.firebaseapp.com`

**Check 3 — 0box deployment mode**:
```bash
# deployment_mode: 0 = bypass Firebase/Twilio (dev mode)
# deployment_mode: 1 = production mode (requires real Firebase tokens)
grep deployment_mode /root/Code/0box/docker.local/config/0box.yaml
```

**Check 4 — 0box SC owner config**:
```bash
grep server_chain.owner /root/Code/0box/docker.local/config/0box.yaml
# Must match on-chain SC owner: 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802
```

### Common Issue: Images / Assets Not Loading (404)

**Cause**: The nginx config is missing location rules for `/zcn.wasm` and `/assets/`.

Each web app subdomain needs these location blocks in its nginx server block:

```nginx
location = /zcn.js {
    proxy_pass http://localhost:PORT/BASEPATH/zcn.js;
    proxy_http_version 1.1;
    add_header Content-Type application/javascript;
}

location = /zcn.wasm {
    proxy_pass http://localhost:PORT/BASEPATH/zcn.wasm;
    proxy_http_version 1.1;
    add_header Content-Type application/wasm;
}

location /assets/ {
    proxy_pass http://localhost:PORT/BASEPATH/assets/;
    proxy_http_version 1.1;
}
```

These are automatically generated by `bash scripts/deploy_local.sh nginx`. After editing the deploy script, always run `nginx` to regenerate:

```bash
bash scripts/deploy_local.sh nginx
nginx -t && nginx -s reload
```

### Common Issue: Network Configuration (block_worker, 0box URL)

**0box config** (`/root/Code/0box/docker.local/config/0box.yaml`):
```yaml
block_worker: http://198.18.0.100:9091   # NOT dev.0chain.net
server_chain:
  owner: 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802
```

**zvault config** (`/root/Code/zvault/docker.local/config/zvault.yaml`):
```yaml
postgres:
  host: postgreszv    # Docker compose service name, NOT localhost
zauth_server: http://172.17.0.1:8080   # Docker bridge gateway
```

**Enterprise blobber config** (`b0docker-compose.yml` for eblobbers):
```yaml
# Must use eb0docker-compose.yml, NOT b0docker-compose.yml
# block_worker must be: http://198.18.0.100:9091
```

### Common Issue: 0box Redis SIGSEGV Crash

**Symptom**: Redis container exits immediately with signal 11 (segfault).

**Cause**: `redis:alpine` pulls Redis v8+ which crashes with the custom `redis.conf`.

**Fix**: Pin to `redis:7.4.3-alpine` in the 0box `docker-compose.yml`. Also fix `redis.conf`:
- Remove the `pidfile` directive
- Remove `rename-command` directives
- Fix healthcheck to use: `redis-cli -a redis_pass ping`

The deploy script applies all these fixes automatically during deployment.

### Rebuilding Web Apps

```bash
# Rebuild all 5 web apps from master branch
bash scripts/deploy_local.sh web-apps

# Rebuild with a custom gosdk branch (for WASM)
bash scripts/deploy_local.sh swap-image web-apps player-fmp4 --gosdk-branch fix/sdk
```

---

## 5. Swap a Service Image

Use `swap-image` to rebuild and restart a single service without doing a full redeploy. This is the primary way to test different branches in isolation.

### Basic Usage

```bash
bash scripts/deploy_local.sh swap-image <service> <branch>
```

### Service Reference

| Service | Command | What it rebuilds |
|---------|---------|-----------------|
| Chain (miners + sharders) | `swap-image 0chain <branch>` | All 4 miners + 2 sharders |
| Regular blobbers | `swap-image blobber <branch>` | All 12 blobbers + 12 validators |
| Enterprise blobbers | `swap-image eblobber <branch>` | All 5 eblobbers + validators |
| 0box | `swap-image 0box <branch>` | 0box service only |
| Web apps | `swap-image web-apps <branch>` | All 5 web apps + WASM |
| zauth | `swap-image zauth-server <branch>` | zauth service only |
| zvault | `swap-image zvault <branch>` | zvault service only |

### Examples

```bash
# Test a specific 0chain fix
bash scripts/deploy_local.sh swap-image 0chain fix/my-sc-fix

# Test a blobber feature branch
bash scripts/deploy_local.sh swap-image blobber feat/new-storage

# Test enterprise blobber with custom gosdk
bash scripts/deploy_local.sh swap-image eblobber staging \
  --gosdk-branch enterprise-blobber

# Test web apps with a custom web-apps branch
bash scripts/deploy_local.sh swap-image web-apps player-fmp4

# Override branch via --branch flag
bash scripts/deploy_local.sh --branch 0box=fix/my-0box-fix swap-image 0box
```

### Branch Resolution Priority

When no branch is specified explicitly, `swap-image` resolves branches in this order:

1. **CLI `--branch` flag** (highest priority): `--branch 0chain=fix/my-branch`
2. **`scripts/deploy_config.yaml`**: `repositories → <repo> → branch`
3. **Fallback**: `master`

### Current Default Branches

| Repo | Default |
|------|---------|
| 0chain | `fix/dkg-broadcast-fee` |
| blobber | `staging` |
| 0box | `staging` |
| eblobber | `staging` |
| gosdk | `staging` |
| web-apps | `master` |

Edit `scripts/deploy_config.yaml` to change permanent defaults.

### Important: Docker Image Tag Collision

The `blobber` and `eblobber` builds both default to `DOCKER_IMAGE_BASE=blobber_base`, which means an eblobber build overwrites the regular blobber image tag. The deploy script sets `DOCKER_IMAGE_BASE=eblobber_base` for enterprise blobbers to prevent this. If you ever see the wrong service running after a swap, check the image tag.

### Enterprise Blobber Notes

Enterprise blobbers use a completely different storage protocol:
- **VersionMarker** instead of WriteMarker
- Requires gosdk branch `enterprise-blobber`
- ZS3 server tests also need the `enterprise-timings` branch with `enterprise-blobber` gosdk

```bash
bash scripts/deploy_local.sh swap-image eblobber staging \
  --gosdk-branch enterprise-blobber
```

---

## 6. Run Test Suites and Monitor Results

### Compile Locally First

Always `go vet` before syncing to catch compile errors:

```bash
cd /Users/saswatabasu/Code/system_test
go vet ./tests/api_tests/
go vet ./tests/cli_tests/
go vet ./tests/sdk_tests/
go vet ./tests/tokenomics_tests/
```

### Sync to Remote

```bash
sshpass -p '<server-password>' rsync -avz \
  --exclude 'zbox' --exclude 'zwallet' --exclude 'mc' --exclude '.git' \
  /Users/saswatabasu/Code/system_test/ root@<server-ip>:/root/Code/system_test/

# Restore Linux CLI binaries (rsync overwrites them with Mac versions)
sshpass -p '<server-password>' ssh root@<server-ip> \
  "cp /root/Code/zboxcli/zbox /root/Code/system_test/tests/cli_tests/zbox && \
   cp /root/Code/zwalletcli/zwallet /root/Code/system_test/tests/cli_tests/zwallet"
```

### Kill Stale Test Processes (MANDATORY)

Before every test run, kill any leftover processes:

```bash
sshpass -p '<server-password>' ssh root@<server-ip> \
  "pkill -f '\.test' || true; pkill -f 'go test' || true; sleep 2; \
   ps aux | grep '\.test' | grep -v grep"
# Verify output shows NOTHING — no test processes running
```

> **CRITICAL**: Never use `killall -9 go` — this kills ALL Go processes including miners, sharders, and blobbers, bringing the entire chain down.

### Running Tests via Deploy Script (Recommended)

```bash
# All suites with auto-retry
bash scripts/deploy_local.sh test

# Specific suites
bash scripts/deploy_local.sh test api
bash scripts/deploy_local.sh test cli
bash scripts/deploy_local.sh test tokenomics
bash scripts/deploy_local.sh test sdk
bash scripts/deploy_local.sh test api cli    # multiple suites

# With retry count
bash scripts/deploy_local.sh test --retries 3

# Filter to specific test name
bash scripts/deploy_local.sh test --filter TestCreateAllocation
bash scripts/deploy_local.sh test cli --filter TestBlobberConfig

# Quick smoke test only
bash scripts/deploy_local.sh smoke
```

### Running Tests Directly (for targeted debugging)

```bash
ssh root@<server-ip>
export PATH=$PATH:/usr/local/go/bin:/root/go/bin
cd /root/Code/system_test

# Single specific test
go test -run TestCreateAllocation -v -timeout 30m ./tests/api_tests/

# Full suite in background (results in log file)
nohup bash -c 'export PATH=$PATH:/usr/local/go/bin:/root/go/bin; \
  go test -v -timeout 60m ./tests/api_tests/ > /tmp/test_api.log 2>&1; \
  echo DONE >> /tmp/test_api.log' > /dev/null 2>&1 &
```

### Log Files

| Suite | Log |
|-------|-----|
| API | `/tmp/test_api.log` |
| CLI | `/tmp/test_cli.log` |
| Tokenomics | `/tmp/test_tokenomics.log` |
| SDK | `/tmp/test_sdk.log` |
| Combined run | `/tmp/test_run_YYYYMMDD.log` |
| run_tests.sh | `/tmp/run_tests.log` |

### Monitor Results Live

#### Option 1 — Web Dashboard (auto-refreshed every 60s)

- **Top-level results**: `https://test.zus.network/test/results`
  - Shows parent test PASS/FAIL/SKIP counts per suite in a table
  - Lists all failed test names at the bottom
- **Subtest results**: `https://test.zus.network/test/subtests`
  - Shows every individual subtest `--- PASS/FAIL/SKIP` with timing

The dashboard is refreshed by a cron job running `refresh-0chain-logs.sh` every minute on the server.

#### Option 2 — Watch log files directly

```bash
# Watch API test progress
ssh root@<server-ip> "tail -f /tmp/test_api.log"

# Count results so far
ssh root@<server-ip> "grep -c '^--- PASS:' /tmp/test_api.log; \
  grep -c '^--- FAIL:' /tmp/test_api.log; \
  grep -c '^--- SKIP:' /tmp/test_api.log"

# See failing tests
ssh root@<server-ip> "grep '^--- FAIL:' /tmp/test_api.log"

# Check if done
ssh root@<server-ip> "grep -E '^(PASS|FAIL|ok|DONE)' /tmp/test_api.log | tail -5"
```

### Pre-Test Reset (Before Full Suite Runs)

Run this before a full suite to ensure a clean state:

#### 1. Kill junk blobbers (fake-URL blobbers left by API tests)

```bash
curl -s "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers" | \
  jq -r '.[] | select(.base_url | test("198\\.18\\.0") | not) | .id' | while read id; do
    zbox kill-blobber --id "$id" --configDir /root/.zcn --wallet local.json || true
done
```

#### 2. Cancel all allocations

```bash
zbox listallocations --configDir /root/.zcn --wallet local.json --json | \
  jq -r '.[].id' | while read id; do
    zbox alloc-cancel --allocation "$id" --configDir /root/.zcn --wallet local.json || true
done
```

#### 3. Fund wallets and providers

```bash
bash scripts/deploy_local.sh fund
bash scripts/deploy_local.sh test-setup
```

#### 4. Set required SC config

```bash
zwallet sc-update-config \
  --keys "min_alloc_size,min_write_price,challenge_enabled,max_block_cost" \
  --values "1024,0.025,true,100000" \
  --configDir /root/.zcn --wallet local.json
```

### Common Test Failure Patterns

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `consensus_not_met`, `no such host` | Junk blobbers on-chain | Kill junk blobbers |
| `MovedToChallenge=0`, challenge tests fail | No seed data in allocations | Upload seed data, wait for challenge |
| `max_delegates reached` | Stake pool exhausted | Unstake all, set `num_delegates=10` |
| `nodeStat: not found` | Chain bug in `fix/dkg-broadcast-fee` | Known bug, skip or use different branch |
| `exec format error` on CLI tests | Mac binary (zbox/zwallet) used on Linux | Restore Linux binaries after rsync |
| `verify_nonce: nonce too low` | Wallet nonce out of sync | Sync nonce from chain before running |
| `unexpected end of JSON input` | Sharder down | Check sharder health, restart if needed |
| Firebase tests return 401 | Firebase token expired | Refresh token or use `deployment_mode: 0` |

### Suite-Specific Notes

**API Tests** — checks REST endpoints and smart contract state:
- Firebase-dependent tests need `FIREBASE_TOKEN` env var or `deployment_mode: 0`
- Challenge timing tests wait 20+ minutes — only run when blobbers have seed data

**CLI Tests** — checks zbox/zwallet command-line tools:
- Must have Linux `zbox` and `zwallet` binaries in `tests/cli_tests/`
- Many tests use `t.RunSequentially()` — cannot parallelize

**SDK Tests** — fastest suite (~1-2 min), good for quick infrastructure validation:
- Runs smoke test: create wallet → create allocation → upload → download

**Tokenomics Tests** — checks reward and payment flows:
- Require active allocations with uploaded data for `MovedToChallenge > 0`
- Use `sc_owner_wallet.json` which must match the on-chain SC owner

---

## Quick Reference

```bash
# SSH to server
sshpass -p '<server-password>' ssh root@<server-ip>

# Set PATH (always)
export PATH=$PATH:/usr/local/go/bin:/root/go/bin

# Chain health
curl http://198.18.0.81:7171/v1/chain/get/stats | jq '{round: .current_round, lfr: .latest_finalized_round}'

# Check all containers
docker ps --format '{{.Names}}: {{.Status}}' | sort

# Sharder sync status
curl -s http://127.0.0.1:7171/v1/chain/get/stats | jq .latest_finalized_round  # sharder1
curl -s http://127.0.0.1:7172/v1/chain/get/stats | jq .latest_finalized_round  # sharder2

# SC config
curl "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config" | jq .fields

# Blobber health
for i in $(seq 1 12); do
  port=$((5050 + $i))
  echo -n "B$i: "
  curl -s --max-time 2 "http://198.18.0.100:$port/v1/storage/challenge/new" -o /dev/null -w "%{http_code}\n"
done

# 0box health
curl -s https://test.zus.network/0box/v2/health | jq .
```
