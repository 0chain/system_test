# Web Apps — Fixing Login, OTP, CORS, Signup, and Allocation Issues

This document covers every known failure mode for the web apps (Vult, Blimp, Bolt, Explorer) running on `test.zus.network`, with root causes, symptoms, and exact fixes.

---

## Table of Contents

1. [Login Failures](#1-login-failures)
2. [OTP / Phone Signup Failures](#2-otp--phone-signup-failures)
3. [CORS Errors](#3-cors-errors)
4. [Standard Allocation Failures](#4-standard-allocation-failures)
5. [Free Allocation Failures](#5-free-allocation-failures)
6. [502 / Service Down](#6-502--service-down)
7. [WASM and Asset Loading Failures](#7-wasm-and-asset-loading-failures)
8. [Quick Diagnostics Reference](#8-quick-diagnostics-reference)

---

## 1. Login Failures

### Auth Chain Recap

Every login goes through three steps:

```
1. Firebase Auth  →  Firebase ID Token        (Google's servers)
2. 0box /v2/csrftoken  →  CSRF Token          (0box validates Firebase token)
3. 0box CreateJwtToken  →  JWT Token          (used for all wallet/storage ops)
```

If any step fails, the user cannot log in.

---

### Issue: "Login failed" / stuck on login screen

**Likely cause**: Firebase service account key missing or expired on 0box.

**How 0box uses it**: 0box validates incoming Firebase ID tokens against Firebase Admin SDK using the service account key. Without it, every token is rejected.

**Fix**:
1. Go to [Firebase Console](https://console.firebase.google.com) → Project `box-dev-ce8bf`
2. Project Settings → Service Accounts → **Generate New Private Key**
3. Copy the downloaded JSON to the server:
```bash
scp /path/to/key.json root@<server-ip>:/root/Code/0box/docker.local/config/0box_firebase_key.json
```
4. Restart 0box:
```bash
ssh root@<server-ip> "cd /root/Code/0box/docker.local && docker compose -p 0box restart 0box"
```

The key is volume-mounted from host, so it persists across container restarts.

---

### Issue: Firebase refuses to issue ID token (auth blocked before it reaches 0box)

**Likely cause**: Deployment domain not in Firebase's authorized domains list.

**Symptom**: Browser console shows Firebase error like `auth/unauthorized-domain`.

**Fix**:
- Firebase Console → Project `box-dev-ce8bf` → Authentication → Settings → **Authorized domains**
- Add `test.zus.network` (or whatever the deployment domain is)
- Also ensure: `localhost`, `box-dev-ce8bf.firebaseapp.com`

This is a Firebase-side CORS enforcement — requests from unauthorized domains are blocked before any token is issued.

---

### Issue: Sign-in method not available (no email/phone option shown in UI)

**Fix**: Firebase Console → Authentication → Sign-in method → Enable:
- **Email/Password**
- **Phone**
- **Google**

---

### Issue: `server_chain.owner` mismatch — 0box rejects wallet operations after login

**Symptom**: Login succeeds (CSRF + JWT issued) but any wallet/storage operation fails. Logs show 0box can't verify the owner.

**Cause**: `0box.yaml` has the wrong `server_chain.owner` — it must match the on-chain SC owner.

**Fix** (set in `0box.yaml` and applied by deploy script):
```yaml
server_chain:
  owner: 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802
```

**Manual check**:
```bash
ssh root@<server-ip> "grep 'owner:' /root/Code/0box/docker.local/config/0box.yaml"
```

---

## 2. OTP / Phone Signup Failures

### Vult/Blimp Signup Flow

```
Email → Firebase ID Token → phone number screen → OTP sent via Twilio
→ POST /v2/twilio/phone/verify/signup { email, otp, phone_number, firebase_token, username }
→ 0box creates user record
```

---

### Issue: OTP SMS never arrives / signup stuck on OTP screen

**Cause**: Twilio is configured but the test environment doesn't have a real Twilio account that can send SMS.

**Fix**: Set `deployment_mode: 0` in `0box.yaml`. This bypasses OTP verification entirely — any code is accepted.

```bash
ssh root@<server-ip> "grep deployment_mode /root/Code/0box/docker.local/config/0box.yaml"
# Should show: deployment_mode: 0
```

If it shows `1`, change it:
```bash
ssh root@<server-ip> "sed -i 's/deployment_mode: 1/deployment_mode: 0/' /root/Code/0box/docker.local/config/0box.yaml"
ssh root@<server-ip> "cd /root/Code/0box/docker.local && docker compose -p 0box restart 0box"
```

> **Note**: Even with `deployment_mode: 0`, the Firebase auth flow still runs on the frontend — the web app still needs to successfully get a Firebase ID token before calling 0box. Only 0box's server-side validation of the token is bypassed.

---

### Issue: OTP sends but verification returns "invalid OTP"

**Cause**: Either Twilio credentials are wrong, or the test number isn't in 0box's `jwt.test_numbers` list.

0box maintains a whitelist of test phone numbers that bypass real OTP validation even when `deployment_mode: 1`:

```yaml
# In 0box.yaml:
jwt:
  test_numbers:
    - "+919876543210"
    - "+919944033150"
    - "+16026666666"
    # ... more
```

**Fix**: Either use a number from the whitelist, or set `deployment_mode: 0`.

---

### Issue: Signup POST fails with "invalid_body"

**Cause**: The FormData sent to `/v2/twilio/phone/verify/signup` is missing `firebase_token`, or the Firebase auth step failed silently and the token is empty.

**Diagnosis**: Open browser devtools → Network → find the signup POST → check FormData fields. `firebase_token` must be present and non-empty.

---

## 3. CORS Errors

### Root Cause

0box, zauth, and zvault all send `Access-Control-Allow-Origin: *` in their responses. Browsers **reject** wildcard origins for credentialed requests (requests that include cookies, CSRF tokens, or `Authorization` headers). The web apps use credentials on every request, so the wildcard fails.

**Fix**: nginx strips the backend's CORS headers and replaces them with the actual request `Origin`, scoped correctly for credentialed use. This is configured per-location in the nginx config generated by the deploy script.

---

### Issue: Browser shows "CORS policy: No 'Access-Control-Allow-Origin' header" or "credential flag is true but wildcard"

**Cause**: nginx CORS location block is missing for `/0box/`, `/zauth/`, or `/zvault/`.

**What the correct nginx block looks like** (for `/0box/`):
```nginx
location /0box/ {
    proxy_pass http://localhost:9081/;
    proxy_set_header Host $host;
    # Strip backend wildcard CORS (browsers reject wildcard with credentials)
    proxy_hide_header Access-Control-Allow-Origin;
    proxy_hide_header Access-Control-Allow-Methods;
    proxy_hide_header Access-Control-Allow-Headers;
    proxy_hide_header Access-Control-Allow-Credentials;
    proxy_hide_header Access-Control-Expose-Headers;
    # Respond with actual request origin (required for credentialed requests)
    add_header 'Access-Control-Allow-Origin' $http_origin always;
    add_header 'Access-Control-Allow-Credentials' 'true' always;
    add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
    add_header 'Access-Control-Allow-Headers' 'Content-Type,Authorization,X-App-ID-TOKEN,X-CSRF-TOKEN,X-Jwt-Token,...' always;
    add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range' always;
    # Preflight
    if ($request_method = 'OPTIONS') {
        add_header 'Access-Control-Allow-Origin' $http_origin;
        add_header 'Access-Control-Allow-Credentials' 'true';
        add_header 'Access-Control-Max-Age' 1728000;
        add_header 'Content-Length' 0;
        return 204;
    }
}
```

The same pattern applies to `/zauth/` (port 8080) and `/zvault/` (port 8090).

**Fix**: Regenerate nginx config from the deploy script, which has the full correct headers:
```bash
bash scripts/deploy_local.sh nginx
nginx -t && nginx -s reload
```

---

### Issue: CORS on Firebase auth itself ("auth/unauthorized-domain")

This is not an nginx issue — it's Firebase rejecting the domain. See [Login Failures → Authorized domains](#issue-firebase-refuses-to-issue-id-token-auth-blocked-before-it-reaches-0box).

---

### Issue: `FBASE_AUTH_DOMAIN` set wrong — Firebase auth succeeds locally but fails from deployment domain

**Cause**: The web app build used `FBASE_AUTH_DOMAIN=test.zus.network` instead of the Firebase project domain.

**Correct value**:
```
FBASE_AUTH_DOMAIN=box-dev-ce8bf.firebaseapp.com   # Firebase project domain — NOT the deployment domain
```

This is set during web app build in the deploy script. If set wrong, rebuild the apps:
```bash
bash scripts/deploy_local.sh web-apps
```

---

## 4. Standard Allocation Failures

### Issue: "No blobbers available" / "no provider match the range" when creating allocation from Blimp/Vult

**Cause**: The 0box `blobbers` table is empty or has stale data. Blimp and Vult query 0box (not the chain directly) for available blobbers.

The table only populates from Kafka health check events, which happen every ~90 minutes. After a fresh deploy, it's empty until the first health check cycle completes.

**Fix** (immediate):
```bash
bash scripts/deploy_local.sh seed-explorer
```

This copies blobber data from the sharder's `events_db` directly into 0box's `blobbers` table, bypassing the wait.

**Requirements for a blobber to appear in allocation queries** — all must be true:
| Column | Required value |
|--------|---------------|
| `not_available` | `false` (NULL is excluded by the SQL WHERE) |
| `brand_id` | Must reference a row in `provider_brand` table (INNER JOIN) |
| `last_health_check` | Within the last hour (Unix epoch) |
| `blobber_type` | `'HotMinus'` (default for regular blobbers) |
| `is_enterprise` | `false` for regular allocations |
| `is_killed` | `false` |
| `is_shutdown` | `false` |

**Verify blobbers are populated**:
```bash
ssh root@<server-ip> \
  "docker exec -e PGPASSWORD=zbox_server postgres-0box \
   psql -U zbox_user -d zbox -c \
   \"SELECT id, base_url, not_available, brand_id, last_health_check, blobber_type, is_enterprise
     FROM blobbers LIMIT 5;\""
```

**Verify `provider_brand` table has the 'Zus' entry** (required for the JOIN):
```bash
ssh root@<server-ip> \
  "docker exec -e PGPASSWORD=zbox_server postgres-0box \
   psql -U zbox_user -d zbox -c \"SELECT * FROM provider_brand;\""
```

If empty, insert it:
```bash
ssh root@<server-ip> \
  "docker exec -e PGPASSWORD=zbox_server postgres-0box \
   psql -U zbox_user -d zbox -c \
   \"INSERT INTO provider_brand (name) VALUES ('Zus') ON CONFLICT DO NOTHING;\""
```

---

### Issue: Allocation creation fails with write price / read price mismatch

**Cause**: SC config `min_write_price` is higher than blobbers' `write_price`, or vice versa. Blobbers configured with `write_price: 0` can't satisfy `min_write_price: 0.001`.

**Fix**: SC config and blobber config must be consistent:
```bash
# SC config (via zwallet)
zwallet sc-update-config \
  --keys "min_write_price" \
  --values "0.025" \
  --configDir /root/.zcn --wallet local.json

# Blobbers must also advertise write_price >= min_write_price
# (set in blobber YAML before start, or via bl-update after start)
```

Current deployed values: `min_write_price=0.025`, blobbers `write_price=0.025`.

---

### Issue: Allocation too small — "size below minimum"

**Cause**: `min_alloc_size` SC setting defaults to `1048576` (1MB). Tests and UI often try to create smaller allocations.

**Fix**:
```bash
zwallet sc-update-config \
  --keys "min_alloc_size" \
  --values "1024" \
  --configDir /root/.zcn --wallet local.json
```

---

## 5. Free Allocation Failures

### How Free Allocations Work

Free allocations are created by 0box on behalf of the user using 0box's own wallet. The flow:

```
User clicks "Free Allocation" in Blimp
→ Blimp calls 0box API
→ 0box signs a FreeStorageMarker using its private key (from 0box.yaml free_storage config)
→ 0box submits a newAllocation transaction on-chain using its own wallet
→ Chain verifies the marker against the registered free_storage_assigner public key
```

---

### Issue: "Free allocation failed" / 0box errors on free allocation endpoint

**Check 1 — 0box wallet has tokens**:
0box needs a funded on-chain wallet to pay for the allocation transaction.

```bash
ssh root@<server-ip> "docker logs 0box-0box-1 2>&1 | grep 'wallet client id' | head -1"
# Get the wallet ID, then check balance:
# zwallet getbalance --clientid <id> --configDir /root/.zcn
```

Fund it if needed:
```bash
bash scripts/deploy_local.sh fund-0box
# or manually:
zwallet send --to_client_id <0box_wallet_id> --tokens 100 \
  --configDir /root/.zcn --wallet local.json --desc "Fund 0box"
```

---

**Check 2 — `free_storage_assginer` in 0box.yaml matches what's registered on-chain**:

The SC has a registered `free_storage_assigner` public key. 0box must sign markers with the matching private key.

In `0box.yaml`:
```yaml
free_storage:
  name: 0chain
  public_key: "545e111275a7c53de7..."   # Must match on-chain registration
  private_key: "8cded3109618dad8b5..."  # Used to sign FreeStorageMarkers
```

In the SC blobber config (registered by the SC owner wallet):
```yaml
# wallets section in 0chain_blobber.yaml:
free_storage_assginer: <same_client_id_as_0box_wallet>
```

The deploy script sets `free_storage_assginer` to the SC owner's client ID (`edb90b850f2e7e7cbd0a...`) in each blobber's config, and updates the `owner_wallet` and `free_storage_assginer` fields in `0box.yaml` to match.

**Verify consistency**:
```bash
# On-chain free_storage_assigner registered for the SC:
curl -s "http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config" \
  | jq '.fields.free_allocation_settings'

# 0box config value:
ssh root@<server-ip> "grep free_storage_assginer /root/Code/0box/docker.local/config/0box.yaml"
```

---

**Check 3 — SC `free_allocation_settings.read_price_range.max` is set**:

If the SC's max read price for free allocations is 0, no blobbers qualify. Set it to match blobbers' read price:

```bash
zwallet sc-update-config \
  --keys "free_allocation_settings.read_price_range.max" \
  --values "0.01" \
  --configDir /root/.zcn --wallet local.json
```

---

**Check 4 — Blobbers seeded in 0box with `is_enterprise=false`**:

Free allocations only use regular (non-enterprise) blobbers. If `seed-explorer` seeded them with `is_enterprise=true`, free allocation queries return empty.

```bash
ssh root@<server-ip> \
  "docker exec -e PGPASSWORD=zbox_server postgres-0box \
   psql -U zbox_user -d zbox -c \
   \"SELECT COUNT(*) FROM blobbers WHERE is_enterprise = false AND not_available = false;\""
# Should return 12 (the regular blobbers)
```

---

## 6. 502 / Service Down

### Issue: 0box returns 502

**Step 1 — Check if the container is running**:
```bash
ssh root@<server-ip> "docker ps | grep 0box"
```

**Step 2 — Check Redis** (0box crashes silently if Redis is down):
```bash
ssh root@<server-ip> "docker ps | grep redis; docker logs 0box-redis --tail 20"
```

If Redis is crash-looping with `signal: 11` (SIGSEGV):

**Cause**: `redis:alpine` pulled Redis 8.x which crashes on this kernel.

**Fix**: Pin to `redis:7.4.3-alpine` in `/root/Code/0box/docker.local/docker-compose.yml`:
```bash
ssh root@<server-ip> "sed -i 's|image: \"redis:alpine\"|image: \"redis:7.4.3-alpine\"|' \
  /root/Code/0box/docker.local/docker-compose.yml"
ssh root@<server-ip> "cd /root/Code/0box/docker.local && \
  docker compose -p 0box up -d --force-recreate redis && sleep 5 && \
  docker compose -p 0box up -d 0box"
```

Also check `redis.conf` — remove these lines if present (cause permission errors and segfaults):
```
# Remove:
pidfile /var/run/redis_6379.pid
rename-command CONFIG ""
rename-command FLUSHALL ""
```

Fix the healthcheck in `docker-compose.yml` (auth is required):
```yaml
healthcheck:
  test: ["CMD", "redis-cli", "-a", "redis_pass", "ping"]
```

---

### Issue: zvault returns 502 / connection refused

**Cause**: `zvault.yaml` has `postgres.host: localhost` which points to the container itself, not the postgres container.

**Fix**:
```bash
ssh root@<server-ip> "sed -i 's|host: localhost|host: postgreszv|' \
  /root/Code/zvault/config/zvault.yaml"
ssh root@<server-ip> "cd /root/Code/zvault/docker.local && \
  docker compose -p zvault up -d --force-recreate zvault"
```

Also check `zauth_server` points to the Docker bridge gateway, not localhost:
```yaml
zauth_server: http://172.17.0.1:8080   # Docker bridge gateway (NOT localhost:8080)
```

---

### Issue: Web app itself returns 502 (Next.js process down)

```bash
# Check PM2 processes
ssh root@<server-ip> "pm2 list"
# Restart a specific app
ssh root@<server-ip> "pm2 restart vult"
# Or restart all
ssh root@<server-ip> "pm2 restart all"
# Rebuild if process keeps crashing
bash scripts/deploy_local.sh web-apps
```

---

## 7. WASM and Asset Loading Failures

### Issue: Wallet operations fail with "WASM not loaded" or zcn.wasm returns 404/wrong content-type

**Cause**: nginx is missing location rules for `/zcn.wasm` and `/assets/` under each web app's subdomain. The WASM file is served under the app's basePath, but browsers request it from the root.

**Fix**: Each web app's nginx server block needs these three location rules (in addition to the main `location /basepath/`):

```nginx
# Correct Content-Type for WASM (browsers require application/wasm)
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
# Next.js static assets under basePath
location /assets/ {
    proxy_pass http://localhost:PORT/BASEPATH/assets/;
    proxy_http_version 1.1;
}
```

**Fix**: Regenerate nginx config from deploy script (includes all location rules):
```bash
bash scripts/deploy_local.sh nginx
nginx -t && nginx -s reload
```

---

### Issue: App loads but wallet/chain features broken — WASM reports network errors

**Cause**: `block_worker` URL in 0box config points to devNet, not local chain.

**Fix**:
```bash
ssh root@<server-ip> "grep block_worker /root/Code/0box/docker.local/config/0box.yaml"
# Must be: block_worker: http://198.18.0.100:9091
```

If wrong:
```bash
ssh root@<server-ip> "sed -i 's|block_worker:.*|block_worker: http://198.18.0.100:9091|' \
  /root/Code/0box/docker.local/config/0box.yaml"
ssh root@<server-ip> "cd /root/Code/0box/docker.local && docker compose -p 0box restart 0box"
```

---

### Issue: WASM build failed — apps using CDN fallback

**Symptom**: Wallet ops work but use an older WASM version from CDN (`USE_CACHED_WASM=true` was set in build env).

**Rebuild WASM from gosdk source**:
```bash
bash scripts/deploy_local.sh swap-image web-apps master
# Or with a specific gosdk branch:
bash scripts/deploy_local.sh swap-image web-apps master --gosdk-branch fix/my-sdk-fix
```

---

## 8. Quick Diagnostics Reference

### Check all services at once
```bash
ssh root@<server-ip> "
  echo '=== 0box ===' && curl -s http://localhost:9081/v2/health | jq -r '.status' 2>/dev/null || echo DOWN
  echo '=== zauth ===' && curl -s http://localhost:8080/v1/health | jq -r '.status' 2>/dev/null || echo DOWN
  echo '=== zvault ===' && curl -s http://localhost:8090/v1/health | jq -r '.status' 2>/dev/null || echo DOWN
  echo '=== Redis ===' && docker exec 0box-redis redis-cli -a redis_pass ping 2>/dev/null || echo DOWN
  echo '=== 0box blobbers seeded ===' && \
    docker exec -e PGPASSWORD=zbox_server postgres-0box \
    psql -U zbox_user -d zbox -t -c \
    'SELECT COUNT(*) FROM blobbers WHERE not_available = false;' 2>/dev/null
"
```

### Check 0box logs for auth errors
```bash
ssh root@<server-ip> "docker logs 0box-0box-1 --tail 50 2>&1 | grep -iE 'error|firebase|cors|invalid|unauthorized'"
```

### Test CSRF endpoint (verifies Firebase key + 0box are working)
```bash
# With deployment_mode: 0, any token is accepted
ssh root@<server-ip> "curl -s -X GET http://localhost:9081/v2/csrftoken \
  -H 'X-App-ID-TOKEN: test_token' | jq ."
```

### Test blobber query (what Blimp calls when creating allocation)
```bash
ssh root@<server-ip> "curl -s 'http://localhost:9081/v2/blobbers?allocation_type=0' | jq 'length'"
# Should return 12 (number of regular blobbers seeded)
```

### Check nginx CORS headers are correct
```bash
curl -I -X OPTIONS https://test.zus.network/0box/v2/csrftoken \
  -H "Origin: https://test.zus.network" \
  -H "Access-Control-Request-Method: GET" | grep -i "access-control"
# Should show Access-Control-Allow-Origin: https://test.zus.network (not *)
```

### Config files location
| Service | Config |
|---------|--------|
| 0box | `/root/Code/0box/docker.local/config/0box.yaml` |
| zauth | `/root/Code/zauth/config/zauth.yaml` |
| zvault | `/root/Code/zvault/config/zvault.yaml` |
| Firebase key | `/root/Code/0box/docker.local/config/0box_firebase_key.json` |
| nginx | `/etc/nginx/sites-enabled/test.zus.network` (generated by deploy script) |

---

## 9. Post-Deploy Dashboard Verification Checklist

**MUST CHECK AFTER EVERY DEPLOY/REDEPLOY** — run these verifications before declaring a deploy complete.

### 9.1 — 0box Is Connected to LOCAL Chain (not devNet)

**Symptom if wrong**: Blockchain page spins, miners/sharders missing from atlus table, allocations data is from production network.

**Check**:
```bash
ssh root@<server-ip> "curl -s http://localhost:9081/ | grep BlockWorker"
# Must show: BlockWorker: http://198.18.0.100:9091
```

**Fix**:
```bash
ssh root@<server-ip> "
  sed -i 's|block_worker:.*|block_worker: http://198.18.0.100:9091|' /root/Code/0box/docker.local/config/0box.yaml
  sed -i 's|^host:.*|host: https://0box.test.zus.network|' /root/Code/0box/docker.local/config/0box.yaml
  cd /root/Code/0box/docker.local && docker compose -p 0box restart 0box
"
```

**Root cause**: The 0box repo ships with `block_worker: https://dev.zus.network/dns`. The deploy script's `start_0box()` patches this. If 0box was restarted manually (not via deploy script), the fix may not have been applied.

---

### 9.2 — Miners and Sharders Show in Atlus Table with Non-Zero Stake

**Symptom if wrong**: Network score = 0, no rewards data, providers table empty.

**Check**:
```bash
ssh root@<server-ip> "curl -s http://localhost:9081/v2/miners | python3 -c \"
import json,sys
for m in json.load(sys.stdin):
    print(m['id'][:16], 'stake:', m.get('total_stake',0))
\""
# All miners should have total_stake > 0
```

**Fix**: Stake 10 ZCN on each miner/sharder (the deploy script's `check_and_fund_providers` now handles this automatically):
```bash
# Manual staking if needed:
ssh root@<server-ip> "
export PATH=\$PATH:/root/Code/zwalletcli
for MID in \$(curl -s 'http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList' | python3 -c \"import json,sys; [print(n.get('simple_miner',n)['id']) for n in json.load(sys.stdin).get('Nodes',[])]\"); do
  zwallet mn-lock --miner_id \"\$MID\" --tokens 10 --configDir /root/.zcn --wallet local.json --silent 2>/dev/null
done
for SID in \$(curl -s 'http://198.18.0.81:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList' | python3 -c \"import json,sys; [print(n.get('simple_miner',n)['id']) for n in json.load(sys.stdin).get('Nodes',[])]\"); do
  zwallet mn-lock --sharder_id \"\$SID\" --tokens 10 --configDir /root/.zcn --wallet local.json --silent 2>/dev/null
done
"
```

**Note**: The `total_stake` shown in 0box reflects what was aggregated via Kafka events. Allow a few minutes after staking for it to appear.

---

### 9.3 — CORS Preflight Works for Miners/Sharders

**Symptom if wrong**: Browser console shows `Preflight response is not successful. Status code: 301` when web apps call `/miner03`, `/sharder01`, etc.

**Root cause**: nginx location `/miner03/` redirects bare `/miner03` (no trailing slash) to `/miner03/`. The nginx sub_filter rewrites internal IPs like `http://198.18.0.73:7073` → `/miner03` (no slash). When the web app fetches this URL, it gets a 301 that fails CORS preflight.

**Fix** (deploy script now includes trailing slashes in sub_filter replacements):
```bash
# sub_filter lines should use trailing slash:
#   sub_filter 'http://198.18.0.73:7073' '/miner03/';
grep "sub_filter.*miner03" /etc/nginx/sites-enabled/test.zus.network
# If missing trailing slash, regenerate nginx:
bash scripts/deploy_local.sh nginx
```

---

### 9.4 — Allocation Utilization Not Negative

**Symptom if wrong**: Atlus shows negative allocation utilization (e.g., -4.9 GB).

**Cause**: The `allocated_storage` column in `snapshots` table goes negative when test allocations expire/finalize and their storage is decremented. This is a data artifact from expired test allocations.

**Check**:
```bash
ssh root@<server-ip> "docker exec postgres-0box psql -U zbox_user -d zbox -c \
  'SELECT round, allocated_storage FROM snapshots ORDER BY round DESC LIMIT 3;'"
```

**Fix**: The display is a UI concern. Negative values come from the chain's own accounting when allocations expire. The crawler's `auto_renew` service should prevent this for crawler-owned allocations. For test allocations (created by test wallets), they will expire normally.

**Important**: The crawler can only renew allocations IT created. Test allocations from test wallets will always expire. This is expected behavior.

---

### 9.5 — Crawler Is Running and Uploading Successfully

**Symptom if wrong**: Network score = 0, no challenge data, empty rewards charts, no allocation utilization — even though the chain is running.

**Check**:
```bash
ssh root@<server-ip> "
# Is crawler running?
docker ps | grep crawler

# Is it uploading (no 'invalid_operation' errors)?
docker logs crawler --since 5m 2>&1 | grep -E 'error|upload|alloc' | head -10

# Monitor log (shows renewal history)
cat /tmp/crawler_monitor.log | tail -20

# Is current allocation valid?
ALLOC=\$(grep -A1 '^allocations:' /root/Code/crawler/docker.local/config/crawler.yaml | grep '  - ' | head -1 | sed 's/.*- //')
/root/Code/zboxcli/zbox getallocation --allocation \"\$ALLOC\" --wallet local.json --configDir /root/.zcn --config config.yaml --silent 2>/dev/null | grep -E 'id:|finalized|expiration'
"
```

**Root cause when broken**: Crawler has a stale allocation from a previous deploy. Blobbers reject uploads with `invalid_operation: Operation needs to be performed by the owner or the payer of the allocation`. This happens because:
1. The crawler config keeps the old allocation ID after a redeploy
2. Old allocations expire (typically ~1-2 hours after creation due to short lock amount) or are from a previous chain state

**Critical: Crawler wallet must own the allocation**. The crawler uses `local.json` for all operations. The allocation must also be created with `local.json`. If the crawler config was generated with a different wallet (e.g., from an old deploy), the wallet IDs won't match and uploads will fail.

**Verify wallet consistency**:
```bash
ssh root@<server-ip> "
echo 'Crawler wallet:' && grep 'wallet_client_id' /root/Code/crawler/docker.local/config/crawler.yaml
echo 'local.json wallet:' && jq -r '.client_id' /root/.zcn/local.json
# These MUST match
"
```

**Fix: Regenerate crawler config and allocation**:
```bash
# Delete stale config and regenerate from local.json
ssh root@<server-ip> "
mv /root/Code/crawler/docker.local/config/crawler.yaml /root/Code/crawler/docker.local/config/crawler.yaml.bak
export PATH=\$PATH:/root/Code/zboxcli:/root/Code/zwalletcli
cd /root/Code/system_test
bash scripts/deploy_local.sh crawler
"
```

**Perpetual monitor**: The deploy script runs `start_crawler_monitor` which starts a background loop (30-min interval) that detects expired/invalid allocations, creates new ones, and restarts the crawler. Check its status:
```bash
ssh root@<server-ip> "cat /tmp/crawler_monitor.pid; cat /tmp/crawler_monitor.log | tail -10"
# Restart monitor if needed:
# bash scripts/deploy_local.sh crawler-monitor
```

---

### 9.6 — Charts and Aggregates Data

**Symptom if wrong**: Rewards charts empty, unique address chart missing, allocated capacity/used storage charts blank.

**Root cause**: Charts read from `snapshots`, `miner_aggregates`, `sharder_aggregates`, `blobber_aggregates` tables. These are populated via the Kafka pipeline (sharder → Kafka → 0box). After a fresh deploy, it takes one Kafka cycle (~2 min) for data to appear.

**Check**:
```bash
ssh root@<server-ip> "docker exec postgres-0box psql -U zbox_user -d zbox -c \
  'SELECT COUNT(*) FROM snapshots; SELECT COUNT(*) FROM miner_aggregates; SELECT COUNT(*) FROM blobber_aggregates;'"
# snapshots should have 1000+ rows after a few minutes
# miner_aggregates should have rows per miner per round
```

**Check Kafka pipeline is flowing**:
```bash
ssh root@<server-ip> "docker logs 0box --tail 5 2>&1 | grep kafka_debug"
# Should show: {"Round": <recent_round>, "LastRound": <previous_round>}
```

If Kafka pipeline is broken, run:
```bash
bash scripts/deploy_local.sh fix-kafka
```

**Note**: `unique_addresses` comes from the `unique_addresses` column in `snapshots`. If it's 0, it means the Kafka events haven't included user transaction events yet. Wait for chain activity or create test transactions.

---

### 9.7 — vult.network Auth Works

**Symptom if wrong**: `Unauthorized. error verifying header token` when opening test.vult.network.

**Check sequence**:
```bash
# 1. Is 0box connected to local chain (not devNet)?
ssh root@<server-ip> "curl -s http://localhost:9081/ | grep BlockWorker"
# Must be local: http://198.18.0.100:9091

# 2. Is 0box running in deployment_mode 0 (bypass Firebase token verification)?
ssh root@<server-ip> "grep deployment_mode /root/Code/0box/docker.local/config/0box.yaml"
# Should show: --deployment_mode 0 (in the docker-compose command)

# 3. Is vult's PM2 process running?
ssh root@<server-ip> "pm2 list | grep vult"
# Should show: online

# 4. Test 0box CSRF endpoint (simulates what vult calls on login):
ssh root@<server-ip> "curl -s -X GET http://localhost:9081/v2/csrftoken \
  -H 'X-App-ID-TOKEN: any_test_token' -w '\nHTTP: %{http_code}\n'"
```

**Fix**: If 0box is on devNet (block_worker wrong), fix as in 9.1. If PM2 vult is crashing with `NO_SECRET` from next-auth, restart PM2:
```bash
ssh root@<server-ip> "pm2 restart vult"
```

---

### 9.8 — Blockchain Page Works (not spinning)

**Symptom if wrong**: Atlus/Explorer blockchain page shows spinner indefinitely.

**Check**: The blockchain page calls 0box REST endpoints for block/transaction data. If 0box is on devNet or not responding, these return 404/empty.

```bash
# Check 0box miners API (what blockchain page uses):
ssh root@<server-ip> "curl -s 'http://localhost:9081/v2/miners' | python3 -c 'import json,sys; print(len(json.load(sys.stdin)), \"miners\")'"
# Should show: 4 miners

# Check 0box is connected to local blocks (not devNet blocks):
ssh root@<server-ip> "curl -s http://localhost:9081/ | grep 'Working on the chain'"
```

**Fix**: Ensure 0box is on local chain (fix 9.1). After restart, wait ~30 seconds for 0box to discover local chain and start processing blocks.

---

### Summary: Quick Post-Deploy Check Script

Run this after every deploy to verify all dashboard components:

```bash
ssh root@<server-ip> bash << 'EOF'
echo "=== 1. 0box chain connection ==="
curl -s http://localhost:9081/ | grep -o 'BlockWorker: [^<]*'

echo ""
echo "=== 2. Miners in 0box ==="
curl -s http://localhost:9081/v2/miners | python3 -c "
import json,sys
try:
    m = json.load(sys.stdin)
    for miner in m:
        print(miner['id'][:16], 'stake:', miner.get('total_stake', 0))
except: print('ERROR')
"

echo ""
echo "=== 3. CORS preflight for miner03 ==="
curl -sk -X OPTIONS 'https://test.zus.network/miner03/' \
  -H 'Origin: https://test.vult.network' \
  -H 'Access-Control-Request-Method: POST' \
  -w 'HTTP: %{http_code}\n' | tail -1

echo ""
echo "=== 4. Kafka pipeline ==="
docker logs 0box --tail 3 2>&1 | grep kafka_debug

echo ""
echo "=== 5. Snapshot counts ==="
docker exec postgres-0box psql -U zbox_user -d zbox -t -c \
  'SELECT COUNT(*) as snapshots FROM snapshots;' 2>&1

echo ""
echo "=== 6. Web apps running ==="
pm2 list | grep -E 'vult|bolt|blimp|explorer|chimney' | awk '{print $2, $10}'
EOF
```
