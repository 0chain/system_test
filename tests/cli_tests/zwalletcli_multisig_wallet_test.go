package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestMultisigWallet(t *testing.T) {
	t.Parallel()

	t.Run("Wallet Creation should fail when signers is < 2", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 1, 1

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])

		require.Equal(t, "Error: too few signers. Minimum numsigners required is 2", output[4])
	})

	t.Run("Wallet Creation should fail when threshold is greater than num-signers", func(t *testing.T) {
		t.Parallel()
		numSigners, threshold := 3, 4

		output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

		require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
		require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
		require.Equal(t, "No wallet in path  ./config/TestMultisigWallet"+
			"-Wallet_Creation_should_fail_when_threshold_is_greater_than_"+
			"num-signers_wallet.json found. Creating wallet...", output[0])
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])

		require.Equal(t, fmt.Sprintf("Error: given threshold (%d) is too high. "+
			"Threshold has to be less than or equal to numsigners (%d)", threshold,
			numSigners), output[4])
	})

	t.Run("Wallet Creation should fail when args not set", func(t *testing.T) {
		t.Parallel()

		output, err := cliutils.RunCommand(fmt.Sprintf("./zwallet createmswallet "+
			"--silent --wallet %s --configDir ./config --config %s", escapedTestName(t)+
			"_wallet.json", configPath))

		require.NotNil(t, err, "expected command to fail", strings.Join(output, "\n"))
		require.True(t, len(output) > 4, "Output was less than number of "+
			"assertions", strings.Join(output, "\n"))
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])

		require.Equal(t, "Error: numsigners flag is missing", output[4])
	})

	t.Run("Wallet Creation should fail when threshold not set", func(t *testing.T) {
		t.Parallel()

		output, err := cliutils.RunCommand(fmt.Sprintf("./zwallet createmswallet "+
			"--numsigners %d --silent --wallet %s --configDir ./config "+
			"--config %s", 3, escapedTestName(t)+"_wallet.json", configPath))

		require.NotNil(t, err, "expected command to fail", strings.Join(output, "\n"))
		require.True(t, len(output) > 4, "Output was less than number "+
			"of assertions", strings.Join(output, "\n"))
		require.Equal(t, "ZCN wallet created!!", output[1])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
		require.Equal(t, "Read pool created successfully", output[3])

		require.Equal(t, "Error: threshold flag is missing", output[4])
	})
}

func createMultiSigWallet(t *testing.T, cliConfigFilename string, numSigners, threshold int) ([]string, error) {
	t.Logf("Creating multisig wallet...")
	return cliutils.RunCommand(fmt.Sprintf(
		"./zwallet createmswallet --numsigners %d --threshold %d --silent --wallet %s --configDir ./config --config %s",
		numSigners, threshold,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
}
