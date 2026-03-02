package api_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	mcAlias = "zs3test"
)

// mcCmd runs an mc CLI command and returns stdout, stderr, and error.
func mcCmd(args ...string) (string, string, error) {
	cmd := exec.Command("mc", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// setupMcAlias configures the mc alias for the zs3 server.
func setupMcAlias(t *test.SystemTest, serverUrl string) {
	_, _, err := mcCmd("alias", "set", mcAlias, serverUrl, parsedConfig.S3AccessKey, parsedConfig.S3SecretKey)
	require.Nil(t, err, "failed to set mc alias for zs3 server")
}

func TestZs3ServerOperations(testSetup *testing.T) {
	if !zs3Available {
		testSetup.Skip("zs3 server is not available, skipping zs3 tests")
	}

	// Check mc binary is available
	if _, err := exec.LookPath("mc"); err != nil {
		testSetup.Skip("mc (MinIO client) binary not found in PATH, skipping zs3 tests")
	}

	t := test.NewSystemTest(testSetup)

	t.SetSmokeTests("CreateBucket should work",
		"ListBuckets should work",
		"ListObjects should work",
		"PutObject should work",
		"GetObject should work",
		"RemoveObject should work")

	// Setup mc alias using the configured zs3 server URL
	setupMcAlias(t, parsedConfig.ZS3ServerUrl)

	bucketName := "zs3-system-test"
	bucketPath := fmt.Sprintf("%s/%s", mcAlias, bucketName)

	// Ensure cleanup after all tests
	testSetup.Cleanup(func() {
		// Best-effort cleanup: remove test bucket and contents
		_, _, _ = mcCmd("rb", "--force", bucketPath)
	})

	t.RunSequentially("CreateBucket should work", func(t *test.SystemTest) {
		_, stderr, err := mcCmd("mb", bucketPath)
		require.Nil(t, err, "mc mb failed: %s", stderr)
	})

	t.RunSequentially("CreateBucket should not error when bucket already exists", func(t *test.SystemTest) {
		// First create
		_, _, _ = mcCmd("mb", bucketPath)
		// Second create — should succeed or return "already exists" (not a hard error for mc)
		_, stderr, err := mcCmd("mb", bucketPath)
		// mc returns exit 1 for "already own it" but that's acceptable
		if err != nil {
			require.Contains(t, stderr, "already", "expected 'already exists' error, got: %s", stderr)
		}
	})

	t.RunSequentially("ListBuckets should work", func(t *test.SystemTest) {
		stdout, stderr, err := mcCmd("ls", mcAlias+"/")
		require.Nil(t, err, "mc ls failed: %s", stderr)
		require.Contains(t, stdout, bucketName, "bucket %s not found in listing: %s", bucketName, stdout)
	})

	t.RunSequentially("PutObject should work", func(t *test.SystemTest) {
		// Create a temp test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-file.txt")
		err := os.WriteFile(testFile, []byte("hello world from zs3 system test"), 0644)
		require.Nil(t, err, "failed to create test file")

		_, stderr, err := mcCmd("cp", testFile, bucketPath+"/test-file.txt")
		require.Nil(t, err, "mc cp (upload) failed: %s", stderr)
	})

	t.RunSequentially("ListObjects should work", func(t *test.SystemTest) {
		// Retry listing to handle ZS3 index propagation delay on fresh allocations
		var stdout, stderr string
		var err error
		for attempt := 0; attempt < 15; attempt++ {
			stdout, stderr, err = mcCmd("ls", bucketPath+"/")
			if err == nil && strings.Contains(stdout, "test-file.txt") {
				break
			}
			time.Sleep(2 * time.Second)
		}
		require.Nil(t, err, "mc ls (objects) failed: %s", stderr)
		require.Contains(t, stdout, "test-file.txt", "uploaded file not found in listing: %s", stdout)
	})

	t.RunSequentially("GetObject should work", func(t *test.SystemTest) {
		tmpDir := t.TempDir()
		downloadFile := filepath.Join(tmpDir, "downloaded.txt")

		_, stderr, err := mcCmd("cp", bucketPath+"/test-file.txt", downloadFile)
		require.Nil(t, err, "mc cp (download) failed: %s", stderr)

		content, err := os.ReadFile(downloadFile)
		require.Nil(t, err, "failed to read downloaded file")
		require.Equal(t, "hello world from zs3 system test", string(content))
	})

	t.RunSequentially("RemoveObject should work", func(t *test.SystemTest) {
		_, stderr, err := mcCmd("rm", bucketPath+"/test-file.txt")
		require.Nil(t, err, "mc rm failed: %s", stderr)

		// Verify object is gone
		stdout, _, _ := mcCmd("ls", bucketPath+"/")
		require.NotContains(t, stdout, "test-file.txt", "file should be removed but still appears in listing")
	})

	t.RunSequentially("PutObject to non-existent bucket should fail", func(t *test.SystemTest) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-file.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.Nil(t, err)

		_, _, err = mcCmd("cp", testFile, mcAlias+"/nonexistent-bucket-12345/test.txt")
		require.NotNil(t, err, "expected error when uploading to non-existent bucket")
	})

	t.RunSequentially("Invalid credentials should fail", func(t *test.SystemTest) {
		badAlias := "zs3badcreds"
		_, _, _ = mcCmd("alias", "set", badAlias, parsedConfig.ZS3ServerUrl, "wrong-key", "wrong-secret")

		_, _, err := mcCmd("ls", badAlias+"/")
		require.NotNil(t, err, "expected error with invalid credentials")

		// Cleanup bad alias
		_, _, _ = mcCmd("alias", "rm", badAlias)
	})

	t.RunSequentially("RemoveBucket should work", func(t *test.SystemTest) {
		// Create a temporary bucket to remove
		tmpBucket := mcAlias + "/zs3-remove-test"
		_, _, _ = mcCmd("mb", tmpBucket)

		_, stderr, err := mcCmd("rb", tmpBucket)
		require.Nil(t, err, "mc rb failed: %s", stderr)

		// Verify bucket is gone
		stdout, _, _ := mcCmd("ls", mcAlias+"/")
		require.NotContains(t, stdout, "zs3-remove-test", "bucket should be removed but still appears")
	})
}
