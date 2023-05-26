package cli_tests

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const maxDescriptionLength = "max_description_length"
const maxDestinations = "max_destinations"
const maxDuration = "max_duration"
const minDuration = "min_duration"
const minLock = "min_lock"

func TestVestingPoolAdd(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")
	t.Parallel()

	var validDuration string
	var vpConfigMap map[string]interface{}
	t.TestSetup("Register wallet + get vesting SC config", func() {
		// get current valid vesting configs
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

		vpConfigMap = configFromKeyValuePair(output)
		validDuration = getValidDuration(t, vpConfigMap)
	})

	// VP-ADD cases
	t.Run("Vesting pool with single destination, valid duration and valid tokens should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
	})

	t.Run("Vesting pool with single destination and description should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with multiple destinations should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with multiple destinations and description should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
	})

	t.Run("Vesting pool with locking insufficient tokens should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, "create_vesting_pool_failed: not enough tokens to create pool provided", output[0], "output did not match expected error message")
	})

	t.Run("Vesting pool with excess locked tokens should work and allow unlocking", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet by locking 0.5 tokens
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.5,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Use vp-info to check excess tokens are shown as can be unlocked
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Len(t, output, 18, "expected output of length 18")
		require.Equal(t, output[2], "can unlock:   400.000 mZCN (excess)")
	})

	t.Run("Vesting pool with start time in future should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 18, "expected output of length 18")
		require.Equal(t, output[7], "start_time:   "+time.Unix(startTime.Unix(), 0).String())
	})

	t.Run("Vesting pool with start time in future for multiple destination wallets should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, time.Second)

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		// FIXME: Output is sometimes len 21, other times 23 (should be 23 always)
		require.GreaterOrEqual(t, len(output), 21, "expected output of length 23")
		require.Equal(t, output[7], "start_time:   "+time.Unix(startTime.Unix(), 0).String())
	})

	t.Run("Vesting pool with start time in past should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: vesting starts before now")
	})

	t.Run("Vesting pool with start time in past for multiple destinations should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: vesting starts before now")
	})

	t.Run("Vesting pool with invalid destination should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

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

	t.Run("Vesting pool with one valid destination and one invalid destination should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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

	t.Run("Vesting pool for duration less than min duration should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: vesting duration is too short")
	})

	t.Run("Vesting pool with duration greater than max duration should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: vesting duration is too long")
	})

	t.Run("Vesting pool with lock less than min lock should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: insufficient amount to lock")
	})

	t.Run("Vesting pool with description greater than max description length should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: entry description is too long")
	})

	t.Run("Vesting pool with destinations greater than max destinations should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

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
			output, err = createWalletForName(t, configPath, targetWalletName)
			require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Equal(t, output[0], "create_vesting_pool_failed: invalid request: too many destinations")
	})

	t.Run("Vesting pool add without destination flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"lock":     0.1,
			"duration": validDuration,
		}), false)
		require.NotNil(t, err, "expected error when adding a new vesting pool without destination")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'd' flag", output[0])
	})

	t.Run("Vesting pool add without duration flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"lock": 0.1,
			"d":    "dummyClientID",
		}), false)
		require.NotNil(t, err, "expected error when adding a new vesting pool without duration")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'duration' flag", output[0])
	})

	t.Run("Vesting pool add without lock flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        "abcdef123456abcdef123456abcdef123456abcdef123456abcdef123456abcd:0.1",
			"duration": "3h30m",
		}), false)
		require.NotNil(t, err, "expected error when adding a new vesting pool without lock")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'lock' flag", output[0])
	})
}

func TestVestingPoolDelete(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")
	// get current valid vesting configs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-DELETE cases
	t.Run("Vesting pool delete should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error deleting vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting pool deleted successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Wallet balance should be greater than 0.9 since non-vested tokens should return
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error getting wallet balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		balance, err := strconv.ParseFloat(regexp.MustCompile(`\d*\.?\d+`).FindString(output[0]), 64)
		require.Nil(t, err, "error parsing float from balance")
		require.Greater(t, balance, 900.000)
	})

	t.Run("Vesting pool delete with invalid pool_id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": "invalidPoolId",
		}), false)
		require.NotNil(t, err, "expected error when deleting invalid vesting pool id")
		require.Len(t, output, 1)
		require.Equal(t, "delete_vesting_pool_failed: can't get pool: value not present", output[0])
	})

	t.Run("Deleting someone else's vesting pool should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error creating new wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "delete_vesting_pool_failed: only pool owner can delete the pool", output[0])
	})

	t.Run("Vesting pool delete without pool id flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolDelete(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error using vp-delete without pool id")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})
}

