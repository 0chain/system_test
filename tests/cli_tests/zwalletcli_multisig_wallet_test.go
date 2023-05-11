package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestMultisigWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Skip("Multisig is broken and will be removed before mainnet - tests remain until this happens")

	t.SetSmokeTests("Wallet Creation should succeed when 0 < threshold <= num-signers")

	t.Parallel()

	t.Run("Wallet Creation should succeed when 0 < threshold <= num-signers", func(t *test.SystemTest) {
		numSigners, threshold := 3, 2

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, true)
		require.Nil(t, err, "error occurred creating multisig wallet", strings.Join(output, "\n"))

		require.True(t, len(output) > 1, "Output was less than number of assertions", strings.Join(output, "\n"))
		require.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should succeed when threshold is equal to num-signers", func(t *test.SystemTest) {
		numSigners, threshold := 3, 3

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, true)

		require.Nil(t, err, "error occurred creating multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 1, "Output was less than number of assertions", strings.Join(output, "\n"))
		require.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
	})

	t.Run("Wallet Creation should fail when threshold is 0", func(t *test.SystemTest) {
		numSigners, threshold := 3, 0

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, false)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "Output was less than number of assertions", strings.Join(output, "\n"))

		require.Contains(t, output, "Error: threshold should be bigger than 0")
	})

	t.Run("Wallet Creation should fail when threshold is -1", func(t *test.SystemTest) {
		numSigners, threshold := 3, -1

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, false)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "Output was less than number of assertions", strings.Join(output, "\n"))

		require.Contains(t, output, "Error: threshold should be bigger than 0")
	})

	t.Run("Wallet Creation should fail when signers is < 2", func(t *test.SystemTest) {
		numSigners, threshold := 1, 1

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, false)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "Output was less than number of assertions", strings.Join(output, "\n"))

		require.Contains(t, output, "Error: too few signers. Minimum numsigners required is 2")
	})

	t.Run("Wallet Creation should fail when threshold is greater than num-signers", func(t *test.SystemTest) {
		numSigners, threshold := 3, 4

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold, false)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "Output was less than number of assertions", strings.Join(output, "\n"))

		require.Contains(t, output, fmt.Sprintf("Error: given threshold (%d) is too high. "+
			"Threshold has to be less than or equal to numsigners (%d)", threshold,
			numSigners))
	})

	t.Run("Wallet Creation should fail when args not set", func(t *test.SystemTest) {
		output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet createmswallet "+
			"--silent --wallet %s --configDir ./config --config %s", escapedTestName(t)+
			"_wallet.json", configPath))

		require.NotNil(t, err, "expected command to fail", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "Output was less than number of "+
			"assertions", strings.Join(output, "\n"))

		require.Contains(t, output, "Error: numsigners flag is missing")
	})

	t.Run("Wallet Creation should fail when threshold not set", func(t *test.SystemTest) {
		output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet createmswallet "+
			"--numsigners %d --silent --wallet %s --configDir ./config "+
			"--config %s", 3, escapedTestName(t)+"_wallet.json", configPath))

		require.NotNil(t, err, "expected command to fail", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: threshold flag is missing", output[0])
	})
}

func createMultiSigWallet(t *test.SystemTest, cliConfigFilename string, numSigners, threshold int, retry bool) ([]string, error) {
	t.Logf("Creating multisig wallet...")
	cmd := fmt.Sprintf(
		"./zwallet createmswallet --numsigners %d --threshold %d --silent --wallet %s --configDir ./config --config %s",
		numSigners, threshold,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
