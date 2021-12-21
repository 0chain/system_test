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
	validDuration := getValidDuration(t, vpConfigMap)

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

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
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

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":           targetWallet.ClientID + ":0.1",
			"lock":        0.1,
			"duration":    validDuration,
			"description": "this is a vesting pool",
		}), true)
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

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":     0.3,
			"duration": validDuration,
		}), true)
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

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":           targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":        0.3,
			"duration":    validDuration,
			"description": "this is a vesting pool",
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with locking insufficient tokens should fail", func(t *testing.T) {
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

		// add a vesting pool for sending 0.5 to target wallet by locking 0.1 tokens
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.5",
			"lock":     0.1,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when creating a vesting pool without insufficent locked tokens")
		require.Len(t, output, 1)
		require.Equal(t, "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}", output[0], "output did not match expected error message")
	})

	t.Run("Vesting pool with excess locked tokens should work and allow unlocking", func(t *testing.T) {
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

		// add a vesting pool for sending 0.1 to target wallet by locking 0.5 tokens
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.5,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Use vp-info to check excess tokens are shown as can be unlocked
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.GreaterOrEqual(t, len(output), 18, "expected output of length 18 atleast")
		require.Equal(t, output[2], "can unlock:   400.000 mZCN (excess)")
	})

	t.Run("Vesting pool with start time in future should work", func(t *testing.T) {
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

		startTime := time.Now().Add(5 * time.Second)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":0.1",
			"lock":       0.1,
			"duration":   validDuration,
			"start_time": startTime.Unix(),
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.GreaterOrEqual(t, len(output), 18, "expected output of length 18 atleast")
		require.Equal(t, output[7], "start_time:   "+time.Unix(startTime.Unix(), 0).String())
	})

	t.Run("Vesting pool with start time in future for multiple destination wallets should work", func(t *testing.T) {
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

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		startTime := time.Now().Add(5 * time.Second)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":       0.3,
			"duration":   validDuration,
			"start_time": startTime.Unix(),
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.GreaterOrEqual(t, len(output), 18, "expected output of length 18 atleast")
		require.Equal(t, output[7], "start_time:   "+time.Unix(startTime.Unix(), 0).String())
	})

	t.Run("Vesting pool with start time in past should fail", func(t *testing.T) {
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

		// subtract 5 seconds from now
		startTime := time.Now().Add(-5 * time.Second)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":0.1",
			"lock":       0.3,
			"duration":   validDuration,
			"start_time": startTime.Unix(),
		}), false)
		require.NotNil(t, err, "expected error when using past start_time")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	t.Run("Vesting pool with start time in past for multiple destinations should fail", func(t *testing.T) {
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

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// subtract 5 seconds from now
		startTime := time.Now().Add(-5 * time.Second)

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":       0.3,
			"duration":   validDuration,
			"start_time": startTime.Unix(),
		}), false)
		require.NotNil(t, err, "expected error when using past start_time")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	t.Run("Vesting pool with invalid destination should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        "abcdef123456:0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when using invalid address")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "parsing destinations: invalid destination id: \"abcdef123456\"")
	})

	t.Run("Vesting pool with one valid destination and one invalid destination should fail", func(t *testing.T) {
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

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1" + " --d abcdef123456:0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when using invalid address")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "parsing destinations: invalid destination id: \"abcdef123456\"")
	})

	t.Run("Vesting pool for duration less than min duration should fail", func(t *testing.T) {
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

		var minDurationInSeconds int64
		if minDurationString, ok := vpConfigMap[minDuration].(string); ok {
			minDurationInSeconds = durationToSeconds(t, minDurationString)
		}
		invalidDuration := strconv.FormatFloat(float64(minDurationInSeconds)-0.0001, 'f', -1, 64) + "s"

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": invalidDuration,
		}), false)
		require.NotNil(t, err, "expected error when using duration less than min duration")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	t.Run("Vesting pool with duration greater than max duration should fail", func(t *testing.T) {
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

		var maxDurationInSeconds int64
		if maxDurationString, ok := vpConfigMap[maxDuration].(string); ok {
			maxDurationInSeconds = durationToSeconds(t, maxDurationString)
		}
		invalidDuration := strconv.FormatFloat(float64(maxDurationInSeconds)+0.0001, 'f', -1, 64) + "s"

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": invalidDuration,
		}), false)
		require.NotNil(t, err, "expected error when using duration greater than max duration")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})
}

func vestingPoolAdd(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Adding a new vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-add "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-add " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func vestingPoolInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("fetching vesting pool info...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-info "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-info " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
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
