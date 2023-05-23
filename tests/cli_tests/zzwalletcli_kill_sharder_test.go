package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const (
	minShardersForKillSharderTest = 2
)

func TestKillSharder(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)

	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	sharderUrl := getSharderUrl(t)
	startSharders := getNodeSlice(t, "getSharderList", sharderUrl)
	if len(startSharders) < minShardersForKillSharderTest {
		t.Skipf("not enough sharders in blockchain, found %d need %d", len(startSharders), minShardersForKillSharderTest)
	}

	var sharderToKill string
	for i := range startSharders {
		if !startSharders[i].IsKilled {
			sharderToKill = startSharders[i].ID
			break
		}
	}
	if sharderToKill == "" {
		t.Skip("all sharders in the blockchain have been killed")
	}

	t.RunSequentially("kill sharder by non-smartcontract owner should fail", func(t *test.SystemTest) {
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = killSharder(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"id": sharderToKill,
		}), true)
		require.Error(t, err, "kill sharder by non-smartcontract owner should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "unauthorized access - only the owner can access"), "")
	})

	t.RunSequentially("Killed sharder does not receive rewards", func(t *test.SystemTest) {
		output, err = executeFaucetWithTokens(t, configPath, 10)
		require.NoError(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = killSharder(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
			"id": sharderToKill,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		cliutil.Wait(t, time.Second)

		sharderAfterKill := getMinersDetail(t, sharderToKill)
		require.True(t, sharderAfterKill.IsKilled, "sharder should be killed")

		// ------------------------------------
		cliutil.Wait(t, 10*time.Second)
		// ------------------------------------

		sharderAfterRewardTest := getMinersDetail(t, sharderToKill)
		require.Equalf(t, sharderAfterKill.TotalReward, sharderAfterRewardTest.TotalReward,
			"killed sharder %s should not receive any more rewards", sharderToKill)
	})
}

func killSharder(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill sharder...")
	cmd := fmt.Sprintf("./zwallet sh-kill %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
