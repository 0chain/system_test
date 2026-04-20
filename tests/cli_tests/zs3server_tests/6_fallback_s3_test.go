package zs3servertests

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// TestZs3serverFallbackS3 verifies zs3server's fallback-to-upstream-S3 feature.
//
// When fallback_s3_enabled=true in zs3server.json and a GetObject request hits
// a key that does NOT exist in the zus allocation, zs3server should:
//   1) fetch the object from the configured upstream S3 endpoint,
//   2) stream bytes to the caller,
//   3) asynchronously cache-back the object into zus so subsequent GETs are local.
//
// Prerequisites:
//   - hosts.yaml with S3 server (zs3server) config
//   - mc binary at ../mc
//   - minio binary at ZS3_MINIO_BIN (default: /root/Code/zs3server/minio)
//   - ZS3_CONFIG_JSON points at zs3server.json (default: /root/Code/zs3server/zcnconfig/zs3server.json)
//   - ZS3_ALLOC_ID, ZS3_CONFIG_DIR set so we can restart zs3server
//
// This test restarts zs3server twice (enable fallback, restore) — it must run
// in isolation (no other tests or benchmarks active against zs3server:9100).
func TestZs3serverFallbackS3(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)

	// Verify zs3server reachable
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable at %s:%s: %v", config.Server, config.HostPort, err)
	}
	conn.Close()

	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Skip("mc binary not available at ../mc")
	}

	minioBin := getenvDefault("ZS3_MINIO_BIN", "/root/Code/zs3server/minio")
	if _, err := os.Stat(minioBin); os.IsNotExist(err) {
		testSetup.Skipf("minio binary not found at %s (set ZS3_MINIO_BIN)", minioBin)
	}

	zs3CfgJSON := getenvDefault("ZS3_CONFIG_JSON", "/root/Code/zs3server/zcnconfig/zs3server.json")
	zs3CfgDir := getenvDefault("ZS3_CONFIG_DIR", "/root/Code/zs3server/zcnconfig")
	zs3AllocID := getenvDefault("ZS3_ALLOC_ID", "")
	if zs3AllocID == "" {
		testSetup.Skip("ZS3_ALLOC_ID env var not set")
	}

	// Upstream minio config
	const (
		upstreamAddr   = ":9200"
		upstreamAK     = "upstreamak"
		upstreamSK     = "upstreamsk1234"
		upstreamBucket = "ups-fallback"
	)
	upstreamDir := fmt.Sprintf("/tmp/upstream_minio_%d", time.Now().Unix())

	t := test.NewSystemTest(testSetup)

	// 1) Start upstream minio
	t.Logf("Starting upstream minio at %s (data dir %s)", upstreamAddr, upstreamDir)
	if err := os.MkdirAll(upstreamDir, 0755); err != nil {
		testSetup.Fatalf("mkdir upstreamDir: %v", err)
	}
	upstreamLog, _ := os.Create(upstreamDir + "/upstream.log")
	upstreamBin := getenvDefault("ZS3_UPSTREAM_MINIO_BIN", minioBin)
	fmt.Println("DEBUG: upstreamBin=", upstreamBin)
	upstreamCmd := exec.Command(upstreamBin, "server", upstreamDir,
		"--address", upstreamAddr, "--console-address", ":9211")
	upstreamCmd.Env = append(os.Environ(),
		"MINIO_ROOT_USER="+upstreamAK,
		"MINIO_ROOT_PASSWORD="+upstreamSK)
	upstreamCmd.Stdout = upstreamLog
	upstreamCmd.Stderr = upstreamLog
	upstreamCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := upstreamCmd.Start(); err != nil {
		testSetup.Fatalf("start upstream minio: %v", err)
	}

	// Teardown: stop upstream, restore config, restart zs3server
	defer func() {
		if upstreamCmd.Process != nil {
			syscall.Kill(-upstreamCmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
			upstreamCmd.Wait()                                     //nolint:errcheck
		}
		_ = os.RemoveAll(upstreamDir)
		// Restore zs3server config + restart
		if bak, err := os.ReadFile(zs3CfgJSON + ".fallback_test_bak"); err == nil {
			_ = os.WriteFile(zs3CfgJSON, bak, 0644)
			_ = os.Remove(zs3CfgJSON + ".fallback_test_bak")
			restartZs3Server(t, minioBin, zs3CfgDir, zs3AllocID, config.HostPort)
		}
	}()

	waitForHTTP(t, "http://localhost"+upstreamAddr+"/minio/health/ready", 20*time.Second)

	// 2) Upload test objects to upstream
	mcAlias := "upsfb"
	_, _ = cliutils.RunCommand(t, fmt.Sprintf("../mc alias set %s http://localhost%s %s %s --api S3v4",
		mcAlias, upstreamAddr, upstreamAK, upstreamSK), 1, time.Second*30)
	defer cliutils.RunCommand(t, fmt.Sprintf("../mc alias rm %s", mcAlias), 1, time.Second*5) //nolint:errcheck

	_, err = cliutils.RunCommand(t, fmt.Sprintf("../mc mb %s/%s --ignore-existing", mcAlias, upstreamBucket), 1, time.Minute)
	if err != nil {
		testSetup.Fatalf("create upstream bucket: %v", err)
	}

	// Upload a small and a 100MB file to upstream, under the SAME bucket name
	// zs3server will use for its allocation (so upstreamBucketFor returns it).
	// We push to a bucket matching the allocation's bucket name convention.
	// The feature uses upstreamBucketFor(localBucket) → in absence of mapping rule,
	// we use the same bucket name on both sides.
	localBucket := upstreamBucket

	smallFile := writeTempFileHelper(testSetup, fmt.Sprintf("fallback-hello-%d", time.Now().UnixNano()))
	defer os.Remove(smallFile)
	smallMD5 := fileMD5Helper(testSetup, smallFile)
	_, err = cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s %s/%s/hello.txt", smallFile, mcAlias, upstreamBucket), 1, time.Minute)
	if err != nil {
		testSetup.Fatalf("upload small to upstream: %v", err)
	}

	bigFile := writeTempRandomHelper(testSetup, 100*1024*1024) // 100 MB
	defer os.Remove(bigFile)
	bigMD5 := fileMD5Helper(testSetup, bigFile)
	_, err = cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s %s/%s/big.bin", bigFile, mcAlias, upstreamBucket), 1, time.Minute*5)
	if err != nil {
		testSetup.Fatalf("upload big to upstream: %v", err)
	}

	// 3) Backup + update zs3server config
	origCfg, err := os.ReadFile(zs3CfgJSON)
	if err != nil {
		testSetup.Fatalf("read zs3server.json: %v", err)
	}
	if err := os.WriteFile(zs3CfgJSON+".fallback_test_bak", origCfg, 0644); err != nil {
		testSetup.Fatalf("backup zs3server.json: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(origCfg, &cfg); err != nil {
		testSetup.Fatalf("parse zs3server.json: %v", err)
	}
	cfg["fallback_s3_enabled"] = true
	cfg["fallback_s3_endpoint"] = "http://localhost" + upstreamAddr
	cfg["fallback_s3_region"] = "us-east-1"
	cfg["fallback_s3_use_ssl"] = false
	cfg["fallback_s3_access_key"] = upstreamAK
	cfg["fallback_s3_secret_key"] = upstreamSK
	newCfg, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(zs3CfgJSON, newCfg, 0644); err != nil {
		testSetup.Fatalf("write zs3server.json: %v", err)
	}
	restartZs3Server(t, minioBin, zs3CfgDir, zs3AllocID, config.HostPort)

	// mc alias for zs3server
	zs3Alias := "zs3fb"
	_, _ = cliutils.RunCommand(t, fmt.Sprintf("../mc alias set %s http://%s:%s %s %s --api S3v4",
		zs3Alias, config.Server, config.HostPort, config.AccessKey, config.SecretKey), 1, time.Minute)
	defer cliutils.RunCommand(t, fmt.Sprintf("../mc alias rm %s", zs3Alias), 1, time.Second*5) //nolint:errcheck

	// Ensure bucket exists in zus (test objects go under a bucket the allocation already has)
	_, _ = cliutils.RunCommand(t, fmt.Sprintf("../mc mb %s/%s --ignore-existing", zs3Alias, localBucket), 1, time.Minute)
	// Ensure test keys don't exist in zus yet (so they trigger fallback)
	_, _ = cliutils.RunCommand(t, fmt.Sprintf("../mc rm %s/%s/hello.txt", zs3Alias, localBucket), 1, time.Second*30)
	_, _ = cliutils.RunCommand(t, fmt.Sprintf("../mc rm %s/%s/big.bin", zs3Alias, localBucket), 1, time.Second*30)
	time.Sleep(2 * time.Second)

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_fallback_fetch_small_file", time.Minute*2, func(t *test.SystemTest) {
		outPath := fmt.Sprintf("/tmp/fb-small-%d", time.Now().UnixNano())
		defer os.Remove(outPath)
		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s/%s/hello.txt %s", zs3Alias, localBucket, outPath), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("zs3 GET hello.txt (expected to fallback): %v", err)
		}
		got := fileMD5Helper(testSetup, outPath)
		if got != smallMD5 {
			testSetup.Fatalf("small file checksum mismatch: want=%s got=%s", smallMD5, got)
		}
		t.Logf("PASS: small file served via fallback, md5=%s", got)
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_fallback_fetch_100MB_checksum", time.Minute*8, func(t *test.SystemTest) {
		outPath := fmt.Sprintf("/tmp/fb-big-%d", time.Now().UnixNano())
		defer os.Remove(outPath)
		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s/%s/big.bin %s", zs3Alias, localBucket, outPath), 1, time.Minute*5)
		if err != nil {
			testSetup.Fatalf("zs3 GET big.bin (expected to fallback): %v", err)
		}
		got := fileMD5Helper(testSetup, outPath)
		if got != bigMD5 {
			testSetup.Fatalf("100MB file checksum mismatch: want=%s got=%s", bigMD5, got)
		}
		t.Logf("PASS: 100MB file served via fallback, md5=%s", got)
	})

	// -----------------------------------------------------------
	t.RunSequentiallyWithTimeout("S3_fallback_cache_back_then_direct", time.Minute*3, func(t *test.SystemTest) {
		// Give cache-back 20 s to finish uploading small file to zus
		time.Sleep(20 * time.Second)

		// Stop upstream so next GET can only succeed if zus holds it
		if upstreamCmd.Process != nil {
			syscall.Kill(-upstreamCmd.Process.Pid, syscall.SIGTERM) //nolint:errcheck
			upstreamCmd.Wait()                                      //nolint:errcheck
			upstreamCmd.Process = nil
		}
		time.Sleep(2 * time.Second)

		outPath := fmt.Sprintf("/tmp/fb-direct-%d", time.Now().UnixNano())
		defer os.Remove(outPath)
		_, err := cliutils.RunCommand(t, fmt.Sprintf("../mc cp %s/%s/hello.txt %s", zs3Alias, localBucket, outPath), 1, time.Minute)
		if err != nil {
			testSetup.Fatalf("zs3 GET hello.txt after upstream down (expected to be cached in zus): %v", err)
		}
		got := fileMD5Helper(testSetup, outPath)
		if got != smallMD5 {
			testSetup.Fatalf("post-cache-back md5 differs: want=%s got=%s", smallMD5, got)
		}
		t.Logf("PASS: cache-back verified — zus serves directly after upstream down")
	})
}