func vestingPoolDelete(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Deleting vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-delete "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-delete " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func TestVestingPoolInfo(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")

	// get current valid vesting configs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-INFO cases
	// Feature to add: vp-info should have a json flag, it already has models in place in gosdk
	t.Run("Vesting pool info with valid pool_id should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 18, "expected output of length 18")
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

	t.Run("Vesting pool info for pool with multiple destinations should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		clientWallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2",
			"lock":     0.3,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 23, "expected output of length 23")
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
	t.Run("Vesting pool info for pool belonging to someone else's wallet must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error getting faucet tokens on foreign wallet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			// adding second wallet this way since map doesn't allow repeated keys
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.3,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// FIXME: should get error
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info", strings.Join(output, "\n"))
	})

	t.Run("Vesting pool info with invalid pool_id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "expected error when using invalid pool_id")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "{\"code\":\"resource_not_found\",\"error\":\"resource_not_found: can't get pool: value not present\"}", output[0])
	})

	t.Run("Vesting pool info without pool id flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		// verify start time using vp-info
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error when using vp-info without pool id flag")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})
}

func TestVestingPoolStop(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")

	// get current valid vesting configs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-STOP cases
	t.Run("Vesting pool stop for pool with one destination should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID,
		}), true)
		require.Nil(t, err, "error stopping vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Stop vesting for "+targetWallet.ClientID+".", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Destination should be removed from vp-info after stopping
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 11, "expected output of length 11 atleast")
		require.Equal(t, "destinations:", output[9])
		require.Equal(t, "client_id:    "+wallet.ClientID, output[10])
		canUnlockAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[2]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		canUnlockUnit := regexp.MustCompile("[um]?ZCN").FindString(output[2])
		canUnlockAmount = unitToZCN(canUnlockAmount, canUnlockUnit)

		// token-accounting for this case: balance tokens should be unlockable
		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking tokens from vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		// Post-mainnet: turn to equal
		require.InEpsilonf(t, 0.9+canUnlockAmount, newBalanceInZCN, 0.00000000001, "expected balance to be [%v] but was [%v]", 0.9+canUnlockAmount, newBalanceInZCN)
	})

	// FIXME: this only stops last destination flag.
	t.RunWithTimeout("Vesting pool stop for multiple destinations should work", 90*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWalletName3 := "targetWallet3" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName3)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		targetWallet3, err := getWalletForName(t, configPath, targetWalletName3)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1" + " --d " + targetWallet2.ClientID + ":0.2" + " --d " + targetWallet3.ClientID + ":0.3",
			"lock":     0.6,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Stopping with multiple destinations
		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID + " --d " + targetWallet2.ClientID,
		}), true)
		require.Nil(t, err, "error stopping vesting pool")
		// FIXME: output only shows stop vesting for last destination flag. Should show all stopped destinations
		require.Len(t, output, 2)
		require.Equal(t, "Stop vesting for "+targetWallet2.ClientID+".", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Destination should be removed from vp-info after stopping
		// FIXME: Multiple d flags don't work, only last flag passed is stopped.
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching pool-info")
		require.Len(t, output, 23, "expected output of length 23")
		canUnlockAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[2]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		canUnlockUnit := regexp.MustCompile("[um]?ZCN").FindString(output[2])
		canUnlockAmount = unitToZCN(canUnlockAmount, canUnlockUnit)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking tokens from vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		// Post-mainnet: turn to equal
		require.InEpsilonf(t, 0.4+canUnlockAmount, newBalanceInZCN, 0.00000000001, "expected balance to be [%v] but was [%v]", 0.9+canUnlockAmount, newBalanceInZCN)
	})

	t.Run("Vesting pool stop for someone else's pool must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error creating new wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		// Stopping with multiple destinations
		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
			"d":       targetWallet.ClientID,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "stop_vesting_failed: only owner can stop a vesting", output[0])
	})

	t.Run("Vesting pool stop without pool id must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"d": targetWallet.ClientID,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting pool stop without destination should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolStop(t, configPath, createParams(map[string]interface{}{
			"pool_id": "dummypoolid",
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'd' flag", output[0])
	})
}

