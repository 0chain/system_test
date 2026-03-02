# CORS & Providers Page Fixes

## CORS Fixes (applied Feb 2026)

### Problem
Atlus (`test.atlus.cloud`) and other web apps making requests to `test.zus.network` (miners, sharders, blobbers, 0box) received CORS errors.

### Root Causes & Fixes Applied

#### 1. Bare miner/sharder paths — OPTIONS preflight missing ACAO headers
- **URL pattern**: `https://test.zus.network/miner03` (exact-match, no trailing slash)
- **Root cause**: nginx `if{}` block with its own `add_header` prevents the parent `location` block's `add_header` from being inherited. ACAO/ACAC were outside the `if(OPTIONS)` block → not sent in 204 response.
- **Fix**: Moved ALL CORS headers inside the `if($request_method = OPTIONS)` block for all 6 exact-match locations (miner01-04, sharder01-02).

#### 2. Duplicate `Access-Control-Allow-Origin` headers from sharder backends
- **Root cause**: Miners/sharders send `Access-Control-Allow-Origin: *` themselves. Without `proxy_hide_header`, nginx ALSO adds `$http_origin` → two ACAO headers → browser rejects.
- **Fix**: Added `proxy_hide_header Access-Control-Allow-Origin` (and other CORS headers) to all miner/sharder prefix locations.

#### 3. Blobbers sending `*` for credentialed requests
- **Root cause**: Blobbers hardcode `Access-Control-Allow-Origin: *`. Browsers reject `*` when requests use `credentials: include`.
- **Fix**: Created `/etc/nginx/snippets/provider-cors.conf` that strips backend `*` and replaces with `$http_origin`. Included in all blobber/validator/eblobber locations.

#### 4. gosdk WASM sends `Access-Control-Allow-Origin` as a REQUEST header
- **Root cause**: gosdk WASM (non-standard behavior) sends `Access-Control-Allow-Origin` as a request header. Nginx's `Access-Control-Allow-Headers` didn't list it → browser blocked the request.
- **Fix**: Added `Access-Control-Allow-Origin` to every `Access-Control-Allow-Headers` value in nginx config.

#### 5. Missing subdomain server blocks — "certificate invalid" errors
- **URLs**: `https://0box.test.zus.network`, `https://zauth.test.zus.network`, `https://zvault.test.zus.network`
- **Root cause**: Web app code hardcodes `https://0box.${domain}` for 0box API calls (e.g. `timestamps-to-rounds`, `latest-snapshot`). With `DOMAIN=test.zus.network` → `https://0box.test.zus.network`. SSL certs existed (from previous certbot run) but NO nginx server block → TLS handshake failed → browser showed "certificate invalid".
- **Fix**: Created nginx server blocks for all 3 subdomains, each proxying to the corresponding backend:
  - `0box.test.zus.network` → `localhost:9081`
  - `zauth.test.zus.network` → `localhost:8080`
  - `zvault.test.zus.network` → `localhost:8090`
- **Deploy script**: `setup_nginx_subdomains()` in `deploy_local.sh` handles these automatically. Certbot certs are requested per-subdomain.

#### Verification commands
```bash
# Bare path OPTIONS preflight
curl -sI -X OPTIONS -H 'Origin: https://test.atlus.cloud' 'https://test.zus.network/miner03'
# → Access-Control-Allow-Origin: https://test.atlus.cloud

# Sharder GET (no duplicate headers)
curl -sI -H 'Origin: https://test.atlus.cloud' 'https://test.zus.network/sharder01/v1/fees_table'
# → single Access-Control-Allow-Origin header

# 0box subdomain
curl -s -H 'Origin: https://test.atlus.cloud' 'https://0box.test.zus.network/v2/latest-snapshot'
# → full JSON with round, total_mint, etc.

curl -s 'https://0box.test.zus.network/v2/timestamps-to-rounds?timestamps=[1740275651]'
# → {"rounds":[46431]}
```

