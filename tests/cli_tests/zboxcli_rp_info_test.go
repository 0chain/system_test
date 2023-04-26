package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestReadPoolInfo(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.RunWithTimeout("Read pool info testing with json parameter", 90*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 5 ZCN
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.9, balance)

		// Lock 1 token in read pool distributed amongst all blobbers
		lockAmount := 1.0
		readPoolParams := createParams(map[string]interface{}{
			"tokens": lockAmount,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		output, err = readPoolInfoWithParams(t, configPath, createParams(map[string]interface{}{
			"json": "",
		}))
		require.Nil(t, err, "Error fetching read pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		rpInfo := climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &rpInfo)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
		require.NotEmpty(t, rpInfo)
	})

	t.RunWithTimeout("Read pool info testing without json parameter", 90*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Wallet balance before lock should be 4.9 ZCN
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, 4.9, balance)

		// Lock 1 token in read pool distributed amongst all blobbers
		lockAmount := 1.0
		readPoolParams := createParams(map[string]interface{}{
			"tokens": lockAmount,
		})
		output, err = readPoolLock(t, configPath, readPoolParams, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		output, err = readPoolInfoWithParams(t, configPath, "")
		require.Nil(t, err, "Error fetching read pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Read pool Balance: 1.00\d ZCN \(\d*\.?\d+ USD\)$`), output[0])
	})
}
func readPoolInfoWithParams(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	cliutils.Wait(t, 30*time.Second) // TODO replace with poller
	t.Logf("Getting read pool info...")
	return cliutils.RunCommand(t, "./zbox rp-info "+params+" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}
