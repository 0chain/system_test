package cli_tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const maxDescriptionLength = "max_description_length"
const maxDestinations = "max_destinations"
const maxDuration = "max_duration"
const minDuration = "min_duration"
const minLock = "min_lock"
const ownerId = "owner_id"

func TestVestingPool(t *testing.T) {
	// t.Parallel()

	// get current valid vesting configs
	output, err := getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output[4:])

	t.Run("Vesting pool with single destination, valid duration and valid tokens should work", func(t *testing.T) {
		t.Parallel()
		fmt.Println(vpConfigMap[maxDuration])

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet")

		targetWallet := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		validDuration := getValidDuration(t, vpConfigMap)

	})
}

func configFromKeyValuePair(output []string) map[string]interface{} {
	config := make(map[string]interface{})
	for _, keyValuePair := range output {
		pair := strings.Split(keyValuePair, "\t")
		key := strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])

		intValue, err := strconv.Atoi(value)
		if err == nil {
			config[key] = intValue
			continue
		}
		floatValue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			config[key] = floatValue
			continue
		}
		// string value
		config[key] = value
	}
	return config
}

func getValidDuration(t *testing.T, vpConfigMap map[string]interface{}) int64 {
	var maxDurationInSeconds int64
	if maxDurationString, ok := vpConfigMap[maxDuration].(string); ok {
		maxDurationInSeconds = durationToSeconds(t, maxDurationString)
	}
	var minDurationInSeconds int64
	if minDurationString, ok := vpConfigMap[minDuration].(string); ok {
		minDurationInSeconds = durationToSeconds(t, minDurationString)
	}

	validDuration := maxDurationInSeconds + minDurationInSeconds/2
	return validDuration
}

func durationToSeconds(t *testing.T, duration string) int64 {
	var seconds int64
	hour, err := strconv.Atoi(strings.Split(duration, "h")[0])
	require.Nil(t, err, "error extracting hours from duration")
	seconds += int64(hour * 60 * 60)
	minute, err := strconv.Atoi(strings.Split(strings.Split(duration, "h")[1], "m")[0])
	require.Nil(t, err, "error extracting minute from duration")
	seconds += int64(minute * 60)
	second, err := strconv.Atoi(strings.Split(strings.Split(strings.Split(duration, "h")[1], "m")[1], "s")[0])
	seconds += int64(second)

	return seconds
}
