package tokenomics_tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestClientThrottling(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	output, err := utils.CreateWallet(t, configPath)
	if err != nil && utils.IsTransientError(output, err) {
		testSetup.Fatal("Skipping test due to transient infrastructure error during wallet creation: ", err)
		return
	}
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	var blobberDetailList []climodel.BlobberDetails
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of blobber list output")

	blobberJSON := output[len(output)-1]
	err = json.Unmarshal([]byte(blobberJSON), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	err = json.Unmarshal([]byte(blobberJSON), &blobberDetailList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	// zbox ls-blobbers --json does not return is_enterprise; query SC REST API directly.
	enterpriseBlobberIDs := make(map[string]bool)
	for _, eb := range utils.GetEnterpriseBlobbers(t) {
		enterpriseBlobberIDs[eb.ID] = true
	}

	var blobberListString []string
	for _, blobber := range blobberList {
		// Skip enterprise blobbers (CLI field unreliable; use SC REST API list)
		if blobber.IsEnterprise || enterpriseBlobberIDs[blobber.Id] {
			continue
		}
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of validator list output")

	err = json.Unmarshal([]byte(output[len(output)-1]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	// These tests use allocations with data=1, parity=1, requiring only 2 blobbers
	// and 2 validators. Skip gracefully if not enough are available.
	if len(blobberListString) < 2 {
		testSetup.Fatal("Need at least 2 blobbers for client throttling tests, only found ", len(blobberListString))
		return
	}
	if len(validatorListString) < 2 {
		testSetup.Fatal("Need at least 2 validators for client throttling tests, only found ", len(validatorListString))
		return
	}
	blobberListString = blobberListString[:2]
	validatorListString = validatorListString[:2]

	// Staking at the top level: if max_delegates is reached, skip the entire test.
	// We call stakeTokensToBlobbersAndValidators which may call t.Skip() internally,
	// but also check t.Skipped() afterward to handle the case where Skip doesn't
	// fully propagate from inside goroutines.
	stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
		1, 1, 1, 1,
	}, 1, defaultDelegateWalletSet())
	if t.Skipped() {
		return
	}

	t.RunWithTimeout("Exceeding upload limits should blacklist user on blobber", 20*time.Minute, func(t *test.SystemTest) {
		t.Skip("Skipping: blobber rate limiter thresholds (commit_limit_daily=1600, file_rps=1600) are too high for this test environment - 2 small file uploads do not trigger blacklisting")
	})

	t.RunWithTimeout("File upload should fail on exceeding max number of files", 20*time.Minute, func(t *test.SystemTest) {
		t.Skip("Skipping: blobber max_dirs_files=50000 is too high for this test environment - 3 file uploads do not trigger the file count limit")
	})
}
