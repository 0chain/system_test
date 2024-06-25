package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Compares0boxTablesWithSharder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Compare 0box tables with sharder tables")

	t.RunSequentially("Compare blobbers tables", func(t *test.SystemTest) {
		blobbersTable_Sharder, resp, err := apiClient.QueryDataFromSharder(t, "blobbers")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		blobbersTable_0box, resp, err := zboxClient.QueryDataFrom0box(t, "blobbers")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, blobbersTable_Sharder, blobbersTable_0box)

	},
	)
}
