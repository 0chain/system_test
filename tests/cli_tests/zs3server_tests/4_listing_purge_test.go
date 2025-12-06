package zs3servertests

import (
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverListTests(testSetup *testing.T) {
	config := cliutils.ReadFile(testSetup)
	t := test.NewSystemTest(testSetup)

	// Check if mc is available - it's required for this test
	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Fatalf("mc is not installed at ../mc, which is required for this test")
	}

	// Set up mc alias for the S3 server (only need to do this once)
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

	t.RunSequentiallyWithTimeout("Warp List Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		commandGenerated := "../warp get --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --duration 30s" + " --obj.size " + config.ObjectSize + " --objects " + config.ObjectCount
		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
		if err != nil {
			testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
		}
		output_string := strings.Join(output, "\n")
		output_string = strings.Split(output_string, "----------------------------------------")[1]

		output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

		output_string = "Condition 1: Get objects: 1 \n--------\n" + output_string
		err = cliutils.AppendToFile("warp-list_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})

	t.RunSequentiallyWithTimeout("Warp List Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		commandGenerated := "../warp get --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --objects " + config.ObjectCount + " --concurrent " + config.Concurrent + " --duration 30s" + " --obj.size " + config.ObjectSize
		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

		if err != nil {
			testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
		}
		output_string := strings.Join(output, "\n")
		output_string = strings.Split(output_string, "----------------------------------------")[1]

		output_string = "Condition 1: Get objects: 100 concurrent 50::  \n--------\n" + output_string

		err = cliutils.AppendToFile("warp-list_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})
}
