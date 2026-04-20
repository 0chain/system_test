package zs3servertests

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverLlama3Checkpoint simulates an ML-training checkpoint workload:
// write N rotating checkpoint objects (each ~1.5 GB by default, configurable)
// through the S3 API, then read them back, measuring write + read throughput.
//
// This models MLPerf Storage v2.0 Llama3 checkpointing — periodic writes of
// model parameters + optimizer state. The rotation pattern tests cache
// behaviour: as newer checkpoints arrive, older ones age out and either stay
// in the hot tier or evict to blobbers.
//
// Env:
//
//	ZS3_LLAMA3_CKPT_BYTES  — per-checkpoint size in bytes, default 1.5 GB
//	ZS3_LLAMA3_NUM_CKPTS   — number of rotating checkpoints, default 4
//	ZS3_LLAMA3_ROTATE      — if "true", delete oldest before writing next
func TestZs3serverLlama3Checkpoint(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	t := test.NewSystemTest(testSetup)

	ckptBytes := int64(intEnv("ZS3_LLAMA3_CKPT_BYTES", 0))
	if ckptBytes == 0 {
		ckptBytes = int64(1536) * 1024 * 1024 // 1.5 GB
	}
	nCkpts := intEnv("ZS3_LLAMA3_NUM_CKPTS", 4)
	rotate := os.Getenv("ZS3_LLAMA3_ROTATE") == "true"

	endpoint := "http://" + config.Server + ":" + config.HostPort
	client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
	bucket := "zs3-llama3-ckpt"
	_ = ensureBucket(client, bucket)

	t.RunSequentiallyWithTimeout(
		fmt.Sprintf("Llama3 ckpt %d × %.2f GB rotate=%v", nCkpts, float64(ckptBytes)/(1<<30), rotate),
		60*time.Minute,
		func(t *test.SystemTest) {
			writeRates := make([]float64, 0, nCkpts)
			keysPresent := []string{}
			payload := randBytes(int(ckptBytes)) // reuse the payload buffer across checkpoints
			for i := 0; i < nCkpts; i++ {
				key := fmt.Sprintf("ckpt_%02d.pt", i)
				t0 := time.Now()
				_, err := client.PutObject(&s3.PutObjectInput{
					Bucket: &bucket, Key: &key, Body: bytes.NewReader(payload),
				})
				if err != nil {
					t.Fatalf("PUT %s: %v", key, err)
				}
				dt := time.Since(t0).Seconds()
				rate := float64(ckptBytes) / (1024 * 1024) / dt
				writeRates = append(writeRates, rate)
				keysPresent = append(keysPresent, key)
				t.Logf("ckpt_%02d: write %.1f MB/s (%.2fs)", i, rate, dt)

				if rotate && len(keysPresent) > nCkpts-1 {
					oldestKey := keysPresent[0]
					_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &oldestKey})
					keysPresent = keysPresent[1:]
				}
			}

			// READ phase — concurrency sweep
			for _, nw := range []int{1, 2, 4} {
				g := benchGet(t, client, bucket, keysPresent, nw)
				t.Logf("READ w=%d: %.1f MB/s p50=%.1fms p95=%.1fms", nw, g.MBPerS, g.P50ms, g.P95ms)
			}
			deleteKeys(client, bucket, keysPresent)

			peak := writeRates[0]
			for _, r := range writeRates {
				if r > peak {
					peak = r
				}
			}
			t.Logf("Llama3 peak write: %.1f MB/s", peak)
		},
	)
}
