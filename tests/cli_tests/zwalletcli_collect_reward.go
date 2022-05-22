package cli_tests

import (
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

// executeFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeCollectReward(t *testing.T, poolID, providerID, providerType, wallet, cliConfigFilename string) ([]string, error) {
	command := "./zwallet collect-reward --provider_id " + providerID
	if len(poolID) > 0 {
		command += " --pool_id " + poolID
	}
	if len(providerType) > 0 {
		command += " --provider_type " + providerType
	}
	command += " --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	t.Logf("Executing collect-reward ...")
	return cliutils.RunCommand(t, command, 3, time.Second*5)
}
