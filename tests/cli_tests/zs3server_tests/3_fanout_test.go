package zs3servertests

import (
	"log"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverFanoutTests(testSetup *testing.T) {
	log.Println("Running Warp Fanout Benchmark...")
	t := test.NewSystemTest(testSetup)

	server, host, accessKey, secretKey, concurrent := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp fanout --copies=50 --obj.size=512KiB --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s" + " --obj.size 1KiB"
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
