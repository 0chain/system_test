package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestLockAndUnlockInterest(t *testing.T) {
	t.Parallel()
	t.Run("parallel", func(t *testing.T) {
		t.Run("Lock and unlock tokens", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 1)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (1.000000) locked successfully", output[0])

			lockTimer := time.NewTimer(time.Minute)

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsBeforeExpire cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsBeforeExpire)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsBeforeExpire.Stats, 1)
			require.NotEqual(t, "", statsBeforeExpire.Stats[0].ID)
			require.True(t, statsBeforeExpire.Stats[0].Locked)
			require.Equal(t, time.Minute, statsBeforeExpire.Stats[0].Duration)
			require.LessOrEqual(t, statsBeforeExpire.Stats[0].TimeLeft, time.Minute)
			require.LessOrEqual(t, statsBeforeExpire.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsBeforeExpire.Stats[0].APR)
			require.GreaterOrEqual(t, statsBeforeExpire.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), statsBeforeExpire.Stats[0].Balance)

			// Wait until lock expires.
			<-lockTimer.C

			// Get balance AFTER locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

			balanceAfterLockExpires := output[0] // no addition earnings since lock has expired.

			// Get locked tokens AFTER locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsAfterExpire cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsAfterExpire)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsAfterExpire.Stats, 1)
			require.NotEqual(t, "", statsAfterExpire.Stats[0].ID)
			require.False(t, statsAfterExpire.Stats[0].Locked)
			require.Equal(t, time.Minute, statsAfterExpire.Stats[0].Duration)
			require.LessOrEqual(t, statsAfterExpire.Stats[0].TimeLeft, time.Duration(0)) // timeleft can be negative
			require.Less(t, statsAfterExpire.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsAfterExpire.Stats[0].APR)
			require.Greater(t, statsAfterExpire.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), statsAfterExpire.Stats[0].Balance)

			time.Sleep(time.Second) // Sleep to let lock try to earn interest after has expired.

			// unlock
			output, err = unlockInterest(t, configPath, true, statsAfterExpire.Stats[0].ID)
			require.Nil(t, err, "unlock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Unlock tokens success", output[0])

			// Get balance AFTER locked tokens are unlocked. Would show rounded off to highest (ZCN).
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens AFTER locked tokens are unlocked.
			output, err = getLockedTokens(t, configPath)
			require.Error(t, err, "missing expected get locked tokens error", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Equal(t, `Failed to get locked tokens.{"code":"resource_not_found","error":"resource_not_found: can't find user node"}`, output[0])

			// Return 1 token to faucet to retain just interest.
			output, err = refillFaucet(t, configPath, 1)
			require.Nil(t, err, "refill faucet execution failed", err, strings.Join(output, "\n"))

			// Check total interest gained - must be equal to after lock has expired.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Equal(t, balanceAfterLockExpires, output[0])
		})

		t.Run("Multiple lock tokens", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 0.8)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 800.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// first lock of 0.8 token for 100 min
			output, err = lockInterest(t, configPath, true, 100, false, 0, true, 0.8)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.800000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance after first lock.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \(\d*\.?\d+ USD\)$`), output[0])

			output, err = executeFaucetWithTokens(t, configPath, 0.5)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 500.\d{1,3} mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// second lock 0.5 token for 5 hrs
			output, err = lockInterest(t, configPath, false, 0, true, 5, true, 0.5)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.500000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance after second lock.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \(\d*\.?\d+ USD\)$`), output[0])

			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 2)

			// check locked tokens. Order retrieved can be different so search appropriate lock

			wantDurationTokenPair := map[time.Duration]int64{
				time.Minute * 100: 8_000_000_000,
				time.Hour * 5:     5_000_000_000,
			}

			for _, stat := range stats.Stats {
				wantToken, ok := wantDurationTokenPair[stat.Duration]
				require.True(t, ok, "Lock duration got %s not expected", stat.Duration)

				require.NotEqual(t, "", stat.ID)
				require.True(t, stat.Locked)
				require.LessOrEqual(t, stat.TimeLeft, stat.Duration)
				require.LessOrEqual(t, stat.StartTime, time.Now().Unix())
				require.Equal(t, 0.1, stat.APR)
				require.GreaterOrEqual(t, stat.TokensEarned, int64(0))
				require.Equal(t, wantToken, stat.Balance)
			}
		})

		t.Run("Lock with durationMin param", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 0.951123)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 951.123 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 0.951123 token for 200 minutes
			output, err = lockInterest(t, configPath, true, 200, false, 0, true, 0.951123)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.951123) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Minute*200, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Minute*200)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(9_511_230_000), stats.Stats[0].Balance)
		})

		t.Run("Lock with durationHr param", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 0.75)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 750.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 0.75 token for 24 hours
			output, err = lockInterest(t, configPath, false, 0, true, 24, true, 0.75)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.750000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Hour*24, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Hour*24)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(7_500_000_000), stats.Stats[0].Balance)
		})

		t.Run("Lock with durationMin and durationHr param", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 0.25)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 250.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 0.25 token for 1 hour and 30 mins
			output, err = lockInterest(t, configPath, true, 30, true, 1, true, 0.25)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.250000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Hour+time.Minute*30, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Hour+time.Minute*30)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(2_500_000_000), stats.Stats[0].Balance)
		})

		t.Run("Lock with minimum tokens allowed", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 0.100_000_001)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 100.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 0.000_000_001 token for 2 hours
			output, err = lockInterest(t, configPath, false, 0, true, 2, true, 0.000_000_001)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			// FIXME precision is lost - should say  `Tokens (0.000000001) locked successfully`
			require.Equal(t, "Tokens (0.000000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 100.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Hour*2, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Hour*2)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10), stats.Stats[0].Balance)
		})

		t.Run("Lock attempt with missing durationHr and durationMin params should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, false, 0, false, 0, true, 1)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: durationHr and durationMin flag is missing. atleast one is required", output[0])
		})

		t.Run("Lock attempt with 0 durationMin param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 0, false, 0, true, 1)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: invalid duration", output[0])
		})

		t.Run("Lock attempt with 0 durationHr param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, false, 0, true, 0, true, 1)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: invalid duration", output[0])
		})

		t.Run("Lock attempt with both 0 durationHr and 0 durationMin params should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 0, true, 0, true, 1)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: invalid duration", output[0])
		})

		t.Run("Lock attempt with missing tokens param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 10, false, 0, false, 0)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: tokens flag is missing", output[0])
		})

		t.Run("Lock attempt with 0 tokens param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 10, false, 0, true, 0)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to lock tokens. {"error": "verify transaction failed"}`, output[0])
		})

		t.Run("Lock attempt with tokens param below minimum allowed should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 10, false, 0, true, 0.000_000_000_9)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to lock tokens. {"error": "verify transaction failed"}`, output[0])
		})

		t.Run("Lock attempt with negative tokens param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 10, false, 0, true, -1)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to lock tokens. submit transaction failed. {"code":"invalid_request","error":"invalid_request: Invalid request (value must be greater than or equal to zero)"}`, output[0])
		})

		t.Run("Lock attempt with tokens param exceeding balance should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = lockInterest(t, configPath, true, 10, false, 0, true, 1.01)
			require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to lock tokens. {"error": "verify transaction failed"}`, output[0])
		})

		t.Run("Unlock attempt with missing pool_id param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = unlockInterest(t, configPath, false, "")
			require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Error: pool_id flag is missing`, output[0])
		})

		t.Run("Unlock attempt with empty pool_id param should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = unlockInterest(t, configPath, true, "")
			require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to unlock tokens. {"error": "verify transaction failed"}`, output[0])
		})

		t.Run("Unlock attempt with pool_id not yet expired should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 1)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (1.000000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Minute, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Minute)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), stats.Stats[0].Balance)

			// Unlock BEFORE locked tokens expire.
			output, err = unlockInterest(t, configPath, true, stats.Stats[0].ID)
			require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to unlock tokens. {"error": "verify transaction failed"}`, output[0])
		})

		t.Run("Unlock attempt with pool_id from someone else should fail", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 1)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (1.000000) locked successfully", output[0])

			lockTimer := time.NewTimer(time.Minute)

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens expire.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \(\d*\.?\d+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens expire.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var stats cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&stats)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, stats.Stats, 1)
			require.NotEqual(t, "", stats.Stats[0].ID)
			require.True(t, stats.Stats[0].Locked)
			require.Equal(t, time.Minute, stats.Stats[0].Duration)
			require.LessOrEqual(t, stats.Stats[0].TimeLeft, time.Minute)
			require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, stats.Stats[0].APR)
			require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), stats.Stats[0].Balance)

			thirdPartyWallet := escapedTestName(t) + "_THIRDPARTY"

			output, err = registerWalletForName(configPath, thirdPartyWallet)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			// Wait until lock expires.
			<-lockTimer.C

			// Unlock attempt by third party wallet.
			output, err = unlockInterestForWallet(thirdPartyWallet, configPath, true, stats.Stats[0].ID)
			require.NotNil(t, err, "Missing expected unlock interest failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, `Failed to unlock tokens. {"error": "verify transaction failed"}`, output[0])
		})
	})
}

