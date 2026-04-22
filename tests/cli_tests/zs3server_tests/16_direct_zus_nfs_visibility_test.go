package zs3servertests

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// TestZs3serverDirectZusNFSVisibility verifies that files uploaded DIRECTLY
// to a Züs allocation (bypassing zs3server's S3 API — e.g. via rclone-zus,
// direct gosdk, or any third-party tool that speaks the Züs protocol) are
// automatically:
//
//  1. surfaced in NFS readdir via FSAL_ZUS → zs3server /internal/list?stub=1
//  2. filled on-demand via FSAL_ZUS → zs3server /internal/prewarm when read
//
// This is the production flow for any app that uses Züs as a cache tier: it
// may write via Züs's native protocol (for linearizability / faster ingest)
// but then read via the POSIX/NFS surface. The test fails if NFS readdir
// misses files that exist on blobbers, or if NFS reads return zero bytes /
// wrong content because the prewarm didn't fire.
//
// Architectural components exercised:
//   - gosdk direct upload (via rclone-zus binary) — no zs3server involvement
//   - FSAL_ZUS zus_list_prewarm in handle.c:readdir — calls /internal/list
//   - zs3server list_router.go — creates sparse stubs with user.zus.stub xattr
//   - FSAL_ZUS zus_rel_stub_check_and_prewarm in file.c:open2/read2
//   - zs3server prewarm_router.go — fetches real bytes from blobber
//   - Optional: fallback_s3 router if the file is only in external S3
//
// Env:
//
//	ZS3_NFS_MOUNT              — NFSv4 mount path, default /mnt/zus_nfs
//	ZS3_DIRECT_BUCKET          — probe bucket, default "zs3-direct-nfs"
//	ZS3_RCLONE_BIN             — rclone-zus binary, default /usr/local/bin/rclone-zus
//	ZS3_RCLONE_REMOTE          — rclone remote name (in ~/.config/rclone/rclone.conf)
//	                              that points at the allocation zs3server serves.
//	                              Default "automation1k".
//	ZS3_LIST_CACHE_TTL_WAIT    — wait after upload before readdir to let
//	                              FSAL_ZUS list cache expire. Default 90s
//	                              (FSAL cache is 60s).
//	ZS3_DIRECT_TIMEOUT         — per-subtest timeout, default 5m.
//
// Pass criteria:
//
//	A. readdir via NFS lists the uploaded file by name within the TTL window.
//	B. read via NFS returns byte-identical content (via prewarm fetch).
//	C. the /internal/list endpoint returned a non-zero "stubbed" count when
//	   invoked (confirms the auto-stub code path actually fired).
func TestZs3serverDirectZusNFSVisibility(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	rcloneBin := getenvDefault("ZS3_RCLONE_BIN", "/usr/local/bin/rclone-zus")
	if _, err := os.Stat(rcloneBin); err != nil {
		testSetup.Skipf("rclone-zus binary %s not found: %v", rcloneBin, err)
	}
	nfsMount := getenvDefault("ZS3_NFS_MOUNT", "/mnt/zus_nfs")
	if _, err := os.Stat(nfsMount); err != nil {
		testSetup.Skipf("NFS mount %s not present: %v", nfsMount, err)
	}

	rcloneRemote := getenvDefault("ZS3_RCLONE_REMOTE", "automation1k")
	bucket := getenvDefault("ZS3_DIRECT_BUCKET", "zs3-direct-nfs")
	listWait := durEnv("ZS3_LIST_CACHE_TTL_WAIT", 90*time.Second)
	timeout := durEnv("ZS3_DIRECT_TIMEOUT", 5*time.Minute)
	endpoint := "http://" + config.Server + ":" + config.HostPort

	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("direct-upload via rclone-zus → NFS readdir surfaces", timeout, func(t *test.SystemTest) {
		// Upload two files via rclone-zus (bypasses zs3server entirely —
		// rclone-zus speaks the Züs gosdk protocol directly to blobbers).
		key1 := fmt.Sprintf("direct-%d-a.bin", time.Now().UnixNano())
		key2 := fmt.Sprintf("direct-%d-b.bin", time.Now().UnixNano())
		payload1 := randBytes(64 * 1024) // 64KB — under direct_threshold, cacheable
		payload2 := randBytes(3 * 1024 * 1024) // 3MB — above direct_threshold

		want1 := md5sum(payload1)
		want2 := md5sum(payload2)

		if err := rcloneUpload(rcloneBin, rcloneRemote, bucket, key1, payload1); err != nil {
			t.Fatalf("rclone-zus upload %s: %v", key1, err)
		}
		if err := rcloneUpload(rcloneBin, rcloneRemote, bucket, key2, payload2); err != nil {
			t.Fatalf("rclone-zus upload %s: %v", key2, err)
		}
		defer rcloneDelete(rcloneBin, rcloneRemote, bucket, key1)
		defer rcloneDelete(rcloneBin, rcloneRemote, bucket, key2)

		// Invalidate FSAL_ZUS list cache (60s TTL) by waiting, or by probing
		// a distinct directory first so the next readdir on our bucket
		// re-issues /internal/list?stub=1. We wait to be safe.
		t.Logf("waiting %s for FSAL list cache to expire and allow auto-stub", listWait)
		time.Sleep(listWait)

		// Trigger NFS readdir — FSAL_ZUS's zusfs_readdir at handle.c:467
		// calls zus_list_prewarm → zs3server /internal/list?stub=1 → stubs
		// appear in /nfs_export → VFS sub-FSAL sees them → returned to NFS.
		dir := filepath.Join(nfsMount, bucket)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("readdir %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, e := range entries {
			seen[e.Name()] = true
		}
		if !seen[key1] {
			t.Errorf("direct-uploaded key %s NOT visible via NFS readdir after %s — FSAL auto-stub path broken (check Ganesha log for 'ZUS list_prewarm failed')", key1, listWait)
		}
		if !seen[key2] {
			t.Errorf("direct-uploaded key %s NOT visible via NFS readdir after %s", key2, listWait)
		}

		// Independently confirm the /internal/list endpoint would have
		// populated stubs — provides a clearer failure signal when FSAL
		// has the bug but zs3server is fine.
		stubbed, listErr := probeListStub(endpoint, bucket, "")
		if listErr != nil {
			t.Logf("/internal/list probe error (non-fatal): %v", listErr)
		} else if stubbed == 0 {
			t.Errorf("/internal/list returned stubbed=0 — zs3server list endpoint is not creating stubs (list_router.go broken?)")
		}

		// Now read each file via NFS — FSAL_ZUS open2/read2 should detect
		// user.zus.stub xattr on the sparse placeholder and call
		// /internal/prewarm to fetch real bytes from blobbers.
		got1, err := os.ReadFile(filepath.Join(dir, key1))
		if err != nil {
			t.Fatalf("NFS read %s: %v (prewarm likely failed)", key1, err)
		}
		if md5sum(got1) != want1 {
			t.Errorf("NFS read %s content mismatch — prewarm either didn't fire or served zeroes (got %d bytes, md5=%s, want md5=%s)",
				key1, len(got1), md5sum(got1), want1)
		}

		got2, err := os.ReadFile(filepath.Join(dir, key2))
		if err != nil {
			t.Fatalf("NFS read %s: %v", key2, err)
		}
		if md5sum(got2) != want2 {
			t.Errorf("NFS read %s content mismatch — expected md5=%s got md5=%s (%d bytes)",
				key2, want2, md5sum(got2), len(got2))
		}
	})
}

