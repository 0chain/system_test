package cli_tests

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/0chain/gosdk/constants"
	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/zcncore"

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
		MinSubmit:       100,
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
	if err != nil {
		return err
	}

	err = zcncore.SetGeneralWalletInfo(walletJSON, parsed.SignatureScheme)
	if err != nil {
		log.Printf("Error in sdk init: %v", err)
		return err
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

// getEnterpriseBlobberIDs queries the sharder REST API to get the set of enterprise blobber IDs.
// The SDK's Blobber struct doesn't include is_enterprise, so we query the raw API.
// Uses pagination with limit=20 (sharder max) to fetch all blobbers.
func getEnterpriseBlobberIDs() map[string]bool {
	const storageSCAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	const pageLimit = 20
	enterpriseIDs := map[string]bool{}

	type blobberWithEnterprise struct {
		ID           string `json:"id"`
		IsEnterprise bool   `json:"is_enterprise"`
	}
	type blobberNodes struct {
		Nodes []blobberWithEnterprise `json:"Nodes"`
	}

	offset := 0
	for {
		endpoint := fmt.Sprintf("/getblobbers?active=true&limit=%d&offset=%d&stakable=false", pageLimit, offset)
		b, err := client.MakeSCRestAPICallToSharder(storageSCAddress, endpoint, nil)
		if err != nil {
			if offset == 0 {
				log.Printf("Warning: failed to query enterprise blobbers: %v", err)
			}
			break
		}

		var wrap blobberNodes
		if err := json.Unmarshal(b, &wrap); err != nil {
			log.Printf("Warning: failed to parse enterprise blobbers response: %v", err)
			break
		}

		for _, node := range wrap.Nodes {
			if node.IsEnterprise {
				enterpriseIDs[node.ID] = true
			}
		}

		// If we got fewer results than the limit, we've fetched all blobbers
		if len(wrap.Nodes) < pageLimit {
			break
		}
		offset += pageLimit
	}

	if len(enterpriseIDs) > 0 {
		log.Printf("Found %d enterprise blobber(s) to exclude from selection", len(enterpriseIDs))
	}
	return enterpriseIDs
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

	// Get enterprise blobber IDs from chain (SDK struct lacks IsEnterprise)
	enterpriseIDs := getEnterpriseBlobberIDs()

	allocationBlobsMap := map[string]bool{}
	for _, b := range a.BlobberDetails {
		allocationBlobsMap[b.BlobberID] = true
	}

	// Minimum free capacity required (1MB) to avoid "free capacity insufficient" errors
	const minFreeCapacity = 1048576

	for _, blobber := range blobbers {
		if _, ok := allocationBlobsMap[string(blobber.ID)]; !ok {
			// Skip restricted blobbers (require auth tickets)
			if blobber.IsRestricted {
				continue
			}
			// Skip enterprise blobbers (require auth tickets)
			if enterpriseIDs[string(blobber.ID)] {
				log.Printf("Skipping enterprise blobber %s", string(blobber.ID))
				continue
			}
			// Skip blobbers with insufficient free capacity
			freeCapacity := int64(blobber.Capacity) - int64(blobber.Allocated)
			if freeCapacity < minFreeCapacity {
				log.Printf("Skipping blobber %s: free capacity %d < %d", string(blobber.ID), freeCapacity, minFreeCapacity)
				continue
			}
			return blobber, nil
		}
	}

	return nil, fmt.Errorf("failed to get blobber not part of allocation (all blobbers are either in allocation, enterprise, restricted, or have insufficient free capacity)")
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

// UploadItem describes a single file to upload in a multi-operation batch.
type UploadItem struct {
	LocalPath  string
	RemotePath string
	Size       int64
}

// doMultiOperation initializes SDK, fetches the allocation, and runs ops as a single transaction.
func doMultiOperation(walletName, configFileName, allocationID string, ops []sdk.OperationRequest) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	walletFile := filepath.Join(wd, "config", walletName+"_wallet.json")
	configFile := filepath.Join(wd, "config", configFileName)

	if err := InitSDK(walletFile, configFile); err != nil {
		return err
	}

	allocation, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return err
	}

	defer func() {
		for _, op := range ops {
			if op.OperationType == constants.FileOperationInsert || op.OperationType == constants.FileOperationUpdate {
				if closer, ok := op.FileReader.(io.Closer); ok {
					_ = closer.Close()
				}
				if op.FileMeta.Path != "" {
					_ = os.RemoveAll(op.FileMeta.Path)
				}
			}
		}
	}()

	return allocation.DoMultiOperation(ops)
}

// MultiUpload uploads multiple files in a single blockchain transaction.
func MultiUpload(walletName, configFileName, allocationID string, items []UploadItem) error {
	ops := make([]sdk.OperationRequest, 0, len(items))
	for _, item := range items {
		f, err := os.Open(item.LocalPath)
		if err != nil {
			return err
		}
		ops = append(ops, sdk.OperationRequest{
			OperationType: constants.FileOperationInsert,
			FileReader:    f,
			FileMeta: sdk.FileMeta{
				Path:       item.LocalPath,
				ActualSize: item.Size,
				RemoteName: filepath.Base(item.RemotePath),
				RemotePath: item.RemotePath,
			},
			Workdir:    "./",
			RemotePath: item.RemotePath,
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, ops)
}

// MultiCopy copies multiple files in a single blockchain transaction.
// ops is a list of [remotePath, destPath] pairs.
func MultiCopy(walletName, configFileName, allocationID string, ops [][2]string) error {
	sdkOps := make([]sdk.OperationRequest, 0, len(ops))
	for _, op := range ops {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationCopy,
			RemotePath:    op[0],
			DestPath:      op[1],
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, sdkOps)
}

// MultiMove moves multiple files in a single blockchain transaction.
// ops is a list of [remotePath, destPath] pairs.
func MultiMove(walletName, configFileName, allocationID string, ops [][2]string) error {
	sdkOps := make([]sdk.OperationRequest, 0, len(ops))
	for _, op := range ops {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationMove,
			RemotePath:    op[0],
			DestPath:      op[1],
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, sdkOps)
}

// MultiDelete deletes multiple files in a single blockchain transaction.
func MultiDelete(walletName, configFileName, allocationID string, remotePaths []string) error {
	sdkOps := make([]sdk.OperationRequest, 0, len(remotePaths))
	for _, p := range remotePaths {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationDelete,
			RemotePath:    p,
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, sdkOps)
}

// MultiRename renames multiple files in a single blockchain transaction.
// ops is a list of [remotePath, newName] pairs.
func MultiRename(walletName, configFileName, allocationID string, ops [][2]string) error {
	sdkOps := make([]sdk.OperationRequest, 0, len(ops))
	for _, op := range ops {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationRename,
			RemotePath:    op[0],
			DestName:      op[1],
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, sdkOps)
}

// MultiRenameAndDelete renames and deletes files in a single blockchain transaction.
// renames is a list of [remotePath, newName] pairs; deletePaths are paths to delete.
func MultiRenameAndDelete(walletName, configFileName, allocationID string, renames [][2]string, deletePaths []string) error {
	sdkOps := make([]sdk.OperationRequest, 0, len(renames)+len(deletePaths))
	for _, op := range renames {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationRename,
			RemotePath:    op[0],
			DestName:      op[1],
		})
	}
	for _, p := range deletePaths {
		sdkOps = append(sdkOps, sdk.OperationRequest{
			OperationType: constants.FileOperationDelete,
			RemotePath:    p,
		})
	}
	return doMultiOperation(walletName, configFileName, allocationID, sdkOps)
}
