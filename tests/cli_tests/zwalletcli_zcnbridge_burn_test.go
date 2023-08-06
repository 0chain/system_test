package cli_tests

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/tokenomics"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestBridgeBurn(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Burning WZCN tokens on balance, should work")

	t.Parallel()

	t.RunWithTimeout("Burning WZCN tokens on balance, should work", time.Minute*10, func(t *test.SystemTest) {
		output, err := burnEth(t, "10000000000", true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")
	})

	t.RunWithTimeout("Get WZCN burn ticket, should work", time.Minute*10, func(t *test.SystemTest) {
		output, err := burnEth(t, "10000000000", true)
		require.Nil(t, err, output)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")

		ethTxHash := getTransactionHash(output, true)

		output, err = getWrappedZcnBurnTicket(t, ethTxHash, true)
		require.Nil(t, err)

		ethereumTxAddress := strings.TrimSpace(strings.Split(output[len(output)-2], ":")[1])
		require.True(t, isEthereumAddress(ethereumTxAddress))

		amount := strings.TrimSpace(strings.Split(output[len(output)-3], ":")[1])
		var amountInt int
		amountInt, err = strconv.Atoi(amount)
		require.Nil(t, err)
		require.Equal(t, 10000000000, amountInt)

		nonce := strings.TrimSpace(strings.Split(output[len(output)-4], ":")[1])
		var nonceInt int
		nonceInt, err = strconv.Atoi(nonce)
		require.Nil(t, err)
		require.GreaterOrEqual(t, nonceInt, 0)
	})

	t.RunWithTimeout("Burning ZCN tokens without ZCN tokens on balance, shouldn't work", time.Minute*10, func(t *test.SystemTest) {
		output, err := burnZcn(t, "1", false)
		require.NotNil(t, err)
		require.Greater(t, len(output), 0)
		require.NotContains(t, output[len(output)-1], "Transaction completed successfully:")
	})

	t.Run("Burning ZCN tokens with available ZCN tokens on balance, should work", func(t *test.SystemTest) {
		output, err := executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Transaction completed successfully:")
	})

	t.RunWithTimeout("Get ZCN burn ticket, should work", time.Minute*10, func(t *test.SystemTest) {
		output, err := executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = burnZcn(t, "1", true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Transaction completed successfully:")

		zcnTxHash := getTransactionHash(output, false)

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
		require.GreaterOrEqual(t, nonceInt, 0)
	})
}

func getTransactionHash(src []string, prefix bool) string {
	var reg *regexp.Regexp
	if prefix {
		reg = regexp.MustCompile("0x[a-f0-9]{64}")
	} else {
		reg = regexp.MustCompile("[a-f0-9]{64}")
	}
	allHashes := reg.FindAllString(strings.Join(src, " "), -1)
	return allHashes[len(allHashes)-1]
}

func isEthereumAddress(src string) bool {
	reg := regexp.MustCompile("0x[a-f0-9]{64}")
	return reg.MatchString(src)
}

func burnZcn(t *test.SystemTest, amount string, retry bool) ([]string, error) {
	t.Logf("Burning ZCN tokens that will be minted for WZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-zcn --token %s --path %s --wallet %s --configDir ./config --config %s",
		amount,
		configDir,
		escapedTestName(t)+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func burnEth(t *test.SystemTest, amount string, retry bool) ([]string, error) {
	t.Logf("Burning WZCN tokens that will be minted for ZCN tokens...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-burn-eth --amount %s --path %s --retries 200",
		amount,
		configDir,
	)

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	if retry {
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getZcnBurnTicket(t *test.SystemTest, hash string, retry bool) ([]string, error) {
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
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getWrappedZcnBurnTicket(t *test.SystemTest, hash string, retry bool) ([]string, error) {
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
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
