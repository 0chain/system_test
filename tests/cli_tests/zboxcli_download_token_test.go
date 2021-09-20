package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
)

func TestFileDownloadTokenMovement(t *testing.T) {
	assert := assert.New(t)

	balance := 0.4 // 800.000 mZCN
	t.Run("Parallel", func(t *testing.T) {
		t.Run("Read pool must have no tokens locked for a newly created allocation", func(t *testing.T) {
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

			output, err = readPoolInfo(t, configPath, allocationID)
			if err != nil {
				t.Errorf("Error fetching read pool due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Len(output, 1)
			assert.Equal("no tokens locked", output[0])
		})

		t.Run("Locked read pool tokens should equal total blobber balance in read pool", func(t *testing.T) {
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

			output, err = readPoolLock(t, configPath, allocationID, 0.4)
			if err != nil {
				t.Errorf("Tokens could not be locked due to error: %v. The CLI output was: %v.", err, strings.Join(output, "\n"))
			}

			assert.Len(output, 1)
			assert.Equal("locked", output[0])

			output, err = readPoolInfo(t, configPath, allocationID)
			if err != nil {
				t.Errorf("Error fetching read pool due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			readPool := []cli_model.ReadPoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &readPool)
			if err != nil {
				t.Errorf("Error unmarshalling read pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), readPool[0].Id)
			assert.Equal(0.4, intToZCN(readPool[0].Balance))
			assert.IsType(int64(1), readPool[0].ExpireAt)
			assert.Equal(allocationID, readPool[0].AllocationId)
			assert.Less(0, len(readPool[0].Blobber))
			assert.Equal(true, readPool[0].Locked)

			balanceInTotal := float64(0)
			for i := 0; i < len(readPool[0].Blobber); i += 1 {
				assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), readPool[0].Blobber[i].BlobberID)
				assert.IsType(int64(1), readPool[0].Blobber[i].Balance)
				balanceInTotal += intToZCN(readPool[0].Blobber[i].Balance)
			}

			assert.InEpsilon(0.4, balanceInTotal, epsilon, "Error should be within epsilon")
		})

		t.Run("Each blobber's read pool balance should reduce by download cost", func(t *testing.T) {
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

			// upload a dummy 5 MB file
			output, err = uploadFile(t, configPath, allocationID, "../../internal/dummy_file/five_MB_test_file", "/")
			if err != nil {
				t.Errorf("Upload file failed due to error: %v", err)
			}

			assert.Equal(2, len(output))
			assert.Equal("Status completed callback. Type = application/octet-stream. Name = five_MB_test_file", output[1])

			// Lock read pool tokens
			output, err = readPoolLock(t, configPath, allocationID, 0.4)
			if err != nil {
				t.Errorf("Tokens could not be locked due to error: %v. The CLI output was: %v.", err, strings.Join(output, "\n"))
			}

			assert.Len(output, 1)
			assert.Equal("locked", output[0])

			// Read pool before download
			output, err = readPoolInfo(t, configPath, allocationID)
			if err != nil {
				t.Errorf("Error fetching read pool due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			initialReadPool := []cli_model.ReadPoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &initialReadPool)
			if err != nil {
				t.Errorf("Error unmarshalling read pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Id)
			assert.Equal(0.4, intToZCN(initialReadPool[0].Balance))
			assert.IsType(int64(1), initialReadPool[0].ExpireAt)
			assert.Equal(allocationID, initialReadPool[0].AllocationId)
			assert.Less(0, len(initialReadPool[0].Blobber))
			assert.Equal(true, initialReadPool[0].Locked)

			for i := 0; i < len(initialReadPool[0].Blobber); i += 1 {
				assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Blobber[i].BlobberID)
				assert.IsType(int64(1), initialReadPool[0].Blobber[i].Balance)
			}

			output, err = getDownloadCostInInt(t, configPath, allocationID, "/five_MB_test_file")
			if err != nil {
				t.Errorf("Could not get download cost due to error: %v. The CLI output was: %v.", err, strings.Join(output, "\n"))
			}
			expectedDownloadCost, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
			if err != nil {
				t.Errorf("Cost couldn't be parsed to float due to error: %v", err)
			}
			unit := strings.Fields(output[0])[1]
			expectedDownloadCost = unitToZCN(expectedDownloadCost, unit)

			// Download the file
			output, err = downloadFile(t, configPath, allocationID, "../../internal/dummy_file/five_MB_test_file_dowloaded", "/five_MB_test_file")
			if err != nil {
				t.Errorf("Downloading the file failed due to error: %v. The CLI output was: %v.", err, strings.Join(output, "\n"))
			}
			defer os.Remove("../../internal/dummy_file/five_MB_test_file_dowloaded")

			assert.Len(output, 2)
			assert.Equal("Status completed callback. Type = application/octet-stream. Name = five_MB_test_file", output[1])

			// Necessary for rp-info to update
			time.Sleep(5 * time.Second)

			// Read pool before download
			output, err = readPoolInfo(t, configPath, allocationID)
			if err != nil {
				t.Errorf("Error fetching read pool due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			finalReadPool := []cli_model.ReadPoolInfo{}
			err = json.Unmarshal([]byte(output[0]), &finalReadPool)
			if err != nil {
				t.Errorf("Error unmarshalling read pool info due to: %v. The CLI output was: %v", err, strings.Join(output, "\n"))
			}

			assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Id)
			assert.Less(intToZCN(finalReadPool[0].Balance), 0.4)
			assert.IsType(int64(1), finalReadPool[0].ExpireAt)
			assert.Equal(allocationID, finalReadPool[0].AllocationId)
			assert.Equal(len(initialReadPool[0].Blobber), len(finalReadPool[0].Blobber))
			assert.Equal(true, finalReadPool[0].Locked)

			for i := 0; i < len(finalReadPool[0].Blobber); i += 1 {
				assert.Regexp(regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Blobber[i].BlobberID)
				assert.IsType(int64(1), finalReadPool[0].Blobber[i].Balance)

				// amount deducted
				assert.InDelta(expectedDownloadCost, intToZCN(initialReadPool[0].Blobber[i].Balance)-intToZCN(finalReadPool[0].Blobber[i].Balance), epsilon, "Error should be within epsilon.")
			}
		})
	})
}

func readPoolInfo(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	return cli_utils.RunCommand("./zbox rp-info --allocation " + allocationID + " --json --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func readPoolLock(t *testing.T, cliConfigFilename string, allocationID string, tokens float64) ([]string, error) {
	return cli_utils.RunCommand(fmt.Sprintf("./zbox rp-lock --allocation %s --tokens %v --duration 900s --silent --wallet %s_wallet.json --configDir ./config --config %s", allocationID, tokens, escapedTestName(t), cliConfigFilename))
}

func getDownloadCostInInt(t *testing.T, cliConfigFilename string, allocationID string, remotepath string) ([]string, error) {
	return cli_utils.RunCommand("./zbox get-download-cost --allocation " + allocationID + " --remotepath " + remotepath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func downloadFile(t *testing.T, cliConfigFilename string, allocation string, localpath string, remotepath string) ([]string, error) {
	return cli_utils.RunCommand("./zbox download --allocation " + allocation + " --localpath " + localpath + " --remotepath " + remotepath + " --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func unitToZCN(expectedDownloadCost float64, unit string) float64 {
	switch unit {
	case "SAS":
		expectedDownloadCost /= 1e10
		return expectedDownloadCost
	case "uZCN":
		expectedDownloadCost /= 1e6
		return expectedDownloadCost
	case "mZCN":
		expectedDownloadCost /= 1e3
		return expectedDownloadCost
	case "ZCN":
		expectedDownloadCost /= 1e1
		return expectedDownloadCost
	}
	return expectedDownloadCost
}
