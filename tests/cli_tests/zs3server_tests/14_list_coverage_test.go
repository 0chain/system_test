package zs3servertests

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverListCoverage fills gaps left by the existing warp-driven LIST
// test (4_listing_purge_test.go) — that covers raw LIST throughput but not
// correctness across protocols or pagination / prefix filtering edge cases.
//
// Subtests:
//   pagination          — write > 1000 objects, paginate through ListObjectsV2
//                          continuation tokens, verify all keys appear once.
//   prefix filter       — write a mixed namespace, list with various prefixes.
//   cross-protocol list — write via S3, verify the same names appear via NFS
//                          readdir (and vice versa).
//   post-delete list    — delete objects, verify LIST doesn't return ghosts.
//   empty bucket list   — LIST on an empty bucket returns zero entries, not err.
//   delimiter hierarchy — LIST with delimiter="/" returns CommonPrefixes
//                          correctly (S3 Files clients rely on this).
func TestZs3serverListCoverage(testSetup *testing.T) {
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
	bucket := getenvDefault("ZS3_LIST_BUCKET", "zs3-list-coverage")
	_ = ensureBucket(client, bucket)

	t.RunSequentiallyWithTimeout("pagination (1500 objects)", 10*time.Minute, func(t *test.SystemTest) {
		N := 1500
		small := []byte("x")
		keys := make([]string, N)
		for i := 0; i < N; i++ {
			keys[i] = fmt.Sprintf("page/obj_%05d.txt", i)
			if _, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &keys[i], Body: bytes.NewReader(small),
			}); err != nil {
				t.Fatalf("PUT %s: %v", keys[i], err)
			}
		}
		defer deleteKeys(client, bucket, keys)

		seen := map[string]bool{}
		var token *string
		pages := 0
		for {
			out, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
				Bucket:            &bucket,
				Prefix:            aws.String("page/"),
				ContinuationToken: token,
				MaxKeys:           aws.Int64(500),
			})
			if err != nil {
				t.Fatalf("LISTv2: %v", err)
			}
			pages++
			for _, o := range out.Contents {
				if o.Key != nil {
					if seen[*o.Key] {
						t.Fatalf("duplicate key in LIST: %s", *o.Key)
					}
					seen[*o.Key] = true
				}
			}
			if !aws.BoolValue(out.IsTruncated) {
				break
			}
			token = out.NextContinuationToken
			if pages > 20 {
				t.Fatalf("too many pages (%d) — continuation token broken?", pages)
			}
		}
		if len(seen) != N {
			t.Fatalf("LIST missed keys: expected %d, got %d (pages=%d)", N, len(seen), pages)
		}
		t.Logf("pagination OK: %d keys across %d pages", N, pages)
	})

	t.RunSequentiallyWithTimeout("prefix filter", 5*time.Minute, func(t *test.SystemTest) {
		put := func(key string) {
			if _, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &key, Body: bytes.NewReader([]byte("x")),
			}); err != nil {
				t.Fatalf("PUT %s: %v", key, err)
			}
		}
		for _, k := range []string{
			"prefixA/a1.txt", "prefixA/a2.txt",
			"prefixB/b1.txt",
			"prefixAB/ab1.txt",
			"other.txt",
		} {
			put(k)
		}
		defer deleteKeys(client, bucket, []string{
			"prefixA/a1.txt", "prefixA/a2.txt", "prefixB/b1.txt", "prefixAB/ab1.txt", "other.txt",
		})

		countPrefix := func(prefix string) int {
			out, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
				Bucket: &bucket, Prefix: aws.String(prefix),
			})
			if err != nil {
				t.Fatalf("LIST prefix=%q: %v", prefix, err)
			}
			return len(out.Contents)
		}
		if c := countPrefix("prefixA/"); c != 2 {
			t.Fatalf("prefixA/ expected 2, got %d", c)
		}
		if c := countPrefix("prefixAB/"); c != 1 {
			t.Fatalf("prefixAB/ expected 1, got %d", c)
		}
		if c := countPrefix("prefix"); c < 4 {
			t.Fatalf("prefix (ambiguous) expected ≥4, got %d", c)
		}
		t.Logf("prefix filter OK")
	})

	t.RunSequentiallyWithTimeout("cross-protocol LIST S3→NFS", 5*time.Minute, func(t *test.SystemTest) {
		nfsMount := os.Getenv("ZS3_NFS_MOUNT")
		if nfsMount == "" {
			t.Skip("ZS3_NFS_MOUNT unset, skipping cross-protocol LIST")
		}
		if _, err := os.Stat(nfsMount); err != nil {
			t.Skipf("NFS mount not accessible: %v", err)
		}

		names := []string{"xp_a.txt", "xp_b.txt", "xp_c.txt"}
		for _, n := range names {
			k := "xprotocol/" + n
			if _, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &k, Body: bytes.NewReader([]byte("x")),
			}); err != nil {
				t.Fatalf("PUT %s: %v", k, err)
			}
		}
		defer func() {
			for _, n := range names {
				k := "xprotocol/" + n
				_, _ = client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &k})
			}
		}()

		// Allow mirror_s3_to_export to materialise stubs
		time.Sleep(2 * time.Second)

		nfsDir := filepath.Join(nfsMount, bucket, "xprotocol")
		entries, err := os.ReadDir(nfsDir)
		if err != nil {
			t.Fatalf("readdir %s: %v", nfsDir, err)
		}
		got := map[string]bool{}
		for _, e := range entries {
			got[e.Name()] = true
		}
		for _, n := range names {
			if !got[n] {
				t.Fatalf("NFS readdir missing %s (S3 wrote, NFS didn't see)", n)
			}
		}
		t.Logf("cross-protocol LIST OK: all %d S3-written names visible via NFS readdir", len(names))
	})

	t.RunSequentiallyWithTimeout("post-delete LIST", 3*time.Minute, func(t *test.SystemTest) {
		keys := []string{"pd_1.txt", "pd_2.txt", "pd_3.txt"}
		for _, k := range keys {
			kk := k
			if _, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &kk, Body: bytes.NewReader([]byte("x")),
			}); err != nil {
				t.Fatalf("PUT %s: %v", k, err)
			}
		}
		for _, k := range keys {
			kk := k
			if _, err := client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucket, Key: &kk}); err != nil {
				t.Fatalf("DEL %s: %v", k, err)
			}
		}
		out, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: &bucket, Prefix: aws.String("pd_"),
		})
		if err != nil {
			t.Fatalf("LIST after DELETE: %v", err)
		}
		if len(out.Contents) != 0 {
			names := []string{}
			for _, o := range out.Contents {
				names = append(names, aws.StringValue(o.Key))
			}
			t.Fatalf("ghost keys in LIST after DELETE: %s", strings.Join(names, ","))
		}
		t.Logf("post-delete LIST OK: no ghosts")
	})

	t.RunSequentiallyWithTimeout("delimiter hierarchy", 3*time.Minute, func(t *test.SystemTest) {
		for _, k := range []string{
			"dir1/a.txt", "dir1/b.txt",
			"dir2/c.txt",
			"root.txt",
		} {
			kk := k
			if _, err := client.PutObject(&s3.PutObjectInput{
				Bucket: &bucket, Key: &kk, Body: bytes.NewReader([]byte("x")),
			}); err != nil {
				t.Fatalf("PUT %s: %v", k, err)
			}
		}
		defer deleteKeys(client, bucket, []string{"dir1/a.txt", "dir1/b.txt", "dir2/c.txt", "root.txt"})

		out, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:    &bucket,
			Delimiter: aws.String("/"),
		})
		if err != nil {
			t.Fatalf("LIST with delimiter: %v", err)
		}
		// Expect root.txt in Contents, dir1/ and dir2/ in CommonPrefixes
		hasRoot := false
		for _, o := range out.Contents {
			if aws.StringValue(o.Key) == "root.txt" {
				hasRoot = true
			}
		}
		if !hasRoot {
			t.Fatalf("root.txt missing from Contents (delimiter='/' should surface root files)")
		}
		prefixes := map[string]bool{}
		for _, cp := range out.CommonPrefixes {
			prefixes[aws.StringValue(cp.Prefix)] = true
		}
		if !prefixes["dir1/"] || !prefixes["dir2/"] {
			t.Fatalf("CommonPrefixes missing dir1/ or dir2/: got %v", prefixes)
		}
		t.Logf("delimiter hierarchy OK")
	})

	t.RunSequentiallyWithTimeout("empty bucket LIST", 1*time.Minute, func(t *test.SystemTest) {
		emptyBucket := bucket + "-empty"
		_, _ = client.CreateBucket(&s3.CreateBucketInput{Bucket: &emptyBucket})
		defer func() { _, _ = client.DeleteBucket(&s3.DeleteBucketInput{Bucket: &emptyBucket}) }()

		out, err := client.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: &emptyBucket})
		if err != nil {
			t.Fatalf("LIST empty bucket: %v", err)
		}
		if len(out.Contents) != 0 {
			t.Fatalf("empty bucket returned %d entries", len(out.Contents))
		}
		t.Logf("empty LIST OK")
	})
}
