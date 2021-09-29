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
	t.Run("parallel", func(t *testing.T) {
		t.Run("Wallet Creation should succeed when 0 < threshold <= num-signers", func(t *testing.T) {

			numSigners, threshold := 3, 2

			output, err := createMultiSigWallet(t, configPath, numSigners, threshold)
			require.Nil(t, err, "error occurred creating multisig wallet", strings.Join(output, "\n"))

			//FIXME: POSSIBLE BUG - blank wallet being created despite it not being used in the multisig create process
			require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
			require.Equal(t, "No wallet in path  ./config/TestMultisigWallet-parallel-Wallet_Creation_should_succeed_when_0_LESS_THAN_threshold_LESS_THAN=_num-signers_wallet.json found. Creating wallet...", output[0])
			require.Equal(t, "ZCN wallet created!!", output[1])
			require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			require.Equal(t, "Read pool created successfully", output[3])

			// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
			require.Equal(t, fmt.Sprintf("registering %d wallets", numSigners+1), output[4])
			require.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
		})

		t.Run("Wallet Creation should succeed when threshold is equal to num-signers", func(t *testing.T) {

			numSigners, threshold := 3, 3

			output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

			require.Nil(t, err, "error occurred creating multisig wallet", strings.Join(output, "\n"))
			require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
			require.Equal(t, "No wallet in path  ./config/TestMultisigWallet-parallel-Wallet_Creation_should_succeed_when_threshold_is_equal_to_num-signers_wallet.json found. Creating wallet...", output[0])
			require.Equal(t, "ZCN wallet created!!", output[1])
			require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			require.Equal(t, "Read pool created successfully", output[3])

			// Total registered wallets = numsigners + 1 (additional wallet for multi-sig)
			require.Equal(t, fmt.Sprintf("registering %d wallets", numSigners+1), output[4])
			require.Equal(t, "Creating and testing a multisig wallet is successful!", output[len(output)-1])
		})

		t.Run("Wallet Creation should fail when threshold is 0", func(t *testing.T) {

			numSigners, threshold := 3, 0

			output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

			require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
			require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
			require.Equal(t, "ZCN wallet created!!", output[1])
			require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			require.Equal(t, "Read pool created successfully", output[3])

			//FIXME: BUG - panic: runtime error: index out of range [0] with length 0
			require.Equal(t, "panic: runtime error: index out of range [0] with length 0", output[4])
		})

		t.Run("Wallet Creation should fail when threshold is -1", func(t *testing.T) {

			numSigners, threshold := 3, -1

			output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

			require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
			require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
			require.Equal(t, "ZCN wallet created!!", output[1])
			require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			require.Equal(t, "Read pool created successfully", output[3])

			//FIXME: BUG - panic: runtime error: makeslice: len out of range
			require.Equal(t, "panic: runtime error: makeslice: len out of range", output[4])
		})

		t.Run("Wallet Creation should fail when signers is < 2", func(t *testing.T) {

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

			numSigners, threshold := 3, 4

			output, err := createMultiSigWallet(t, configPath, numSigners, threshold)

			require.NotNil(t, err, "expected error to occur creating a multisig wallet", strings.Join(output, "\n"))
			require.True(t, len(output) > 4, "Output was less than number of assertions", strings.Join(output, "\n"))
			require.Equal(t, "No wallet in path  ./config/TestMultisigWallet"+
				"-parallel-Wallet_Creation_should_fail_when_threshold_is_greater_than_"+
				"num-signers_wallet.json found. Creating wallet...", output[0])
			require.Equal(t, "ZCN wallet created!!", output[1])
			require.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			require.Equal(t, "Read pool created successfully", output[3])

			require.Equal(t, fmt.Sprintf("Error: given threshold (%d) is too high. "+
				"Threshold has to be less than or equal to numsigners (%d)", threshold,
				numSigners), output[4])
		})

		t.Run("Wallet Creation should fail when args not set", func(t *testing.T) {

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
	})
}

func createMultiSigWallet(t *testing.T, cliConfigFilename string, numSigners, threshold int) ([]string, error) {
	return cliutils.RunCommand(fmt.Sprintf(
		"./zwallet createmswallet --numsigners %d --threshold %d --silent --wallet %s --configDir ./config --config %s",
		numSigners, threshold,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename))
}
