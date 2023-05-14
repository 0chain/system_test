package cli_tests

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"

	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/gosdk/zboxcore/fileref"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/crypto"
)

// getBlobberNotPartOfAllocation returns a blobber not part of current allocation
func InitSDK(wallet string) error {
	f, err := os.Open(wallet)
	if err != nil {
		return err
	}
	clientBytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	walletJSON := string(clientBytes)

	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig := config.Parse(configPath)

	err = sdk.InitStorageSDK(
		walletJSON,
		parsedConfig.BlockWorker,
		"",
		crypto.BLS0Chain,
		nil,
		0,
	)
	return err
}

func GetBlobberNotPartOfAllocation(walletname, allocationID string) (string, error) {
	err := InitSDK(walletname)
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

func GetRandomBlobber(walletname, except_blobber string) (string, error) {
	err := InitSDK(walletname)
	if err != nil {
		return "", err
	}
	blobbers, err := sdk.GetBlobbers(true)
	if err != nil {
		return "", err
	}

	var randomBlobber string
	for range blobbers {
		randomIndex, err := generateRandomIndex(int64(len(blobbers)))
		if err != nil {
			return "", err
		}

		blobber := blobbers[randomIndex.Int64()].ID
		if blobber != common.Key(except_blobber) {
			randomBlobber = string(blobber)
			break
		}
	}

	if len(randomBlobber) > 0 {
		return randomBlobber, nil
	}
	return "", fmt.Errorf("failed to get blobbers")
}

func VerifyFileRefFromBlobber(walletname, allocationID, blobberID, remoteFile string) (*fileref.FileRef, error) {
	err := InitSDK(walletname)
	if err != nil {
		return nil, err
	}
	return sdk.GetFileRefFromBlobber(allocationID, blobberID, remoteFile)

}
