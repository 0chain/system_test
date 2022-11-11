package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
)

func TestGetSCStats(t *testing.T) {
	t.Parallel()

	t.Run("Get miner stats call should return successfully", func(t *testing.T) {
		t.Parallel()

		minerGetStatsResponse, resp, err := apiClient.V1MinerGetStats(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, minerGetStatsResponse)
		require.NotZero(t, minerGetStatsResponse.BlockFinality)
		require.NotZero(t, minerGetStatsResponse.LastFinalizedRound)
		require.NotZero(t, minerGetStatsResponse.BlocksFinalized)
		require.GreaterOrEqual(t, minerGetStatsResponse.StateHealth, int64(-1))
		require.NotZero(t, minerGetStatsResponse.CurrentRound)
		require.GreaterOrEqual(t, minerGetStatsResponse.RoundTimeout, int64(0))
		require.GreaterOrEqual(t, minerGetStatsResponse.Timeouts, int64(0))
		require.NotZero(t, minerGetStatsResponse.AverageBlockSize)
		require.NotNil(t, minerGetStatsResponse.NetworkTime)
	})

	t.Run("Get sharder stats call should return successfully", func(t *testing.T) {
		t.Parallel()

		sharderGetStatsResponse, resp, err := apiClient.V1SharderGetStats(client.HttpOkStatus)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, sharderGetStatsResponse)
		require.NotZero(t, sharderGetStatsResponse.LastFinalizedRound)
		require.GreaterOrEqual(t, sharderGetStatsResponse.StateHealth, int64(-1))
		require.NotZero(t, sharderGetStatsResponse.AverageBlockSize)
		require.GreaterOrEqual(t, sharderGetStatsResponse.PrevInvocationCount, uint64(0))
		require.NotZero(t, sharderGetStatsResponse.PrevInvocationScanTime)
		require.GreaterOrEqual(t, sharderGetStatsResponse.MeanScanBlockStatsTime, float64(0))
	})
}
