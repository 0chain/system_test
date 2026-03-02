# swap-image Command Reference

Rebuild and restart a single service from a specific branch without full redeployment.

## Syntax

```bash
bash scripts/deploy_local.sh swap-image <repo> [branch] [--gosdk-branch <branch>]
```

- **repo**: Service to rebuild (see table below)
- **branch**: Git branch to build from. If omitted, uses `deploy_config.yaml` → `master` fallback
- **--gosdk-branch**: Override the gosdk branch used for WASM or dependent builds

## Commands by Service

### Web-Apps (Vult, Bolt, Blimp, Explorer, Chimney)

```bash
# Swap to a branch (builds WASM + all 5 apps + restarts PM2)
bash scripts/deploy_local.sh swap-image web-apps player-fmp4

# Swap with a different gosdk branch for WASM build
bash scripts/deploy_local.sh swap-image web-apps player-fmp4 --gosdk-branch fix/my-gosdk

# Rebuild on current branch with new gosdk (keeps web-apps branch unchanged)
bash scripts/deploy_local.sh swap-image web-apps --gosdk-branch fix/new-gosdk-branch
```

**What happens**: Checkout branch → build zcn.wasm from gosdk via Docker → `yarn install --frozen-lockfile` → build each app → restart PM2 processes.

### 0box

```bash
bash scripts/deploy_local.sh swap-image 0box transcoder-updated
```

**What happens**: Checkout branch → rebuild `zbox_base` → `docker build` (same as CI's `build.zbox.sh`) → tag as `0chaindev/0box:staging` → restart container.

### Blobber (regular blobbers + validators)

```bash
# Swap all regular blobbers to a branch
bash scripts/deploy_local.sh swap-image blobber feat/public-share-with-revoke

# Swap to staging (default from config)
bash scripts/deploy_local.sh swap-image blobber
```

**What happens**: Checkout branch → rebuild `blobber_base` (with BuildKit) → build blobber image → build validator image → restart all blobber + validator containers.

### Enterprise Blobber (eblobber)

```bash
# Swap eblobber to a branch (uses gosdk from config)
bash scripts/deploy_local.sh swap-image eblobber staging

# Swap eblobber with a specific gosdk branch
bash scripts/deploy_local.sh swap-image eblobber staging --gosdk-branch enterprise-blobber
```

**What happens**: Checkout gosdk → checkout eblobber branch → rebuild base → build eblobber image → restart enterprise blobber containers.

### 0chain (miners + sharders)

```bash
bash scripts/deploy_local.sh swap-image 0chain fix/dkg-broadcast-fee
```

**What happens**: Checkout branch → always rebuild `zchain_build_base`/`zchain_run_base` (with BuildKit) → build miner image → build sharder image → stop all miners/sharders → start sharders first → start miners → verify all running.

### Zauth / Zvault

```bash
bash scripts/deploy_local.sh swap-image zauth-server staging
bash scripts/deploy_local.sh swap-image zvault staging
```

**What happens**: Checkout branch → `docker build` from `docker.local/Dockerfile` → restart container.

### zs3server (S3 gateway)

```bash
bash scripts/deploy_local.sh swap-image zs3server feat/enterprise-timings --gosdk-branch enterprise-blobber
```

**What happens**: Checkout gosdk → checkout zs3server branch → `docker compose build` → restart container.

### CLI Tools (zboxcli / zwalletcli)

```bash
bash scripts/deploy_local.sh swap-image zboxcli staging
bash scripts/deploy_local.sh swap-image zwalletcli staging
```

**What happens**: Checkout branch → `go build` → copy binary to test directories. No Docker images.

## Branch Resolution

When no branch is specified, swap-image resolves in this order:

1. **CLI argument**: `swap-image blobber fix/my-branch` (highest priority)
2. **`scripts/deploy_config.yaml`**: reads `repositories → <repo> → branch`
3. **Fallback**: `master`

## gosdk Dependency

gosdk is **not a standalone service** — it's built as a dependency:

| Service | Uses gosdk for | Config field |
|---|---|---|
| web-apps | zcn.wasm (WASM binary) | `web-apps.gosdk_branch` |
| eblobber | Go module dependency | `eblobber.gosdk_branch` |
| zs3server | Go module dependency | `zs3server.gosdk_branch` |

To change which gosdk branch is used, either:
- Pass `--gosdk-branch fix/branch` on the command line
- Edit `gosdk_branch` under the service in `scripts/deploy_config.yaml`

## Build Details

- All Docker builds use **BuildKit** (`DOCKER_BUILDKIT=1`) for compatibility with newer Dockerfiles
- Base images (`blobber_base`, `zbox_base`) are **always rebuilt** during swap to handle Go version changes across branches
- Build output is truncated to last 10-20 lines; errors propagate correctly via `pipefail`
- 0box is built using the repo's `build.zbox.sh` script (matching CI), then retagged as `0chaindev/0box:staging` for docker-compose compatibility
- gosdk is automatically restored to its previous branch after swap completes (prevents system_test `replace ../gosdk` breakage)

## Automatic Config Fixes on Restart

These fixes are applied automatically during the restart phase (git checkout resets config files):

| Service | Fix Applied | Why |
|---------|------------|-----|
| **0chain** | `delegate_wallet` → SC owner, `num_delegates` → 200, Kafka config (server_chain + root), 60s sharder wait before miners, verify miners running | delegate_wallet immutable after registration; Kafka defaults to disabled/wrong host; miners need sharders for LFB sync |
| **blobber** | `fix_blobber_config()` + `fix_validator_config()`: delegate_wallet, block_worker, storage_version, read_price, service_charge, num_delegates | All config resets on git checkout; blobbers 10-12 use specific compose files (invalid IPs otherwise) |
| **eblobber** | `block_worker` → local 0dns, `delegate_wallet` → SC owner, `service_charge` ≤ 0.5, `num_delegates` ≥ 10 | Config ships with dev.0chain.net, placeholder wallet, and invalid defaults |
| **0box** | Pin redis to `redis:7.4.3-alpine`, fix Kafka config (enabled + local host), copy Firebase key, seed snapshots | `redis:alpine` (v8+) SIGSEGV; Kafka defaults disabled; Firebase key not in git; empty snapshots = deadlock |
| **zauth-server** | Sync JWT secret with 0box | JWT mismatch → "signature is invalid" on all auth endpoints |
| **zvault** | `postgres.host` → `postgreszv`, `zauth_server` → Docker bridge gateway, sync JWT secret with 0box | Config ships with `localhost` (wrong inside Docker); JWT mismatch breaks auth |
| **zs3server** | Build `zbox_base` + main Dockerfile, use local compose | Default compose has no build, uses DockerHub images |

## Examples

```bash
# Developer workflow: new gosdk → rebuild blobbers → run tests
bash scripts/deploy_local.sh swap-image blobber --gosdk-branch fix/my-sdk
bash scripts/deploy_local.sh test

# New 0box feature → rebuild → run API tests
bash scripts/deploy_local.sh swap-image 0box fix/my-feature
bash scripts/deploy_local.sh test api

# Web-app change with custom WASM → rebuild → verify
bash scripts/deploy_local.sh swap-image web-apps fix/ui-change --gosdk-branch fix/sdk-change

# Rebuild chain from a branch → reconfigure → verify
bash scripts/deploy_local.sh swap-image 0chain fix/consensus-change
bash scripts/deploy_local.sh chain
bash scripts/deploy_local.sh verify
```