---

## Providers Page Data Issues (Feb 2026)

### Symptoms
Explorer (Atlus) providers page shows zeros for:
- `unique_addresses` (should be 646+)
- `successful_challenges` (should be 25,000+)
- `total_challenges` (should be 25,000+)
- `total_rewards` (should be 282 trillion SAS)
- Miner/sharder `reward` field = 0

Graph endpoints also return all-zero arrays for challenges, token supply.

### Root Cause: Deploy Ordering
In a **correct fresh deploy** (`redeploy`), this should NOT happen:
1. Chain starts at block 0
2. Kafka starts fresh
3. 0box starts and consumes from Kafka offset 0
4. All challenge/reward/user events flow through Kafka → 0box aggregates them correctly

The snapshots show zeros because **0box was started (or restarted) after the chain already had significant history**. The Kafka consumer started from a later offset and missed all historical challenge/reward/user events. `events_db` has the data (challenges, provider_rewards with per-provider breakdown, users) but 0box never received those events through Kafka.

### Real Fix: Correct Deploy Ordering
Ensure the deploy script starts all services in order and that 0box is not started against a chain with existing history unless Kafka offsets are reset. The `fix-kafka` command seeds `is_published` markers — 0box should be started AFTER this, so it begins from a consistent point.

In a fresh `redeploy` this is automatic. The snapshot patch below is only needed when 0box is added to an existing chain or after a restart that loses Kafka state.

### Workaround: Snapshot Patch (for existing chains)
`fix_snapshot_aggregates()` in `deploy_local.sh`:
- Queries `sharder-postgres-1:events_db` for real counts (`provider_rewards.total_rewards` has per-provider data)
- Patches the latest (and all zero-value historical) rows in `0box.snapshots`

```bash
# Run manually:
bash scripts/deploy_local.sh fix-snapshots

# Also runs automatically as part of:
bash scripts/deploy_local.sh seed-explorer
```

Note: `events_db.provider_rewards` has `provider_id` and `total_rewards` per provider, so in theory per-type breakdown (miner/sharder/blobber) is possible if 0box joins against the miners/sharders/blobbers tables. Currently the patch only updates the top-level `total_rewards`.

### Miner/Sharder rewards = 0 on SC REST API
The `getMinerList`/`getSharderList` endpoints return `reward=0` and `total_reward=0` for all nodes. This is a separate issue — the chain's reward distribution mechanism may not be active or the `simple_miner.total_reward` field isn't being updated. Actual rewards DO exist in `events_db.provider_rewards`. This is a chain-level issue, not a deploy script issue.

### Blobber challenges_passed (SC REST API)
Blobbers DO show correct `challenges_passed` and `challenges_completed` values on the SC REST API (`getblobbers` endpoint). Only the 0box snapshot aggregates are wrong.

---

## Quick Reference: What Shows Where

| Data | Source | Status |
|------|--------|--------|
| Blobber list | `sharder SC REST /getblobbers` | ✓ Correct |
| Blobber challenges_passed | `sharder SC REST /getblobbers` | ✓ Correct |
| Miner/sharder list | `sharder SC REST /getMinerList` | ✓ Correct |
| Miner/sharder stake | `sharder SC REST` | ✓ Correct |
| Miner/sharder rewards | `sharder SC REST` | ✗ = 0 (chain bug) |
| Explorer unique_addresses | `0box /v2/latest-snapshot` | ✗ = 0 → fixed by `fix-snapshots` |
| Explorer total_challenges | `0box /v2/latest-snapshot` | ✗ = 0 → fixed by `fix-snapshots` |
| Explorer total_rewards | `0box /v2/latest-snapshot` | ✗ = 0 → fixed by `fix-snapshots` |
| Graph data | `0box /v2/graph-*` | ✗ = 0 → partially fixed by `fix-snapshots` |
