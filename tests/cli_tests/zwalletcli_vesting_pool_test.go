package cli_tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const maxDescriptionLength = "max_description_length"
const maxDestinations = "max_destinations"
const maxDuration = "max_duration"
const minDuration = "min_duration"
const minLock = "min_lock"
const ownerId = "owner_id"

func TestVestingPool(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)

	t.Run("Vesting pool with single destination, valid duration and valid tokens should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		validDuration := getValidDuration(t, vpConfigMap)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}))
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with single destination and description should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		validDuration := getValidDuration(t, vpConfigMap)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":           targetWallet.ClientID + ":0.1",
			"lock":        0.1,
			"duration":    validDuration,
			"description": "this is a vesting pool",
		}))
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with multiple destinations should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		validDuration := getValidDuration(t, vpConfigMap)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":     0.3,
			"duration": validDuration,
		}))
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with multiple destinations and description should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		validDuration := getValidDuration(t, vpConfigMap)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":           targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":        0.3,
			"duration":    validDuration,
			"description": "this is a vesting pool",
		}))
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})
}

func vestingPoolAdd(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Adding a new vesting pool...")
	return cliutils.RunCommand(t, "./zwallet vp-add "+params+
		" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
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

func getValidDuration(t *testing.T, vpConfigMap map[string]interface{}) string {
	var maxDurationInSeconds int64
	if maxDurationString, ok := vpConfigMap[maxDuration].(string); ok {
		maxDurationInSeconds = durationToSeconds(t, maxDurationString)
	}
	var minDurationInSeconds int64
	if minDurationString, ok := vpConfigMap[minDuration].(string); ok {
		minDurationInSeconds = durationToSeconds(t, minDurationString)
	}

	validDuration := strconv.FormatInt((maxDurationInSeconds+minDurationInSeconds)/2, 10) + "s"
	return validDuration
}

func durationToSeconds(t *testing.T, duration string) int64 {
	var seconds int64
	if strings.Contains(duration, "h") {
		hour, err := strconv.Atoi(strings.Split(duration, "h")[0])
		require.Nil(t, err, "error extracting hours from duration")
		seconds += int64(hour * 60 * 60)
		duration = strings.Split(duration, "h")[1]
	}
	if strings.Contains(duration, "m") {
		minute, err := strconv.Atoi(strings.Split(duration, "m")[0])
		require.Nil(t, err, "error extracting minute from duration")
		seconds += int64(minute * 60)
		duration = strings.Split(duration, "m")[1]
	}
	if strings.Contains(duration, "s") {
		second, err := strconv.Atoi(strings.Split(duration, "s")[0])
		require.Nil(t, err, "error extracting seconds from duration")
		seconds += int64(second)
	}
	return seconds
}
