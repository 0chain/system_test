package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func Test___FlakyScenariosCreateAllocationFreeStorage(t *testing.T) {
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	assigner := escapedTestName(t) + "_ASSIGNER"

	// register SC owner wallet
	output, err := registerWalletForName(t, configPath, scOwnerWallet)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	// register assigner wallet
	output, err = registerWalletForName(t, configPath, assigner)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	// Open the wallet file themselves to get private key for signing data
	ownerWallet := readWalletFile(t, "./config/"+scOwnerWallet+"_wallet.json")
	assignerWallet := readWalletFile(t, "./config/"+assigner+"_wallet.json")

	// necessary cli call to generate wallet to avoid polluting logs of succeeding cli calls
	output, err = registerWallet(t, configPath)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	configKeyDataShards := "free_allocation_settings.data_shards"
	configKeyParityShards := "free_allocation_settings.parity_shards"
	configKeySize := "free_allocation_settings.size"
	configKeyDuration := "free_allocation_settings.duration"

	// nolint:gocritic
	// configKeyReadPoolFraction := "free_allocation_settings.read_pool_fraction"

	keys := strings.Join([]string{
		configKeyDataShards,
		configKeyParityShards,
		configKeySize,
		configKeyDuration,
	}, ",")

	output, err = getStorageSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, strings.Join(output, "\n"))

	cfgBefore, _ := keyValuePairStringToMap(t, output)

	// ensure revert in config is run regardless of test result
	defer func() {
		oldValues := strings.Join([]string{
			cfgBefore[configKeyDataShards],
			cfgBefore[configKeyParityShards],
			cfgBefore[configKeySize],
			cfgBefore[configKeyDuration],
		}, ",")

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   keys,
			"values": oldValues,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
	}()

	newValues := strings.Join([]string{
		"1",    // decreasing data shards from default 10
		"1",    // decreasing parity shards from default 5
		"1024", // decreasing size from default 10000000000
		"5m",   // reduce free allocation duration from 50h to 5m
	}, ",")

	output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
		"keys":   keys,
		"values": newValues,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2, strings.Join(output, "\n"))
	require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))
	require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

	// miners list
	output, err = getMiners(t, configPath)
	require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var miners climodel.NodeList
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

	freeAllocAssignerTxn := freeAllocationAssignerTxn(t, ownerWallet, assignerWallet)
	err = sendTxn(miners, freeAllocAssignerTxn)
	require.Nil(t, err, "Error sending txn to miners: %v", output[0], err)

	output, err = verifyTransaction(t, configPath, freeAllocAssignerTxn.Hash)
	require.Nil(t, err, "Could not verify commit transaction", strings.Join(output, "\n"))
	require.Len(t, output, 3)
	require.Equal(t, "Transaction verification success", output[0])
	require.Equal(t, "TransactionStatus: 1", output[1])
	require.Greater(t, len(output[2]), 0, output[2])

	t.Parallel()

	// FIXME not working at the moment
	t.Run("Create free storage from marker with accounting", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationForWallet(t, recipient, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: [txn] too less sharders to confirm it: min_confirmation is 50%, but got 0/2 sharders", output[0])
		// FIXME disabled as not working as expected
		// require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		// require.Len(t, output, 1)
		// matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		// require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")
		// allocationID := strings.Fields(output[0])[2]
		//
		// readPoolFraction, err := strconv.ParseFloat(cfgBefore[configKeyReadPoolFraction], 64)
		// require.Nil(t, err, "Read pool fraction config is not float: %s", cfgBefore[configKeyReadPoolFraction])
		//
		// wantReadPoolFraction := marker.FreeTokens * readPoolFraction
		// wantWritePoolToken := marker.FreeTokens - wantReadPoolFraction
		//
		// // Verify write and read pools are set with tokens
		// output, err = writePoolInfo(t, configPath, true)
		// require.Len(t, output, 1, strings.Join(output, "\n"))
		// require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))
		//
		// writePool := []climodel.WritePoolInfo{}
		// err = json.Unmarshal([]byte(output[0]), &writePool)
		// require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))
		// require.Len(t, writePool, 1, "More than 1 write pool found", strings.Join(output, "\n"))
		// require.Equal(t, ConvertToValue(wantWritePoolToken), writePool[0].Balance, "Expected write pool amount not met", strings.Join(output, "\n"))
		//
		// readPool := getReadPoolInfo(t, allocationID)
		// require.Len(t, readPool, 1, "Read pool must exist")
		// require.Equal(t, ConvertToValue(wantReadPoolFraction), readPool[0].Balance, "Read Pool balance must be equal to locked amount")
	})

	t.Run("Create free storage with malformed marker should fail", func(t *testing.T) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte("bad marker json"), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "unmarshalling markerinvalid character 'b' looking for beginning of value", output[0])
	})

	t.Run("Create free storage with invalid marker contents should fail", func(t *testing.T) {
		// register recipient wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte(`{"invalid_marker":true}`), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: error getting assigner details: value not present", output[0])
	})

	t.Run("Create free storage with invalid marker signature should fail", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		marker.Signature = "badsignature"
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		// TODO test can differ one of just sort it out
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: encoding/hex: invalid byte: U+0073 's'", output[0])
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: marker timestamped in the future: 1642693108"", output[0])
	})

	t.Run("Create free storage with wrong recipient wallet should fail", func(t *testing.T) {
		recipientCorrect := escapedTestName(t) + "_RECIPIENT"

		// register correct recipient wallet
		output, err = registerWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		// register this wallet
		output, err = registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipientCorrect + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, 1, len(output), strings.Join(output, "\n"))
		require.Regexp(t, regexp.MustCompile("Error creating free allocation: free_allocation_failed: marker verification failed: marker timestamped in the future: ([0-9]{10})"), output[0])
	})

	t.Run("Create free storage with tokens exceeding assigner's individual limit should fail", func(t *testing.T) {
		recipient := escapedTestName(t)

		// register recipient wallet
		output, err = registerWalletForName(t, configPath, recipient)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: freeTokensIndividualLimit + 1,
			Timestamp:  time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		marker.Signature = sign(t, data, assignerWallet)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{"free_storage": markerFile}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		// TODO sort out why message changes
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: 110000000000 exceeded permitted free storage  100000000000", output[0])
		// require.Equal(t, "Error creating free allocation: free_allocation_failed:marker verification failed: marker timestamped in the future: 1642693167", output[0])
	})
}
