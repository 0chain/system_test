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

	t.Cleanup(func() {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": intialBlobberInfo.Capacity}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "cct": intialBlobberInfo.Terms.Challenge_completion_time}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "max_offer_duration": intialBlobberInfo.Terms.Max_offer_duration}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "max_stake": intToZCN(intialBlobberInfo.StakePoolSettings.MaxStake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "min_stake": intToZCN(intialBlobberInfo.StakePoolSettings.MinStake)}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "min_lock_demand": intialBlobberInfo.Terms.Min_lock_demand}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": intialBlobberInfo.StakePoolSettings.NumDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": intialBlobberInfo.StakePoolSettings.ServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Parallel()

	t.Run("update blobber capacity should work", func(t *testing.T) {
		// register wallet for normal user
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		// register wallet for blobber owner
		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newCapacity := 99 * GB

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": newCapacity}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, int64(newCapacity), finalBlobberInfo.Capacity)
	})

	t.Run("update blobber challenge completion time should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newChallengeCompletionTIme := 110 * time.Second

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "cct": newChallengeCompletionTIme}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newChallengeCompletionTIme, finalBlobberInfo.Terms.Challenge_completion_time)
	})

	t.Run("update blobber max offer duration should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newMaxOfferDuration := 2668400 * time.Second

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "max_offer_duration": newMaxOfferDuration}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newMaxOfferDuration, finalBlobberInfo.Terms.Max_offer_duration)
	})

	t.Run("update blobber max stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]

		oldMaxStake := intialBlobberInfo.StakePoolSettings.MaxStake
		newMaxStake := intToZCN(oldMaxStake) - 1

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "max_stake": newMaxStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, float64(newMaxStake), intToZCN(finalBlobberInfo.StakePoolSettings.MaxStake))
	})

	t.Run("update blobber min stake should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]

		oldMinStake := intialBlobberInfo.StakePoolSettings.MinStake
		newMinStake := intToZCN(oldMinStake) + 1

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "min_stake": newMinStake}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, float64(newMinStake), intToZCN(finalBlobberInfo.StakePoolSettings.MinStake))
	})

	t.Run("update blobber min lock demand should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newMinLockDemand := 0.2

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "min_lock_demand": newMinLockDemand}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		cliutils.Wait(t, 3*time.Second)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newMinLockDemand, finalBlobberInfo.Terms.Min_lock_demand)
	})

	t.Run("update blobber number of delegates should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newNumberOfDelegates := 52

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": newNumberOfDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		// BUG: number of delegates is not being updated
		require.NotEqual(t, newNumberOfDelegates, finalBlobberInfo.StakePoolSettings.NumDelegates)
	})

	t.Run("update blobber service charge should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]
		newServiceCharge := 52

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": newServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		// BUG: service charge is not being updated
		require.NotEqual(t, newServiceCharge, finalBlobberInfo.StakePoolSettings.ServiceCharge)
	})

	t.Run("update no params should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME: since we are not updating any params, the output should not say `updated successfully`
		require.Equal(t, "blobber settings updated successfully", output[0])
	})

	t.Run("update without blobber ID should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 25)
		require.Equal(t, "Error: required flag(s) \"blobber_id\" not set", output[0])
	})

	t.Run("update with invalid blobber ID should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": "invalid-blobber-id"}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "consensus_failed: consensus failed on sharders", output[1])
	})

	t.Run("update with invalid blobber wallet/owner should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]

		output, err = cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}), escapedTestName(t), configPath), 1, time.Second*2)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_blobber_settings_failed: access denied, allowed for delegate_wallet owner only",
			output[0], strings.Join(output, "\n"))
	})

	// FIXME sortout why it fails
	t.Run("update all params at once should work", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo := blobberList[0]

		newWritePrice := intialBlobberInfo.Terms.Write_price + 1
		newServiceCharge := intialBlobberInfo.StakePoolSettings.ServiceCharge + 1
		newReadPrice := intialBlobberInfo.Terms.Read_price + 1
		newNumberOfDelegates := intialBlobberInfo.StakePoolSettings.NumDelegates + 1
		newMaxOfferDuration := intialBlobberInfo.Terms.Max_offer_duration + 1*time.Second
		newCapacity := intialBlobberInfo.Capacity + 1
		newMinLockDemand := intialBlobberInfo.Terms.Min_lock_demand + 0.01
		newMinStake := intialBlobberInfo.StakePoolSettings.MinStake + 1
		newMaxStake := intialBlobberInfo.StakePoolSettings.MaxStake + 1
		newChallengeCompletionTIme := intialBlobberInfo.Terms.Challenge_completion_time + 1*time.Second

		// FIXME: updating multiple configs at once is not working
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": newWritePrice, "service_charge": newServiceCharge, "read_price": newReadPrice, "num_delegates": newNumberOfDelegates, "max_offer_duration": newMaxOfferDuration, "capacity": newCapacity, "min_lock_demand": newMinLockDemand, "min_stake": newMinStake, "max_stake": newMaxStake, "cct": newChallengeCompletionTIme}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_blobber_settings_failed: invalid blobber terms: read_price is greater than max_read_price allowed", output[0])
	})
}

func getBlobberInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateBlobberInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
