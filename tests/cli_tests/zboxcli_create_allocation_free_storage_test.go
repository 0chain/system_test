package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/0chain/system_test/internal/api/util/crypto"

	"github.com/herumi/bls-go-binary/bls"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
)

const (
	storageSmartContractAddress = `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7`
	minerSmartContractAddress   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"

	freeTokensIndividualLimit = 100.0
	freeTokensTotalLimit      = 10000.0
	configKeyReadPoolFraction = "free_allocation_settings.read_pool_fraction"
)

func TestCreateAllocationFreeStorage(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create free storage from marker with accounting")

	var assignerWallet *climodel.WalletFile
	var cfg map[string]string

	t.TestSetup("Create free storage allocation wallet", func() {
		err := bls.Init(bls.CurveFp254BNb)
		require.NoError(t, err, "Error initializing BLS")

		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		assigner := escapedTestName(t) + "_ASSIGNER"

		// create SC owner wallet
		output, err := createWalletForName(t, configPath, scOwnerWallet)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		// create assigner wallet
		output, err = createWalletForName(t, configPath, assigner)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		assignerWallet = readWalletFile(t, "./config/"+assigner+"_wallet.json")

		// necessary cli call to generate wallet to avoid polluting logs of succeeding cli calls
		output, err = createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = getStorageSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfg, _ = keyValuePairStringToMap(output)

		// miners list
		output, err = getMiners(t, configPath)
		require.Nil(t, err, "get miners failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var miners climodel.NodeList
		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[0], err)
		require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))

		input := map[string]interface{}{
			"name":  assignerWallet.ClientID,
			"key":   assignerWallet.ClientKey,
			"limit": freeTokensIndividualLimit,
			"max":   freeTokensTotalLimit,
		}
		output, err = createFreeStorageAllocation(t, configPath, scOwnerWallet, createParams(input))
		require.NoError(t, err)
		t.Log(output)
	})

	t.RunSequentiallyWithTimeout("Create free storage from marker with accounting", 60*time.Second, func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// create recipient wallet
		output, err := createWalletForName(t, configPath, recipient)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Nonce:      time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)

		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationForWallet(t, recipient, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]

		readPoolFraction, err := strconv.ParseFloat(cfg[configKeyReadPoolFraction], 64)
		require.Nil(t, err, "Read pool fraction config is not float: %s", cfg[configKeyReadPoolFraction])

		wantReadPoolFraction := marker.FreeTokens * readPoolFraction
		wantWritePoolToken := marker.FreeTokens - wantReadPoolFraction

		allocation := getAllocation(t, allocationID)
		require.Equal(t, ConvertToValue(wantWritePoolToken), allocation.WritePool, "Expected write pool amount not met", strings.Join(output, "\n"))
		require.Equal(t, uint16(63), allocation.FileOptions, "Expected file_options to be 63 (all allowed) by default", strings.Join(output, "\n"))

		readPool := getReadPoolInfo(t)
		require.Equal(t, ConvertToValue(wantReadPoolFraction), readPool.Balance, "Read Pool balance must be equal to locked amount")
	})

	t.RunSequentially("Create free storage with malformed marker should fail", func(t *test.SystemTest) {
		// create recipient wallet
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte("bad marker json"), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "unmarshalling markerinvalid character 'b' looking for beginning of value", output[0])
	})

	t.RunSequentially("Create free storage with invalid marker contents should fail", func(t *test.SystemTest) {
		// create recipient wallet
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		markerFile := "./config/" + escapedTestName(t) + "_MARKER.json"

		err = os.WriteFile(markerFile, []byte(`{"invalid_marker":true}`), 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker can be used only by its recipient", output[0])
	})

	t.RunSequentially("Create free storage with invalid marker signature should fail", func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// create recipient wallet
		output, err := createWalletForName(t, configPath, recipient)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Nonce:      2,
		}

		marker.Signature = "badsignature"
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker verification failed: encoding/hex: invalid byte: U+0073 's'", output[0])
	})

	t.RunSequentially("Create free storage with wrong recipient wallet should fail", func(t *test.SystemTest) {
		recipientCorrect := escapedTestName(t) + "_RECIPIENT"

		// create correct recipient wallet
		output, err := createWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipientCorrect)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		// create this wallet
		output, err = createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: 5,
			Nonce:      time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipientCorrect + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, 1, len(output), strings.Join(output, "\n"))
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker can be used only by its recipient", output[0])
	})

	t.RunSequentially("Create free storage with tokens exceeding assigner's individual limit should fail", func(t *test.SystemTest) {
		recipient := escapedTestName(t)

		// create recipient wallet
		output, err := createWalletForName(t, configPath, recipient)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		recipientWallet, err := getWalletForName(t, configPath, recipient)
		require.Nil(t, err, "Error occurred when retrieving new owner wallet")

		marker := climodel.FreeStorageMarker{
			Recipient:  recipientWallet.ClientID,
			FreeTokens: freeTokensIndividualLimit + 1,
			Nonce:      time.Now().Unix(),
		}

		forSignatureBytes, err := json.Marshal(&marker)
		require.Nil(t, err, "Could not marshal marker")

		data := hex.EncodeToString(forSignatureBytes)
		rawHash, err := hex.DecodeString(data)
		require.Nil(t, err, "failed to decode hex %s", data)
		require.NotNil(t, rawHash, "failed to decode hex %s", data)
		secretKey := crypto.ToSecretKey(t, assignerWallet)
		marker.Signature = crypto.Sign(t, string(rawHash), secretKey)
		marker.Assigner = assignerWallet.ClientID

		forFileBytes, err := json.Marshal(marker)
		require.Nil(t, err, "Could not marshal marker")

		markerFile := "./config/" + recipient + "_MARKER.json"

		err = os.WriteFile(markerFile, forFileBytes, 0600)
		require.Nil(t, err, "Could not write file marker")

		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(map[string]interface{}{
			"free_storage": markerFile,
		}))
		require.NotNil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1)
		require.Equal(t, "Error creating free allocation: free_allocation_failed: marker verification failed: 1010000000000 exceeded permitted free storage  1000000000000", output[0])
	})
}

func readWalletFile(t *test.SystemTest, file string) *climodel.WalletFile {
	wallet := &climodel.WalletFile{}

	f, err := os.Open(file)
	require.Nil(t, err, "wallet file %s not found", file)

	ownerWalletBytes, err := io.ReadAll(f)
	require.Nil(t, err, "error reading wallet file %s", file)

	err = json.Unmarshal(ownerWalletBytes, wallet)
	require.Nil(t, err, "error marshaling wallet content")

	return wallet
}

func createFreeStorageAllocation(t *test.SystemTest, configFile, from, params string) ([]string, error) {
	t.Logf("Creating new free storage allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox add --silent %s --wallet %s --configDir ./config --config %s",
		params,
		from+"_wallet.json",
		configFile), 3, time.Second*5)
}
