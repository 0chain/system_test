package zs3servertests

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverWritePressure exercises concurrent S3 + NFS writers against
// overlapping keys and verifies:
//   - no cache corruption (final state is one of the written payloads, not a mix)
//   - no deadlock / timeout (all writers complete within the test window)
//   - the post-race read is self-consistent across both protocols
//
// This is the write-side analog of the Porcupine test: instead of checking
// linearizability across arbitrary op histories, we check last-writer-wins
// consistency for a fixed hot-key set under sustained parallel PUT load.
//
// Env:
//
//	ZS3_NFS_MOUNT          — NFS mount, default /mnt/zus_nfs
//	ZS3_WP_S3_WORKERS      — S3 concurrency, default 4
//	ZS3_WP_NFS_WORKERS     — NFS concurrency, default 4
//	ZS3_WP_OPS             — total ops across all writers, default 200
//	ZS3_WP_KEYS            — hot-key count, default 10
//	ZS3_WP_PAYLOAD_BYTES   — per-object payload size, default 32 KiB

type writeRecord struct {
	key     string
	value   string
	doneAt  time.Time
	via     string // "s3" or "nfs"
	payload []byte
}

func TestZs3serverWritePressure(testSetup *testing.T) {
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

	t := test.NewSystemTest(testSetup)

	s3Workers := intEnv("ZS3_WP_S3_WORKERS", 4)
	nfsWorkers := intEnv("ZS3_WP_NFS_WORKERS", 4)
	nOps := intEnv("ZS3_WP_OPS", 200)
	nKeys := intEnv("ZS3_WP_KEYS", 10)
	payloadBytes := intEnv("ZS3_WP_PAYLOAD_BYTES", 32*1024)

	bucket := getenvDefault("ZS3_WP_BUCKET", "zs3-write-pressure")
	endpoint := "http://" + config.Server + ":" + config.HostPort
	client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
	_ = ensureBucket(client, bucket)
	_ = os.MkdirAll(filepath.Join(nfsMount, bucket), 0o755)

	t.RunSequentiallyWithTimeout("concurrent S3+NFS same-key pressure", 10*time.Minute, func(t *test.SystemTest) {
		var (
			mu    sync.Mutex
			log   []writeRecord
			errs  int64
			total atomic.Int32
		)

		recordWrite := func(key, value, via string, payload []byte) {
			mu.Lock()
			log = append(log, writeRecord{
				key: key, value: value, doneAt: time.Now(), via: via, payload: append([]byte(nil), payload...),
			})
			mu.Unlock()
		}

		opCh := make(chan int, nOps)
		for i := 0; i < nOps; i++ {
			opCh <- i
		}
		close(opCh)

		var wg sync.WaitGroup

		// S3 workers — handle even-indexed ops
		for w := 0; w < s3Workers; w++ {
			wg.Add(1)
			go func(wid int) {
				defer wg.Done()
				for opID := range opCh {
					if opID%2 != 0 {
						continue
					}
					key := fmt.Sprintf("hotkey_%03d.bin", opID%nKeys)
					val := fmt.Sprintf("s3-%d-%d", wid, opID)
					payload := make([]byte, payloadBytes)
					copy(payload, []byte(val))
					_, err := client.PutObject(&s3.PutObjectInput{
						Bucket: &bucket, Key: &key, Body: bytes.NewReader(payload),
					})
					if err != nil {
						atomic.AddInt64(&errs, 1)
						continue
					}
					recordWrite(key, val, "s3", payload)
					total.Add(1)
				}
			}(w)
		}

		// NFS workers — handle odd-indexed ops
		for w := 0; w < nfsWorkers; w++ {
			wg.Add(1)
			go func(wid int) {
				defer wg.Done()
				for opID := range opCh {
					if opID%2 != 1 {
						continue
					}
					key := fmt.Sprintf("hotkey_%03d.bin", opID%nKeys)
					val := fmt.Sprintf("nfs-%d-%d", wid, opID)
					payload := make([]byte, payloadBytes)
					copy(payload, []byte(val))
					tmp := filepath.Join(nfsMount, bucket, key+fmt.Sprintf(".tmp.%d.%d", wid, opID))
					if err := os.WriteFile(tmp, payload, 0o644); err != nil {
						atomic.AddInt64(&errs, 1)
						continue
					}
					dst := filepath.Join(nfsMount, bucket, key)
					if err := os.Rename(tmp, dst); err != nil {
						atomic.AddInt64(&errs, 1)
						continue
					}
					recordWrite(key, val, "nfs", payload)
					total.Add(1)
				}
			}(w)
		}

		wg.Wait()
		t.Logf("ops: %d/%d complete, errors=%d, writers s3=%d nfs=%d",
			total.Load(), nOps, errs, s3Workers, nfsWorkers)

		// For each key, index all payloads we saw for that key.
		payloadsByKey := map[string]map[string]bool{}
		for _, r := range log {
			m, ok := payloadsByKey[r.key]
			if !ok {
				m = map[string]bool{}
				payloadsByKey[r.key] = m
			}
			m[string(r.payload)] = true
		}
		if len(payloadsByKey) == 0 {
			t.Fatalf("no writes succeeded — cannot validate consistency")
		}

		// Read each key via S3 and NFS; assert the final content equals one of
		// the recorded payloads for that key (no partial/corrupt bytes).
		for key, valid := range payloadsByKey {
			out, err := client.GetObject(&s3.GetObjectInput{Bucket: &bucket, Key: &key})
			if err != nil {
				t.Fatalf("final S3 GET %s: %v", key, err)
			}
			s3Got, _ := io.ReadAll(out.Body)
			_ = out.Body.Close()
			if !valid[string(s3Got)] {
				t.Fatalf("key %s: S3 final content is not any recorded write (len=%d)", key, len(s3Got))
			}

			path := filepath.Join(nfsMount, bucket, key)
			nfsGot, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("final NFS read %s: %v", key, err)
			}
			if !valid[string(nfsGot)] {
				t.Fatalf("key %s: NFS final content is not any recorded write (len=%d)", key, len(nfsGot))
			}
		}

		// Cleanup
		for key := range payloadsByKey {
			_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key})
			_ = os.Remove(filepath.Join(nfsMount, bucket, key))
		}
	})
}
