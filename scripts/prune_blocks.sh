#!/bin/bash
# Prune block files older than KEEP_HOURS hours (default 168 = 1 week)
# Usage: ./prune_blocks.sh [hours]
# Deployed on test.zus.network (<server-ip>) via cron: 0 3 * * *
KEEP_HOURS="${1:-168}"
KEEP_MINS=$(( KEEP_HOURS * 60 ))

prune_blocks() {
    local blocks_dir="$1"
    local label="$2"
    echo "=== [$label] Removing blocks older than ${KEEP_HOURS}h (${KEEP_MINS} min) ==="
    BEFORE=$(find "$blocks_dir" -name '*.dat.zlib' | wc -l)
    find "$blocks_dir" -name '*.dat.zlib' -mmin +${KEEP_MINS} -delete
    find "$blocks_dir" -mindepth 1 -type d -empty -delete
    AFTER=$(find "$blocks_dir" -name '*.dat.zlib' | wc -l)
    echo "Done: $BEFORE -> $AFTER files (removed $((BEFORE - AFTER)))"
}

prune_blocks /root/Code/0chain/docker.local/sharder1/data/blocks sharder1
prune_blocks /root/Code/0chain/docker.local/sharder2/data/blocks sharder2
