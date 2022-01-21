package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	apr  = 0.1
	year = time.Hour * 24 * 366
)

func TestLockAndUnlockInterest(t *testing.T) {
	t.Parallel()

	t.Run("Lock and unlock tokens", func(t *testing.T) {
		t.Parallel()

		// lock 1 token for 1 min
		// all interest are already earned after lock.
		tokensToLock := float64(1)
		lockDuration := time.Minute
		wantInterestEarned := computeExpectedLockInterest(tokensToLock, lockDuration)
		wantInterestEarnedAsBalance := balancePrint(wantInterestEarned)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock tokens
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
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

		// Get locked tokens BEFORE locked tokens expire.
		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var statsBeforeExpire climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &statsBeforeExpire)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
		require.Len(t, statsBeforeExpire.Stats, 1)
		require.NotEqual(t, "", statsBeforeExpire.Stats[0].ID)
		require.True(t, statsBeforeExpire.Stats[0].Locked)
		require.Equal(t, lockDuration, statsBeforeExpire.Stats[0].Duration)
		require.LessOrEqual(t, statsBeforeExpire.Stats[0].TimeLeft, lockDuration)
		require.LessOrEqual(t, statsBeforeExpire.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, statsBeforeExpire.Stats[0].APR)
		require.Equal(t, wantInterestEarned, statsBeforeExpire.Stats[0].TokensEarned)
		require.Equal(t, int64(tokensToLock*1e10), statsBeforeExpire.Stats[0].Balance)

		// Wait until lock expires.
		<-lockTimer.C

		// Get balance AFTER locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

		// Get locked tokens AFTER locked tokens expire.
		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var statsAfterExpire climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &statsAfterExpire)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
		require.Len(t, statsAfterExpire.Stats, 1)
		require.NotEqual(t, "", statsAfterExpire.Stats[0].ID)
		require.False(t, statsAfterExpire.Stats[0].Locked)
		require.Equal(t, lockDuration, statsAfterExpire.Stats[0].Duration)
		require.LessOrEqual(t, statsAfterExpire.Stats[0].TimeLeft, time.Duration(0)) // timeleft can be negative
		require.Less(t, statsAfterExpire.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, statsAfterExpire.Stats[0].APR)
		require.Equal(t, wantInterestEarned, statsAfterExpire.Stats[0].TokensEarned)
		require.Equal(t, int64(tokensToLock*1e10), statsAfterExpire.Stats[0].Balance)

		cliutil.Wait(t, time.Second*5) // Sleep to let lock try to earn interest after has expired.

		// unlock
		output, err = unlockInterest(t, configPath, createParams(map[string]interface{}{
			"pool_id": statsAfterExpire.Stats[0].ID,
		}), true)
		require.Nil(t, err, "unlock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Unlock tokens success", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Get balance AFTER locked tokens are unlocked. Would show rounded off to highest (ZCN).
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Get locked tokens AFTER locked tokens are unlocked.
		output, err = getLockedTokens(t, configPath)
		require.Error(t, err, "missing expected get locked tokens error", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `Failed to get locked tokens.{"code":"resource_not_found","error":"resource_not_found: can't find user node"}`, output[0])

		// Return 1 token to faucet to retain just interest.
		output, err = refillFaucet(t, configPath, 1)
		require.Nil(t, err, "refill faucet execution failed", strings.Join(output, "\n"))

		// Check total interest.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])
	})

	t.Run("Multiple lock tokens", func(t *testing.T) {
		t.Parallel()

		// first lock: 0.8 token for 100 min
		// all interest are already earned after lock.
		tokensToLock1 := float64(0.8)
		lockDuration1 := time.Minute * 100
		wantInterestEarnedFromLock1 := computeExpectedLockInterest(tokensToLock1, lockDuration1)
		wantInterestEarnedFromLock1AsBalance := balancePrint(wantInterestEarnedFromLock1)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 800.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// first lock of 0.8 token for 100 min
		params := createParams(map[string]interface{}{
			"durationMin": 100,
			"tokens":      0.8,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (0.800000) locked successfully", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance after first lock.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedFromLock1AsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

		// first lock: 0.5 token for 100 min
		tokensToLock2 := float64(0.5)
		lockDuration2 := time.Hour * 5
		wantInterestEarnedFromLock2 := computeExpectedLockInterest(tokensToLock2, lockDuration2)
		wantInterestEarnedFromBothLocksAsBalance := balancePrint(wantInterestEarnedFromLock1 + wantInterestEarnedFromLock2)

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock2)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+balancePrint(wantInterestEarnedFromLock1+int64(tokensToLock2*1e10))+` \(\d*\.?\d+ USD\)$`), output[0])

		// second lock 0.5 token for 5 hrs
		params = createParams(map[string]interface{}{
			"durationHr": 5,
			"tokens":     0.5,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (0.500000) locked successfully", output[0])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance after second lock.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedFromBothLocksAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

		output, err = getLockedTokens(t, configPath)
		require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Locked tokens:", output[0])

		var stats climodel.LockedInterestPoolStats
		err = json.Unmarshal([]byte(output[1]), &stats)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
		require.Len(t, stats.Stats, 2)

		// check locked tokens. Order retrieved can be different so search appropriate lock

		wantDurationTokenPair := map[time.Duration]struct {
			tokensLocked   float64
			interestEarned int64
		}{
			lockDuration1: {
				tokensLocked:   tokensToLock1,
				interestEarned: wantInterestEarnedFromLock1,
			},
			lockDuration2: {
				tokensLocked:   tokensToLock2,
				interestEarned: wantInterestEarnedFromLock2,
			},
		}

		for _, stat := range stats.Stats {
			want, ok := wantDurationTokenPair[stat.Duration]
			require.True(t, ok, "Lock duration got %s not expected", stat.Duration)

			require.NotEqual(t, "", stat.ID)
			require.True(t, stat.Locked)
			require.LessOrEqual(t, stat.TimeLeft, stat.Duration)
			require.LessOrEqual(t, stat.StartTime, time.Now().Unix())
			require.Equal(t, apr, stat.APR)
			require.Equal(t, want.interestEarned, stat.TokensEarned)
			require.Equal(t, int64(want.tokensLocked*1e10), stat.Balance)
		}
	})

	t.Run("Lock with maximum durationMin param", func(t *testing.T) {
		t.Parallel()

		// lock 0.951123 token for 1 year
		// all interest are already earned after lock.
		tokensToLock := float64(0.951123)
		lockDuration := year
		wantInterestEarned := computeExpectedLockInterest(tokensToLock, lockDuration)
		wantInterestEarnedAsBalance := balancePrint(wantInterestEarned)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 951.123 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock 0.951123 token for 200 minutes
		params := createParams(map[string]interface{}{
			"durationMin": int64(lockDuration.Minutes()),
			"tokens":      0.951123,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (0.951123) locked successfully", output[0])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

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
		require.Equal(t, lockDuration, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, lockDuration)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, wantInterestEarned)
		require.Equal(t, int64(tokensToLock*1e10), stats.Stats[0].Balance)
	})

	t.Run("Lock with maximum durationHr param", func(t *testing.T) {
		t.Parallel()

		// lock 0.75 token for 1 year
		// all interest are already earned after lock.
		tokensToLock := float64(0.75)
		lockDuration := year
		wantInterestEarned := computeExpectedLockInterest(tokensToLock, lockDuration)
		wantInterestEarnedAsBalance := balancePrint(wantInterestEarned)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 750.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock 0.75 token for 1 year
		params := createParams(map[string]interface{}{
			"durationHr": int64(lockDuration.Hours()),
			"tokens":     0.75,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (0.750000) locked successfully", output[0])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

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
		require.Equal(t, lockDuration, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, lockDuration)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.GreaterOrEqual(t, stats.Stats[0].TokensEarned, wantInterestEarned)
		require.Equal(t, int64(tokensToLock*1e10), stats.Stats[0].Balance)
	})

	t.Run("Lock with both durationMin and durationHr param", func(t *testing.T) {
		t.Parallel()

		// lock 0.25 token for 1hr and 30 mins
		// all interest are already earned after lock.
		tokensToLock := float64(0.25)
		lockDurationHr := time.Hour
		lockDurationMin := time.Minute
		wantLockDuration := lockDurationHr + lockDurationMin
		wantInterestEarned := computeExpectedLockInterest(tokensToLock, wantLockDuration)
		wantInterestEarnedAsBalance := balancePrint(wantInterestEarned)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, tokensToLock)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 250.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock
		params := createParams(map[string]interface{}{
			"durationMin": int64(lockDurationMin.Minutes()),
			"durationHr":  int64(lockDurationHr.Hours()),
			"tokens":      tokensToLock,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "Tokens (0.250000) locked successfully", output[0])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: `+wantInterestEarnedAsBalance+` \(\d*\.?\d+ USD\)$`), output[0])

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
		require.Equal(t, wantLockDuration, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, wantLockDuration)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.Equal(t, wantInterestEarned, stats.Stats[0].TokensEarned)
		require.Equal(t, int64(tokensToLock*1e10), stats.Stats[0].Balance)
	})

	t.Run("Lock with minimum tokens allowed", func(t *testing.T) {
		t.Parallel()

		// lock 0.000_000_001 token for 1 year. Given APR is 10%, 1 year interest would be 0.000_000_000_1 (1 SAS).
		// all interest are already earned after lock.
		tokensToLock := float64(0.000_000_001)
		lockDuration := year
		wantInterestEarned := computeExpectedLockInterest(tokensToLock, lockDuration)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		// Get 0.100_000_001 token from faucet (as it does not allow just 10 SAS)
		output, err = executeFaucetWithTokens(t, configPath, 0.100_000_001)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 100.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// lock
		params := createParams(map[string]interface{}{
			"durationHr": int64(lockDuration.Hours()),
			"tokens":     tokensToLock,
		})
		output, err = lockInterest(t, configPath, params, true)
		require.Nil(t, err, "lock interest failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)
		// FIXME precision is lost - should say  `Tokens (0.000000001) locked successfully`
		require.Equal(t, "Tokens (0.000000) locked successfully", output[0])

		// Sleep for a bit before checking balance so there is balance already from interest.
		cliutil.Wait(t, time.Second)

		// Get balance BEFORE locked tokens expire.
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 100.000 mZCN \(\d*\.?\d+ USD\)$`), output[0])

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
		require.Equal(t, lockDuration, stats.Stats[0].Duration)
		require.LessOrEqual(t, stats.Stats[0].TimeLeft, lockDuration)
		require.LessOrEqual(t, stats.Stats[0].StartTime, time.Now().Unix())
		require.Equal(t, apr, stats.Stats[0].APR)
		require.Equal(t, wantInterestEarned, stats.Stats[0].TokensEarned)
		require.Equal(t, int64(tokensToLock*1e10), stats.Stats[0].Balance)
	})

	t.Run("Lock attempt with tokens exceeding balance param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 5,
			"tokens":      1.1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed locking tokens:lock amount is greater than balance", output[0])
	})

	t.Run("Lock attempt with missing durationHr and durationMin params should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"tokens": 1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: durationHr and durationMin flag is missing. atleast one is required", output[0])
	})

	t.Run("Lock attempt with 0 durationMin param and missing durationHr param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 0,
			"tokens":      1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: invalid duration", output[0])
	})

	t.Run("Lock attempt with 0 durationHr param and missing durationMin param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationHr": 0,
			"tokens":     1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: invalid duration", output[0])
	})

	t.Run("Lock attempt with durationMin over 1 year", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		over1YearMins := int64(year.Minutes()) + 1
		params := createParams(map[string]interface{}{
			"durationMin": int64(over1YearMins),
			"tokens":      1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed locking tokens:duration (8784h1m0s) is longer than max lock period (8784h0m0s)", output[0])
	})

	t.Run("Lock attempt with durationHr over 1 year", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		over1YearHrs := int64(year.Hours()) + 1

		params := createParams(map[string]interface{}{
			"durationHr": int64(over1YearHrs),
			"tokens":     1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed locking tokens:duration (8785h0m0s) is longer than max lock period (8784h0m0s)", output[0])
	})

	t.Run("Lock attempt with both 0 durationHr and 0 durationMin params should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 0,
			"durationHr":  0,
			"tokens":      1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: invalid duration", output[0])
	})

	t.Run("Lock attempt with missing tokens param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 10,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: tokens flag is missing", output[0])
	})

	t.Run("Lock attempt with 0 tokens param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 10,
			"tokens":      0,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		//nolint:misspell
		require.Equal(t, "failed locking tokens:insufficent amount to dig an interest pool", output[0])
	})

	t.Run("Lock attempt with tokens param below minimum allowed should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 10,
			"tokens":      0.000_000_000_9,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		//nolint:misspell
		require.Equal(t, "failed locking tokens:insufficent amount to dig an interest pool", output[0])
	})

	t.Run("Lock attempt with negative tokens param should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 10,
			"tokens":      -1,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `Failed to lock tokens. submit transaction failed. {"code":"invalid_request","error":"invalid_request: Invalid request (value must be greater than or equal to zero)"}`, output[0])
	})

	t.Run("Lock attempt with tokens param exceeding balance should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		params := createParams(map[string]interface{}{
			"durationMin": 10,
			"tokens":      1.01,
		})
		output, err = lockInterest(t, configPath, params, false)
		require.NotNil(t, err, "Missing expected lock interest failure", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "failed locking tokens:lock amount is greater than balance", output[0])
	})

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
		require.Equal(t, "failed to unlock tokens:pool () doesn't exist", output[0])
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
		require.Equal(t, "failed to unlock tokens:pool (abcdef) doesn't exist", output[0])
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
		require.Equal(t, "failed to unlock tokens:error emptying pool emptying pool failed: pool is still locked", output[0])
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
		reg := regexp.MustCompile("failed to unlock tokens:pool \\([a-z0-9]{64}\\) doesn't exist")
		require.Regexp(t, reg, output[0])
	})
}

