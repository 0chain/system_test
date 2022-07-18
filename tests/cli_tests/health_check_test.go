package cli_tests

import (
	"encoding/json"
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestHealthChecksBlobber(t *testing.T) {
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1, strings.Join(output, "\n"))

	var blobberList []climodel.BlobberDetails
	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(blobberList), 0, "blobber list is empty")

	intialBlobberInfo := blobberList[0]
	loc, _ := time.LoadLocation("UTC")

	t.Run("health check blobber", func(t *testing.T) {
		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "health_check_period",
			"values": "1h0m0s",
		}, false)
		require.Nil(t, err)
		cliutils.Wait(t, 30*time.Second)

		healthCheckPeriod := getHealthCheckPeriodSCConfig(t)
		require.Equal(t, "1h0m0s", healthCheckPeriod)

		require.False(t, intialBlobberInfo.IsShutDown)
		require.WithinDuration(
			t,
			time.Now().In(loc),
			time.Unix(intialBlobberInfo.LastHealthCheck, 0),
			1*time.Hour,
		)

		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "health_check_period",
			"values": "0h0m30s",
		}, false)
		require.Nil(t, err)
		cliutils.Wait(t, 30*time.Second)
		healthCheckPeriod = getHealthCheckPeriodSCConfig(t)
		require.Equal(t, "30s", healthCheckPeriod)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))
		fmt.Println(time.Unix(finalBlobberInfo.LastHealthCheck, 0))
		//require.Equal(t, "0h0m30s")
	})
}

func getHealthCheckPeriodSCConfig(t *testing.T) string {
	healthCheckPeriodMatcher := regexp.MustCompile("health_check_period[ \\t]+\\w+")
	output, err := getStorageSCConfig(t, configPath, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	healthCheckPeriodMatches := healthCheckPeriodMatcher.FindAllStringSubmatch(strings.Join(output, "\n"), 1)
	require.Len(t, healthCheckPeriodMatches, 1)
	healthCheckPeriodInfo := strings.Split(healthCheckPeriodMatches[0][0], "health_check_period")
	require.Len(t, healthCheckPeriodInfo, 2)
	return strings.TrimSpace(healthCheckPeriodInfo[1])
}
