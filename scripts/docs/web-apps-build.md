# Web-Apps Build Process

## Overview

The deploy script builds 5 web apps from the `0chain/web-apps` monorepo:
- **Vult** - Personal cloud storage
- **Bolt** - Token management
- **Blimp** - Enterprise storage
- **Explorer** (Atlus) - Blockchain explorer
- **Chimney** - Data marketplace

Chalk and Atlus (standalone) are excluded from our test deployment.

## Build Approach

We match the **staging CI** pattern from `0chain/web-apps/.github/workflows/staging-webapp-cicd.yaml`:

| Aspect | Staging CI | Our Deploy |
|--------|-----------|------------|
| Package manager | `yarn install --frozen-lockfile` | Same |
| Build command | `yarn workspace <app> build` | Same |
| WASM | Built from gosdk + egosdk | Built from gosdk only (same zcn.wasm for all apps) |
| Firebase | box-staging-edd6f | **box-dev-ce8bf** (our test project) |
| Serving | Docker containers | PM2 + nginx reverse proxy |
| packages/shared/.env | Yes, per-app | Same |

## WASM Build

`zcn.wasm` is built from the `0chain/gosdk` repo using Docker:

```bash
docker run --rm -v "$gosdk_dir":/gosdk -w /gosdk golang:1.22.5 sh -c \
    "git config --global --add safe.directory /gosdk; make wasm-build"
```

The same `zcn.wasm` is copied to ALL apps (`packages/*/public/zcn.wasm`).

**Note**: Staging CI also builds `enterprise-zcn.wasm` from `0chain/egosdk` for Blimp's enterprise storage features. We skip this — all apps use the standard `zcn.wasm`. Enterprise storage operations in Blimp will not work.

If WASM build fails, apps fall back to CDN download via `USE_CACHED_WASM=true`.

## Environment Variables

### Per-App Differences

| Variable | Vult | Blimp | Bolt | Explorer | Chimney |
|----------|------|-------|------|----------|---------|
| ZBOX_APP | vult | blimp | bolt | **atlus** | chimney |
| GOSDK_VERSION | 1.20.9 | 1.20.9 | - | - | - |
| EGOSDK_VERSION | - | 1.19.0 | - | - | - |
| USE_CACHED_WASM | true | true | - | - | - |
| TRANSLATION_API_KEY | yes | yes | (empty) | yes | yes |
| ATLUS_URL | - | - | yes | - | yes |
| DATALAKE_API_URL | - | yes | - | - | - |
| Auth0 | yes | yes | yes | **no** | yes |

### Firebase (box-dev-ce8bf)

```
FBASE_API_KEY=AIzaSyBvR4zq1ZtYbQ-5Sha1P9PGLZr3QPn2jaY
FBASE_AUTH_DOMAIN=box-dev-ce8bf.firebaseapp.com
FBASE_PROJECT_ID=box-dev-ce8bf
FBASE_STORAGE_BUCKET=box-dev-ce8bf.firebasestorage.app
FBASE_MESSAGING_SENDER_ID=893964718514
FBASE_APP_ID=1:893964718514:web:6d6f2ee9f96211e64954ab
```

### Secrets (from `scripts/.secrets.env`)

These must be set in `.secrets.env` (gitignored):

```bash
FBASE_API_KEY=...          # Optional override (defaults to box-dev)
WEBHOOK_API_TOKEN=...      # Required
ZENDESK_KEY=...            # Required
ETH_NODE_URL=...           # Required (Ethereum RPC)
NFT_NODE_URL=...           # Required (Polygon RPC)
RECAPTCHA_KEY=...          # Required
ALCHEMY_API_KEY=...        # Required
AUTH0_SECRET=...           # Required
AUTH0_ISSUER_BASE_URL=...  # Required
AUTH0_CLIENT_ID=...        # Required
AUTH0_CLIENT_SECRET=...    # Required
```

### packages/shared/.env

Following the staging CI pattern, `packages/shared/.env` is written as a copy of each app's `.env` before that app's build. This ensures shared components have access to the same env vars at build time.

## basePath Injection

Since we serve apps under nginx subpaths (`/vult/`, `/bolt/`, etc.), each app's `next.config.js` needs a `basePath` property. The build script injects this automatically by detecting the config pattern:

- `module.exports = {` → Blimp
- `const nextConfig = {` → Vult
- `nextTranslate({` → Chimney/Bolt/Explorer
- `withTM({` → Bolt/Explorer

## Ports

| App | Port | Nginx Path |
|-----|------|------------|
| Explorer | 3001 | /explorer/ |
| Bolt | 3002 | /bolt/ |
| Vult | 3003 | /vult/ |
| Chimney | 3005 | /chimney/ |
| Blimp | 3006 | /blimp/ |

## Commands

```bash
# Full build (WASM + yarn + all apps)
bash scripts/deploy_local.sh web-apps

# Rebuild with specific gosdk branch
bash scripts/deploy_local.sh swap-image web-apps --gosdk-branch fix/my-branch

# Just restart app servers (no rebuild)
# (manually call start_web_apps from deploy script)
```

## Differences from Staging CI

1. **No enterprise-zcn.wasm** — We use the same zcn.wasm everywhere
2. **Firebase project** — We use box-dev-ce8bf, staging uses box-staging-edd6f
3. **basePath injection** — Required for our nginx subpath routing (staging uses Docker per-container)
4. **PM2 serving** — We use PM2 + `npx next start`, staging uses Docker with `yarn workspace <app> start`
5. **Contract addresses** — Different network, different addresses (expected)
