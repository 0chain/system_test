# Infrastructure Fixes Reference

Known issues discovered during local deployment and their fixes. These are automatically handled by `deploy_local.sh` in `fix_blobber_config()` and related functions.

## 0box Redis Crash (SIGSEGV)

**Symptom**: `0box-redis` container crash-loops with `signal: 11` (SIGSEGV) immediately on startup.

**Root Cause**: The 0box repo's `docker-compose.yml` uses `image: "redis:alpine"` which now pulls Redis 8.x. Redis 8.6.0+ crashes with a null-pointer dereference on certain kernel versions (tested on `6.8.0-85-generic`).

**DevNet uses**: `redis:7.4.3-alpine` (see `0helm/values/dev_activeset/0box/docker-compose-0box.yml`)

**Fix**: The deploy script pins `redis:alpine` to `redis:7.4.3-alpine` in the 0box docker-compose. This is done automatically in `fix_blobber_config()`.

**Manual fix**:
```bash
sed -i 's|image: "redis:alpine"|image: "redis:7.4.3-alpine"|' /root/Code/0box/docker.local/docker-compose.yml
cd /root/Code/0box/docker.local && docker compose -p 0box up -d --force-recreate redis
# Wait for redis healthy, then restart 0box
docker compose -p 0box up -d 0box
```

## zvault Postgres Connection Refused

**Symptom**: `zvault-zvault-1` exits with `failed to connect to host=localhost database=zvault: connection refused`.

**Root Cause**: The zvault repo's default `config/zvault.yaml` has `postgres.host: localhost`. Inside the Docker container, `localhost` means the container itself, not the postgres container. The postgres runs in a separate container (`zvault-postgreszv-1`) accessible via the compose service name `postgreszv`.

**DevNet uses**: `postgres.host: zvault-postgres` with a custom docker-compose that renames the service (see `0helm/values/dev_activeset/zvault/zvault.yaml`).

**Fix**: The deploy script changes `host: localhost` to `host: postgreszv` (the compose service name) in `zvault.yaml`. This is done automatically in `fix_blobber_config()`.

**Manual fix**:
```bash
sed -i 's|host: localhost|host: postgreszv|' /root/Code/zvault/config/zvault.yaml
cd /root/Code/zvault/docker.local && docker compose -p zvault up -d --force-recreate zvault
```

## Enterprise Blobber block_worker

**Symptom**: `eblobber-N` containers crash with `block_worker: https://dev.0chain.net/dns: no such host`.

**Root Cause**: The eblobber repo's `config/0chain_blobber.yaml` ships with `block_worker: https://dev.0chain.net/dns` which is the devNet URL. For local deployment, it must point to `http://198.18.0.100:9091` (local 0dns).

**Fix**: The deploy script replaces `block_worker` in the enterprise blobber config. This is done in both `fix_blobber_config()` and the `swap-image eblobber` restart section.

**Manual fix**:
```bash
sed -i 's|block_worker:.*|block_worker: http://198.18.0.100:9091|' /root/Code/eblobber/config/0chain_blobber.yaml
# Restart enterprise blobbers
for i in 1 2 3 4 5; do docker restart eblobber-$i 2>/dev/null; done
```

## Enterprise Blobber Port Conflicts

**Symptom**: `eblobber-N` containers fail to start with `port is already allocated` when using compose project names `eblobber1`, `eblobber2`, etc.

**Root Cause**: The enterprise blobber `b0docker-compose.yml` uses `505${BLOBBER}` port pattern, which overlaps with regular blobbers (blobber-1 uses 5051, blobber-2 uses 5052, etc.). The deploy script avoids this by using higher BLOBBER indices (e.g., BLOBBER=1 through 5 are mapped to enterprise-specific ports like `507N` via `eb0docker-compose.yml`).

**Key**: Enterprise blobbers MUST use `eb0docker-compose.yml` (not `b0docker-compose.yml`) which has non-overlapping port ranges. The deploy script handles this correctly.

## DevNet vs Local Docker-Compose Differences

| Aspect | DevNet (0helm) | Local (deploy_local.sh) |
|--------|---------------|------------------------|
| Redis image | `redis:7.4.3-alpine` | `redis:alpine` (auto-pinned to 7.4.3) |
| zvault postgres host | `zvault-postgres` (custom service name) | `postgreszv` (repo default service name, auto-fixed) |
| zauth postgres host | `zauth-postgres` (custom service name) | `postgres` (repo default, works with compose project) |
| eblobber block_worker | Points to devNet 0dns | Auto-fixed to `http://198.18.0.100:9091` |
| Docker images | Pre-built from DockerHub (`0chaindev/*:staging`) | Built locally from source |
| Network | `testnet0` with static IPs | `testnet0` with static IPs (same) |