func lockInterest(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	cmd := "./zwallet lock " + params + " --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if retry {
		return cliutil.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutil.RunCommandWithoutRetry(cmd)
	}
}

func unlockInterest(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return unlockInterestForWallet(t, escapedTestName(t), cliConfigFilename, params, retry)
}

func unlockInterestForWallet(t *testing.T, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	cmd := "./zwallet unlock " + params + " --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if retry {
		return cliutil.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutil.RunCommandWithoutRetry(cmd)
	}
}

func getLockedTokens(t *testing.T, cliConfigFilename string) ([]string, error) {
	cliutil.Wait(t, 5*time.Second)
	return cliutil.RunCommand(t, "./zwallet getlockedtokens --silent --wallet "+escapedTestName(t)+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func refillFaucet(t *testing.T, cliConfigFilename string, tokens float64) ([]string, error) {
	return cliutil.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName refill --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		escapedTestName(t),
		cliConfigFilename,
	), 3, time.Second*20)
}

func computeExpectedLockInterest(tokens float64, duration time.Duration) int64 {
	return int64(tokens * 1e10 * apr * duration.Minutes() / year.Minutes())
}

func balancePrint(b int64) string {
	switch {
	case b/1e10 > 0:
		return fmt.Sprintf("%.3f ZCN", float64(b)/1e10)
	case b/1e7 > 0:
		return fmt.Sprintf("%.3f mZCN", float64(b)/1e7)
	case b/1e4 > 0:
		return fmt.Sprintf("%.3f uZCN", float64(b)/1e4)
	}
	return fmt.Sprintf("%d SAS", b)
}
