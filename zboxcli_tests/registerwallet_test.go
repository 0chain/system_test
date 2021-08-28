package zboxcli_tests

import (
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldReturnWalletRegisteredOnRunningRegisterCommand(t *testing.T) {
	t.Run("Checks terminal output", func(t *testing.T) {
		// Run the "./zbox register" command from the latest build
		// fetched dynamically using the getLatestBuild.sh script
		//err := exec.Command("sh", "-c", "../getLatestZboxBuild.sh").Run()

		// if err != nil {
		// 	t.Errorf(err.Error())
		// }

		cmd := exec.Command("./zbox.exe", "register")
		output, err := cmd.Output()

		if err != nil {
			t.Errorf(err.Error())
		}

		assert.Equal(t, "Wallet registered\n", string(output))
	})

	t.Run("Execute faucet against Wallet", func(t *testing.T) {
		// Get faucet tokens on zwallet by running the fuacet method on
		// latest release of zwalletcli
		// err := exec.Command("sh", "-c", "../getLatestZwalletBuild.sh").Run()

		// if err != nil {
		// 	t.Errorf(err.Error())
		// }

		cmd := exec.Command("./zwallet.exe", "faucet", "--methodName", "pour", "--input", "\"{Pay day}\"")
		output, err := cmd.Output()

		if err != nil {
			t.Errorf(err.Error())
		}

		assert.Regexp(t, regexp.MustCompile("Execute faucet smart contract success with txn :  [a-zA-Z0-9]"), string(output))
	})
}
