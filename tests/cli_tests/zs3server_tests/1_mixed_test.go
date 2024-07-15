package zs3servertests

import (
	"log"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverMixedWarpTests(testSetup *testing.T) {
	log.Println("Running Warp Mixed Benchmark...")
	t := test.NewSystemTest(testSetup)
	config := cliutils.ReadFile(testSetup)
	_, _ = cliutils.RunMinioServer(config.AccessKey, config.SecretKey)
	time.Sleep(1 * time.Second)
	t.Logf("Minio server Started")
}
