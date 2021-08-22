package zboxcli_tests

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZboxCliRegisterWallet(t *testing.T) {

	// Run the "./zbox register" command from the latest build
	// fetched dynamically using the getLatestBuild.sh script
	err := exec.Command("sh", "-c", "../getLatestBuild.sh").Run()

	if err != nil {
		cmd := exec.Command("./zbox register")
		output, err := cmd.Output()

		if err != nil {
			assert.Equal(t, "Wallet registered\n", string(output))
		}
	}
}
