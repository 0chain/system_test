package zs3servertests

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverWriteThroughFallback verifies that a PUT through zs3server
// also lands on the configured upstream S3 (write-through or write-back
// semantics — the S3 Files analog where writes go to EFS and sync to S3).
//
// STATUS 2026-04-19: the feature is NOT yet implemented in
// cmd/gateway/zcn/fallback_s3.go (only read-fallback exists — tryFallbackFetch
// + fallbackStat). This test is intentionally a failing regression gate that
// will turn GREEN once the write-through / write-back code lands.
//
// The test is SKIPPED when the gateway reports no upstream — it only fails
// when fallback is enabled AND the upstream doesn't see a recent PUT. That
// lets the test tree stay clean on non-fallback deployments.
//
// Env:
//
//	ZS3_UPSTREAM_ENDPOINT   — upstream S3 endpoint (e.g. http://localhost:9200)
//	ZS3_UPSTREAM_ACCESS_KEY — default "upstream"
//	ZS3_UPSTREAM_SECRET_KEY — default "upstream123"
//	ZS3_UPSTREAM_BUCKET     — bucket on upstream, default same as zs3 bucket
//	ZS3_WRITE_THROUGH_MODE  — "mirror" | "primary_upstream" | "primary_zus"
//	                         controls the expected semantics; default "mirror"
func TestZs3serverWriteThroughFallback(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	upstreamEndpoint := os.Getenv("ZS3_UPSTREAM_ENDPOINT")
	if upstreamEndpoint == "" {
		testSetup.Skip("ZS3_UPSTREAM_ENDPOINT unset — write-through requires an upstream to observe")
	}
	upAK := getenvDefault("ZS3_UPSTREAM_ACCESS_KEY", "upstream")
	upSK := getenvDefault("ZS3_UPSTREAM_SECRET_KEY", "upstream123")

	t := test.NewSystemTest(testSetup)

	bucket := getenvDefault("ZS3_WT_BUCKET", "zs3-write-through")
	upBucket := getenvDefault("ZS3_UPSTREAM_BUCKET", bucket)
	mode := getenvDefault("ZS3_WRITE_THROUGH_MODE", "mirror")

	zs3Endpoint := "http://" + config.Server + ":" + config.HostPort
	zs3Client := newS3Client(t, zs3Endpoint, config.AccessKey, config.SecretKey)
	if err := ensureBucket(zs3Client, bucket); err != nil {
		t.Fatalf("zs3 mb: %v", err)
	}

	upstreamClient := newUpstreamS3Client(t, upstreamEndpoint, upAK, upSK)
	_ = ensureBucket(upstreamClient, upBucket)

	t.RunSequentiallyWithTimeout("write-through semantic: "+mode, 5*time.Minute, func(t *test.SystemTest) {
		key := fmt.Sprintf("wt-probe-%d.bin", time.Now().UnixNano())
		data := randBytes(1 << 20) // 1 MB — small enough to go through WAL path
		wantMD5 := hex.EncodeToString(md5.New().Sum(data)[:0]) // placeholder; we'll hash after write
		h := md5.New()
		_, _ = h.Write(data)
		wantMD5 = hex.EncodeToString(h.Sum(nil))

		// PUT through zs3server
		if _, err := zs3Client.PutObject(&s3.PutObjectInput{
			Bucket: &bucket, Key: &key, Body: bytes.NewReader(data),
		}); err != nil {
			t.Fatalf("zs3 PUT: %v", err)
		}

		// Poll upstream — allow async sync up to 30s
		deadline := time.Now().Add(30 * time.Second)
		var found bool
		for time.Now().Before(deadline) {
			out, err := upstreamClient.GetObject(&s3.GetObjectInput{Bucket: &upBucket, Key: &key})
			if err == nil {
				got, _ := io.ReadAll(out.Body)
				_ = out.Body.Close()
				h2 := md5.New()
				_, _ = h2.Write(got)
				gotMD5 := hex.EncodeToString(h2.Sum(nil))
				if gotMD5 == wantMD5 {
					found = true
					break
				}
				t.Logf("upstream body md5 mismatch want=%s got=%s", wantMD5, gotMD5)
			}
			time.Sleep(500 * time.Millisecond)
		}

		if !found {
			switch mode {
			case "mirror", "primary_upstream":
				t.Fatalf("write-through feature not implemented — zs3 PUT did not reach upstream within 30s (key=%s)", key)
			case "primary_zus":
				t.Logf("primary_zus mode: upstream never syncs on write (expected). To test upstream copy, enable archive mode.")
			default:
				t.Fatalf("unknown mode %q; upstream never saw the write", mode)
			}
		} else {
			t.Logf("write-through OK: upstream has byte-identical copy of %s (md5=%s)", key, wantMD5)
		}

		// cleanup
		_, _ = zs3Client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &key})
		_, _ = upstreamClient.DeleteObject(&s3.DeleteObjectInput{Bucket: &upBucket, Key: &key})
	})
}

func newUpstreamS3Client(t *test.SystemTest, endpoint, ak, sk string) *s3.S3 {
	sess, err := awsSession.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(ak, sk, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("upstream session: %v", err)
	}
	return s3.New(sess)
}
