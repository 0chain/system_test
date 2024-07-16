package zs3servertests

import (
	"log"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverPutWarpTests(testSetup *testing.T) {
	log.Println("Running Warp Put Benchmark...")
	t := test.NewSystemTest(testSetup)
	config := cliutils.ReadFile(testSetup)

	commandGenerated := "./warp put --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + "  --concurrent " + config.Concurrent + " --duration 30s" + " --obj.size " + config.ObjectSize
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

	if err != nil {
		testSetup.Fatalf("Error running warp put: %v\nOutput: %s", err, output)
	}
	log.Println("Warp Put Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]

	output_string = "Condition 2 : Put  \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}
