package zs3servertests

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverPutWarpTests(testSetup *testing.T) {
	// Check if hosts.yaml config exists, skip if not available
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml config not found, ZS3 server tests not configured, skipping test")
	}

	config := cliutils.ReadFile(testSetup)

	// Check if ZS3 server is reachable, skip if not available
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("ZS3 server not available at %s:%s, skipping test: %v", config.Server, config.HostPort, err)
	}
	conn.Close()

	// Check if warp binary is available
	if _, err := os.Stat("../warp"); os.IsNotExist(err) {
		testSetup.Skip("warp binary not available at ../warp, skipping test")
	}

	t := test.NewSystemTest(testSetup)

	// Check if mc is available - it's required for this test
	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Skip("mc is not installed at ../mc, skipping test")
	}

	// Remove alias if it exists (ignore errors)
	_, _ = cliutils.RunCommand(t, "../mc alias rm warp-test", 1, time.Second*5)
	
	// Set up mc alias for the S3 server
	aliasCommand := "../mc alias set warp-test http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
	output, err := cliutils.RunCommand(t, aliasCommand, 1, time.Minute*2)
	if err != nil {
		testSetup.Fatalf("Failed to set mc alias: %v\nOutput: %s", err, output)
	}

	// Verify alias was set correctly by listing it
	aliasListOutput, aliasListErr := cliutils.RunCommand(t, "../mc alias list", 1, time.Minute*2)
	if aliasListErr != nil {
		testSetup.Fatalf("Failed to list mc aliases: %v\nOutput: %s", aliasListErr, aliasListOutput)
	}
	aliasListStr := strings.Join(aliasListOutput, "\n")
	if !strings.Contains(aliasListStr, "warp-test") {
		testSetup.Fatalf("Alias 'warp-test' was not found in alias list. Output: %s", aliasListStr)
	}

	// Create the bucket that warp expects (warp-benchmark-bucket)
	bucketCommand := "../mc mb warp-test/warp-benchmark-bucket"
	output, err = cliutils.RunCommand(t, bucketCommand, 1, time.Minute*2)
	if err != nil {
		// Check if the error is because the bucket already exists (which is fine)
		outputStr := strings.Join(output, "\n")
		if strings.Contains(outputStr, "already exists") || strings.Contains(outputStr, "BucketAlreadyExists") {
			// Bucket already exists, which is fine - continue
			t.Logf("Bucket already exists, continuing...")
		} else if strings.Contains(outputStr, "does not exist") || strings.Contains(outputStr, "Unable to make bucket") {
			// Alias might not be properly configured, try to recreate it
			t.Logf("Alias issue detected (error: %s), retrying alias setup...", outputStr)
			_, aliasErr := cliutils.RunCommand(t, aliasCommand, 1, time.Minute*2)
			if aliasErr != nil {
				testSetup.Fatalf("Failed to recreate mc alias: %v", aliasErr)
			}
			// Verify alias again after recreation
			aliasListOutput, aliasListErr = cliutils.RunCommand(t, "../mc alias list", 1, time.Minute*2)
			if aliasListErr == nil {
				aliasListStr = strings.Join(aliasListOutput, "\n")
				if !strings.Contains(aliasListStr, "warp-test") {
					testSetup.Fatalf("Alias 'warp-test' still not found after retry. Output: %s", aliasListStr)
				}
			}
			// Retry bucket creation
			output, err = cliutils.RunCommand(t, bucketCommand, 1, time.Minute*2)
			if err != nil {
				outputStr = strings.Join(output, "\n")
				if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "BucketAlreadyExists") {
					testSetup.Fatalf("Failed to create bucket after retry: %v\nOutput: %s", err, outputStr)
				}
			} else {
				t.Logf("Bucket created successfully after retry")
			}
		} else {
			testSetup.Fatalf("Failed to create bucket: %v\nOutput: %s", err, outputStr)
		}
	} else {
		t.Logf("Bucket created successfully")
	}

	commandGenerated := "../warp put --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + "  --concurrent " + config.Concurrent + " --duration 30s" + " --obj.size " + config.ObjectSize
	output, err = cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

	if err != nil {
		testSetup.Fatalf("Error running warp put: %v\nOutput: %s", err, output)
	}
	output_string := strings.Join(output, "\n")

	// warp output uses Unicode box-drawing separator (U+2500), not ASCII hyphens.
	// warp put may produce a single report section with no separator at all.
	unicodeSeparator := strings.Repeat("\u2500", 34)
	parts := strings.Split(output_string, unicodeSeparator)
	if len(parts) > 1 {
		output_string = strings.TrimSpace(parts[1])
	}
	// Strip cleanup section if present
	if idx := strings.Index(output_string, "warp: Starting cleanup"); idx >= 0 {
		output_string = strings.TrimSpace(output_string[:idx])
	}

	output_string = "Condition 2 : Put  \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}
