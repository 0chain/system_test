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
		// Check if mc is available
		if _, err := os.Stat("../mc"); os.IsNotExist(err) {
			t.Logf("Warning: ../mc is not installed, skipping bucket creation")
		} else {
			// Set up mc alias for the S3 server
			aliasCommand := "../mc alias set warp-test http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
			_, err := cliutils.RunCommand(t, aliasCommand, 1, time.Minute*2)
			if err != nil {
				t.Logf("Warning: Failed to set mc alias: %v", err)
			}

			// Create the bucket that warp expects (warp-benchmark-bucket)
			bucketCommand := "../mc mb warp-test/warp-benchmark-bucket"
			_, err = cliutils.RunCommand(t, bucketCommand, 1, time.Minute*2)
			if err != nil {
				// Bucket might already exist, which is fine
				t.Logf("Note: Bucket creation result: %v (bucket may already exist)", err)
			}
		}

		commandGenerated := "../warp mixed --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --objects=" + "22" + " --duration=" + "30s" + "  --obj.size=" + "256B"
		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
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
