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

func TestShutDownBlobber(t *testing.T) {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	blobbers := []climodel.BlobberInfo{}
	output, err = listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)
	err = json.Unmarshal([]byte(output[0]), &blobbers)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

	// Pick a random blobber to shutdown
	blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

	t.Run("Shutting down blobber by blobber's delegate wallet should work", func(t *testing.T) {
		output, err := shutdownBlobberForWallet(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}), blobberOwnerWallet)
		require.Nil(t, err)
		require.Len(t, output, 1)
		require.Equal(t, "shut down blobber", output[0])

		blobbers := getBlobbersList(t)
		var blobberNew climodel.BlobberInfo
		for _, blobberNew := range blobbers {
			if blobberNew.Id == blobber.Id {
				break
			}
		}
		require.Equal(t, true, blobberNew.IsShutDown)
	})

	t.Run("shutted down blobber should not be listed", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		blobbers := getBlobbersList(t)
		require.NotContains(t, blobbers, blobber)
	})

	t.Run("Should not be able to use shutted down blobber for new allocations", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Should throw error as one of the 6 blobbers is shutdown
		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"size":   1024,
			"tokens": 1,
			"data":   5,
			"parity": 1,
		}))
		require.NotNil(t, err, "expected error but got none", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error creating allocation: failed_get_allocation_blobbers: failed to get blobbers for allocation: not enough blobbers to honor the allocation", output[0])
	})
}

func shutdownBlobber(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	return shutdownBlobberForWallet(t, cliConfigFilename, params, escapedTestName(t))
}

func shutdownBlobberForWallet(t *testing.T, cliConfigFilename, params, wallet string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox shut-down-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}
