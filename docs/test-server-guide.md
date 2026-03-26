# Test Server Guide

## Servers

| Name | IP | SSH | DNS |
|------|----|-----|-----|
| test | 37.27.65.188 | `ssh root@37.27.65.188` | test.zus.network |
| test1 | 65.108.232.28 | `ssh root@65.108.232.28` | test1.zus.network |
| test2 | 144.76.58.147 | `ssh root@144.76.58.147` | test2.zus.network |

> Passwords are shared via internal channels. Do not commit credentials to this repo.

Each server runs the full stack: 4 miners, 2 sharders, 12 blobbers, 12 validators, 5 enterprise blobbers, 0box, 0dns, kafka, elasticsearch, crawler, zauth, zvault, and monitoring.

## Repo Locations on Servers

All repos are at `/root/Code/<repo>`:
```
/root/Code/0chain      # miners + sharders
/root/Code/blobber     # blobbers + validators
/root/Code/0box        # 0box
/root/Code/0dns        # 0dns
/root/Code/gosdk       # gosdk (dependency)
/root/Code/system_test # deploy scripts + tests
/root/Code/crawler     # crawler
```

## Deploy Script

The deploy script is at `/root/Code/system_test/scripts/deploy_local.sh`.

### Swap Image (Rebuild & Restart ONE Service)

This is the **safest way** to update code. It pulls the branch, builds, and restarts containers. **Does NOT delete chain data.**

```bash
cd /root/Code/system_test/scripts

# Swap 0chain (miners + sharders)
./deploy_local.sh swap-image 0chain                        # uses branch from deploy_config.yaml
./deploy_local.sh swap-image 0chain fix/my-branch          # specific branch

# Swap blobber (blobbers + validators)
./deploy_local.sh swap-image blobber
./deploy_local.sh swap-image blobber fix/my-branch

# Swap with custom gosdk branch
./deploy_local.sh swap-image blobber staging --gosdk-branch fix/sdk-branch

# Other services
./deploy_local.sh swap-image 0box
./deploy_local.sh swap-image 0dns
./deploy_local.sh swap-image crawler
./deploy_local.sh swap-image zauth-server
./deploy_local.sh swap-image zvault
./deploy_local.sh swap-image web-apps
```

### Branch Configuration

Default branches are in `/root/Code/system_test/scripts/deploy_config.yaml`:
```yaml
repositories:
  0chain:
    branch: fix/dkg-broadcast-fee
  blobber:
    branch: fix/finalize-worker-and-logs
  0box:
    branch: fix/kafka-multi-sharder-chaos
  gosdk:
    branch: fix/lfb-aware-sharder-selection
```

To change defaults, edit `deploy_config.yaml` or pass the branch on command line.

### Verify Deployment

```bash
./deploy_local.sh verify    # Health check all services
./deploy_local.sh smoke     # Quick smoke test
```

## Running Tests

```bash
cd /root/Code/system_test/scripts

# Run all tests
./deploy_local.sh test

# Run specific test suites
./deploy_local.sh test api          # API tests
./deploy_local.sh test cli          # CLI tests
./deploy_local.sh test tokenomics   # Tokenomics tests
./deploy_local.sh test sdk          # SDK tests
```

## Chain Configuration (After Deploy)

```bash
cd /root/Code/system_test/scripts
./deploy_local.sh chain    # Reconfigure SC settings + hardforks
```

## Useful Commands

### Check Chain Status
```bash
# From the server
curl -s http://localhost:7171/v1/chain/get/stats | python3 -c 'import json,sys; d=json.load(sys.stdin); print("round:", d.get("current_round"), "lfb:", d.get("latest_finalized_round"))'

# From anywhere
curl -s https://test.zus.network/sharder01/v1/chain/get/stats
```

### Check Container Status
```bash
docker ps --format 'table {{.Names}}\t{{.Status}}' | sort
```

### View Logs
```bash
# Miner/Sharder application logs
tail -f /root/Code/0chain/docker.local/miner1/log/0chain.log
tail -f /root/Code/0chain/docker.local/sharder1/log/0chain.log

# Blobber logs
docker logs blobber-1 --tail 50

# 0box logs
docker logs 0box --tail 50
```

### Restart Individual Services
```bash
# Restart a single miner/sharder (no rebuild)
docker restart miner-1
docker restart sharder-1

# Restart a blobber
docker restart blobber-1

# Restart 0box
docker restart 0box
```

## ⚠️ Important Notes

1. **Do NOT run `clean.sh`** — it deletes all chain data and requires full redeploy
2. **Do NOT run `docker system prune`** without checking — it can remove volumes with chain data
3. **`swap-image` is safe** — it rebuilds and restarts without deleting data
4. **Start order matters for full restart**: sharders first, then ALL miners simultaneously, then blobbers
5. **Blobber PebbleDB**: Volume-mounted at `blobber${N}/pebble/`. Do not delete — contains challenge Merkle trees
6. **Daily cleanup cron** runs at 4am: prunes unused Docker images, build cache, and large container logs

## Diagnostics URLs

| Service | URL |
|---------|-----|
| Sharder diagnostics | `https://test.zus.network/sharder01/_diagnostics` |
| Miner diagnostics | `http://localhost:7171/_diagnostics` (from server) |
| 0box latest snapshot | `https://test.zus.network/v2/latest-snapshot` |
| Portainer | `http://<server-ip>:9000` |
| Kafka UI | `http://<server-ip>:8080` |
| pgAdmin (sharder) | `http://<server-ip>:5050` |
