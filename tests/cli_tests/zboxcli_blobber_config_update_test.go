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

func TestBlobberConfigUpdate(t *testing.T) {
	if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
	}

	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1, strings.Join(output, "\n"))

	var blobberList []climodel.BlobberDetails
	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(blobberList), 0, "blobber list is empty")

	intialBlobberDetails := blobberList[0]

	t.Cleanup(func() {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "capacity": intialBlobberDetails.Capacity}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "max_offer_duration": intialBlobberDetails.Terms.Max_offer_duration}))
		require.Nil(t, err, strings.Join(output, "\n"))

		max_stake, err := intialBlobberDetails.StakePoolSettings.MaxStake.Int64()
		require.Nil(t, err)
		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "max_stake": intToZCN(max_stake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		min_stake, err := intialBlobberDetails.StakePoolSettings.MinStake.Int64()
		require.Nil(t, err)
		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "min_stake": intToZCN(min_stake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "min_lock_demand": intialBlobberDetails.Terms.Min_lock_demand}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "num_delegates": intialBlobberDetails.StakePoolSettings.MaxNumDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "service_charge": intialBlobberDetails.StakePoolSettings.ServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "read_price": intToZCN(intialBlobberDetails.Terms.Read_price)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "write_price": intToZCN(intialBlobberDetails.Terms.Write_price)}))
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Run("update blobber capacity should work", func(t *testing.T) {
		// register wallet for normal user
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newCapacity := 99 * GB

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "capacity": newCapacity}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, int64(newCapacity), finalBlobberDetails.Capacity)
	})

	t.Run("update blobber max offer duration should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newMaxOfferDuration := 2668400 * time.Second

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "max_offer_duration": newMaxOfferDuration}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newMaxOfferDuration, finalBlobberDetails.Terms.Max_offer_duration)
	})

	t.Run("update blobber max stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldMaxStake, err := intialBlobberDetails.StakePoolSettings.MaxStake.Int64()
		require.Nil(t, err)
		newMaxStake := intToZCN(oldMaxStake) - 1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "max_stake": newMaxStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		max_stake, err := finalBlobberDetails.StakePoolSettings.MaxStake.Int64()
		require.Nil(t, err)
		require.Equal(t, float64(newMaxStake), intToZCN(max_stake))
	})

	t.Run("update blobber min stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldMinStake, err := intialBlobberDetails.StakePoolSettings.MinStake.Int64()
		require.Nil(t, err)
		newMinStake := intToZCN(oldMinStake) + 1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "min_stake": newMinStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		min_stake, err := finalBlobberDetails.StakePoolSettings.MinStake.Int64()
		require.Nil(t, err)
		require.Equal(t, float64(newMinStake), intToZCN(min_stake))
	})

	t.Run("update blobber min lock demand should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newMinLockDemand := 0.2

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "min_lock_demand": newMinLockDemand}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		cliutils.Wait(t, 3*time.Second)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newMinLockDemand, finalBlobberDetails.Terms.Min_lock_demand)
	})

	t.Run("update blobber number of delegates should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newNumberOfDelegates := 15

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "num_delegates": newNumberOfDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newNumberOfDelegates, finalBlobberDetails.StakePoolSettings.MaxNumDelegates)
	})

	t.Run("update blobber service charge should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newServiceCharge := 0.1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "service_charge": newServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newServiceCharge, finalBlobberDetails.StakePoolSettings.ServiceCharge)
	})

	t.Run("update no params should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME: since we are not updating any params, the output should not say `updated successfully`
		require.Equal(t, "blobber settings updated successfully", output[0])
	})

	t.Run("update without blobber ID should fail", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 25)
		require.Equal(t, "Error: required flag(s) \"blobber_id\" not set", output[0])
	})

	t.Run("update with invalid blobber ID should fail", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": "invalid-blobber-id"}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "internal_error: missing blobber: invalid-blobber-id", output[1])
	})

	t.Run("update with invalid blobber wallet/owner should fail", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID}), escapedTestName(t), configPath), 1, time.Second*2)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_blobber_settings_failed: access denied, allowed for delegate_wallet owner only",
			output[0], strings.Join(output, "\n"))
	})

	t.Run("update blobber read price should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldReadPrice := intialBlobberDetails.Terms.Read_price
		newReadPrice := intToZCN(oldReadPrice) + 1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "read_price": newReadPrice}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newReadPrice, intToZCN(finalBlobberDetails.Terms.Read_price))
	})

	t.Run("update blobber write price should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		oldWritePrice := intialBlobberDetails.Terms.Write_price
		newWritePrice := intToZCN(oldWritePrice) + 1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "write_price": newWritePrice}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberDetails.Terms.Write_price))
	})

	t.Run("update all params at once should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		newWritePrice := intToZCN(intialBlobberDetails.Terms.Write_price) + 1
		newServiceCharge := intialBlobberDetails.StakePoolSettings.ServiceCharge + 0.1
		newReadPrice := intToZCN(intialBlobberDetails.Terms.Read_price) + 1
		newNumberOfDelegates := intialBlobberDetails.StakePoolSettings.MaxNumDelegates + 1
		newMaxOfferDuration := intialBlobberDetails.Terms.Max_offer_duration + 1*time.Second
		newCapacity := intialBlobberDetails.Capacity + 1
		newMinLockDemand := intialBlobberDetails.Terms.Min_lock_demand + 0.01
		min_stake, err := intialBlobberDetails.StakePoolSettings.MinStake.Int64()
		require.Nil(t, err)
		newMinStake := intToZCN(min_stake) + 1
		max_stake, err := intialBlobberDetails.StakePoolSettings.MaxStake.Int64()
		require.Nil(t, err)
		newMaxStake := intToZCN(max_stake) - 1

		output, err = updateBlobberDetails(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberDetails.ID, "write_price": newWritePrice, "service_charge": newServiceCharge, "read_price": newReadPrice, "num_delegates": newNumberOfDelegates, "max_offer_duration": newMaxOfferDuration, "capacity": newCapacity, "min_lock_demand": newMinLockDemand, "min_stake": newMinStake, "max_stake": newMaxStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberDetails(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberDetails.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberDetails climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberDetails)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberDetails.Terms.Write_price))
		require.Equal(t, newServiceCharge, finalBlobberDetails.StakePoolSettings.ServiceCharge)
		require.Equal(t, newReadPrice, intToZCN(finalBlobberDetails.Terms.Read_price))
		require.Equal(t, newNumberOfDelegates, finalBlobberDetails.StakePoolSettings.MaxNumDelegates)
		require.Equal(t, newMaxOfferDuration, finalBlobberDetails.Terms.Max_offer_duration)
		require.Equal(t, newCapacity, finalBlobberDetails.Capacity)
		require.Equal(t, newMinLockDemand, finalBlobberDetails.Terms.Min_lock_demand)
		min_stake, err = finalBlobberDetails.StakePoolSettings.MinStake.Int64()
		require.Nil(t, err)
		require.Equal(t, newMinStake, intToZCN(min_stake))
		max_stake, err = finalBlobberDetails.StakePoolSettings.MaxStake.Int64()
		require.Nil(t, err)
		require.Equal(t, newMaxStake, intToZCN(max_stake))
	})
}

func getBlobberDetails(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateBlobberDetails(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
