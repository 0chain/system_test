#!/bin/bash
# Compare upload/download throughput: mc vs aws s3 vs rclone-zus vs rclone (S3).
# Target: zs3server on localhost:9100 (backed by ZUS blobbers / eblobbers).
#
# Cold-cache: drops host page cache before every DOWN leg (propagates to
# containers since they share the kernel), so every download tool pays the
# same "fetch from blobber disk" cost without benefit of OS page cache from
# the immediately-preceding upload.
#
# USAGE: ./upload_compare.sh [BUCKET]
# OUTPUT: /tmp/upload_compare_results/summary.{txt,csv}

set -o pipefail

BUCKET="${1:-uploadcmp}"
WORK=/tmp/upload_compare_work
OUT=/tmp/upload_compare_results
ENDPOINT="${ENDPOINT:-http://localhost:9100}"
AK="${AWS_ACCESS_KEY_ID:-rootroot}"
SK="${AWS_SECRET_ACCESS_KEY:-rootroot}"

mkdir -p "$WORK/src" "$WORK/dst_mc" "$WORK/dst_aws" "$WORK/dst_rczus" "$WORK/dst_rcs3" "$OUT"
rm -rf "$WORK/src"/* "$WORK/dst_"*/*

echo "=== generating test corpus (155 files, ~550 MB) ==="
for i in $(seq 1 100); do dd if=/dev/urandom of="$WORK/src/small_${i}.bin"  bs=4K  count=1    status=none; done
for i in $(seq 1 50);  do dd if=/dev/urandom of="$WORK/src/medium_${i}.bin" bs=1M  count=1    status=none; done
for i in $(seq 1 5);   do dd if=/dev/urandom of="$WORK/src/large_${i}.bin"  bs=1M  count=100  status=none; done
TOTAL_BYTES=$(du -sb "$WORK/src" | awk '{print $1}')
FILE_COUNT=$(find "$WORK/src" -type f | wc -l)
echo "corpus: ${FILE_COUNT} files, ${TOTAL_BYTES} bytes"

# Compute source hashes
( cd "$WORK/src" && find . -type f | sort | xargs sha256sum ) > "$WORK/src_hashes.txt"

summary_csv="$OUT/summary.csv"
summary_txt="$OUT/summary.txt"
: > "$summary_csv"; : > "$summary_txt"
echo "tool,op,seconds,bytes,MBps,integrity_ok" >> "$summary_csv"

drop_caches() {
  sync
  echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'
}

time_op() {  # name op seconds bytes dst_dir
  local name="$1" op="$2" sec="$3" bytes="$4" dst="$5"
  local mbps
  mbps=$(awk -v b="$bytes" -v s="$sec" 'BEGIN{ if(s==0) print 0; else printf "%.1f", (b/1048576)/s }')
  local ok="-"
  if [ "$op" = "DOWN" ]; then
    ( cd "$dst" && find . -type f | sort | xargs sha256sum ) > "$WORK/${name}_hashes.txt"
    if diff -q "$WORK/src_hashes.txt" "$WORK/${name}_hashes.txt" >/dev/null 2>&1; then ok="yes"; else ok="NO"; fi
  fi
  printf "%-12s %-4s %8.2fs %10d bytes %7s MB/s  integrity=%s\n" "$name" "$op" "$sec" "$bytes" "$mbps" "$ok" | tee -a "$summary_txt"
  echo "$name,$op,$sec,$bytes,$mbps,$ok" >> "$summary_csv"
}

run_timed() {  # var cmd...
  local __var="$1"; shift
  local t0 t1
  t0=$(date +%s.%N)
  "$@" >/dev/null 2>&1
  local rc=$?
  t1=$(date +%s.%N)
  local dt
  dt=$(awk -v a="$t0" -v b="$t1" 'BEGIN{printf "%.3f", b-a}')
  printf -v "$__var" '%s' "$dt"
  return $rc
}

# --- Ensure bucket exists (via mc) ---
if command -v mc >/dev/null 2>&1; then
  mc alias set zs3 "$ENDPOINT" "$AK" "$SK" >/dev/null 2>&1
  mc mb "zs3/$BUCKET" >/dev/null 2>&1
  mc rm --recursive --force "zs3/$BUCKET" >/dev/null 2>&1
