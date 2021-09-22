package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const epsilon float64 = 1e-03
const tokenUnit float64 = 1e+10

func TestFileUploadTokenMovement(t *testing.T) {
	t.Parallel()

	balance := 0.8 // 800.000 mZCN
	t.Run("Parallel", func(t *testing.T) {
		t.Run("Challenge pool should be 0 before any write", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1.0)
			require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

			allocParam := createParams(map[string]interface{}{
				"lock":   balance,
				"size":   10485760,
				"expire": "1h",
			})
			output, err = createNewAllocation(t, configPath, allocParam)
			require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

			allocationID := strings.Fields(output[0])[2]
			output, err = challengePoolInfo(t, configPath, allocationID)
			require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

			require.Len(t, output, 4)
			require.Regexp(t, regexp.MustCompile(fmt.Sprintf("POOL ID: ([a-f0-9]{64}):challengepool:%s", allocationID)), output[0])
			require.Equal(t, "0", strings.Fields(output[3])[0])
		})

		t.Run("Total balance in blobber pool equals locked tokens", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1.0)
			require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

			allocParam := createParams(map[string]interface{}{
				"lock":   balance,
				"size":   10485760,
				"expire": "1h",
			})
			output, err = createNewAllocation(t, configPath, allocParam)
			require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation ouput did not match expected")

			allocationID := strings.Fields(output[0])[2]

			output, err = writePoolInfo(t, configPath)
			require.Nil(t, err, "Failed to fetch Write Pool info", strings.Join(output, "\n"))

			writePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &writePool)
			require.Nil(t, err, "Error unmarshalling write pool", strings.Join(output, "\n"))

			require.Equal(t, allocationID, writePool[0].Id)
			require.InDelta(t, 0.8, intToZCN(writePool[0].Balance), epsilon)
			require.IsType(t, int64(1), writePool[0].ExpireAt)
			require.Equal(t, allocationID, writePool[0].AllocationId)
			require.Less(t, 0, len(writePool[0].Blobber))
			require.Equal(t, true, writePool[0].Locked)

			totalBalanceInBlobbers := float64(0)
			for _, blobber := range writePool[0].Blobber {
				totalBalanceInBlobbers += intToZCN(blobber.Balance)
			}
			require.InDelta(t, 0.8, totalBalanceInBlobbers, epsilon, "Sum of balances should be within epsilon.")
		})

		t.Run("Tokens should move from each blobber's pool balance to challenge pool acc. to blobber write price and uploaded file size", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1.0)
			require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

			allocParam := createParams(map[string]interface{}{
				"lock":   balance,
				"size":   10485760,
				"expire": "1h",
			})
			output, err = createNewAllocation(t, configPath, allocParam)
			require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
			require.Regexp(t, matcher, output[0], "Allocation creation ouput did not match expected")

			allocationID := strings.Fields(output[0])[2]

			output, err = writePoolInfo(t, configPath)
			require.Nil(t, err, "Failed to fetch Write Pool", strings.Join(output, "\n"))

			initialWritePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &initialWritePool)
			require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

			require.Equal(t, allocationID, initialWritePool[0].Id)
			require.InDelta(t, 0.8, intToZCN(initialWritePool[0].Balance), epsilon)
			require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
			require.Equal(t, allocationID, initialWritePool[0].AllocationId)
			require.Less(t, 0, len(initialWritePool[0].Blobber))
			require.Equal(t, true, initialWritePool[0].Locked)

			// Get blobber write price in tok / GB
			blobberWritePrice := make(map[string]float64) // blobber_id -> writeprice in tok / GB
			for _, blobber := range initialWritePool[0].Blobber {
				blobInfo, err := getBlobberInfoJSONByID(t, configPath, blobber.BlobberID)
				require.Nil(t, err, "Could not fetch blobber info", strings.Join(output, "\n"))

				blobberInfo := cli_model.BlobberInfo{}
				err = json.Unmarshal([]byte(blobInfo[0]), &blobberInfo)
				require.Nil(t, err, "Error Unmarshalling the blobber info json", strings.Join(output, "\n"))

				currWritePrice := intToZCN(blobberInfo.Terms.Write_price)
				blobberWritePrice[blobber.BlobberID] = currWritePrice
			}

			// upload a dummy 5 MB file
			output, err = uploadFile(t, configPath, allocationID, "../../internal/dummy_file/five_MB_test_file", "/")
			require.Nil(t, err, "Upload file failed", strings.Join(output, "\n"))

			require.Len(t, output, 2)
			require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = five_MB_test_file", output[1])

			// Necessary for wp-info to update
			time.Sleep(5 * time.Second)

			// Get the new Write-Pool info after upload
			output, err = writePoolInfo(t, configPath)
			require.Nil(t, err, "Failed to fetch Write Pool info", strings.Join(output, "\n"))

			finalWritePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &finalWritePool)
			require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

			require.Equal(t, allocationID, finalWritePool[0].Id)
			require.InDelta(t, 0.8, intToZCN(finalWritePool[0].Balance), epsilon)
			require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
			require.Equal(t, allocationID, finalWritePool[0].AllocationId)
			require.Less(t, 0, len(finalWritePool[0].Blobber))
			require.Equal(t, true, finalWritePool[0].Locked)

			// Blobber pool balance should reduce by (write price*filesize) for each blobber
			for i := 0; i < len(finalWritePool[0].Blobber); i += 1 {
				require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
				require.IsType(t, int64(1), finalWritePool[0].Blobber[i].Balance)

				// deduce tokens
				require.InDelta(t, intToZCN(initialWritePool[0].Blobber[i].Balance)-intToZCN(finalWritePool[0].Blobber[i].Balance), blobberWritePrice[finalWritePool[0].Blobber[i].BlobberID]*0.005, epsilon, "Error should be within epsilon")
			}
		})

	})
}

func writePoolInfo(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cli_utils.RunCommand("./zbox wp-info --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func challengePoolInfo(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	return cli_utils.RunCommand("./zbox cp-info --allocation " + allocationID + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func getBlobberInfoJSONByID(t *testing.T, cliConfigFilename string, blobberID string) ([]string, error) {
	return cli_utils.RunCommand("./zbox bl-info --blobber_id " + blobberID + " --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func uploadFile(t *testing.T, cliConfigFilename string, allocation string, localpath string, remotepath string) ([]string, error) {
	return cli_utils.RunCommand("./zbox upload --allocation " + allocation + " --localpath " + localpath + " --remotepath " + remotepath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func intToZCN(balance int64) float64 {
	return float64(balance) / tokenUnit
}