func TestVestingPoolTokenAccounting(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")

	t.Run("Vesting pool with one destination should move some balance to pending which should be unlockable", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 3.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		startTime := time.Now().Add(1 * time.Second).Unix()

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":2",
			"lock":       2,
			"duration":   "2m",
			"start_time": startTime,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching client balance")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d+\.?\d* USD\)`), output[0])

		// Get vp-info and current time
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		currTime := time.Now().Unix()
		require.Nil(t, err, "error fetching pool info")
		require.Len(t, output, 18, "expected output of length 18")
		ratio := math.Min((float64(currTime)-float64(startTime))/120, 1) // 120 is duration
		expectedVestedAmount := 2 * ratio
		// Round to 6 decimal places
		expectedVestedAmount = math.Round(expectedVestedAmount*1e6) / 1e6
		actualVestedAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[15]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		unit := regexp.MustCompile("[um]?ZCN").FindString(output[15])
		actualVestedAmount = unitToZCN(actualVestedAmount, unit)
		require.GreaterOrEqualf(t, actualVestedAmount, expectedVestedAmount,
			"transferred amount [%v] should have been greater than or equal to expected transferred amount [%v]", actualVestedAmount, expectedVestedAmount)

		cliutils.Wait(t, time.Second)

		// Target wallet should be able to unlock tokens from vesting pool
		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName)
		require.Nil(t, err, "error unlocking tokens from vesting pool by target wallet")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.GreaterOrEqualf(t, newBalanceInZCN, actualVestedAmount,
			"amount in wallet after unlock should be greater or equal to transferred amount")
	})

	t.RunWithTimeout("Vesting pool with multiple destinations should move some balance to pending which should be unlockable", 90*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 4.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet2, err := getWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching destination wallet")

		startTime := time.Now().Add(1 * time.Second).Unix()

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":1" + " --d " + targetWallet2.ClientID + ":2",
			"lock":     3,
			"duration": "2m",
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching balance for client wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d+\.?\d* USD\)`), output[0])

		// Get vp-info and current time
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		currTime := time.Now().Unix()
		require.Nil(t, err, "error fetching pool info")
		require.GreaterOrEqual(t, len(output), 23, "expected output of length 23 or more")
		ratio := math.Min((float64(currTime)-float64(startTime))/120, 1) // 120 is duration
		expectedVestedAmount1 := 1 * ratio
		expectedVestedAmount1 = math.Round(expectedVestedAmount1*1e6) / 1e6
		actualVestedAmount1, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[15]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		unit := regexp.MustCompile("[um]?ZCN").FindString(output[15])
		actualVestedAmount1 = unitToZCN(actualVestedAmount1, unit)
		require.GreaterOrEqualf(t, actualVestedAmount1, expectedVestedAmount1,
			"transferred amount [%v] should have been greater than or equal to expected transferred amount [%v]", actualVestedAmount1, expectedVestedAmount1)

		cliutils.Wait(t, 1*time.Second)

		// Target wallet 1 should be able to unlock tokens from vesting pool
		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName)
		require.Nil(t, err, "error unlocking tokens from vesting pool by target wallet")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.GreaterOrEqualf(t, newBalanceInZCN, actualVestedAmount1,
			"amount in wallet after unlock should be greater or equal to transferred amount")

		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		currTime = time.Now().Unix()
		require.Nil(t, err, "error fetching pool info")
		ratio = math.Min((float64(currTime)-float64(startTime))/120, 1) // 120 is duration
		expectedVestedAmount2 := 2 * ratio
		expectedVestedAmount2 = math.Round(expectedVestedAmount2*1e6) / 1e6
		actualVestedAmount2, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[22]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		unit = regexp.MustCompile("[um]?ZCN").FindString(output[22])
		actualVestedAmount2 = unitToZCN(actualVestedAmount2, unit)
		require.GreaterOrEqualf(t, actualVestedAmount2, expectedVestedAmount2,
			"transferred amount [%v] should have been greater than or equal to expected transferred amount [%v]", actualVestedAmount2, expectedVestedAmount2)

		cliutils.Wait(t, 1*time.Second)

		// Target wallet 2 should be able to unlock tokens from vesting pool
		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName2)
		require.Nil(t, err, "error unlocking tokens from vesting pool by target wallet")
		require.Len(t, output, 2)
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = getBalanceForWallet(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance = regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err = strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN = unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.GreaterOrEqualf(t, newBalanceInZCN, actualVestedAmount2,
			"amount in wallet after unlock should be greater or equal to transferred amount")
	})
}

func TestVestingPoolTrigger(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")

	// get current valid vesting configs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-TRIGGER cases
	// FIXME: vp-trigger is not working, tokens still keep on being sent in a timely manner as opposed to being release all at once.
	t.Run("Vesting pool trigger for one destination pool should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error trigerring vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting triggered successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// vp-info should show vested tokens as sent
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Len(t, output, 18)
		// FIXME:
		// output[1] should be 0 SAS, output[3]: "1.000 mZCN (real value)", output[13]: "1.000 mZCN"
		// output[14] should be 0 SAS, output[4]: 0 SAS

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 1.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])
	})

	t.Run("Vesting pool trigger for a pool with multiple destinations should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWalletName2 := "targetWallet2" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName2)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

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
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error trigerring vesting pool")
		require.Len(t, output, 2)
		require.Equal(t, "Vesting triggered successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Vp-info should show that all tokens are transferred to destination wallets
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Len(t, output, 24, "expected output of length 24 atleast")
		// FIXME:
		// Balance should be 0 SAS, Sent should be 3.000 mZCN, sent for ID 1 should be 1.000 mZCN
		// Sent for ID 2 should be 2.000 mZCN

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 1.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])

		output, err = getBalanceForWallet(t, configPath, targetWalletName2)
		require.Nil(t, err, "error fetching wallet balance")
		// FIXME: Balance should be 2.000 mZCN
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]?ZCN \(\d*\.?\d+ USD\)`), output[0])
	})

	t.Run("Triggering someone else's pool must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error creating new wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		require.Equal(t, "trigger_vesting_pool_failed: only owner can trigger the pool", output[0])
	})

	t.Run("Vesting pool trigger without pool id flag should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "expected error trigerring vesting pool without pool id")
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting pool trigger with invalid pool id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolTrigger(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "expected error trigerring vesting pool with invalid pool id")
		require.Len(t, output, 1)
		require.Equal(t, "trigger_vesting_pool_failed: can't get pool: value not present", output[0])
	})
}