else
  echo "mc not installed — aborting" ; exit 1
fi

# --- 1) mc mirror ---
echo "=== mc mirror ==="
run_timed t_up mc mirror --overwrite --quiet "$WORK/src" "zs3/$BUCKET/mc"
time_op "mc" UP "$t_up" "$TOTAL_BYTES" ""
echo "  (drop_caches before DOWN)"; drop_caches
run_timed t_dn mc mirror --overwrite --quiet "zs3/$BUCKET/mc" "$WORK/dst_mc"
time_op "mc" DOWN "$t_dn" "$TOTAL_BYTES" "$WORK/dst_mc"

# --- 2) aws s3 sync ---
if command -v aws >/dev/null 2>&1; then
  echo "=== aws s3 sync ==="
  export AWS_ACCESS_KEY_ID="$AK" AWS_SECRET_ACCESS_KEY="$SK"
  AWSCMD=(aws --endpoint-url "$ENDPOINT" s3)
  run_timed t_up "${AWSCMD[@]}" sync "$WORK/src" "s3://$BUCKET/aws" --no-progress
  time_op "aws-s3" UP "$t_up" "$TOTAL_BYTES" ""
  echo "  (drop_caches before DOWN)"; drop_caches
  run_timed t_dn "${AWSCMD[@]}" sync "s3://$BUCKET/aws" "$WORK/dst_aws" --no-progress
  time_op "aws-s3" DOWN "$t_dn" "$TOTAL_BYTES" "$WORK/dst_aws"
else
  echo "aws CLI not installed — skipping"
fi

# --- 3) rclone-zus (custom backend, direct to blobbers) ---
RCLONE_ZUS="${RCLONE_ZUS:-/root/Code/rclone_zus/rclone}"
if [ -x "$RCLONE_ZUS" ] && [ -n "${ZUS_ALLOC:-}" ] && [ -n "${ZUS_WALLET:-}" ]; then
  echo "=== rclone-zus (direct blobber) ==="
  export RCLONE_CONFIG=/tmp/rczus.conf
  cat > "$RCLONE_CONFIG" <<EOF
[zus]
type = zus
allocation_id = ${ZUS_ALLOC}
wallet_file = ${ZUS_WALLET}
block_worker = ${BLOCK_WORKER:-http://198.18.0.98:9091}
EOF
  run_timed t_up "$RCLONE_ZUS" copy "$WORK/src" "zus:/uploadcmp/rczus" --transfers=16 --checkers=16
  time_op "rclone-zus" UP "$t_up" "$TOTAL_BYTES" ""
  echo "  (drop_caches before DOWN)"; drop_caches
  run_timed t_dn "$RCLONE_ZUS" copy "zus:/uploadcmp/rczus" "$WORK/dst_rczus" --transfers=16 --checkers=16
  time_op "rclone-zus" DOWN "$t_dn" "$TOTAL_BYTES" "$WORK/dst_rczus"
else
  echo "rclone-zus skipped (binary or ZUS_ALLOC/ZUS_WALLET env missing)"
fi

# --- 4) rclone (standard S3) ---
if command -v rclone >/dev/null 2>&1; then
  echo "=== rclone S3 ==="
  export RCLONE_CONFIG=/tmp/rcs3.conf
  cat > "$RCLONE_CONFIG" <<EOF
[s3]
type = s3
provider = Other
endpoint = ${ENDPOINT}
access_key_id = ${AK}
secret_access_key = ${SK}
force_path_style = true
EOF
  run_timed t_up rclone copy "$WORK/src" "s3:/${BUCKET}/rcs3" --transfers=16 --checkers=16
  time_op "rclone-s3" UP "$t_up" "$TOTAL_BYTES" ""
  echo "  (drop_caches before DOWN)"; drop_caches
  run_timed t_dn rclone copy "s3:/${BUCKET}/rcs3" "$WORK/dst_rcs3" --transfers=16 --checkers=16
  time_op "rclone-s3" DOWN "$t_dn" "$TOTAL_BYTES" "$WORK/dst_rcs3"
else
  echo "rclone not installed — skipping"
fi

echo
echo "=== cold-cache summary ==="
cat "$summary_txt"
echo
echo "csv: $summary_csv"
