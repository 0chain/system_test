package zs3servertests

import (
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverMixedWarpTests(testSetup *testing.T) {
	config := cliutils.ReadFile(testSetup)
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Warp Mixed Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		// Check if mc is available - it's required for this test
		if _, err := os.Stat("../mc"); os.IsNotExist(err) {
			testSetup.Fatalf("mc is not installed at ../mc, which is required for this test")
		}

		// Set up mc alias for the S3 server
		aliasCommand := "../mc alias set warp-test http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
		output, err := cliutils.RunCommand(t, aliasCommand, 1, time.Minute*2)
		if err != nil {
			testSetup.Fatalf("Failed to set mc alias: %v\nOutput: %s", err, output)
		}

		// Create the bucket that warp expects (warp-benchmark-bucket)
		bucketCommand := "../mc mb warp-test/warp-benchmark-bucket"
		output, err = cliutils.RunCommand(t, bucketCommand, 1, time.Minute*2)
		if err != nil {
			// Check if the error is because the bucket already exists (which is fine)
			outputStr := strings.Join(output, "\n")
			if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "BucketAlreadyExists") {
				testSetup.Fatalf("Failed to create bucket: %v\nOutput: %s", err, outputStr)
			}
			// Bucket already exists, which is fine - continue
			t.Logf("Bucket already exists, continuing...")
		} else {
			t.Logf("Bucket created successfully")
		}

		commandGenerated := "../warp mixed --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --objects=" + "22" + " --duration=" + "30s" + "  --obj.size=" + "256B"
		output, err = cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
		if err != nil {
			testSetup.Fatalf("Error running warp mixed: %v\nOutput: %s", err, output)
		}
		output_string := strings.Join(output, "\n")
		err = cliutils.AppendToFile("warp-mixed_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})
}