// --- helpers ---

func getenvDefault(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}

func waitForHTTP(t *test.SystemTest, url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, _ := cliutils.RunCommand(t, fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' %s", url), 1, time.Second*3)
		if strings.Contains(strings.Join(out, ""), "200") {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
}

func restartZs3Server(t *test.SystemTest, minioBin, zs3CfgDir, allocID, hostPort string) {
	// Kill existing
	_, _ = cliutils.RunCommand(t, "pkill -9 -f minio.gateway.zcn", 1, time.Second*5)
	time.Sleep(3 * time.Second)

	// Relaunch
	logPath := filepath.Join(os.TempDir(), fmt.Sprintf("zs3server_restart_%d.log", time.Now().Unix()))
	cmd := exec.Command(minioBin, "gateway", "zcn",
		"--address", ":"+hostPort,
		"--console-address", ":9101",
		"--configDir", zs3CfgDir,
		"--allocationId", allocID)
	cmd.Env = append(os.Environ(),
		"MINIO_ROOT_USER=rootroot",
		"MINIO_ROOT_PASSWORD=rootroot",
		"MINIO_BROWSER=OFF")
	logF, _ := os.Create(logPath)
	cmd.Stdout = logF
	cmd.Stderr = logF
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("restart zs3server: %v", err)
	}
	// Detach — let it run independently
	go cmd.Wait() //nolint:errcheck

	waitForHTTP(t, fmt.Sprintf("http://localhost:%s/minio/health/ready", hostPort), 30*time.Second)
}
