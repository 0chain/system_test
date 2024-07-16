package zs3servertests

import (
	"os"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestMain(m *testing.M) {
	config := cliutils.ReadFile(nil)
	_, _ = cliutils.RunMinioServer(config.AccessKey, config.SecretKey)
	exitRun := m.Run()
	os.Exit(exitRun)
}