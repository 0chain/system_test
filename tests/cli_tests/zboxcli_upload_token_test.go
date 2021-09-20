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
	"github.com/stretchr/testify/assert"
)

const epsilon float64 = 1e-04
const tokenUnit float64 = 1e+10

func TestFileUploadTokenMovement(t *testing.T) {
	assert := assert.New(t)

	balance := 0.8 // 800.000 mZCN
	t.Run("Parallel", func(t *testing.T) {
		t.Run("Challenge pool should be 0 before any write", func(t *testing.T) {
			t.Parallel()

			if _, err := registerWallet(t, configPath); err != nil {
				t.Errorf("Failed to register wallet due to error: %v", err)
			}

			if _, err := executeFaucetWithTokens(t, configPath, 1.0); err != nil {
				t.Errorf("Failed to execute faucet transaction due to error: %v", err)
			}

			output, err := newAllocation(t, configPath, balance)
			if err != nil {
				t.Errorf("Failed to create new allocation with output: %v", strings.Join(output, "\n"))
			}

			allocationID := strings.Fields(output[0])[2]
			output, err = challengePoolInfo(t, configPath, allocationID)
			if err != nil {
				t.Errorf("Could not fetch challenge pool info due to error: %v", err)
			}

			assert.Equal(4, len(output))
			assert.Regexp(regexp.MustCompile(fmt.Sprintf("POOL ID: ([a-f0-9]{64}):challengepool:%s", allocationID)), output[0])
			assert.Equal("0", strings.Fields(output[3])[0])
		})

		t.Run("Total balance in blobber pool equals locked tokens", func(t *testing.T) {
			t.Parallel()

			if _, err := registerWallet(t, configPath); err != nil {
				t.Errorf("Failed to register wallet due to error: %v", err)
			}

			if _, err := executeFaucetWithTokens(t, configPath, 1.0); err != nil {
				t.Errorf("Failed to execute faucet transaction due to error: %v", err)
			}

			output, err := newAllocation(t, configPath, balance)
			if err != nil {
				t.Errorf("Failed to create new allocation with output: %v", strings.Join(output, "\n"))
			}

			assert.Equal(1, len(output))
			assert.Regexp(regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation ouput did not match expected")

			allocationID := strings.Fields(output[0])[2]

			output, err = writePoolInfo(t, configPath)
			if err != nil {
				t.Errorf("Failed to fetch Write Pool info with output: %v", strings.Join(output, "\n"))
			}

			writePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &writePool)
			if err != nil {
				t.Errorf("Error unmarshalling write pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Equal(allocationID, writePool[0].Id)
			assert.Equal(0.8, intToZCN(writePool[0].Balance))
			assert.IsType(int64(1), writePool[0].ExpireAt)
			assert.Equal(allocationID, writePool[0].AllocationId)
			assert.Less(0, len(writePool[0].Blobber))
			assert.Equal(true, writePool[0].Locked)

			totalBalanceInBlobbers := float64(0)
			for _, blobber := range writePool[0].Blobber {
				totalBalanceInBlobbers += intToZCN(blobber.Balance)
			}
			assert.InDelta(0.8, totalBalanceInBlobbers, epsilon, "Sum of balances should be within error of 10^-2.")
		})

		t.Run("Tokens should move from each blobber's pool balance to challenge pool acc. to blobber write price and uploaded file size", func(t *testing.T) {
			t.Parallel()

			if _, err := registerWallet(t, configPath); err != nil {
				t.Errorf("Failed to register wallet due to error: %v", err)
			}

			if _, err := executeFaucetWithTokens(t, configPath, 1.0); err != nil {
				t.Errorf("Failed to execute faucet transaction due to error: %v", err)
			}

			output, err := newAllocation(t, configPath, balance)
			if err != nil {
				t.Errorf("Failed to create new allocation with output: %v", strings.Join(output, "\n"))
			}

			assert.Equal(1, len(output))
			matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
			assert.Regexp(matcher, output[0], "Allocation creation ouput did not match expected")

			allocationID := strings.Fields(output[0])[2]

			output, err = writePoolInfo(t, configPath)
			if err != nil {
				t.Errorf("Failed to fetch Write Pool info with output: %v", strings.Join(output, "\n"))
			}

			initialWritePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &initialWritePool)
			if err != nil {
				t.Errorf("Error unmarshalling write pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Equal(allocationID, initialWritePool[0].Id)
			assert.Equal(0.8, intToZCN(initialWritePool[0].Balance))
			assert.IsType(int64(1), initialWritePool[0].ExpireAt)
			assert.Equal(allocationID, initialWritePool[0].AllocationId)
			assert.Less(0, len(initialWritePool[0].Blobber))
			assert.Equal(true, initialWritePool[0].Locked)

			// Get blobber write price in tok / GB
			blobberWritePrice := make(map[string]float64) // blobber_id -> writeprice in tok / GB
			for _, blobber := range initialWritePool[0].Blobber {
				blobInfo, err := getBlobberInfoJSONByID(t, configPath, blobber.BlobberID)
				if err != nil {
					t.Errorf("Could not fetch blobber info due to error: %v", err)
				}

				blobberInfo := cli_model.BlobberInfo{}
				err = json.Unmarshal([]byte(blobInfo[0]), &blobberInfo)
				if err != nil {
					t.Errorf("Error Unmarshalling the blobber info json due to: %v", err)
				}

				currWritePrice := intToZCN(blobberInfo.Terms.Write_price)
				blobberWritePrice[blobber.BlobberID] = currWritePrice
			}

			// upload a dummy 5 MB file
			output, err = uploadFile(t, configPath, allocationID, "../../internal/dummy_file/five_MB_test_file", "/")
			if err != nil {
				t.Errorf("Upload file failed due to error: %v", err)
			}

			assert.Equal(2, len(output))
			assert.Equal("Status completed callback. Type = application/octet-stream. Name = five_MB_test_file", output[1])

			// Necessary for wp-info to update
			time.Sleep(5 * time.Second)

			// Get the new Write-Pool info after upload
			output, err = writePoolInfo(t, configPath)
			if err != nil {
				t.Errorf("Failed to fetch Write Pool info due to error %v", err)
			}

			finalWritePool := []cli_model.WritePoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &finalWritePool)
			if err != nil {
				t.Errorf("Error unmarshalling write pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Equal(allocationID, finalWritePool[0].Id)
			assert.Equal(0.8, intToZCN(finalWritePool[0].Balance))
			assert.IsType(int64(1), finalWritePool[0].ExpireAt)
			assert.Equal(allocationID, finalWritePool[0].AllocationId)
			assert.Less(0, len(finalWritePool[0].Blobber))
			assert.Equal(true, finalWritePool[0].Locked)

			// Blobber pool balance should reduce by (write price*filesize) for each blobber
			for i := 0; i < len(finalWritePool[0].Blobber); i += 1 {
				assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
				assert.IsType(int64(1), finalWritePool[0].Blobber[i].Balance)

				// deduce tokens
				assert.InDelta(intToZCN(initialWritePool[0].Blobber[i].Balance)-intToZCN(finalWritePool[0].Blobber[i].Balance), blobberWritePrice[finalWritePool[0].Blobber[i].BlobberID]*0.005, epsilon, "Error should be within epsilon")
			}
		})

	})
}

func newAllocation(t *testing.T, cliConfigFilename string, lock float64) ([]string, error) {
	return cli_utils.RunCommand(fmt.Sprintf("./zbox newallocation --lock %v --expire 300s --size 10485760 --silent --wallet %v_wallet.json  --configDir ./config --config %s", lock, escapedTestName(t), cliConfigFilename))
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
