package cli_tests

import (
	"regexp"
	"testing"

	cli_utils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/assert"
)

func TestCreateAllocation(t *testing.T) {

	t.Run("Create allocations with small expiry", func(t *testing.T) {
		t.Parallel()

		output, err := cli_utils.RegisterWallet(configPath)
		assert.NotNil(t, err)

		if len(output) == 1 {
			assert.Equal(t, "Wallet registered", output[0])
		} else {
			// This is true for the first round only since the wallet is created here
			assert.Equal(t, "ZCN wallet created!!", output[1])
			assert.Equal(t, "Creating related read pool for storage smart-contract...", output[2])
			assert.Equal(t, "Read pool created successfully", output[3])
		}

		_, err = cli_utils.ExecuteFaucet(configPath)
		assert.NotNil(t, err)

		_, err = cli_utils.GetBalance(configPath)
		assert.NotNil(t, err)

		var lock float64 = 0.5
		options := map[string]string{"expire": "300s"}
		output, err = cli_utils.NewAllocation(configPath, lock, options)
		assert.NotNil(t, err)

		assert.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0])

	})

	t.Run("Create allocations with largest possible expiry", func(t *testing.T) {
		t.Parallel()

	})
}
