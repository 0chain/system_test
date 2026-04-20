package zs3servertests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverMLPerfThroughput runs MLPerf-style PUT/GET workloads against the
// S3 gateway and emits a JSON artifact for regression diffing.
//
// Workloads (shape matches MLPerf Storage v2.0 — scaled to fit a single-node run):
//   - resnet     : 5000 × 50–300 KB (random small-file read, ImageNet)
//   - unet3d     :  10 × 200 MB    (sequential mid-file, KiTS19 volumes)
//   - bert       :  30 × 20–40 MB  (Wikipedia preprocessed chunks)
//   - cosmoflow  :  20 × 128 MB    (CosmoFlow tfrecords)
//   - dlrm       :   3 × 1 GB      (DLRM day-partitions)
//
// Set ZS3_MLPERF_OUT to write the JSON somewhere other than /tmp/.
// Set ZS3_MLPERF_WORKLOADS="resnet,bert" to skip the long ones.
func TestZs3serverMLPerfThroughput(testSetup *testing.T) {
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
	endpoint := "http://" + config.Server + ":" + config.HostPort
	client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)

	bucket := getenvDefault("ZS3_MLPERF_BUCKET", "zs3-mlperf-bench")
	if err := ensureBucket(client, bucket); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	workloadFilter := getenvDefault("ZS3_MLPERF_WORKLOADS", "resnet,bert,cosmoflow")
	getSweep := []int{1, 4, 8, 16}

	results := map[string]any{
		"endpoint":   endpoint,
		"bucket":     bucket,
		"started_at": time.Now().Unix(),
		"workloads":  map[string]any{},
	}

	for _, wl := range splitCSV(workloadFilter) {
		t.RunSequentiallyWithTimeout("MLPerf "+wl, 40*time.Minute, func(t *test.SystemTest) {
			specs := workloadSpecs(wl)
			if specs == nil {
				t.Skipf("unknown workload %s", wl)
			}
			t.Logf("generating %d objects", len(specs))
			putStats := benchPut(t, client, bucket, specs, 16)
			t.Logf("PUT: %.1f obj/s %.1f MB/s p50=%.1fms p95=%.1fms err=%d",
				putStats.ObjPerS, putStats.MBPerS, putStats.P50ms, putStats.P95ms, putStats.Errors)

			keys := make([]string, len(specs))
			for i, s := range specs {
				keys[i] = s.key
			}
			getResults := map[int]*phaseStats{}
			auResults := map[int]float64{}
			for _, nw := range getSweep {
				g := benchGet(t, client, bucket, keys, nw)
				getResults[nw] = g
				au := computeAU(wl, g.MBPerS)
				auResults[nw] = au
				t.Logf("GET w=%d: %.1f obj/s %.1f MB/s p50=%.1fms p95=%.1fms err=%d  AU=%.1f%%",
					nw, g.ObjPerS, g.MBPerS, g.P50ms, g.P95ms, g.Errors, au*100)
			}
			// cleanup
			deleteKeys(client, bucket, keys)

			results["workloads"].(map[string]any)[wl] = map[string]any{
				"put":       putStats,
				"get":       getResults,
				"au_by_workers": auResults,
			}
		})
	}

	results["finished_at"] = time.Now().Unix()
	outPath := getenvDefault("ZS3_MLPERF_OUT", fmt.Sprintf("/tmp/zs3_mlperf_%d.json", time.Now().Unix()))
	b, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile(outPath, b, 0o644)
	t.Logf("MLPerf results written to %s", outPath)
}

// ---- helpers (shared by tests 7-13 via package scope) ----

type objSpec struct {
	key     string
	payload []byte
}

type phaseStats struct {
	Elapsed float64 `json:"elapsed_s"`
	Count   int     `json:"obj_count"`
	MBTotal float64 `json:"mb_total"`
	ObjPerS float64 `json:"obj_per_s"`
	MBPerS  float64 `json:"mb_per_s"`
	Errors  int     `json:"errors"`
	P50ms   float64 `json:"p50_ms"`
	P95ms   float64 `json:"p95_ms"`
}

