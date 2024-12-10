package cli_tests

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/zcncore"
	"io"
	"math/big"
	"os"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/fileref"
	"github.com/0chain/gosdk/zboxcore/sdk"
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

	err = client.Init(context.Background(), conf.Config{
		BlockWorker:     parsed.BlockWorker,
		SignatureScheme: parsed.SignatureScheme,
		ChainID:         parsed.ChainID,
		MaxTxnQuery:     5,
		QuerySleepTime:  5,
		MinSubmit:       10,
		MinConfirmation: 10,
	})
	if err != nil {
		return err
	}

	err = client.InitSDK(
		"{}",
		parsed.BlockWorker,
		parsed.ChainID,
		parsed.SignatureScheme,
		0, true,
	)

	err = zcncore.SetGeneralWalletInfo(walletJSON, parsed.SignatureScheme)
	if err != nil {
		fmt.Println("Error in sdk init", err)
		os.Exit(1)
	}

	if client.GetClient().IsSplit {
		zcncore.RegisterZauthServer(parsed.ZauthServer)
	}
	return err
}

// GetBlobberIDNotPartOfAllocation returns a blobber not part of current allocation
func GetBlobberIDNotPartOfAllocation(walletname, configFile, allocationID string) (string, error) {
	blobber, err := getBlobberNotPartOfAllocation(walletname, configFile, allocationID)

	if err != nil {
		return "", err
	}
	return string(blobber.ID), err
}

func getBlobberNotPartOfAllocation(walletname, configFile, allocationID string) (*sdk.Blobber, error) {
	err := InitSDK(walletname, configFile)
	if err != nil {
		return nil, err
	}

	a, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return nil, err
	}

	blobbers, err := sdk.GetBlobbers(true, false)
	if err != nil {
		return nil, err
	}

	allocationBlobsMap := map[string]bool{}
	for _, b := range a.BlobberDetails {
		allocationBlobsMap[b.BlobberID] = true
	}

	for _, blobber := range blobbers {
		if _, ok := allocationBlobsMap[string(blobber.ID)]; !ok {
			return blobber, nil
		}
	}

	return nil, fmt.Errorf("failed to get blobber not part of allocation")
}

// GetBlobberIdAndUrlNotPartOfAllocation returns a blobber not part of current allocation
func GetBlobberIdAndUrlNotPartOfAllocation(walletName, configFile, allocationID string) (blobberId, blobberUrl string, err error) {
	blobber, err := getBlobberNotPartOfAllocation(walletName, configFile, allocationID)
	if err != nil || blobber == nil {
		return "", "", err
	}
	return string(blobber.ID), blobber.BaseURL, err
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

	if randomBlobber != "" {
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
