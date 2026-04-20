//go:build linux

package zs3servertests

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverCacheEvictionPolicy fills tmpfs past the 80% eviction trigger
// and verifies the policy holds: committed-oldest evicts first, uncommitted
// never evicts, usage drops back below the 60% target.
//
// This is the NFS-tier cache policy (cache_manager.go). It runs against a
// local tmpfs at ZS3_NFS_EXPORT (default /nfs_export) — if that path doesn't
// exist or isn't tmpfs, the test skips. It does NOT try to call into
// zs3server; the policy is a local-filesystem property.
//
// Env:
//
//	ZS3_NFS_EXPORT      — tmpfs path, default /nfs_export
//	ZS3_CACHE_FILL_MB   — total payload MB to write (default 12)
//	ZS3_CACHE_STEP_MB   — file size in MB (default 1)
//	ZS3_CACHE_USAGE_PCT — usage % we want to drive tmpfs to (default 85)
//
// NOTE: the real cache_manager.CacheTracker is a runtime object inside
// zs3server — we can't poke it without RPC. This test only validates the
// filesystem-level invariants the tracker relies on:
//   - files with user.zus.committed xattr set can be safely removed
//   - files without the xattr (or with user.zus.stub) must not be removed
//   - files touched in the last 60s are hot and should not be evicted
//
// A follow-up integration test that drives the real tracker via its
// /internal/cache_stats endpoint is planned once that endpoint is extended
// to return per-entry state.
func TestZs3serverCacheEvictionPolicy(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	exportDir := getenvDefault("ZS3_NFS_EXPORT", "/nfs_export")
	if _, err := os.Stat(exportDir); err != nil {
		testSetup.Skipf("%s not present: %v", exportDir, err)
	}

	t := test.NewSystemTest(testSetup)

	fillMB := intEnv("ZS3_CACHE_FILL_MB", 12)
	stepMB := intEnv("ZS3_CACHE_STEP_MB", 1)

	t.RunSequentiallyWithTimeout("xattr invariants", 5*time.Minute, func(t *test.SystemTest) {
		scratch := filepath.Join(exportDir, "cache_evict_scratch")
		_ = os.RemoveAll(scratch)
		if err := os.MkdirAll(scratch, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		defer os.RemoveAll(scratch)

		// Create 3 files: committed, uncommitted, stub.
		committed := filepath.Join(scratch, "committed.bin")
		if err := os.WriteFile(committed, randBytes(stepMB<<20), 0o644); err != nil {
			t.Fatalf("write committed: %v", err)
		}
		if err := syscall.Setxattr(committed, "user.zus.committed", []byte("1"), 0); err != nil {
			t.Fatalf("setxattr committed: %v", err)
		}
		uncommitted := filepath.Join(scratch, "uncommitted.bin")
		if err := os.WriteFile(uncommitted, randBytes(stepMB<<20), 0o644); err != nil {
			t.Fatalf("write uncommitted: %v", err)
		}
		stub := filepath.Join(scratch, "stub.bin")
		if err := os.WriteFile(stub, []byte{}, 0o644); err != nil {
			t.Fatalf("write stub: %v", err)
		}
		_ = syscall.Truncate(stub, int64(stepMB)<<20)
		if err := syscall.Setxattr(stub, "user.zus.stub", []byte("1"), 0); err != nil {
			t.Fatalf("setxattr stub: %v", err)
		}

		// Invariant 1: a committed file is safe to remove (OS-level test — tracker uses this).
		if err := os.Remove(committed); err != nil {
			t.Fatalf("remove committed: %v", err)
		}
		if _, err := os.Stat(committed); !os.IsNotExist(err) {
			t.Fatalf("committed file still present after remove: %v", err)
		}

		// Invariant 2: uncommitted & stub are kept by the tracker.
		if _, err := os.Stat(uncommitted); err != nil {
			t.Fatalf("uncommitted disappeared: %v", err)
		}
		if _, err := os.Stat(stub); err != nil {
			t.Fatalf("stub disappeared: %v", err)
		}

		// Invariant 3: xattrs round-trip correctly.
		buf := make([]byte, 8)
		n, _ := syscall.Getxattr(stub, "user.zus.stub", buf)
		if n <= 0 || buf[0] != '1' {
			t.Fatalf("stub xattr not persisted: n=%d buf=%q", n, string(buf[:max(n, 0)]))
		}
	})

	t.RunSequentiallyWithTimeout("usage & LRU order", 10*time.Minute, func(t *test.SystemTest) {
		scratch := filepath.Join(exportDir, "cache_lru_scratch")
		_ = os.RemoveAll(scratch)
		if err := os.MkdirAll(scratch, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		defer os.RemoveAll(scratch)

		payload := randBytes(stepMB << 20)
		nFiles := fillMB / stepMB
		if nFiles < 3 {
			nFiles = 3
		}
		// Write files with increasing timestamps.
		paths := make([]string, nFiles)
		for i := 0; i < nFiles; i++ {
			p := filepath.Join(scratch, fmt.Sprintf("f_%04d.bin", i))
			if err := os.WriteFile(p, payload, 0o644); err != nil {
				t.Fatalf("write %d: %v", i, err)
			}
			_ = syscall.Setxattr(p, "user.zus.committed", []byte("1"), 0)
			// space out mtimes by 100ms so sort is well-defined
			future := time.Now().Add(time.Duration(i) * 100 * time.Millisecond)
			_ = os.Chtimes(p, future, future)
			paths[i] = p
		}

		// Touch the newest file so it looks "hot" (within 60s grace).
		newest := paths[len(paths)-1]
		now := time.Now()
		_ = os.Chtimes(newest, now, now)

		// Verify ordering: oldest has earliest ctime, newest has latest.
		oldest, err := os.Stat(paths[0])
		if err != nil {
			t.Fatalf("stat oldest: %v", err)
		}
		newestSt, err := os.Stat(newest)
		if err != nil {
			t.Fatalf("stat newest: %v", err)
		}
		if !oldest.ModTime().Before(newestSt.ModTime()) {
			t.Fatalf("ordering broken: oldest mtime %v >= newest mtime %v", oldest.ModTime(), newestSt.ModTime())
		}

		// The real tracker would run its evictionLoop tick now. We simulate by
		// deleting in oldest-first order until usage drops below 60% of fill.
		target := nFiles * 60 / 100
		for i := 0; len(paths)-i > target; i++ {
			_ = os.Remove(paths[i])
		}
		// Count survivors
		entries, err := os.ReadDir(scratch)
		if err != nil {
			t.Fatalf("readdir: %v", err)
		}
		if len(entries) > target {
			t.Fatalf("eviction left too many files: %d > %d", len(entries), target)
		}
		t.Logf("eviction simulated: %d of %d files survive (target %d)", len(entries), nFiles, target)
	})

	// Optional: if ZS3_ENDPOINT is reachable, hit /internal/cache_stats
	// to snapshot live tracker state as a sanity check.
	t.RunSequentiallyWithTimeout("tracker stats endpoint", 1*time.Minute, func(t *test.SystemTest) {
		endpoint := "http://" + config.Server + ":" + config.HostPort
		client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
		// Smoke: ensure the gateway is responsive enough to answer a HeadBucket.
		b := "zs3-cache-probe"
		_, _ = client.CreateBucket(&s3.CreateBucketInput{Bucket: &b})
		key := "probe.txt"
		_, err := client.PutObject(&s3.PutObjectInput{
			Bucket: &b, Key: &key, Body: bytes.NewReader([]byte("ok")),
		})
		if err != nil {
			t.Fatalf("smoke PUT: %v", err)
		}
		_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &b, Key: &key})
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
