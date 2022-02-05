package cli_tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestLockConfig(t *testing.T) {
	t.Parallel()

	t.Run("get lock config should work", func(t *testing.T) {
		t.Parallel()

		output, err := getLockConfig(t, configPath, true)
		require.Nil(t, err, "get lock config failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1, "Expected output length to be at least 2", strings.Join(output, "\n"))
		require.Equal(t, "Configuration:", output[len(output)-2], strings.Join(output, "\n"))
		require.NotNil(t, output[len(output)-1], strings.Join(output, "\n"))

		var lockConfig climodel.LockConfig
		err = json.Unmarshal([]byte(output[len(output)-1]), &lockConfig)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[len(output)-1], err)

		require.NotEmpty(t, lockConfig.ID, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.SimpleGlobalNode, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.SimpleGlobalNode.MaxMint, strings.Join(output, "\n"))
		require.NotNil(t, lockConfig.SimpleGlobalNode.TotalMinted, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.SimpleGlobalNode.MinLock, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.SimpleGlobalNode.Apr, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.SimpleGlobalNode.OwnerId, strings.Join(output, "\n"))
		require.NotEmpty(t, lockConfig.MinLockPeriod, strings.Join(output, "\n"))
	})
}

func getLockConfig(t *testing.T, cliConfigFilename string, retry bool) ([]string, error) {
	t.Log("getting lock config...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet lockconfig --silent --configDir ./config --config %s", cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet lockconfig --silent --configDir ./config --config %s", cliConfigFilename))
	}
}
