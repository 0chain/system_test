package cli_tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnhealthyBlobberScenarios(t *testing.T) {
	t.Run("Cancel Allocation Should Work when blobber fails challenges", func(t *testing.T) {
		t.Skip("Skipping cancel allocation test until correct chain setup is implemented")
		t.Parallel()

		allocationID := setupAllocation(t, configPath)

		output, err := cancelAllocation(t, configPath, allocationID)

		require.Nil(t, err, "error canceling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		assertOutputMatchesAllocationRegex(t, regexp.MustCompile(`^Allocation canceled with txId : [a-f0-9]{64}$`), output[0])
	})
}
