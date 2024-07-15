package zs3servertests

import (
	"log"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverWarpTests(testSetup *testing.T) {
	log.Println("Running Warp List Benchmark...")
	t := test.NewSystemTest(testSetup)
	config := cliutils.ReadFile(testSetup)
	_, _ = cliutils.RunMinioServer(config.AccessKey, config.SecretKey)


	t.RunWithTimeout("Warp List Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		commandGenerated := ".../warp get --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --duration 30s" + " --obj.size " + config.ObjectSize
		log.Println("Command Generated: ", commandGenerated)
		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
		if err != nil {
			testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
		}
		log.Println("Warp List Output:\n", output)
		output_string := strings.Join(output, "\n")
		output_string = strings.Split(output_string, "----------------------------------------")[1]

		output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

		output_string = "Condition 1: Get objects: 1 \n--------\n" + output_string
		err = cliutils.AppendToFile("warp-list_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})
}

func TestZs3serverConcurrentListTests(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	config := cliutils.ReadFile(testSetup)
	_, _ = cliutils.RunMinioServer(config.AccessKey, config.SecretKey)


	t.RunWithTimeout("Warp List Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		commandGenerated := ".../warp get --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --objects " + config.ObjectCount + " --concurrent " + config.Concurrent + " --duration 30s" + " --obj.size " + config.ObjectSize
		log.Println("Command Generated: ", commandGenerated)

		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

		if err != nil {
			testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
		}
		log.Println("Warp List Output:\n", output)
		output_string := strings.Join(output, "\n")
		output_string = strings.Split(output_string, "----------------------------------------")[1]

		output_string = "Condition 1: Get objects: 100 concurrent 50::  \n--------\n" + output_string
		log.Println("APending to file with this stat ", output_string)

		err = cliutils.AppendToFile("warp-list_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})
}
