package zs3servertests

import (
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverPutWarpTests(testSetup *testing.T) {
	config := cliutils.ReadFile(testSetup)
	t := test.NewSystemTest(testSetup)

	// Check if mc is available - it's required for this test
	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Fatalf("mc is not installed at ../mc, which is required for this test")
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

	// Test alias connectivity by trying to list buckets (this verifies the alias works)
	testAliasOutput, testAliasErr := cliutils.RunCommand(t, "../mc ls warp-test", 1, time.Minute*2)
	if testAliasErr != nil {
		testAliasStr := strings.Join(testAliasOutput, "\n")
		// If listing fails, the alias might not be working - try to recreate it
		t.Logf("Alias connectivity test failed (error: %s), retrying alias setup...", testAliasStr)
		_, aliasErr := cliutils.RunCommand(t, aliasCommand, 1, time.Minute*2)
		if aliasErr != nil {
			testSetup.Fatalf("Failed to recreate mc alias: %v", aliasErr)
		}
		// Test alias again after recreation
		testAliasOutput, testAliasErr = cliutils.RunCommand(t, "../mc ls warp-test", 1, time.Minute*2)
		if testAliasErr != nil {
			testAliasStr = strings.Join(testAliasOutput, "\n")
			testSetup.Fatalf("Alias connectivity test failed after retry: %v\nOutput: %s", testAliasErr, testAliasStr)
		}
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
		} else {
			// For any other error, try one more time after a brief wait
			t.Logf("Bucket creation failed (error: %s), retrying after brief wait...", outputStr)
			time.Sleep(2 * time.Second)
			output, err = cliutils.RunCommand(t, bucketCommand, 1, time.Minute*2)
			if err != nil {
				outputStr = strings.Join(output, "\n")
				if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "BucketAlreadyExists") {
					testSetup.Fatalf("Failed to create bucket after retry: %v\nOutput: %s", err, outputStr)
				}
			} else {
				t.Logf("Bucket created successfully after retry")
			}
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
	output_string = strings.Split(output_string, "----------------------------------------")[1]

	output_string = "Condition 2 : Put  \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}