func TestVestingPoolUnlock(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")

	// get current valid vesting configs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-UNLOCK cases
	t.Run("Vesting pool unlock with excess tokens in pool should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.2,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 2, "expected output of length 1")
		require.Equal(t, "Tokens unlocked successfully.", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Vp-info should show (can unlock) as 0, wallet should have increased by 0.1
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Equal(t, "can unlock:   0 SAS (excess)", output[2])

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "error fetching wallet balance")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 900.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Vesting pool unlock by destination wallet should work", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		cliutils.Wait(t, time.Second*5)

		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName)
		require.Nil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 2, "expected output of length 1")
		require.Equal(t, "Tokens unlocked successfully.", output[0])

		// Vp-info should show (can unlock) as 0, wallet should have increased by 0.1
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching vesting pool info")
		require.Equal(t, "can unlock:   0 SAS (excess)", output[2])

		// Target wallet balance should get unlocked tokens
		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching wallet balance")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ [um]ZCN \(\d*\.?\d+ USD\)$`), output[0])
		balance, err := strconv.ParseFloat(regexp.MustCompile(`\d*\.?\d+`).FindString(output[0]), 64)
		require.Nil(t, err, "error parsing float from balance")
		require.Greater(t, balance, 0.0)
	})

	t.Run("Unlocking someone else's vesting pool should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error creating new wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "expected error stopping someone elses's vesting pool")
		require.Len(t, output, 1)
		reg := regexp.MustCompile("unlock_vesting_pool_failed: vesting pool: destination [a-z0-9]{64} not found in the pool")
		require.Regexp(t, reg, output[0])
	})

	t.Run("Vesting pool unlock for one destination and no excess tokens in pool should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = createWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error creating target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, time.Second)

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "unlock_vesting_pool_failed: draining pool: no excess tokens to unlock", output[0])
	})

	t.Run("Vesting unlock without pool id must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "missing required 'pool_id' flag", output[0])
	})

	t.Run("Vesting unlock with invalid pool id must fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = vestingPoolUnlock(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef123456",
		}), false)
		require.NotNil(t, err, "error unlocking vesting pool tokens")
		require.Len(t, output, 1, "expected output of length 1")
		require.Equal(t, "unlock_vesting_pool_failed: can't get pool: value not present", output[0])
	})
}

