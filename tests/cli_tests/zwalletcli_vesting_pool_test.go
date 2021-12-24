package cli_tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const maxDescriptionLength = "max_description_length"
const maxDestinations = "max_destinations"
const maxDuration = "max_duration"
const minDuration = "min_duration"
const minLock = "min_lock"

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
		require.NotNil(t, err, "expected error when creating a vesting pool without insufficient locked tokens")
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

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
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

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
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

	t.Run("Vesting pool with lock less than min lock should fail", func(t *testing.T) {
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

		var invalidLockAmount float64
		if minLockAmount, ok := vpConfigMap[minLock].(float64); ok {
			invalidLockAmount = minLockAmount - 0.0001
		}

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":" + strconv.FormatFloat(invalidLockAmount, 'f', -1, 64),
			"lock":     invalidLockAmount,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when using lock less than min lock")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	t.Run("Vesting pool with description greater than max description length should fail", func(t *testing.T) {
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

		var invalidDescription string
		if maxDescriptionLengthAllowed, ok := vpConfigMap[maxDescriptionLength].(int); ok {
			invalidDescription = cliutils.RandomAlphaNumericString(int(maxDescriptionLengthAllowed + 1))
		}

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":           targetWallet.ClientID + ":0.1",
			"lock":        0.1,
			"duration":    validDuration,
			"description": invalidDescription,
		}), false)
		require.NotNil(t, err, "expected error when using description length greater than max allowed")
		require.Len(t, output, 1)
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	t.Run("Vesting pool with destinations greater than max destinations should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		var invalidDestinations int
		if maxDestinationsAllowed, ok := vpConfigMap[maxDestinations].(int); ok {
			invalidDestinations = maxDestinationsAllowed + 1
		}

		output, err = executeFaucetWithTokens(t, configPath, float64(invalidDestinations)*0.1)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWallets := make([]*climodel.Wallet, invalidDestinations)
		var destinationString string
		for i := 0; i < invalidDestinations; i++ {
			targetWalletName := "targetWallet" + strconv.Itoa(i) + escapedTestName(t)
			output, err = registerWalletForName(t, configPath, targetWalletName)
			require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

			targetWallets[i], err = getWalletForName(t, configPath, targetWalletName)
			require.Nil(t, err, "error fetching destination wallet")

			destinationString += targetWallets[i].ClientID + ":0.1 --d "
		}
		destinationString = destinationString[:len(destinationString)-5]
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        destinationString,
			"lock":     float64(invalidDestinations) * 0.1,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when using more destinations than allowed")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, output[0], "Failed to add vesting pool: {\"error\": \"verify transaction failed\"}")
	})

	// Feature to add: vp-info should have a json flag, it already has models in place in gosdk
	t.Run("Vesting pool info with valid pool_id should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		clientWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": validDuration,
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
		require.Equal(t, output[0], "pool_id:      "+poolId)
		require.Equal(t, output[1], "balance:      300.000 mZCN")
		require.Equal(t, output[2], "can unlock:   200.000 mZCN (excess)")
		require.Equal(t, output[3], "sent:         0 SAS (real value)")
		require.Equal(t, output[4], "pending:      100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested:       \d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[5])
		require.Equal(t, output[6], "description:")
		require.Regexp(t, regexp.MustCompile("start_time:   [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[7])
		require.Regexp(t, regexp.MustCompile("expire_at:    [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[8])
		require.Equal(t, output[9], "destinations:")
		require.Equal(t, output[10], "- id:          "+targetWallet.ClientID)
		require.Equal(t, output[11], "vesting:     100.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[12])
		require.Equal(t, output[13], "sent:        0 SAS (real value)")
		require.Equal(t, output[14], "pending:     100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[15])
		require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[16])
		require.Equal(t, output[17], "client_id:    "+clientWallet.ClientID)
	})

	t.Run("Vesting pool info for pool with multiple destinations should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		clientWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

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
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.GreaterOrEqual(t, len(output), 18, "expected output of length 18 atleast")
		require.Equal(t, output[0], "pool_id:      "+poolId)
		require.Equal(t, output[1], "balance:      300.000 mZCN")
		require.Equal(t, output[2], "can unlock:   0 SAS (excess)")
		require.Equal(t, output[3], "sent:         0 SAS (real value)")
		require.Equal(t, output[4], "pending:      300.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested:       \d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[5])
		require.Equal(t, output[6], "description:")
		require.Regexp(t, regexp.MustCompile("start_time:   [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[7])
		require.Regexp(t, regexp.MustCompile("expire_at:    [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[8])
		require.Equal(t, output[9], "destinations:")
		require.Equal(t, output[10], "- id:          "+targetWallet.ClientID)
		require.Equal(t, output[11], "vesting:     100.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[12])
		require.Equal(t, output[13], "sent:        0 SAS (real value)")
		require.Equal(t, output[14], "pending:     100.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[15])
		require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[16])
		require.Equal(t, output[17], "- id:          "+targetWallet2.ClientID)
		require.Equal(t, output[18], "vesting:     200.000 mZCN")
		require.Regexp(t, regexp.MustCompile(`can unlock: {2}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[19])
		// FIXME: multiple destinations info not printing complete info for all destinations
		// require.Equal(t, output[20], "sent:        0 SAS (real value)")
		require.Equal(t, output[20], "pending:     200.000 mZCN (not sent, real value)")
		require.Regexp(t, regexp.MustCompile(`vested: {6}\d*\.?\d+ [um]ZCN \(virtual, time based value\)`), output[21])
		// FIXME: multiple destinations info not printing complete info for all destinations
		// require.Regexp(t, regexp.MustCompile("last unlock: [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9 +-]{5} [A-Z]{3}"), output[23])
		require.Equal(t, output[22], "client_id:    "+clientWallet.ClientID)
	})

	// FIXME: vp-info can show information of vp belonging to other wallets
	t.Run("Vesting pool info for pool belonging to someone else's wallet must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error getting faucet tokens on foreign wallet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// FIXME: should get error
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info", strings.Join(output, "\n"))
	})

	t.Run("Vesting pool info ith invalid pool_id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "expected error when using invalid pool_id")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "{\"code\":\"resource_not_found\",\"error\":\"resource_not_found: can't get pool: value not present\"}", output[0])
	})

	t.Run("Vesting pool list before and after adding pool must work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "no vesting pools", output[0])

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
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "-  "+poolId, output[0])
	})

	// FIXME: Is this expected behavior that any wallet can list any wallet's vesting pools?
	t.Run("Listing pools for someone else's client-id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error registering new wallet", strings.Join(output, "\n"))

		foreignWallet, err := getWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error fetching wallet")

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": foreignWallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "-  "+poolId, output[0])
	})

	t.Run("Vesting pool list with invalid client id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": "abcdef123456",
		}), false)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "no vesting pools", output[0])
	})
}

func vestingPoolAdd(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Adding a new vesting pool...")
	return vestingPoolAddForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolAddForWallet(t *testing.T, cliConfigFilename, params string, retry bool, walletName string) ([]string, error) {
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-add "+params+
			" --silent --wallet "+walletName+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-add " + params +
			" --silent --wallet " + walletName + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
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

func vestingPoolList(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Listing vesting pools...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-list "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-list " + params +
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
