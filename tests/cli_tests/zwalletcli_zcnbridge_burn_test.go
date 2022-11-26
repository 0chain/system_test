package cli_tests

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/tokenomics"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBridgeBurn(t *testing.T) {
	t.Parallel()

	t.Run("Burning WZCN tokens on balance, should work", func(t *testing.T) {
		t.Parallel()

		output, err := burnEth(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")
	})

	t.Run("Get WZCN burn ticket, should work", func(t *testing.T) {
		t.Parallel()

		output, err := burnEth(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err, output)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")

		reg := regexp.MustCompile("0x[a-f0-9]{64}")
		allHashes := reg.FindAllString(strings.Join(output, " "), -1)
		ethTxHash := allHashes[len(allHashes)-1]

		output, err = getWrappedZcnBurnTicket(t, ethTxHash, true)
		require.Nil(t, err)

		ethereumTxAddress := strings.TrimSpace(strings.Split(output[len(output)-2], ":")[1])
		require.True(t, reg.MatchString(ethereumTxAddress))

		amount := strings.TrimSpace(strings.Split(output[len(output)-3], ":")[1])
		var amountInt int
		amountInt, err = strconv.Atoi(amount)
		require.Nil(t, err)
		require.Equal(t, 1, amountInt)

		nonce := strings.TrimSpace(strings.Split(output[len(output)-4], ":")[1])
		var nonceInt int
		nonceInt, err = strconv.Atoi(nonce)
		require.Nil(t, err)
		require.Equal(t, 1, nonceInt)
	})

	t.Run("Burning ZCN tokens without ZCN tokens on balance, shouldn't work", func(t *testing.T) {
		t.Parallel()

		output, err := burnZcn(t, "1", bridgeClientConfigFile, true)
		require.NotNil(t, err)
		require.Greater(t, len(output), 0)
	})

	t.Run("Burning ZCN tokens with available ZCN tokens on balance, should work", func(t *testing.T) {
		t.Parallel()

		output, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
	})

	t.Run("Get ZCN burn ticket, should work", func(t *testing.T) {
		t.Parallel()

		output, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", bridgeClientConfigFile, true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)

		reg := regexp.MustCompile("[a-f0-9]{64}")
		allHashes := reg.FindAllString(strings.Join(output, " "), -1)
		zcnTxHash := allHashes[len(allHashes)-1]

		output, err = getZcnBurnTicket(t, zcnTxHash, true)
		require.Nil(t, err)

		amount := strings.TrimSpace(strings.Split(output[len(output)-2], ":")[1])
		var amountFloat float64
		amountFloat, err = strconv.ParseFloat(amount, 32)
		require.Nil(t, err)
		require.Equal(t, int64(1), tokenomics.ZcnToInt(amountFloat))

		nonce := strings.TrimSpace(strings.Split(output[len(output)-3], ":")[1])
		var nonceInt int
		nonceInt, err = strconv.Atoi(nonce)
		require.Nil(t, err)
		require.Equal(t, 1, nonceInt)
	})
}

//nolint
func burnZcn(t *testing.T, amount, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning ZCN tokens that will be minted for WZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-zcn --token %s --path %s --bridge_config %s --wallet %s --configDir ./config --config %s",
		amount,
		configDir,
		bridgeClientConfigFile,
		escapedTestName(t)+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func burnEth(t *testing.T, amount, bridgeClientConfigFile string, retry bool) ([]string, error) {
	t.Logf("Burning WZCN tokens that will be minted for ZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-eth --amount %s --path %s --bridge_config %s",
		amount,
		configDir,
		bridgeClientConfigFile,
	)
	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func getZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get ZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-zcn-burn --hash %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

//nolint
func getWrappedZcnBurnTicket(t *testing.T, hash string, retry bool) ([]string, error) {
	t.Logf("Get WZCN burn ticket...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-get-wzcn-burn --hash %s --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		hash,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
