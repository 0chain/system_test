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

	server, host, accessKey, secretKey, concurrent, objectSize, _ := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp put --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s" + " --obj.size "+objectSize
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