func newS3Client(t *test.SystemTest, endpoint, access, secret string) *s3.S3 {
	sess, err := awsSession.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(access, secret, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("aws session: %v", err)
	}
	return s3.New(sess)
}

func ensureBucket(client *s3.S3, bucket string) error {
	_, err := client.HeadBucket(&s3.HeadBucketInput{Bucket: &bucket})
	if err == nil {
		return nil
	}
	_, err = client.CreateBucket(&s3.CreateBucketInput{Bucket: &bucket})
	return err
}

func benchPut(t *test.SystemTest, client *s3.S3, bucket string, specs []objSpec, workers int) *phaseStats {
	start := time.Now()
	latencies := make([]float64, len(specs))
	var errors int64
	var mu sync.Mutex
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, s := range specs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, s objSpec) {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			_, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &s.key, Body: bytes.NewReader(s.payload),
			})
			lat := time.Since(t0).Seconds() * 1000
			if err != nil {
				atomic.AddInt64(&errors, 1)
				t.Logf("PUT error %s: %v", s.key, err)
			}
			mu.Lock()
			latencies[i] = lat
			mu.Unlock()
		}(i, s)
	}
	wg.Wait()
	elapsed := time.Since(start).Seconds()
	totalB := int64(0)
	for _, s := range specs {
		totalB += int64(len(s.payload))
	}
	return buildStats(latencies, elapsed, len(specs), totalB, int(errors))
}

func benchGet(t *test.SystemTest, client *s3.S3, bucket string, keys []string, workers int) *phaseStats {
	start := time.Now()
	latencies := make([]float64, len(keys))
	var errors int64
	totalB := int64(0)
	var mu sync.Mutex
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, k string) {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			out, err := client.GetObject(&s3.GetObjectInput{Bucket: &bucket, Key: &k})
			lat := time.Since(t0).Seconds() * 1000
			if err != nil {
				atomic.AddInt64(&errors, 1)
				mu.Lock()
				latencies[i] = 0
				mu.Unlock()
				return
			}
			b, _ := io.ReadAll(out.Body)
			_ = out.Body.Close()
			mu.Lock()
			totalB += int64(len(b))
			latencies[i] = lat
			mu.Unlock()
		}(i, k)
	}
	wg.Wait()
	elapsed := time.Since(start).Seconds()
	return buildStats(latencies, elapsed, len(keys), totalB, int(errors))
}

func buildStats(latencies []float64, elapsed float64, n int, bytes int64, errs int) *phaseStats {
	valid := make([]float64, 0, len(latencies))
	for _, l := range latencies {
		if l > 0 {
			valid = append(valid, l)
		}
	}
	sort.Float64s(valid)
	p50, p95 := 0.0, 0.0
	if len(valid) > 0 {
		p50 = valid[len(valid)/2]
		p95 = valid[(len(valid)*95)/100]
	}
	mb := float64(bytes) / (1024 * 1024)
	return &phaseStats{
		Elapsed: elapsed, Count: n, MBTotal: mb,
		ObjPerS: float64(n) / elapsed, MBPerS: mb / elapsed,
		Errors: errs, P50ms: p50, P95ms: p95,
	}
}

func deleteKeys(client *s3.S3, bucket string, keys []string) {
	for i := 0; i < len(keys); i += 1000 {
		end := i + 1000
		if end > len(keys) {
			end = len(keys)
		}
		objs := make([]*s3.ObjectIdentifier, 0, end-i)
		for _, k := range keys[i:end] {
			kk := k
			objs = append(objs, &s3.ObjectIdentifier{Key: &kk})
		}
		_, _ = client.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: &bucket,
			Delete: &s3.Delete{Objects: objs},
		})
	}
}

