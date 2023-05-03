package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
)

func TestBlockRewardsForBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := utils.GetLatestFinalizedBlock(t)

	fmt.Println("prevBlock", prevBlock)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	readPrices := [][]float64{{0, 0.01}, {0, 0}}
	writePrices := [][]float64{{0.1, 0.9}}
	readData := [][]int{{1, 1}, {1, 9}}
	totalData := 0.1 * GB
	stakes := [][]float64{{1.0, 1.0, 1.0, 1.0}, {1.0, 2.0, 1.0, 2.0}}

	var descriptions []string
	descriptions = append(descriptions, "Blobber Block Reward Test - 1")

	idx := 0

	for _, readPrice := range readPrices {
		for _, writePrice := range writePrices {
			for _, readData := range readData {
				for _, stake := range stakes {
					t.RunSequentiallyWithTimeout(descriptions[idx], (500*time.Minute)+(40*time.Second), func(t *test.SystemTest) {
						stakeTokensToBlobbersAndValidators(t, blobberList, validatorList, configPath, stake, 1)

						output, err := utils.CreateWallet(t, configPath)
						require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

						// 1. Create an allocation with 1 data shard and 1 parity shard.
						allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
							"size":   1 * GB,
							"tokens": 1,
							"data":   1,
							"parity": 1,
							"expire": "20m",
						})
						fmt.Println("Allocation ID : ", allocationId)

						remotepath := "/dir/"
						filesize := totalData
						filename := utils.GenerateRandomTestFileName(t)

						err = utils.CreateFileWithSize(filename, int64(filesize))
						require.Nil(t, err)

						output, err = utils.UploadFile(t, configPath, map[string]interface{}{
							// fetch the latest block in the chain
							"allocation": allocationId,
							"remotepath": remotepath + filepath.Base(filename),
							"localpath":  filename,
						}, true)
						require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

						for i := 0; i < readData[0]; i++ {
							err = os.Remove(filename)

							remoteFilepath := remotepath + filepath.Base(filename)

							output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
								"allocation": allocationId,
								"remotepath": remoteFilepath,
								"localpath":  os.TempDir() + string(os.PathSeparator),
								"blobber_id": blobberList[0].Id,
							}), true)
							require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
						}

						for i := 0; i < readData[1]; i++ {
							err = os.Remove(filename)

							remoteFilepath := remotepath + filepath.Base(filename)

							output, err = utils.DownloadFile(t, configPath, utils.CreateParams(map[string]interface{}{
								"allocation": allocationId,
								"remotepath": remoteFilepath,
								"localpath":  os.TempDir() + string(os.PathSeparator),
								"blobber_id": blobberList[1].Id,
							}), true)
							require.Nil(t, err, "error downloading file", strings.Join(output, "\n"))
						}

						// Sleep for 10 minutes
						time.Sleep(10 * time.Minute)
						curBlock := utils.GetLatestFinalizedBlock(t)

						blobber1PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[0].Id)
						blobber2PassedChallenges := countPassedChallengesForBlobberAndAllocation(t, allocationId, blobberList[1].Id)

						blobberBlockRewards := getBlockRewards(t, strconv.FormatInt(prevBlock.Round, 10), strconv.FormatInt(curBlock.Round, 10), blobberList[0].Id, blobberList[1].Id)

						blobber1ProviderRewards := float64(blobberBlockRewards[0])
						blobber2ProviderRewards := float64(blobberBlockRewards[1])
						blobber1DelegateRewards := float64(blobberBlockRewards[2])
						blobber2DelegateRewards := float64(blobberBlockRewards[3])
						blobber1TotalRewards := float64(blobberBlockRewards[4])
						blobber2TotalRewards := float64(blobberBlockRewards[5])

						blobber1Weight := calculateWeight(writePrice[0], readPrice[0], totalData, float64(readData[0])*totalData, stake[0], blobber1PassedChallenges)
						blobber2Weight := calculateWeight(writePrice[1], readPrice[1], totalData, float64(readData[1])*totalData, stake[1], blobber2PassedChallenges)

						// print all values
						fmt.Println("blobber1ProviderRewards", blobber1ProviderRewards)
						fmt.Println("blobber2ProviderRewards", blobber2ProviderRewards)
						fmt.Println("blobber1DelegateRewards", blobber1DelegateRewards)
						fmt.Println("blobber2DelegateRewards", blobber2DelegateRewards)
						fmt.Println("blobber1TotalRewards", blobber1TotalRewards)
						fmt.Println("blobber2TotalRewards", blobber2TotalRewards)

						require.InEpsilon(t, blobber1TotalRewards/blobber2TotalRewards, blobber1Weight/blobber2Weight, 0.05, "Total rewards not distributed correctly")

						prevBlock = utils.GetLatestFinalizedBlock(t)

						unstakeTokensForBlobbersAndValidators(t, blobberList, validatorList, configPath, 1)

					})

					break
				}
			}

		}
	}
}

