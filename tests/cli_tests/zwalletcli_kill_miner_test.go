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

func TestKillMiner(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

	sharderUrl := getSharderUrl(t)
	t.RunSequentiallyWithTimeout("Killed miner does not receive rewards", 200*time.Second, func(t *test.SystemTest) {
		minerIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minerIds) > 0, "no miners found")

		startMiners := getNodes(t, minerIds, sharderUrl)
		var minerToKill string
		for i := range startMiners.Nodes {
			if !startMiners.Nodes[i].IsKilled {
				minerToKill = startMiners.Nodes[i].ID
				break
			}
		}
		require.True(t, len(minerToKill) > 0, "no un-killed miners found")

		output, err := killMiner(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
			"id": minerToKill,
		}), true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		cliutil.Wait(t, time.Second)

		minerAfterKill := getMinersDetail(t, minerToKill)
		require.True(t, minerAfterKill.IsKilled, "miner should be killed")

		// ------------------------------------
		cliutil.Wait(t, 10*time.Second)
		// ------------------------------------

		minerAfterRewardTest := getMinersDetail(t, minerToKill)
		require.Equalf(t, minerAfterKill.TotalReward, minerAfterRewardTest.TotalReward,
			"killed miner %s should not receive any more rewards", minerToKill)
	})

	t.RunSequentially("kill miner by non-smartcontract owner should fail", func(t *test.SystemTest) {

		minerIds := getSortedMinerIds(t, sharderUrl)
		require.True(t, len(minerIds) > 0, "no miners found")

		output, err := killMiner(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"id": minerIds[0],
		}), true)
		require.Error(t, err, "kill miner by non-smartcontract owner should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "unauthorized access - only the owner can access"), "")
	})

}

func killMiner(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill miner...")
	cmd := fmt.Sprintf("./zwallet mn-kill %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
