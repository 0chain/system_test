package cli_tests

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/fileref"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/gosdk/zcncore"
)

func InitSDK(wallet, configFile string) error {
	f, err := os.Open(wallet)
	if err != nil {
		return err
	}
	clientBytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	walletJSON := string(clientBytes)

	parsed, err := conf.LoadConfigFile(configFile)
	if err != nil {
		return err
	}

	marshal, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	err = zcncore.Init(string(marshal))

	err = sdk.InitStorageSDK(
		walletJSON,
		parsed.BlockWorker,
		parsed.ChainID,
		parsed.SignatureScheme,
		nil,
		0,
	)
	return err
}

// GetBlobberNotPartOfAllocation returns a blobber not part of current allocation
func GetBlobberNotPartOfAllocation(walletname, configFile, allocationID string) (string, error) {
	err := InitSDK(walletname, configFile)
	if err != nil {
		return "", err
	}

	a, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return "", err
	}

	blobbers, err := sdk.GetBlobbers(true)
	if err != nil {
		return "", err
	}

	allocationBlobsMap := map[string]bool{}
	for _, b := range a.BlobberDetails {
		allocationBlobsMap[b.BlobberID] = true
	}

	for _, blobber := range blobbers {
		if _, ok := allocationBlobsMap[string(blobber.ID)]; !ok {
			return string(blobber.ID), nil
		}
	}

	return "", fmt.Errorf("failed to get blobber not part of allocation")
}

func generateRandomIndex(sliceLen int64) (*big.Int, error) {
	// Generate a random index within the range of the slice
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(sliceLen))
	if err != nil {
		return nil, err
	}
	return randomIndex, nil
}

// GetRandomBlobber gets a random blobber from allocation
func GetRandomBlobber(walletname, configFile, allocationID, except_blobber string) (string, error) {
	err := InitSDK(walletname, configFile)
	if err != nil {
		return "", err
	}

	a, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return "", err
	}

	blobbers := []string{}

	for _, blobber := range a.BlobberDetails {
		blobbers = append(blobbers, blobber.BlobberID)
	}

	var randomBlobber string
	for range blobbers {
		randomIndex, err := generateRandomIndex(int64(len(blobbers)))
		if err != nil {
			return "", err
		}

		blobber := blobbers[randomIndex.Int64()]
		if blobber != except_blobber {
			randomBlobber = string(blobber)
			break
		}
	}

	if len(randomBlobber) > 0 {
		return randomBlobber, nil
	}
	return "", fmt.Errorf("failed to get blobbers")
}

func VerifyFileRefFromBlobber(walletname, configFile, allocationID, blobberID, remoteFile string) (*fileref.FileRef, error) {
	err := InitSDK(walletname, configFile)
	if err != nil {
		return nil, err
	}
	return sdk.GetFileRefFromBlobber(allocationID, blobberID, remoteFile)
}