func TestVestingPoolUpdateConfig(testSetup *testing.T) {
	testSetup.Skip("Enable post mainnet when vesting sc is enabled")
	t := test.NewSystemTest(testSetup)
	t.Skip("turn on post mainnet")
	if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
		t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
	}

	// unused wallet, just added to avoid having the creating new wallet outputs
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

	t.RunSequentially("should allow update of max_destinations", func(t *test.SystemTest) {
		configKey := "max_destinations"
		newValue := "4"

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgBefore, _ := keyValuePairStringToMap(output)

		// ensure revert in config is run regardless of test result
		defer func() {
			oldValue := cfgBefore[configKey]
			output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
				"keys":   configKey,
				"values": oldValue,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
			require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
			require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))
		}()

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "vesting smart contract settings updated", output[0], strings.Join(output, "\n"))
		require.Regexp(t, `Hash: [0-9a-f]+`, output[1], strings.Join(output, "\n"))

		output, err = getVestingPoolSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(output)
		require.Equal(t, newValue, cfgAfter[configKey], "new value %s for config was not set", newValue, configKey)

		// test transaction to verify chain is still working
		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
	})

	t.RunSequentiallyWithTimeout("update max_destinations to invalid value should fail", 60*time.Second, func(t *test.SystemTest) {
		configKey := "max_destinations"
		newValue := "x"

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: value x cannot be converted to time.Duration, failing to set config key max_destinations",
			output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update by non-smartcontract owner should fail", func(t *test.SystemTest) {
		configKey := "max_destinations"
		newValue := "4"

		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, escapedTestName(t), map[string]interface{}{
			"keys":   configKey,
			"values": newValue,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentiallyWithTimeout("update with bad config key should fail", 90*time.Second, func(t *test.SystemTest) {
		configKey := "unknown_key"
		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   configKey,
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_config: config setting unknown_key not found", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update with missing keys param should fail", func(t *test.SystemTest) {
		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"values": 1,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update with missing values param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = updateVestingPoolSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys": "max_destinations",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "number keys must equal the number values", output[0], strings.Join(output, "\n"))
	})
}

func getVestingPoolSCConfig(t *test.SystemTest, cliConfigFilename string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Retrieving vesting config...")

	cmd := "./zwallet vp-config --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateVestingPoolSCConfig(t *test.SystemTest, walletName string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating vesting config...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zwallet vp-update-config %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		configPath,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func vestingPoolUnlock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return vestingPoolUnlockForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolUnlockForWallet(t *test.SystemTest, cliConfigFilename, params string, retry bool, wallet string) ([]string, error) {
	t.Log("Unlocking a vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-unlock "+params+
			" --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-unlock " + params +
			" --silent --wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func vestingPoolTrigger(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return vestingPoolTriggerForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolTriggerForWallet(t *test.SystemTest, cliConfigFilename, params string, retry bool, wallet string) ([]string, error) {
	t.Log("Triggering vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-trigger "+params+
			" --silent --wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-trigger " + params +
			" --silent --wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func vestingPoolStop(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Stopping vesting pool...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-stop "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-stop " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func vestingPoolInfo(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("fetching vesting pool info...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-info "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-info " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func vestingPoolAdd(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Adding a new vesting pool...")
	return vestingPoolAddForWallet(t, cliConfigFilename, params, retry, escapedTestName(t))
}

func vestingPoolAddForWallet(t *test.SystemTest, cliConfigFilename, params string, retry bool, walletName string) ([]string, error) {
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-add "+params+
			" --silent --wallet "+walletName+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-add " + params +
			" --silent --wallet " + walletName + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}

func configFromKeyValuePair(output []string) map[string]interface{} { //nolint
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

func getValidDuration(t *test.SystemTest, vpConfigMap map[string]interface{}) string { //nolint
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

func durationToSeconds(t *test.SystemTest, duration string) int64 {
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