func lockInterest(t *testing.T, cliConfigFilename string, withDurationMin bool, durationMin int64,
	withDurationHr bool, durationHr int64, withTokens bool, tokens float64) ([]string, error) {
	cmd := "./zwallet lock --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if withDurationMin {
		cmd += ` --durationMin "` + strconv.FormatInt(durationMin, 10) + `"`
	}
	if withDurationHr {
		cmd += ` --durationHr "` + strconv.FormatInt(durationHr, 10) + `"`
	}
	if withTokens {
		cmd += ` --tokens "` + strconv.FormatFloat(tokens, 'f', 10, 64) + `"`
	}
	return cli_utils.RunCommand(cmd)
}

func unlockInterest(t *testing.T, cliConfigFilename string, withPoolID bool, poolID string) ([]string, error) {
	return unlockInterestForWallet(escapedTestName(t), cliConfigFilename, withPoolID, poolID)
}

func unlockInterestForWallet(wallet, cliConfigFilename string, withPoolID bool, poolID string) ([]string, error) {
	cmd := "./zwallet unlock --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if withPoolID {
		cmd += ` --pool_id "` + poolID + `"`
	}
	return cli_utils.RunCommand(cmd)
}

func getLockedTokens(t *testing.T, cliConfigFilename string) ([]string, error) {
	return cli_utils.RunCommand("./zwallet getlockedtokens --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func refillFaucet(t *testing.T, cliConfigFilename string, tokens float64) ([]string, error) {
	return cli_utils.RunCommand(
		fmt.Sprintf("./zwallet faucet --methodName refill --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
			tokens,
			escapedTestName(t),
			cliConfigFilename,
		))
}
