package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestKillSharder(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	_ = initialiseTest(t, escapedTestName(t)+"_TARGET", true)

	sharderUrl := getSharderUrl(t)
	// wait for a sharder to be registered
	var sharderIds []string
	for {
		sharderIds = getSortedSharderIds(t, sharderUrl)
		if len(sharderIds) > 0 {
			break
		}
		cliutil.Wait(t, time.Second)
	}

	t.RunSequentiallyWithTimeout("kill sharder by non-smartcontract owner should fail", 1000*time.Second, func(t *test.SystemTest) {
		var sharderIds []string
		sharderIds, _ = waitForNSharder(t, sharderUrl, 1)
		require.True(t, len(sharderIds) > 0, "no sharders found, should be impossible")

		output, err := killSharder(t, escapedTestName(t), configPath, createParams(map[string]interface{}{
			"id": sharderIds[0],
		}), true)
		require.Error(t, err, "kill sharder by non-smartcontract owner should fail")
		require.Len(t, output, 1)
		require.True(t, strings.Contains(output[0], "unauthorized access - only the owner can access"), "")
	})

	t.RunSequentiallyWithTimeout("Killed sharder does not receive rewards", 1000*time.Second, func(t *test.SystemTest) {
		var startSharders climodel.NodeList
		_, startSharders = waitForNSharder(t, sharderUrl, 2)
		var sharderToKill string
		for i := range startSharders.Nodes {
			if !startSharders.Nodes[i].IsKilled {
				sharderToKill = startSharders.Nodes[i].ID
				break
			}
		}
		require.True(t, len(sharderToKill) > 0, "no un-killed sharders found")

		output, err := killSharder(t, scOwnerWallet, configPath, createParams(map[string]interface{}{
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
