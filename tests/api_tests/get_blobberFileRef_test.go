package api_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/endpoint"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	// cli_tests "github.com/0chain/system_test/tests/cli_tests"

	"github.com/stretchr/testify/require"
	//nolint
)

func TestGetBlobberFileRefs(t *testing.T) {
	t.Parallel()
	configPath := "./api_tests_config.yaml"
	t.Run("Get file ref with allocation id, remote path and ref type should work", func(t *testing.T) {
		t.Parallel()
		_, err := registerWalletViaCli(t, configPath)
		require.Nil(t, err)
		wallet := readWalletFile(t, "./config/"+escapedTestName(t)+"_wallet.json")
		registeredWallet, keyPair := registerWalletForMnemonic(t, wallet.Mnemonic)
		// keyPair := crypto.GenerateKeys(t, wallet.Mnemonic)
		executeFaucetTransactionResponse, confirmation := executeFaucet(t, registeredWallet, keyPair)
		require.NotNil(t, executeFaucetTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		availableBlobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 2, 1, time.Minute*20)
		require.NotNil(t, availableBlobbers)
		require.NotNil(t, blobberRequirements)

		blobberRequirements.Blobbers = availableBlobbers

		createAllocationTransactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.NotNil(t, createAllocationTransactionResponse)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)

		allocation := getAllocation(t, createAllocationTransactionResponse.Entity.Hash)
		t.Logf((" this is allocation id %s"), allocation.ID)
		require.NotNil(t, allocation)

		sysKeyPair := []sys.KeyPair{sys.KeyPair{
			PrivateKey: keyPair.PrivateKey.SerializeToHexStr(),
			PublicKey:  keyPair.PublicKey.SerializeToHexStr(),
		}}

		url := "http://demo.0chain.net/blobber01"
		clientSignature, _ := SignHash(createAllocationTransactionResponse.Entity.Hash, "bls0chain", sysKeyPair)
		// // require.NotNil(t, allocation)
		allocationId := allocation.ID
		refType := "f" // or can be "d"
		getBlobberFileUploadRequest(t, url, registeredWallet, keyPair, allocationId, refType, clientSignature)
		blobberFileRefRequest := getBlobberFileRefRequest(t, url, registeredWallet, keyPair, allocationId, refType, clientSignature)
		require.NotNil(t, blobberFileRefRequest)
		// _, httpResponse, err := v1BlobberGetFileRefs(t, blobberFileRefRequest)
		// require.Nil(t, err)
		// require.Equal(t, endpoint.HttpOkStatus, httpResponse.Status(), httpResponse)
	})
}

func getBlobberFileRefRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string, clientSignature string) model.BlobberGetFileRefsRequest {
	t.Logf("get blobber file request object...")
	blobberFileRequest := model.BlobberGetFileRefsRequest{
		URL:             url,
		ClientID:        registeredWallet.ClientID,
		ClientKey:       registeredWallet.ClientKey,
		ClientSignature: clientSignature,
		AllocationID:    allocationId,
		RefType:         refType,
	}
	return blobberFileRequest
}

func SignHash(hash string, signatureScheme string, keys []sys.KeyPair) (string, error) {
	retSignature := ""
	for _, kv := range keys {
		ss := zcncrypto.NewSignatureScheme(signatureScheme)
		err := ss.SetPrivateKey(kv.PrivateKey)
		if err != nil {
			return "", err
		}

		if len(retSignature) == 0 {
			retSignature, err = ss.Sign(hash)
		} else {
			retSignature, err = ss.Add(retSignature, hash)
		}
		if err != nil {
			return "", err
		}
	}
	return retSignature, nil
}

func getBlobberFileUploadRequest(t *testing.T, url string, registeredWallet *model.Wallet, keyPair *model.KeyPair, allocationId string, refType string, clientSignature string) {
	t.Logf("get blobber file upload request object...")

	// generating random file
	filename := generateRandomTestFileName(t)
	_, err := createFileWithSize(filename, 1024)
	require.Nil(t, err)

	configPath := "./api_tests_config.yaml"
	output, err := uploadFileForWallet(t, escapedTestName(t), configPath, map[string]interface{}{
		"allocation": allocationId,
		"remotepath": "/",
		"localpath":  filename,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
}

func generateRandomTestFileName(t *testing.T) string {
	path := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))

	//FIXME: Filenames longer than 100 characters are rejected see https://github.com/0chain/zboxcli/issues/249
	randomFilename := RandomAlphaNumericString(10)
	return fmt.Sprintf("%s%s%s_test.txt", path, string(os.PathSeparator), randomFilename)
}

func RandomAlphaNumericString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret)
}

func uploadFileForWallet(t *testing.T, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Uploading file...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func createParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func escapedTestName(t *testing.T) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func registerWalletViaCli(t *testing.T, cliConfigFilename string) ([]string, error) {
	return registerWalletViaCliForName(t, cliConfigFilename, escapedTestName(t))
}

func registerWalletViaCliForName(t *testing.T, cliConfigFilename, name string) ([]string, error) {
	t.Logf("Registering wallet...")
	return cliutils.RunCommand(t, "./zbox register --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func readWalletFile(t *testing.T, file string) *climodel.WalletFile {
	wallet := &climodel.WalletFile{}

	f, err := os.Open(file)
	require.Nil(t, err, "wallet file %s not found", file)

	ownerWalletBytes, err := io.ReadAll(f)
	require.Nil(t, err, "error reading wallet file %s", file)

	err = json.Unmarshal(ownerWalletBytes, wallet)
	require.Nil(t, err, "error marshaling wallet content")

	return wallet
}
