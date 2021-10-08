package cli_tests

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestUnhealthyBlobberScenarios(t *testing.T) {
	t.Run("Cancel Allocation Should Work when blobber fails challenges", func(t *testing.T) {
		t.Skip("Skipping cancel allocation test until correct chain setup is implemented")
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID)

		require.Nil(t, err, "error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, reCancelAllocation, output[0])
	})
}
