package zboxcli_tests

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldReturnWalletRegisteredOnRunningRegisterCommand(t *testing.T) {
	t.Run("Checks terminal output", func(t *testing.T) {
		// Run the "./zbox register" command from the latest build
		// fetched dynamically using the getLatestBuild.sh script
		err := exec.Command("sh", "-c", "../getLatestBuild.sh").Run()

		if err != nil {
			t.Errorf(err.Error())
		}

		cmd := exec.Command("./zbox", "register")
		output, err := cmd.Output()

		if err != nil {
			t.Errorf(err.Error())
		}

		assert.Equal(t, "Wallet registered\n", string(output))
	})
}
