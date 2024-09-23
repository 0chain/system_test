package zs3servertests

import (
	"log"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverMixedWarpTests(testSetup *testing.T) {
	log.Println("Running Warp Mixed Benchmark...")
	t := test.NewSystemTest(testSetup)
	_, err := cliutils.RunMinioServer("rootroot", "rootroot")
	if err!=nil{
		t.Logf(err.Error())
	}
	config := cliutils.ReadFile(testSetup)
	t.Logf("Minio server Started")

	t.RunSequentiallyWithTimeout("Warp Mixed Benchmark", 40*time.Minute, func(t *test.SystemTest) {
		commandGenerated := "../warp mixed --host=" + config.Server + ":" + config.HostPort + " --access-key=" + config.AccessKey + " --secret-key=" + config.SecretKey + " --objects=" + "22" + " --duration=" + "30s" + "  --obj.size=" + "256B"
		log.Println("Command Generated: ", commandGenerated)

		output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
		if err != nil {
			testSetup.Fatalf("Error running warp mixed: %v\nOutput: %s", err, output)
		}
		log.Println("Warp mixed Output:\n", output)
		output_string := strings.Join(output, "\n")
		err = cliutils.AppendToFile("warp-mixed_output.txt", output_string)

		if err != nil {
			testSetup.Fatalf("Error appending to file: %v\n", err)
		}
	})
}
