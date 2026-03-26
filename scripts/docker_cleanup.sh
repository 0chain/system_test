#!/bin/bash
# Daily Docker cleanup — removes unused images, build cache, and truncates large container logs
# Deployed via cron: 0 4 * * * /root/scripts/docker_cleanup.sh >> /var/log/docker_cleanup.log 2>&1

echo "=== Docker cleanup started at $(date) ==="

# Disk before
BEFORE=$(df -h / | tail -1 | awk '{print $4, $5}')
echo "Disk before: $BEFORE"

# 1. Remove unused Docker images (not used by any container)
echo "--- Pruning unused images ---"
docker image prune -af --filter "until=72h" 2>/dev/null | tail -2

# 2. Remove build cache older than 3 days
echo "--- Pruning build cache ---"
docker builder prune -af --filter "until=72h" 2>/dev/null | tail -2

# 3. Remove dangling volumes (not used by any container)
echo "--- Pruning dangling volumes ---"
docker volume prune -f 2>/dev/null | tail -2

# 4. Truncate Docker container logs > 500MB
echo "--- Truncating large container logs ---"
for f in /var/lib/docker/containers/*/*-json.log; do
    sz=$(stat -c%s "$f" 2>/dev/null || echo 0)
    if [ "$sz" -gt 524288000 ]; then
        cid=$(basename $(dirname "$f"))
        name=$(docker inspect --format "{{.Name}}" "$cid" 2>/dev/null | tr -d "/")
        echo "Truncating log for ${name:-$cid} ($((sz/1048576))MB)"
        truncate -s 0 "$f"
    fi
done

# Disk after
AFTER=$(df -h / | tail -1 | awk '{print $4, $5}')
echo "Disk after: $AFTER"
echo "=== Docker cleanup done at $(date) ==="
