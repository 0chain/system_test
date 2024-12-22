package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0TenderlyValidatorConfigUpdate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("update validator number of delegates should work")

	var intialValidatorInfo climodel.Validator
	t.TestSetup("Get validator config", func() {
		// blobber delegate wallet and validator delegate wallet are same
		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}

		createWallet(t)

		output, err := listValidators(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var validatorList []climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &validatorList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(validatorList), 0, "validator list is empty")

		intialValidatorInfo = validatorList[0]

		t.Cleanup(func() {
			createWallet(t)

			output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "num_delegates": intialValidatorInfo.NumDelegates}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "service_charge": intialValidatorInfo.ServiceCharge}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})
	})

	t.RunSequentially("update validator number of delegates should work", func(t *test.SystemTest) {
		createWallet(t)

		newNumberOfDelegates := 15

		output, err := updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "num_delegates": newNumberOfDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getValidatorInfo(t, configPath, createParams(map[string]interface{}{"json": "", "validator_id": intialValidatorInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalValidatorInfo climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &finalValidatorInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newNumberOfDelegates, finalValidatorInfo.NumDelegates)
	})

	t.RunSequentially("update validator service charge should work", func(t *test.SystemTest) {
		createWallet(t)

		newServiceCharge := 0.1

		output, err := updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "service_charge": newServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getValidatorInfo(t, configPath, createParams(map[string]interface{}{"json": "", "validator_id": intialValidatorInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalValidatorInfo climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &finalValidatorInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newServiceCharge, finalValidatorInfo.ServiceCharge)
	})
}

func listValidators(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting validator list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-validators %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func getValidatorInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting validator info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox validator-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateValidatorInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating validator info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox validator-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