// rcloneUpload uploads payload to a rclone remote via `rclone copyto`.
// This path goes through rclone-zus → zus pool → gosdk → blobbers,
// bypassing zs3server S3 API entirely.
func rcloneUpload(bin, remote, bucket, key string, payload []byte) error {
	tmp, err := os.CreateTemp("", "zs3-direct-*.bin")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(payload); err != nil {
		return err
	}
	tmp.Close()
	cmd := exec.Command(bin, "copyto", tmp.Name(),
		fmt.Sprintf("%s:%s/%s", remote, bucket, key),
		"--transfers", "1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func rcloneDelete(bin, remote, bucket, key string) {
	_ = exec.Command(bin, "deletefile",
		fmt.Sprintf("%s:%s/%s", remote, bucket, key)).Run()
}

// probeListStub hits the zs3server internal list endpoint with stub=1 and
// returns how many stubs it created. Used to distinguish "FSAL auto-call
// broken" from "zs3server endpoint broken" in failure messages.
func probeListStub(endpoint, bucket, prefix string) (int, error) {
	url := fmt.Sprintf("%s/internal/list?bucket=%s&stub=1", endpoint, bucket)
	if prefix != "" {
		url += "&prefix=" + prefix
	}
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	// List response is JSON { entries: [...], stubbed: N } — we only need the
	// count, so a cheap substring match avoids importing encoding/json here.
	body, _ := io.ReadAll(resp.Body)
	var stubbed int
	if idx := bytes.Index(body, []byte(`"stubbed":`)); idx >= 0 {
		_, _ = fmt.Sscanf(string(body[idx:]), `"stubbed":%d`, &stubbed)
	}
	return stubbed, nil
}

// md5sum returns the lowercase hex md5 of b.
func md5sumLocal(b []byte) string {
	h := md5.Sum(b)
	return hex.EncodeToString(h[:])
}