func getBlockRewards(t *test.SystemTest, startBlockNumber, endBlockNumber, blobber1, blobber2 string) []int64 {
	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/block-rewards?start_block_number=" + startBlockNumber + "&end_block_number=" + endBlockNumber)
	var response map[string]interface{}

	res, _ := http.Get(url)

	// decode and save the res body to response
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil
	}

	var result []int64

	var blobber1TotalReward int64
	blobber1TotalReward = 0
	var blobber2TotalReward int64
	blobber2TotalReward = 0

	var blobber1ProviderReward int64
	blobber1ProviderReward = 0
	var blobber2ProviderReward int64
	blobber2ProviderReward = 0

	for _, providerReward := range response["provider_rewards"].([]interface{}) {
		providerId := providerReward.(map[string]interface{})["provider_id"].(string)
		amount := int64(providerReward.(map[string]interface{})["amount"].(float64))

		if providerId == blobber1 {
			blobber1TotalReward += amount
			blobber1ProviderReward += amount
		} else if providerId == blobber2 {
			blobber2TotalReward += amount
			blobber2ProviderReward += amount
		}
	}

	var blobber1DelegateReward int64
	blobber1DelegateReward = 0
	var blobber2DelegateReward int64
	blobber2DelegateReward = 0

	for _, delegateRewards := range response["delegate_rewards"].([]interface{}) {
		providerId := delegateRewards.(map[string]interface{})["provider_id"].(string)
		amount := int64(delegateRewards.(map[string]interface{})["amount"].(float64))

		if providerId == blobber1 {
			blobber1TotalReward += amount
			blobber1DelegateReward += amount
		} else if providerId == blobber2 {
			blobber2TotalReward += amount
			blobber2DelegateReward += amount
		}
	}

	result = append(result, blobber1ProviderReward)
	result = append(result, blobber2ProviderReward)
	result = append(result, blobber1DelegateReward)
	result = append(result, blobber2DelegateReward)

	result = append(result, blobber1TotalReward)
	result = append(result, blobber2TotalReward)

	return result
}

func getZeta(wp, rp float64) float64 {

	i := float64(1)
	k := float64(0.9)
	mu := float64(0.2)

	if wp == 0 {
		return 0
	}

	return i - (k * (rp / (rp + (mu * wp))))
}

func getGamma(X, R float64) float64 {

	A := float64(10)
	B := float64(1)
	alpha := float64(0.2)

	if X == 0 {
		return 0
	}

	factor := math.Abs((alpha*X - R) / (alpha*X + R))
	return A - B*factor
}

func calculateWeight(wp, rp, X, R, stakes, challenges float64) float64 {

	zeta := getZeta(wp, rp)
	gamma := getGamma(X, R)

	return (zeta*gamma + 1) * stakes * challenges
}

func resetNetwork(readPrice, writePrice float64) {
	fmt.Println("Read Price : ", readPrice)
	fmt.Println("Write Price : ", writePrice)

	time.Sleep(15 * time.Minute)
}

func countPassedChallengesForBlobberAndAllocation(t *test.SystemTest, allocationID, blobberID string) float64 {
	challenges, _ := getAllChallenges(t, allocationID)

	var passedChallenges float64
	passedChallenges = 0

	for _, challenge := range challenges {
		if challenge.Passed && challenge.BlobberID == blobberID {
			passedChallenges++
		}
	}

	return passedChallenges
}
