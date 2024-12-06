package zs3servertests

import (
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
