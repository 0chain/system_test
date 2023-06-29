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

func TestBlobberConfigUpdate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("update blobber capacity should work")

	var intialBlobberInfo climodel.BlobberDetails
	t.TestSetup("Create wallet, execute faucet, get blobber details", func() {
		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = createWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		intialBlobberInfo = blobberList[0]

		t.Cleanup(func() {
			output, err := createWallet(t, configPath)
			require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": intialBlobberInfo.Capacity}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": intialBlobberInfo.StakePoolSettings.MaxNumDelegates}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": intialBlobberInfo.StakePoolSettings.ServiceCharge}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": intToZCN(intialBlobberInfo.Terms.Read_price)}))
			require.Nil(t, err, strings.Join(output, "\n"))

			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": intToZCN(intialBlobberInfo.Terms.Write_price)}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		// init enough tokens to blobber owner wallet to issue txns
		for i := 0; i < 3; i++ {
			_, err = executeFaucetWithTokensForWallet(t, blobberOwnerWallet, configPath, 9)
			require.NoError(t, err)
		}
	})

	t.RunSequentially("update blobber capacity should work", func(t *test.SystemTest) {
		// create wallet for normal user
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		newCapacity := 301 * GB

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": newCapacity}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, int64(newCapacity), finalBlobberInfo.Capacity)
	})

	t.RunSequentially("update blobber number of delegates should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		newNumberOfDelegates := 15

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": newNumberOfDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newNumberOfDelegates, finalBlobberInfo.StakePoolSettings.MaxNumDelegates)
	})

	t.RunSequentially("update blobber service charge should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		newServiceCharge := 0.1

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": newServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newServiceCharge, finalBlobberInfo.StakePoolSettings.ServiceCharge)
	})

	t.RunSequentially("update no params should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME: since we are not updating any params, the output should not say `updated successfully`
		require.Equal(t, "blobber settings updated successfully", output[0])
	})

	t.RunSequentially("update without blobber ID should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 26)
		require.Equal(t, "Error: required flag(s) \"blobber_id\" not set", output[0])
	})

	t.RunSequentially("update with invalid blobber ID should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": "invalid-blobber-id"}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "internal_error: error retrieving blobber invalid-blobber-id, error record not found", output[1])
	})

	t.RunSequentially("update with invalid blobber wallet/owner should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}), escapedTestName(t), configPath), 1, time.Second*2)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_blobber_settings_failed: access denied, allowed for delegate_wallet owner only",
			output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update blobber read price should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		oldReadPrice := intialBlobberInfo.Terms.Read_price
		newReadPrice := intToZCN(oldReadPrice) + 1

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": newReadPrice}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newReadPrice, intToZCN(finalBlobberInfo.Terms.Read_price))
	})

	t.RunSequentially("update blobber write price should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		oldWritePrice := intialBlobberInfo.Terms.Write_price
		newWritePrice := intToZCN(oldWritePrice) + 1

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": newWritePrice}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberInfo.Terms.Write_price))
	})

	t.RunSequentially("update all params at once should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		newWritePrice := intToZCN(intialBlobberInfo.Terms.Write_price) + 1
		newServiceCharge := intialBlobberInfo.StakePoolSettings.ServiceCharge + 0.1
		newReadPrice := intToZCN(intialBlobberInfo.Terms.Read_price) + 1
		newNumberOfDelegates := intialBlobberInfo.StakePoolSettings.MaxNumDelegates + 1
		newCapacity := intialBlobberInfo.Capacity + 1
		newNotAvailable := !intialBlobberInfo.NotAvailable

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":     intialBlobberInfo.ID,
			"write_price":    newWritePrice,
			"service_charge": newServiceCharge,
			"read_price":     newReadPrice,
			"num_delegates":  newNumberOfDelegates,
			"capacity":       newCapacity,
			"not_available":  newNotAvailable,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		if !newNotAvailable {
			t.Cleanup(func() { setNotAvailability(t, intialBlobberInfo.ID, true) })
		}

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberInfo.Terms.Write_price))
		require.Equal(t, newServiceCharge, finalBlobberInfo.StakePoolSettings.ServiceCharge)
		require.Equal(t, newReadPrice, intToZCN(finalBlobberInfo.Terms.Read_price))
		require.Equal(t, newNumberOfDelegates, finalBlobberInfo.StakePoolSettings.MaxNumDelegates)
		require.Equal(t, newCapacity, finalBlobberInfo.Capacity)
		require.Equal(t, newNotAvailable, finalBlobberInfo.NotAvailable)
	})
}

func getBlobberInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateBlobberInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
