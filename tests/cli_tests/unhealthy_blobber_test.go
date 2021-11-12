package cli_tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until we set up a byzantime chain/investigate conductor tests
*/
func TestUnhealthyBlobberScenarios(t *testing.T) {
	t.Run("Cancel Allocation Should Work when blobber fails challenges", func(t *testing.T) {
		t.Skip("allocations can only be canceled if 20% of blobbers fail 20 challenges, which we can't force on a vanilla chain as a black box test")
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID)

		require.Nil(t, err, "error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`), output[0])
	})

	t.Run("Cancel Other's Allocation Should Fail", func(t *testing.T) {
		t.Skip("allocations can only be canceled if 20% of blobbers fail 20 challenges, which we can't force on a vanilla chain as a black box test")
		t.Parallel()

		var otherAllocationID string
		// This test creates a separate wallet and allocates there, test nesting needed to create other wallet json
		t.Run("Get Other Allocation ID", func(t *testing.T) {
			otherAllocationID = setupAllocation(t, configPath)
		})

		// otherAllocationID should not be cancelable from this level
		output, err := cancelAllocation(t, configPath, otherAllocationID)

		require.NotNil(t, err, "expected error canceling allocation", strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		//FIXME: POSSIBLE BUG: Error message shows error in creating instead of error in canceling
		require.Equal(t, "Error creating allocation:[txn] too less sharders to confirm it: min_confirmation is 50%, "+
			"but got 0/2 sharders", output[len(output)-1])
	})

	t.Run("Cancel_allocation_immediately_should_succeed", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID)
		require.NoError(t, err, "cancel allocation failed but should succeed", strings.Join(output, "\n"))
	})
}
