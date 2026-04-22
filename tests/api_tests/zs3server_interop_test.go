package api_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// TestZs3ServerInterop verifies universal S3 interoperability: the same
// Züs wallet/allocation is accessible via AWS CLI, mc (MinIO client), and
// rclone. Objects written by any client must be immediately visible to the
// other two. Tests cover PUT, GET, LIST, DELETE, and overwrite across all
// writer→reader combinations.
//
// Required config fields (api_tests_config.yaml):
//
//	zs3_server_url, s3_access_key, s3_secret_key
//
// Optional env overrides:
//
//	AWS_BIN    — path to aws binary (default: aws)
//	RCLONE_BIN — path to rclone binary (default: rclone)
func TestZs3ServerInterop(testSetup *testing.T) {
	if !zs3Available {
		zs3Available = isServiceResponding(parsedConfig.ZS3ServerUrl)
	}
	if !zs3Available {
		testSetup.Fatalf("zs3 server not available at %s", parsedConfig.ZS3ServerUrl)
	}

	awsBin := "aws"
	if v := os.Getenv("AWS_BIN"); v != "" {
		awsBin = v
	}
	rcloneBin := "rclone"
	if v := os.Getenv("RCLONE_BIN"); v != "" {
		rcloneBin = v
	}

	for _, bin := range []struct{ name, path string }{
		{"aws", awsBin}, {"mc", mcPath}, {"rclone", rcloneBin},
	} {
		if _, err := exec.LookPath(bin.path); err != nil {
			testSetup.Fatalf("%s binary not found (set %s_BIN to override): %v",
				bin.name, strings.ToUpper(bin.name), err)
		}
	}

	const interopAlias = "zs3interop"
	_, _, err := mcCmd("alias", "set", interopAlias,
		parsedConfig.ZS3ServerUrl, parsedConfig.S3AccessKey, parsedConfig.S3SecretKey)
	require.Nil(testSetup, err, "mc alias set for interop failed")
	testSetup.Cleanup(func() { mcCmd("alias", "rm", interopAlias) }) //nolint

	endpoint := parsedConfig.ZS3ServerUrl
	accessKey := parsedConfig.S3AccessKey
	secretKey := parsedConfig.S3SecretKey

	bucket := "zs3-interop-test"
	mcBucket := interopAlias + "/" + bucket

	// aws CLI helpers
	awsEnv := append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+accessKey,
		"AWS_SECRET_ACCESS_KEY="+secretKey,
		"AWS_DEFAULT_REGION=us-east-1",
	)
	awsRun := func(t *test.SystemTest, args ...string) (string, string, error) {
		cmd := exec.Command(awsBin, args...)
		cmd.Env = awsEnv
		var so, se strings.Builder
		cmd.Stdout, cmd.Stderr = &so, &se
		err := cmd.Run()
		return strings.TrimSpace(so.String()), strings.TrimSpace(se.String()), err
	}
	awsPut := func(t *test.SystemTest, key string, data []byte) {
		f := writeTempFile(t, data)
		_, stderr, err := awsRun(t, "s3", "cp", f,
			fmt.Sprintf("s3://%s/%s", bucket, key),
			"--endpoint-url", endpoint, "--no-progress")
		require.Nil(t, err, "aws s3 cp PUT %s: %s", key, stderr)
	}
	awsGet := func(t *test.SystemTest, key string) []byte {
		dst := filepath.Join(t.TempDir(), "aws-dl")
		_, stderr, err := awsRun(t, "s3", "cp",
			fmt.Sprintf("s3://%s/%s", bucket, key), dst,
			"--endpoint-url", endpoint, "--no-progress")
		require.Nil(t, err, "aws s3 cp GET %s: %s", key, stderr)
		b, e := os.ReadFile(dst)
		require.Nil(t, e)
		return b
	}
	awsDelete := func(t *test.SystemTest, key string) {
		_, stderr, err := awsRun(t, "s3", "rm",
			fmt.Sprintf("s3://%s/%s", bucket, key),
			"--endpoint-url", endpoint)
		require.Nil(t, err, "aws s3 rm %s: %s", key, stderr)
	}
	awsExists := func(key string) bool {
		_, _, err := awsRun(nil, "s3", "ls",
			fmt.Sprintf("s3://%s/%s", bucket, key),
			"--endpoint-url", endpoint)
		return err == nil
	}
	awsList := func(t *test.SystemTest, prefix string) []string {
		stdout, stderr, err := awsRun(t, "s3", "ls",
			fmt.Sprintf("s3://%s/%s", bucket, prefix),
			"--endpoint-url", endpoint, "--recursive")
		require.Nil(t, err, "aws s3 ls: %s", stderr)
		var keys []string
		for _, line := range strings.Split(stdout, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				keys = append(keys, parts[3])
			}
		}
		return keys
	}

	// mc helpers (reuses package-level mcCmd / mcPath)
	mcPut := func(t *test.SystemTest, key string, data []byte) {
		f := writeTempFile(t, data)
		_, stderr, err := mcCmd("cp", f, mcBucket+"/"+key)
		require.Nil(t, err, "mc cp PUT %s: %s", key, stderr)
	}
	mcGet := func(t *test.SystemTest, key string) []byte {
		dst := filepath.Join(t.TempDir(), "mc-dl")
		_, stderr, err := mcCmd("cp", mcBucket+"/"+key, dst)
		require.Nil(t, err, "mc cp GET %s: %s", key, stderr)
		b, e := os.ReadFile(dst)
		require.Nil(t, e)
		return b
	}
	mcDelete := func(t *test.SystemTest, key string) {
		_, stderr, err := mcCmd("rm", mcBucket+"/"+key)
		require.Nil(t, err, "mc rm %s: %s", key, stderr)
	}
	mcExists := func(key string) bool {
		_, _, err := mcCmd("stat", mcBucket+"/"+key)
		return err == nil
	}
	mcList := func(t *test.SystemTest, prefix string) []string {
		stdout, stderr, err := mcCmd("ls", "--recursive", mcBucket+"/"+prefix)
		require.Nil(t, err, "mc ls: %s", stderr)
		var keys []string
		for _, line := range strings.Split(stdout, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// "2024-01-01 00:00:00 UTC  123 path/to/key"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				keys = append(keys, parts[len(parts)-1])
			}
		}
		return keys
	}

	// rclone helpers
	rcloneEnv := append(os.Environ(),
		"RCLONE_S3_PROVIDER=Other",
		"RCLONE_S3_ENDPOINT="+endpoint,
		"RCLONE_S3_ACCESS_KEY_ID="+accessKey,
		"RCLONE_S3_SECRET_ACCESS_KEY="+secretKey,
		"RCLONE_S3_PATH_STYLE=true",
	)
	rcloneRun := func(t *test.SystemTest, args ...string) (string, string, error) {
		cmd := exec.Command(rcloneBin, args...)
		cmd.Env = rcloneEnv
		var so, se strings.Builder
		cmd.Stdout, cmd.Stderr = &so, &se
		err := cmd.Run()
		return strings.TrimSpace(so.String()), strings.TrimSpace(se.String()), err
	}
	rcloneRemote := func(key string) string {
		return fmt.Sprintf(":s3:%s/%s", bucket, key)
	}
	rclonePut := func(t *test.SystemTest, key string, data []byte) {
		f := writeTempFile(t, data)
		_, stderr, err := rcloneRun(t, "copyto", f, rcloneRemote(key))
		require.Nil(t, err, "rclone copyto PUT %s: %s", key, stderr)
	}
	rcloneGet := func(t *test.SystemTest, key string) []byte {
		dst := filepath.Join(t.TempDir(), "rclone-dl")
		_, stderr, err := rcloneRun(t, "copyto", rcloneRemote(key), dst)
		require.Nil(t, err, "rclone copyto GET %s: %s", key, stderr)
		b, e := os.ReadFile(dst)
		require.Nil(t, e)
		return b
	}
	rcloneDelete := func(t *test.SystemTest, key string) {
		_, stderr, err := rcloneRun(t, "deletefile", rcloneRemote(key))
		require.Nil(t, err, "rclone deletefile %s: %s", key, stderr)
	}
	rcloneExists := func(key string) bool {
		out, _, _ := rcloneRun(nil, "lsf", rcloneRemote(key))
		return len(strings.TrimSpace(out)) > 0
	}
	rcloneList := func(t *test.SystemTest, prefix string) []string {
		stdout, _, _ := rcloneRun(t, "lsf", "--recursive", rcloneRemote(prefix))
		var keys []string
		for _, line := range strings.Split(stdout, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				keys = append(keys, prefix+line)
			}
		}
		return keys
	}

	uniqueKey := func(tag string) string {
		return fmt.Sprintf("interop/%s/%d", tag, time.Now().UnixNano())
	}

	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests(
		"AwsCLI write mc read rclone read",
		"mc write AwsCLI read rclone read",
		"rclone write AwsCLI read mc read",
	)

	// Create the interop bucket once; tear down after all subtests.
	_, stderr, err := mcCmd("mb", "--ignore-existing", mcBucket)
	require.Nil(testSetup, err, "mc mb interop bucket: %s", stderr)
	testSetup.Cleanup(func() { mcCmd("rb", "--force", mcBucket) }) //nolint

	// ── Cross-client PUT/GET visibility ───────────────────────────────────

	type client struct {
		name string
		put  func(t *test.SystemTest, key string, data []byte)
		get  func(t *test.SystemTest, key string) []byte
	}
	clients := []client{
		{"AwsCLI", awsPut, awsGet},
		{"mc", mcPut, mcGet},
		{"rclone", rclonePut, rcloneGet},
	}

	for _, w := range clients {
		for _, r1 := range clients {
			for _, r2 := range clients {
				if r1.name == w.name || r2.name == w.name || r1.name == r2.name {
					continue
				}
				w, r1, r2 := w, r1, r2
				name := fmt.Sprintf("%s write %s read %s read", w.name, r1.name, r2.name)
				t.RunSequentially(name, func(t *test.SystemTest) {
					payload := []byte("interop-payload written by " + w.name)
					key := uniqueKey(w.name + "-" + r1.name + "-" + r2.name)
					testSetup.Cleanup(func() { awsDelete(nil, key) })

					w.put(t, key, payload)
					require.Equal(t, payload, r1.get(t, key),
						"%s wrote; %s should read identical bytes", w.name, r1.name)
					require.Equal(t, payload, r2.get(t, key),
						"%s wrote; %s should read identical bytes", w.name, r2.name)
				})
			}
		}
	}

	// ── LIST agreement ────────────────────────────────────────────────────

	t.RunSequentially("All clients agree on listed objects", func(t *test.SystemTest) {
		prefix := fmt.Sprintf("interop/list/%d/", time.Now().UnixNano())
		keys := map[string][]byte{
			prefix + "aws-obj":    []byte("by-aws"),
			prefix + "mc-obj":     []byte("by-mc"),
			prefix + "rclone-obj": []byte("by-rclone"),
		}
		testSetup.Cleanup(func() {
			for k := range keys {
				awsDelete(nil, k)
			}
		})
		awsPut(t, prefix+"aws-obj", keys[prefix+"aws-obj"])
		mcPut(t, prefix+"mc-obj", keys[prefix+"mc-obj"])
		rclonePut(t, prefix+"rclone-obj", keys[prefix+"rclone-obj"])

		want := []string{prefix + "aws-obj", prefix + "mc-obj", prefix + "rclone-obj"}
		sort.Strings(want)

		awsKeys := awsList(t, prefix)
		sort.Strings(awsKeys)
		require.Equal(t, want, awsKeys, "AWS CLI list disagrees")

		mcKeys := mcList(t, prefix)
		sort.Strings(mcKeys)
		require.Equal(t, want, mcKeys, "mc list disagrees")

		rcloneKeys := rcloneList(t, prefix)
		sort.Strings(rcloneKeys)
		require.Equal(t, want, rcloneKeys, "rclone list disagrees")
	})

	// ── DELETE propagation ────────────────────────────────────────────────

	for _, deleter := range []struct {
		name string
		del  func(*test.SystemTest, string)
	}{
		{"AwsCLI", awsDelete},
		{"mc", mcDelete},
		{"rclone", rcloneDelete},
	} {
		deleter := deleter
		t.RunSequentially(fmt.Sprintf("%s delete seen by other two clients", deleter.name),
			func(t *test.SystemTest) {
				key := uniqueKey("del-" + deleter.name)
				awsPut(t, key, []byte("will-be-deleted"))
				require.True(t, awsExists(key), "object must exist before delete")

				deleter.del(t, key)

				require.False(t, awsExists(key),
					"aws: object still visible after %s deleted it", deleter.name)
				require.False(t, mcExists(key),
					"mc: object still visible after %s deleted it", deleter.name)
				require.False(t, rcloneExists(key),
					"rclone: object still visible after %s deleted it", deleter.name)
			})
	}

	// ── Overwrite propagation ─────────────────────────────────────────────

	t.RunSequentially("Successive overwrites visible to all clients", func(t *test.SystemTest) {
		key := uniqueKey("overwrite")
		testSetup.Cleanup(func() { awsDelete(nil, key) })

		for _, w := range clients {
			w := w
			content := []byte("version-by-" + w.name)
			w.put(t, key, content)
			for _, r := range clients {
				require.Equal(t, content, r.get(t, key),
					"%s should see version written by %s", r.name, w.name)
			}
		}
	})
}

// writeTempFile writes data to a temp file and returns its path.
func writeTempFile(t interface {
	TempDir() string
	Helper()
}, data []byte) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "payload")
	if err := os.WriteFile(f, data, 0o600); err != nil {
		panic(err)
	}
	return f
}
