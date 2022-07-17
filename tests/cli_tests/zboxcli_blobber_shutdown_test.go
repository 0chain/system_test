package cli_tests

import (
	"fmt"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestShutDownBlobber(t *testing.T) {
	t.Run("shutting down blobber by blobber owner should work", func(t *testing.T) {
		output, err := shutdownBlobberForWallet(t, configPath, "", "wallets/blobber02_owner_dev3")
		require.Nil(t, err)
		require.Len(t, output, 1)
		require.Equal(t, "shut down blobber", output[0])
	})
}

func shutdownBlobber(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return shutdownBlobberForWallet(t, cliConfigFilename, params, escapedTestName(t))
}

func shutdownBlobberForWallet(t *testing.T, cliConfigFilename, params, wallet string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox shut-down-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}
