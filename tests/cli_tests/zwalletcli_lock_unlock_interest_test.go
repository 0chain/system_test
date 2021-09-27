package cli_tests

import (
	"encoding/json"
	"fmt"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
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
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 1)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (1.000000) locked successfully", output[0])

			lockTimer := time.NewTimer(time.Minute)

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsBeforeLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsBeforeLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsBeforeLapse.Stats, 1)
			require.NotEqual(t, "", statsBeforeLapse.Stats[0].ID)
			require.True(t, statsBeforeLapse.Stats[0].Locked)
			require.Equal(t, time.Minute, statsBeforeLapse.Stats[0].Duration)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].TimeLeft, time.Minute)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsBeforeLapse.Stats[0].APR)
			require.GreaterOrEqual(t, statsBeforeLapse.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), statsBeforeLapse.Stats[0].Balance)

			// Wait until timer reaches 1 min
			<-lockTimer.C

			// Get balance AFTER locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,4} SAS \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			balanceAfterLockLapse := output[0] // no addition earnings since lock has lapsed

			// Get locked tokens AFTER locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsAfterLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsAfterLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsAfterLapse.Stats, 1)
			require.NotEqual(t, "", statsAfterLapse.Stats[0].ID)
			require.False(t, statsAfterLapse.Stats[0].Locked)
			require.Equal(t, time.Minute, statsAfterLapse.Stats[0].Duration)
			require.LessOrEqual(t, statsAfterLapse.Stats[0].TimeLeft, time.Duration(0)) // timeleft can be negative
			require.Less(t, statsAfterLapse.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsAfterLapse.Stats[0].APR)
			require.Greater(t, statsAfterLapse.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(10_000_000_000), statsAfterLapse.Stats[0].Balance)

			time.Sleep(time.Second) // Sleep to let lock try to earn interest after has lapsed.

			// unlock
			output, err = unlockInterest(t, configPath, true, statsAfterLapse.Stats[0].ID)
			require.Nil(t, err, "unlock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Unlock tokens success", output[0])

			// Get balance AFTER locked tokens are unlocked. Would show rounded off to highest (ZCN).
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens AFTER locked tokens are unlocked.
			output, err = getLockedTokens(t, configPath)
			require.Error(t, err, "missing expected get locked tokens error", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Equal(t, `Failed to get locked tokens.{"code":"resource_not_found","error":"resource_not_found: can't find user node"}`, output[0])

			// Return 1 token to faucet to retain just interest.
			output, err = refillFaucet(t, configPath, 1)
			require.Nil(t, err, "refill faucet execution failed", err, strings.Join(output, "\n"))

			// Check total interest gained - must be equal to after lock has lapsed.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Equal(t, balanceAfterLockLapse, output[0])
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
			require.Regexp(t, regexp.MustCompile(`Balance: 800.000 mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

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
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			output, err = executeFaucetWithTokens(t, configPath, 0.5)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 500.\d{1,3} mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

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
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

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
			require.Regexp(t, regexp.MustCompile(`Balance: 951.123 mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, true, 200, false, 0, true, 0.951123)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.951123) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsBeforeLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsBeforeLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsBeforeLapse.Stats, 1)
			require.NotEqual(t, "", statsBeforeLapse.Stats[0].ID)
			require.True(t, statsBeforeLapse.Stats[0].Locked)
			require.Equal(t, time.Minute*200, statsBeforeLapse.Stats[0].Duration)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].TimeLeft, time.Minute*200)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsBeforeLapse.Stats[0].APR)
			require.GreaterOrEqual(t, statsBeforeLapse.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(9_511_230_000), statsBeforeLapse.Stats[0].Balance)
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
			require.Regexp(t, regexp.MustCompile(`Balance: 750.000 mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// lock 1 token for 1 min
			output, err = lockInterest(t, configPath, false, 0, true, 24, true, 0.75)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.750000) locked successfully", output[0])

			// Sleep for a bit before checking balance so there is balance already from interest.
			time.Sleep(time.Second)

			// Get balance BEFORE locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: \d{1,3}\.\d{1,3} uZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsBeforeLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsBeforeLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsBeforeLapse.Stats, 1)
			require.NotEqual(t, "", statsBeforeLapse.Stats[0].ID)
			require.True(t, statsBeforeLapse.Stats[0].Locked)
			require.Equal(t, time.Hour*24, statsBeforeLapse.Stats[0].Duration)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].TimeLeft, time.Hour*24)
			require.LessOrEqual(t, statsBeforeLapse.Stats[0].StartTime, time.Now().Unix())
			require.Equal(t, 0.1, statsBeforeLapse.Stats[0].APR)
			require.GreaterOrEqual(t, statsBeforeLapse.Stats[0].TokensEarned, int64(0))
			require.Equal(t, int64(7_500_000_000), statsBeforeLapse.Stats[0].Balance)
		})

		t.Run("Lock with minimum tokens allowed", func(t *testing.T) {
		})

		t.Run("Lock attempt with missing durationHr and durationMin params should fail", func(t *testing.T) {
			// Register wallet
			// Faucet
		})
		t.Run("Lock attempt with empty durationHr param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with empty durationMin param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with both durationHr and durationMin params should fail", func(t *testing.T) {
		})
		//t.Run("Lock attempt with durationMin below minimum allowed should fail", func(t *testing.T) {
		//})
		//t.Run("Lock attempt with durationHr below minimum allowed should fail", func(t *testing.T) {
		//})
		t.Run("Lock attempt with missing tokens param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with empty tokens param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with 0 tokens param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with tokens param below minimum allowed should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with negative tokens param should fail", func(t *testing.T) {
		})
		t.Run("Lock attempt with tokens param exceeding balance should fail", func(t *testing.T) {
		})
		t.Run("Unlock attempt with missing pool_id param should fail", func(t *testing.T) {
		})
		t.Run("Unlock attempt with empty pool_id param should fail", func(t *testing.T) {
		})
		t.Run("Unlock attempt with pool_id not yet meeting duration should fail", func(t *testing.T) {
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
	cmd := "./zwallet unlock --silent --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename
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
