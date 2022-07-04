package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestValidatorConfigUpdate(t *testing.T) {
	// blobber delegate wallet and validator delegate wallet are same
	if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
	}

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	output, err = listValidators(t, configPath, createParams(map[string]interface{}{"json": ""}))
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1, strings.Join(output, "\n"))

	var validatorList []climodel.Validator
	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(validatorList), 0, "validator list is empty")

	intialValidatorInfo := validatorList[0]

	t.Cleanup(func() {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "max_stake": intToZCN(intialValidatorInfo.MaxStake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "min_stake": intToZCN(intialValidatorInfo.MinStake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "num_delegates": intialValidatorInfo.NumDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "service_charge": intialValidatorInfo.ServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Run("update blobber max stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldMaxStake := intialValidatorInfo.MaxStake
		newMaxStake := intToZCN(oldMaxStake) - 1

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "max_stake": newMaxStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getValidatorInfo(t, configPath, createParams(map[string]interface{}{"json": "", "validator_id": intialValidatorInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalValidatorInfo climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &finalValidatorInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, float64(newMaxStake), intToZCN(finalValidatorInfo.MaxStake))
	})

	t.Run("update blobber min stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldMinStake := intialValidatorInfo.MinStake
		newMinStake := intToZCN(oldMinStake) + 1

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "min_stake": newMinStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getValidatorInfo(t, configPath, createParams(map[string]interface{}{"json": "", "validator_id": intialValidatorInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalValidatorInfo climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &finalValidatorInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, float64(newMinStake), intToZCN(finalValidatorInfo.MinStake))
	})

	t.Run("update validator number of delegates should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newNumberOfDelegates := 15

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "num_delegates": newNumberOfDelegates}))
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

	t.Run("update blobber service charge should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newServiceCharge := 0.1

		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{"validator_id": intialValidatorInfo.ID, "service_charge": newServiceCharge}))
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

func listValidators(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting validator list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-validators %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func getValidatorInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting validator info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox validator-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateValidatorInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating validator info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox validator-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