func workloadSpecs(wl string) []objSpec {
	switch wl {
	case "resnet":
		out := make([]objSpec, 0, 5000)
		for cls := 0; cls < 100; cls++ {
			for img := 0; img < 50; img++ {
				sz, _ := rand.Int(rand.Reader, big.NewInt(250*1024))
				size := int(sz.Int64()) + 50*1024
				out = append(out, objSpec{
					key:     fmt.Sprintf("imagenet/n%08d/img_%05d.JPEG", cls, img),
					payload: randBytes(size),
				})
			}
		}
		return out
	case "unet3d":
		return nObjs(10, 200*1024*1024, "unet3d/case_%05d/imaging.nii.gz")
	case "bert":
		out := make([]objSpec, 30)
		for i := 0; i < 30; i++ {
			sz, _ := rand.Int(rand.Reader, big.NewInt(20*1024*1024))
			out[i] = objSpec{
				key:     fmt.Sprintf("bert/wiki_%04d.tfrecord", i),
				payload: randBytes(int(sz.Int64()) + 20*1024*1024),
			}
		}
		return out
	case "cosmoflow":
		return nObjs(20, 128*1024*1024, "cosmoflow/sample_%04d.tfrecord")
	case "dlrm":
		return nObjs(3, 1024*1024*1024, "dlrm/day_%04d.tfrecord")
	}
	return nil
}

func nObjs(n, size int, keyFmt string) []objSpec {
	out := make([]objSpec, n)
	for i := 0; i < n; i++ {
		out[i] = objSpec{key: fmt.Sprintf(keyFmt, i), payload: randBytes(size)}
	}
	return out
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// unexported-helper smoke so tests don't get flagged for unused imports when conditionally compiled
var _ = filepath.Join

// ---- MLPerf Storage v2.0 Accelerator Utilization (AU) ----
//
// AU is the headline MLPerf Storage metric: fraction of wall-clock time the
// simulated accelerator is doing compute instead of waiting on storage.
//
//   AU = compute_time / (compute_time + storage_wait_time)
//
// storage_wait_time = batch_bytes / observed_MBps
// compute_time      = batch_size × per_sample_compute_ms
//
// Reference compute times are per-sample on an NVIDIA A100 as published in
// the MLPerf Storage v2.0 rules (https://mlcommons.org/benchmarks/storage/).
// Target: AU ≥ 90% — "storage is fast enough to keep the GPU fed."
//
// Assumes single-accelerator no-overlap (worst case). A prefetching pipeline
// can push AU higher, but that's a framework-level optimisation, not a
// storage property.
type workloadAUParams struct {
	BatchSize        int
	SampleBytes      int64
	ComputeMsPerSample float64
}

var auParams = map[string]workloadAUParams{
	// values from MLPerf Storage v2.0 reference runs on A100
	"resnet":    {BatchSize: 48, SampleBytes: 175 * 1024, ComputeMsPerSample: 8.3},
	"unet3d":    {BatchSize: 4, SampleBytes: 200 * 1024 * 1024, ComputeMsPerSample: 4125},
	"bert":      {BatchSize: 64, SampleBytes: 31 * 1024 * 1024, ComputeMsPerSample: 100},
	"cosmoflow": {BatchSize: 8, SampleBytes: 128 * 1024 * 1024, ComputeMsPerSample: 45},
	"dlrm":      {BatchSize: 16, SampleBytes: 1024 * 1024 * 1024, ComputeMsPerSample: 25},
}

func computeAU(workload string, mbPerS float64) float64 {
	p, ok := auParams[workload]
	if !ok || mbPerS <= 0 {
		return 0
	}
	batchBytes := float64(p.BatchSize) * float64(p.SampleBytes)
	storageWaitMs := batchBytes / (mbPerS * 1024 * 1024) * 1000
	computeMs := float64(p.BatchSize) * p.ComputeMsPerSample
	return computeMs / (computeMs + storageWaitMs)
}
