package cli_tests

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

func TestBlobberStorageRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.RunWithTimeout("Finalize Expired Allocation Should Work", 5*time.Minute, func(t *test.SystemTest) {
		//TODO: unacceptably slow

		options := map[string]interface{}{
			"expire": "1h",
			"size":   "1024",
			"parity": "1",
			"lock":   "0.5",
			"data":   "1",
		}
		output, err := createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		cliutils.Wait(t, 4*time.Minute)

		output, err = finalizeAllocation(t, configPath, allocationID, false)

		require.Nil(t, err, "unexpected error updating allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		matcher := regexp.MustCompile("Allocation finalized with txId .*$")
		require.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")
	})
}
