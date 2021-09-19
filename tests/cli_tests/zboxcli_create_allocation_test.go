package cli_tests

import (
	"regexp"
	"testing"

	cli_utils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/assert"
)

func TestCreateAllocation(t *testing.T) {
	// networkConfig, err := cli_utils.GetNetworkConfiguration(configPath)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// var miners []interface{} = (networkConfig["miners"]).([]interface{})

	// if len(miners) < 1 {
	// 	t.Fatal(err)
	// }

	type Test struct {
		lock    float64
		name    string
		options map[string]string
	}

	var successScenarioTests = []Test{
		{lock: 0.5, name: "Create allocation with smallest expiry (5m)", options: map[string]string{"expire": "5m", "size": "256000"}},
		{lock: 0.5, name: "Create allocation with smallest possible size (1024)", options: map[string]string{"expire": "1h", "size": "1024"}},
		{lock: 0.5, name: "Create allocation with parity 1", options: map[string]string{"expire": "1h", "size": "1024", "parity": "1"}},
		{lock: 0.5, name: "Create allocation with data shard 20", options: map[string]string{"expire": "1h", "size": "128000", "data": "20"}},
		{lock: 0.5, name: "Create allocation with read price range 0-0.03", options: map[string]string{"expire": "1h", "size": "128000", "read_price": "0-0.03"}},
		{lock: 0.5, name: "Create allocation with write price range 0-0.03", options: map[string]string{"expire": "1h", "size": "128000", "write_price": "0-0.03"}},
	}

	for _, tt := range successScenarioTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := setupWallet(t, configPath)
			assert.Nil(t, err)

			output, err := cli_utils.NewAllocation(configPath, tt.lock, tt.options)
			assert.Nil(t, err)

			assert.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0])
		})
	}

	t.Run("Create allocation with invalid expiry", func(t *testing.T) {
		t.Parallel()
		_, err := setupWallet(t, configPath)
		assert.Nil(t, err)

		var lock float64 = 0.5
		options := map[string]string{"expire": "-1", "size": "1024"}
		output, err := cli_utils.NewAllocation(configPath, lock, options)
		assert.NotNil(t, err)

		assert.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration -1", output[len(output)-1])
	})

	// t.Run("Create allocation with invalid expiry", func(t *testing.T) {
	// 	// t.Parallel()
	// 	_, err := setupWallet(t, configPath)
	// 	assert.Nil(t, err)

	// 	var lock float64 = 0.5
	// 	options := map[string]string{"expire": "-1", "size": "1024"}
	// 	output, err := cli_utils.NewAllocation(configPath, lock, options)
	// 	assert.NotNil(t, err)

	// 	assert.Equal(t, "invalid argument \"-1\" for \"--expire\" flag: time: missing unit in duration -1", output[len(output)-1])
	// })

	// var failureScenarioTests = []Test {
	// 	{lock: 0.5, name: "Create allocation with invalid parity bigger than", options: map[string]string{"expire": "1h", "size": "1024", "parity": strconv.Itoa(len(miners) + 1)}},
	// 	{lock: 0.5, name: "Create allocation with invalid parity bigger than", options: map[string]string{"expire": "1h", "size": "1024", "parity": strconv.Itoa(len(miners) + 1)}},
	// }

}

func setupWallet(t *testing.T, configPath string) ([]string, error) {
	output, err := cli_utils.RegisterWallet(configPath)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}

	_, err = cli_utils.ExecuteFaucet(configPath)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}
	_, err = cli_utils.GetBalance(configPath)
	if err != nil {
		cli_utils.Logger.Errorf(err.Error())
		return nil, err
	}

	return output, nil
}
