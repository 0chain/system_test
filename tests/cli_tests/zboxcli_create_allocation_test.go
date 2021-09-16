package cli_tests

import (
	"regexp"
	"testing"

	cli_utils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/assert"
)

func TestCreateAllocation(t *testing.T) {
	t.Run("Create allocation with small expiry", func(t *testing.T) {
		t.Parallel()

		output, err := cli_utils.RegisterWallet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.ExecuteFaucet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.GetBalance(configPath)
		assert.Nil(t, err)

		var lock float64 = 0.5
		options := map[string]string{"expire": "1h"}
		output, err = cli_utils.NewAllocation(configPath, lock, options)
		assert.Nil(t, err)

		assert.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0])

	})

	t.Run("Create allocation with invalid expiry", func(t *testing.T) {
		t.Parallel()
		output, err := cli_utils.RegisterWallet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.ExecuteFaucet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.GetBalance(configPath)
		assert.Nil(t, err)

		var lock float64 = 0.5
		options := map[string]string{"expire": "-1"}
		output, err = cli_utils.NewAllocation(configPath, lock, options)
		assert.NotNil(t, err)

		assert.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration -1", output[len(output)-1])
	})

	t.Run("Create allocation with smallest possible size", func(t *testing.T) {
		t.Parallel()
		output, err := cli_utils.RegisterWallet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.ExecuteFaucet(configPath)
		assert.Nil(t, err)

		_, err = cli_utils.GetBalance(configPath)
		assert.Nil(t, err)

		var lock float64 = 0.5
		options := map[string]string{"expire": "-1"}
		output, err = cli_utils.NewAllocation(configPath, lock, options)
		assert.NotNil(t, err)

		assert.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration -1", output[len(output)-1])
	})
}
