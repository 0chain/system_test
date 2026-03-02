# Web-Apps Update Plan

## Quick Reference

```bash
# Rebuild all 5 web apps on current branch (master)
bash scripts/deploy_local.sh web-apps

# Swap to a different branch + rebuild
bash scripts/deploy_local.sh swap-image web-apps <branch>

# Swap with custom gosdk for WASM build
bash scripts/deploy_local.sh swap-image web-apps <branch> --gosdk-branch <gosdk-branch>
```

## What Gets Built

| App | Port | URL | nginx path |
|-----|------|-----|------------|
| Vult | 3003 | test.vult.network | /vult |
| Bolt | 3002 | test.bolt.holdings | /bolt |
| Blimp | 3006 | test.blimp.software | /blimp |
| Explorer (Atlus) | 3001 | test.zus.network, test.atlus.cloud | /explorer |
| Chimney | 3005 | test.chimney.software | /chimney |

Chalk is excluded from test deployment.

## Build Pipeline

1. **Generate .env files** — from `scripts/.secrets.env` template, per-app substitution
2. **Build zcn.wasm** — from gosdk (`staging` branch by default)
3. **Install dependencies** — `yarn install` (monorepo with workspaces)
4. **Build shared package** — dependency for all apps
5. **Inject basePath** — `/vult`, `/bolt`, etc. into each `next.config.js`
6. **Build each app** — `yarn workspace <app> build` (Next.js production build)
7. **Start via PM2** — `npx next start -p <port>` per app
8. **Health check** — HTTP 200 on each port

## Updating Web Apps

### Scenario 1: Code change in web-apps repo

```bash
# On remote
cd /root/Code/web-apps
git fetch origin && git checkout <branch> && git pull

# Rebuild
cd /root/Code/system_test
bash scripts/deploy_local.sh web-apps
```

Or use swap-image (handles git checkout automatically):
```bash
bash scripts/deploy_local.sh swap-image web-apps <branch>
```

### Scenario 2: GoSDK/WASM update

If gosdk changes affect the WASM SDK used by web apps:
```bash
bash scripts/deploy_local.sh swap-image web-apps master --gosdk-branch <new-gosdk-branch>
```

This rebuilds `zcn.wasm` from the specified gosdk branch and copies it to all apps.

### Scenario 3: Environment/secret change

1. Edit `scripts/.secrets.env` (never edit `.env` files directly on server)
2. Run: `bash scripts/deploy_local.sh web-apps`

This regenerates all `.env` files and rebuilds (env values are baked at build time in Next.js).

### Scenario 4: Nginx routing change

```bash
bash scripts/deploy_local.sh nginx
```

## Configuration

### Branch Resolution (priority order)
1. CLI flag: `--branch web-apps=<branch>`
2. `scripts/deploy_config.yaml` → `repositories.web-apps.branch`
3. Fallback: `master`

### Required Secrets (`scripts/.secrets.env`)

| Secret | Purpose |
|--------|---------|
| FBASE_API_KEY | Firebase authentication (required) |
| AUTH0_SECRET, AUTH0_CLIENT_ID, AUTH0_CLIENT_SECRET | Auth0 (Blimp, Vult) |
| ALCHEMY_API_KEY | Ethereum RPC |
| ETH_NODE_URL, NFT_NODE_URL | Ethereum/Polygon nodes |
| RECAPTCHA_KEY | Google reCAPTCHA |
| WEBHOOK_API_TOKEN | Webhook notifications |
| ZENDESK_KEY | Support widget |

### WASM Caching

Apps use `USE_CACHED_WASM=true` — at runtime, they download pre-built WASM from CDN using `GOSDK_VERSION=1.20.4`. The local `zcn.wasm` build serves as fallback.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| All apps return 404 | PM2 not running | `bash scripts/deploy_local.sh web-apps` |
| App loads but Firebase errors | Wrong FBASE_API_KEY or domain not authorized | Check `.secrets.env`, add domain in Firebase Console |
| "no provider match" on Blimp | 0box blobber table empty | `bash scripts/deploy_local.sh seed-explorer` |
| WASM download fails | CDN version mismatch | Rebuild with matching gosdk branch |
| basePath routing broken | stale next.config.js | Delete `.next/` directories, rebuild |

## Architecture Notes

- **Monorepo**: yarn workspaces, shared package as common dependency
- **Build output**: `.next/` directory per app (Next.js SSR/static)
- **Process manager**: PM2 (auto-restart, log management)
- **Reverse proxy**: nginx handles SSL, domain routing, WebSocket upgrade
- **No Docker**: Apps run as native Node.js processes (unlike staging which uses Docker)
- **Env baked at build**: Next.js `NEXT_PUBLIC_*` vars are compiled in — must rebuild after changes
