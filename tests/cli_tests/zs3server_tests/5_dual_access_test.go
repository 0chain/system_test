package zs3servertests

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// TestZs3serverDualAccess verifies that data written via S3 is readable via NFS
// and vice versa (dual-access consistency).
//
// Prerequisites:
//   - hosts.yaml with S3 server config
//   - NFS mounted (set ZS3_NFS_MOUNT env var, default /mnt/zus_nfs)
//   - mc binary at ../mc
func TestZs3serverDualAccess(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml config not found, skipping test")
	}

	config := cliutils.ReadFile(testSetup)

	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("ZS3 server not available at %s:%s, skipping: %v", config.Server, config.HostPort, err)
	}
	conn.Close()

	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Skip("mc binary not available at ../mc, skipping test")
	}

	nfsMount := os.Getenv("ZS3_NFS_MOUNT")
	if nfsMount == "" {
		nfsMount = "/mnt/zus_nfs"
	}
	if _, err := os.Stat(nfsMount); os.IsNotExist(err) {
		testSetup.Skipf("NFS mount not found at %s (set ZS3_NFS_MOUNT env var), skipping", nfsMount)
	}

	t := test.NewSystemTest(testSetup)

	bucket := fmt.Sprintf("dual-test-%d", time.Now().Unix())

	// Setup mc alias
	_, _ = cliutils.RunCommand(t, "../mc alias rm zs3dual", 1, time.Second*5)
	aliasCmd := fmt.Sprintf("../mc alias set zs3dual http://%s:%s %s %s --api S3v4",
		config.Server, config.HostPort, config.AccessKey, config.SecretKey)
	output, err := cliutils.RunCommand(t, aliasCmd, 1, time.Minute)
	if err != nil {
		testSetup.Fatalf("Failed to set mc alias: %v\nOutput: %s", err, output)
	}

	// Create bucket
	_, err = cliutils.RunCommand(t, fmt.Sprintf("../mc mb zs3dual/%s", bucket), 1, time.Minute)
	if err != nil {
		testSetup.Fatalf("Failed to create bucket: %v", err)
	}

	nfsBucket := nfsMount + "/" + bucket
	time.Sleep(2 * time.Second)

	// Cleanup at end
	defer func() {
		cliutils.RunCommand(t, fmt.Sprintf("../mc rb --force zs3dual/%s", bucket), 1, time.Minute) //nolint:errcheck
		cliutils.RunCommand(t, "../mc alias rm zs3dual", 1, time.Second*5)                          //nolint:errcheck
	}()

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_PUT_then_NFS_READ_small_file", time.Minute*2, func(t *test.SystemTest) {
		content := fmt.Sprintf("s3-to-nfs-%d", time.Now().UnixNano())
		tmpFile := writeTempFileHelper(testSetup, content)
		defer os.Remove(tmpFile)

		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s zs3dual/%s/s3-to-nfs.txt", tmpFile, bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 PUT failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		data, err := os.ReadFile(nfsBucket + "/s3-to-nfs.txt")
		if err != nil {
			testSetup.Fatalf("NFS read failed: %v", err)
		}
		if string(data) != content {
			testSetup.Fatalf("content mismatch: S3 wrote %q, NFS read %q", content, string(data))
		}
		t.Logf("PASS: S3 PUT -> NFS READ content match")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("NFS_WRITE_then_S3_READ_small_file", time.Minute*2, func(t *test.SystemTest) {
		content := fmt.Sprintf("nfs-to-s3-%d", time.Now().UnixNano())

		if err := os.WriteFile(nfsBucket+"/nfs-to-s3.txt", []byte(content), 0644); err != nil {
			testSetup.Fatalf("NFS write failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		output, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cat zs3dual/%s/nfs-to-s3.txt", bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 GET failed: %v", err)
		}
		s3Content := strings.Join(output, "")
		if s3Content != content {
			testSetup.Fatalf("content mismatch: NFS wrote %q, S3 read %q", content, s3Content)
		}
		t.Logf("PASS: NFS WRITE -> S3 READ content match")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_PUT_100KB_then_NFS_READ_checksum", time.Minute*2, func(t *test.SystemTest) {
		tmpFile := writeTempRandomHelper(testSetup, 100*1024)
		defer os.Remove(tmpFile)
		expectedMD5 := fileMD5Helper(testSetup, tmpFile)

		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s zs3dual/%s/medium-s3.bin", tmpFile, bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 PUT 100KB failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		actualMD5 := fileMD5Helper(testSetup, nfsBucket+"/medium-s3.bin")
		if expectedMD5 != actualMD5 {
			testSetup.Fatalf("checksum mismatch: S3 PUT -> NFS READ (expected %s, got %s)", expectedMD5, actualMD5)
		}
		t.Logf("PASS: S3 PUT 100KB -> NFS READ checksum match")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("NFS_WRITE_100KB_then_S3_READ_checksum", time.Minute*2, func(t *test.SystemTest) {
		tmpFile := writeTempRandomHelper(testSetup, 100*1024)
		defer os.Remove(tmpFile)
		expectedMD5 := fileMD5Helper(testSetup, tmpFile)

		data, err := os.ReadFile(tmpFile)
		if err != nil {
			testSetup.Fatalf("read temp file failed: %v", err)
		}
		if err := os.WriteFile(nfsBucket+"/medium-nfs.bin", data, 0644); err != nil {
			testSetup.Fatalf("NFS write 100KB failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		s3Tmp := fmt.Sprintf("/tmp/s3-read-%d", time.Now().UnixNano())
		defer os.Remove(s3Tmp)
		_, err = cliutils.RunCommand(t, fmt.Sprintf("../mc cp zs3dual/%s/medium-nfs.bin %s", bucket, s3Tmp), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 GET 100KB failed: %v", err)
		}
		actualMD5 := fileMD5Helper(testSetup, s3Tmp)
		if expectedMD5 != actualMD5 {
			testSetup.Fatalf("checksum mismatch: NFS WRITE -> S3 READ (expected %s, got %s)", expectedMD5, actualMD5)
		}
		t.Logf("PASS: NFS WRITE 100KB -> S3 READ checksum match")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_LIST_sees_NFS_written_files", time.Minute*2, func(t *test.SystemTest) {
		output, err := cliutils.RunCommand(t, fmt.Sprintf("../mc ls zs3dual/%s/", bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 LIST failed: %v", err)
		}
		listing := strings.Join(output, "\n")
		if !strings.Contains(listing, "nfs-to-s3.txt") {
			testSetup.Fatalf("S3 LIST does not see NFS-written file nfs-to-s3.txt. Listing:\n%s", listing)
		}
		if !strings.Contains(listing, "medium-nfs.bin") {
			testSetup.Fatalf("S3 LIST does not see NFS-written file medium-nfs.bin. Listing:\n%s", listing)
		}
		t.Logf("PASS: S3 LIST sees NFS-written files")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("NFS_LIST_sees_S3_written_files", time.Minute*2, func(t *test.SystemTest) {
		entries, err := os.ReadDir(nfsBucket)
		if err != nil {
			testSetup.Fatalf("NFS readdir failed: %v", err)
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		nameStr := strings.Join(names, ",")
		if !strings.Contains(nameStr, "s3-to-nfs.txt") {
			testSetup.Fatalf("NFS LIST does not see S3-written file s3-to-nfs.txt. Files: %s", nameStr)
		}
		if !strings.Contains(nameStr, "medium-s3.bin") {
			testSetup.Fatalf("NFS LIST does not see S3-written file medium-s3.bin. Files: %s", nameStr)
		}
		t.Logf("PASS: NFS LIST sees S3-written files")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_DELETE_then_NFS_verify_gone", time.Minute*2, func(t *test.SystemTest) {
		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc rm zs3dual/%s/s3-to-nfs.txt", bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 DELETE failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		if _, err := os.Stat(nfsBucket + "/s3-to-nfs.txt"); !os.IsNotExist(err) {
			testSetup.Fatalf("file still visible via NFS after S3 DELETE")
		}
		t.Logf("PASS: S3 DELETE -> NFS file gone")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("NFS_DELETE_then_S3_verify_gone", time.Minute*2, func(t *test.SystemTest) {
		if err := os.Remove(nfsBucket + "/nfs-to-s3.txt"); err != nil {
			testSetup.Fatalf("NFS DELETE failed: %v", err)
		}
		time.Sleep(2 * time.Second)

		output, _ := cliutils.RunCommand(t, fmt.Sprintf("../mc stat zs3dual/%s/nfs-to-s3.txt", bucket), 1, time.Minute)
		outStr := strings.ToLower(strings.Join(output, "\n"))
		if !strings.Contains(outStr, "not found") && !strings.Contains(outStr, "does not exist") {
			testSetup.Fatalf("file still visible via S3 after NFS DELETE: %s", outStr)
		}
		t.Logf("PASS: NFS DELETE -> S3 file gone")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_overwrite_then_NFS_sees_new_content", time.Minute*2, func(t *test.SystemTest) {
		v1 := writeTempFileHelper(testSetup, "version1")
		defer os.Remove(v1)
		cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s zs3dual/%s/overwrite.txt", v1, bucket), 1, time.Minute) //nolint:errcheck
		time.Sleep(2 * time.Second)

		v2 := writeTempFileHelper(testSetup, "version2")
		defer os.Remove(v2)
		cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s zs3dual/%s/overwrite.txt", v2, bucket), 1, time.Minute) //nolint:errcheck
		time.Sleep(2 * time.Second)

		data, err := os.ReadFile(nfsBucket + "/overwrite.txt")
		if err != nil {
			testSetup.Fatalf("NFS read overwrite.txt failed: %v", err)
		}
		if string(data) != "version2" {
			testSetup.Fatalf("NFS does not see S3 overwrite: got %q, want %q", string(data), "version2")
		}
		t.Logf("PASS: S3 overwrite -> NFS sees new content")
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("NFS_overwrite_then_S3_sees_new_content", time.Minute*2, func(t *test.SystemTest) {
		os.WriteFile(nfsBucket+"/nfs-overwrite.txt", []byte("nfs-v1"), 0644) //nolint:errcheck
		time.Sleep(2 * time.Second)

		os.WriteFile(nfsBucket+"/nfs-overwrite.txt", []byte("nfs-v2"), 0644) //nolint:errcheck
		time.Sleep(2 * time.Second)

		output, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cat zs3dual/%s/nfs-overwrite.txt", bucket), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("S3 GET nfs-overwrite.txt failed: %v", err)
		}
		if strings.Join(output, "") != "nfs-v2" {
			testSetup.Fatalf("S3 does not see NFS overwrite: got %q, want %q", strings.Join(output, ""), "nfs-v2")
		}
		t.Logf("PASS: NFS overwrite -> S3 sees new content")
	})
}

// --- helpers ---

func writeTempFileHelper(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "dual-test-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func writeTempRandomHelper(t *testing.T, size int) string {
	t.Helper()
	f, err := os.CreateTemp("", "dual-test-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	// Deterministic pseudo-random for reproducibility
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i * 37) ^ (i >> 8))
	}
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write temp random: %v", err)
	}
	f.Close()
	return f.Name()
}

func fileMD5Helper(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s for MD5: %v", path, err)
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("read %s for MD5: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
