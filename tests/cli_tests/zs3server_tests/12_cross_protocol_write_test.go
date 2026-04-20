package zs3servertests

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverCrossProtocolWrite verifies that:
//   - A write via S3 is visible + byte-identical via NFS within the cache window.
//   - A write via NFS is visible + byte-identical via S3 within the cache window.
//
// This exercises the mirror_s3_to_export and nfs_blobber_sync coordination
// between /mcache (MinIO writeback) and /nfs_export (tmpfs for Ganesha).
//
// Pass criteria: both directions succeed, md5 matches, within cache visibility
// timeout (default 10s). A miss after the timeout means the mirror logic
// didn't fire, which is a regression we want to catch.
//
// Env:
//
//	ZS3_NFS_MOUNT           — NFSv4 mount path, default /mnt/zus_nfs
//	ZS3_CROSS_TIMEOUT       — visibility timeout, default 10s
//	ZS3_CROSS_BUCKET        — bucket for probes, default "zs3-cross-protocol"
func TestZs3serverCrossProtocolWrite(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	nfsMount := getenvDefault("ZS3_NFS_MOUNT", "/mnt/zus_nfs")
	if _, err := os.Stat(nfsMount); err != nil {
		testSetup.Skipf("NFS mount %s not present: %v", nfsMount, err)
	}
	timeout := durEnv("ZS3_CROSS_TIMEOUT", 10*time.Second)
	bucket := getenvDefault("ZS3_CROSS_BUCKET", "zs3-cross-protocol")

	t := test.NewSystemTest(testSetup)

	endpoint := "http://" + config.Server + ":" + config.HostPort
	client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
	_ = ensureBucket(client, bucket)

	// Ensure the NFS bucket dir exists
	_ = os.MkdirAll(filepath.Join(nfsMount, bucket), 0o755)

	t.RunSequentiallyWithTimeout("S3 PUT → NFS read", 5*time.Minute, func(t *test.SystemTest) {
		key := fmt.Sprintf("s3-to-nfs-%d.bin", time.Now().UnixNano())
		payload := randBytes(512 * 1024) // 512 KB — WAL-eligible
		want := md5sum(payload)

		if _, err := client.PutObject(&s3.PutObjectInput{
			Bucket: &bucket, Key: &key, Body: bytes.NewReader(payload),
		}); err != nil {
			t.Fatalf("S3 PUT: %v", err)
		}

		path := filepath.Join(nfsMount, bucket, key)
		deadline := time.Now().Add(timeout)
		var got []byte
		for time.Now().Before(deadline) {
			data, err := os.ReadFile(path)
			if err == nil && len(data) == len(payload) {
				got = data
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if got == nil {
			t.Fatalf("NFS did not see S3-written key %s within %s", key, timeout)
		}
		if md5sum(got) != want {
			t.Fatalf("S3→NFS md5 mismatch: want=%s got=%s", want, md5sum(got))
		}
		t.Logf("S3 PUT → NFS read OK (md5=%s)", want)

		_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key})
		_ = os.Remove(path)
	})

	t.RunSequentiallyWithTimeout("NFS write → S3 GET", 5*time.Minute, func(t *test.SystemTest) {
		key := fmt.Sprintf("nfs-to-s3-%d.bin", time.Now().UnixNano())
		payload := randBytes(512 * 1024)
		want := md5sum(payload)

		path := filepath.Join(nfsMount, bucket, key)
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatalf("NFS write: %v", err)
		}

		deadline := time.Now().Add(timeout)
		var got []byte
		for time.Now().Before(deadline) {
			out, err := client.GetObject(&s3.GetObjectInput{Bucket: &bucket, Key: &key})
			if err == nil {
				data, _ := io.ReadAll(out.Body)
				_ = out.Body.Close()
				if len(data) == len(payload) {
					got = data
					break
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		if got == nil {
			t.Fatalf("S3 did not see NFS-written key %s within %s", key, timeout)
		}
		if md5sum(got) != want {
			t.Fatalf("NFS→S3 md5 mismatch: want=%s got=%s", want, md5sum(got))
		}
		t.Logf("NFS write → S3 GET OK (md5=%s)", want)

		_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key})
		_ = os.Remove(path)
	})
}

func md5sum(b []byte) string {
	h := md5.New()
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
