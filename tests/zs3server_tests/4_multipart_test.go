package zs3servertests

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverMultipartTests(testSetup *testing.T) {
	log.Println("Running Warp Multipart Benchmark...")
	timeout := time.Duration(200 * time.Minute)
	err := os.Setenv("GO_TEST_TIMEOUT", timeout.String())
	if err != nil {
		log.Printf("Error setting environment variable: %v", err)
	}

	t := test.NewSystemTest(testSetup)

	config := cliutils.ReadFile(testSetup)

	commandGenerated := "./warp multipart --parts=500 --part.size=512KiB --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + "  --concurrent " + config.Concurrent + " --duration 30s"
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
	if err != nil {
		testSetup.Fatalf("Error running warp multipart: %v\nOutput: %s", err, output)
	}
	log.Println("Warp Multipart Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]
	output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

	output_string = "Condition 1: Retention : objects: 1 \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}
