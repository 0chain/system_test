package api_tests

import (
	"encoding/json"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Compares0boxTablesWithSharder(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Compare 0box tables with sharder tables")

	t.RunSequentially("Compare blobbers tables", func(t *test.SystemTest) {
		blobbersTable_Sharder, resp, err := apiClient.V1SCRestGetAllBlobbers(t, client.HttpOkStatus)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		var blobberIds []string
		for _, blobber := range blobbersTable_Sharder {
			blobberIds = append(blobberIds, blobber.ID)
		}
		blobberIdsJson, err := json.Marshal(blobberIds)
		require.NoError(t, err)
		blobberIdsStr := string(blobberIdsJson)
		blobbersTable_0box, resp, err := zboxClient.GetBlobbers(t, blobberIdsStr)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Equal(t, len(blobbersTable_Sharder), len(blobbersTable_0box))
		for i, blobber := range blobbersTable_Sharder {
			require.Equal(t, blobber.ID, blobbersTable_0box[i].ID)
			require.Equal(t, blobber.BaseURL, blobbersTable_0box[i].BaseURL)
			require.Equal(t, blobber.Terms, blobbersTable_0box[i].Terms)
			require.Equal(t, blobber.Capacity, blobbersTable_0box[i].Capacity)
			require.Equal(t, blobber.Allocated, blobbersTable_0box[i].Allocated)
			require.Equal(t, blobber.LastHealthCheck, blobbersTable_0box[i].LastHealthCheck)
			require.Equal(t, blobber.PublicKey, blobbersTable_0box[i].PublicKey)
			require.Equal(t, blobber.StakePoolSettings, blobbersTable_0box[i].StakePoolSettings)
			require.Equal(t, blobber.TotalStake, blobbersTable_0box[i].TotalStake)
			require.Equal(t, blobber.SavedData, blobbersTable_0box[i].SavedData)
			require.Equal(t, blobber.ReadData, blobbersTable_0box[i].ReadData)
			require.Equal(t, blobber.ChallengesPassed, blobbersTable_0box[i].ChallengesPassed)
			require.Equal(t, blobber.ChallengesCompleted, blobbersTable_0box[i].ChallengesCompleted)
		}

	})
}
