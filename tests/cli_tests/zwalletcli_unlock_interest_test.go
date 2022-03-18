package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestUnlockInterest(t *testing.T) {
	t.Parallel()

	t.Run("Unlock attempt with missing pool_id param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = unlockInterest(t, configPath, "", false)
		require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `Error: pool_id flag is missing`, output[0])
	})

	t.Run("Unlock attempt with empty pool_id param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": `""`,
		}), false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed to unlock tokens: pool () doesn't exist", output[0])
	})

	t.Run("Unlock attempt with bad pool_id param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": "abcdef",
		}), false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed to unlock tokens: pool (abcdef) doesn't exist", output[0])
	})

	t.Run("Unlock attempt with pool_id not yet expired should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock 1 token for 1 min
		params := createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      1,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (1.000000) locked successfully", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

		// Get locked tokens BEFORE locked tokens expire.
		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var stats climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &stats)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
		require.Len(t, stats.Stats, 1)
		require.NotEqual(t, "", stats.Stats[0].ID)
		require.True(t, stats.Stats[0].Locked)
		require.Equal(t, time.Minute, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Minute)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
		require.Equal(t, int64(10_000_000_000), stats.Stats[0].Balance)

		// Unlock BEFORE locked tokens expire.
		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": stats.Stats[0].ID,
		}), false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed to unlock tokens: error emptying pool emptying pool failed: pool is still locked", output[0])
	})

	t.Run("Unlock attempt with pool_id from someone else should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock 1 token for 1 min
		params := createParams(map[string]interface{}{
			"durationMin": 1,
			"tokens":      1,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (1.000000) locked successfully", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		lockTimer := time.NewTimer(time.Minute)

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

		// Get locked tokens BEFORE locked tokens expire.
		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var stats climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &stats)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
		require.Len(t, stats.Stats, 1)
		require.NotEqual(t, "", stats.Stats[0].ID)
		require.True(t, stats.Stats[0].Locked)
		require.Equal(t, time.Minute, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Minute)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
		require.Equal(t, int64(10_000_000_000), stats.Stats[0].Balance)

		thirdPartyWallet := escapedTestName(t) + "_THIRDPARTY"

		output, err = registerWalletForName(t, configPath, thirdPartyWallet)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		// Wait until lock expires.
		<-lockTimer.C

		// Unlock attempt by third party wallet.
		output, err = unlockInterestForWallet(t, thirdPartyWallet, configPath, createParams(map[string]interface{}{
			"pool_id": stats.Stats[0].ID,
		}), false)
		require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
		reg := regexp.MustCompile(`failed to unlock tokens: pool \([a-z0-9]{64}\) doesn't exist`)
		require.Regexp(t, reg, output[0])
	})
}
