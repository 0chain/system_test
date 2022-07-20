package cli_tests

import (
	"encoding/json"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestHealthChecks(t *testing.T) {
	timeLoc, _ := time.LoadLocation("UTC")
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	t.Run("health check blobber", func(t *testing.T) {
		healthCheckPeriod := getHealthCheckPeriodSCConfig(t)
		require.Equal(t, "1h0m0s", healthCheckPeriod)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		for _, blobberInfo := range blobberList {
			require.False(t, blobberInfo.IsShutDown)
			require.WithinDuration(
				t,
				time.Now().In(timeLoc),
				time.Unix(blobberInfo.LastHealthCheck, 0),
				1*time.Hour,
			)
		}
	})

	t.Run("health check validators", func(t *testing.T) {
		output, err = listValidators(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var validatorList []climodel.Validator
		err = json.Unmarshal([]byte(output[0]), &validatorList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(validatorList), 0, "validator list is empty")

		for _, validatorInfo := range validatorList {
			require.False(t, validatorInfo.IsShutDown)
			require.WithinDuration(
				t,
				time.Now().In(timeLoc),
				time.Unix(validatorInfo.LastHealthCheck, 0),
				1*time.Hour,
			)
		}
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
