package cli_tests

import (
	"encoding/json"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
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
			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			// lock 0.5 token for 1 min
			output, err = lockInterest(t, configPath, true, 1, false, 0, true, 0.5)
			require.Nil(t, err, "lock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Tokens (0.500000) locked successfully", output[0])

			lockTimer := time.NewTimer(time.Minute)

			// Get balance BEFORE locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Regexp(t, regexp.MustCompile(`Balance: 500.000 mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens BEFORE locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsBeforeLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsBeforeLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsBeforeLapse.Stats, 1)
			assert.NotEqual(t, "", statsBeforeLapse.Stats[0].ID)
			assert.True(t, statsBeforeLapse.Stats[0].Locked)
			assert.Equal(t, time.Minute, statsBeforeLapse.Stats[0].Duration)
			assert.LessOrEqual(t, statsBeforeLapse.Stats[0].TimeLeft, time.Minute)
			assert.LessOrEqual(t, statsBeforeLapse.Stats[0].StartTime, time.Now().Unix())
			assert.Equal(t, 0.1, statsBeforeLapse.Stats[0].APR)
			assert.GreaterOrEqual(t, statsBeforeLapse.Stats[0].TokensEarned, int64(0))
			assert.Equal(t, int64(5_000_000_000), statsBeforeLapse.Stats[0].Balance)

			// Wait until timer reaches 1 min
			<-lockTimer.C

			// Get balance AFTER locked tokens lapse.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			assert.Regexp(t, regexp.MustCompile(`Balance: 500.000 mZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens AFTER locked tokens lapse.
			output, err = getLockedTokens(t, configPath)
			require.Nil(t, err, "get locked tokens failed", strings.Join(output, "\n"))
			require.Equal(t, 2, len(output))
			require.Equal(t, "Locked tokens:", output[0])

			var statsAfterLapse cli_model.LockedInterestPoolStats
			err = json.NewDecoder(strings.NewReader(output[1])).Decode(&statsAfterLapse)
			require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[1], err)
			require.Len(t, statsAfterLapse.Stats, 1)
			assert.NotEqual(t, "", statsAfterLapse.Stats[0].ID)
			assert.False(t, statsAfterLapse.Stats[0].Locked)
			assert.Equal(t, time.Minute, statsAfterLapse.Stats[0].Duration)
			assert.LessOrEqual(t, statsAfterLapse.Stats[0].TimeLeft, time.Duration(0)) // timeleft can be negative
			assert.Less(t, statsAfterLapse.Stats[0].StartTime, time.Now().Unix())
			assert.Equal(t, 0.1, statsAfterLapse.Stats[0].APR)
			assert.Greater(t, statsAfterLapse.Stats[0].TokensEarned, int64(0))
			assert.Equal(t, int64(5_000_000_000), statsAfterLapse.Stats[0].Balance)

			// unlock
			output, err = unlockInterest(t, configPath, true, statsAfterLapse.Stats[0].ID)
			require.Nil(t, err, "unlock interest failed", err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Unlock tokens success", output[0])

			// Get balance AFTER locked tokens are unlocked.
			output, err = getBalance(t, configPath)
			require.Nil(t, err, "get balance failed", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			assert.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9]*\.?[0-9]+ USD\)$`), output[0])

			// Get locked tokens AFTER locked tokens are unlocked.
			output, err = getLockedTokens(t, configPath)
			require.Error(t, err, "missing expected get locked tokens error", strings.Join(output, "\n"))
			require.Equal(t, 1, len(output))
			require.Equal(t, `Failed to get locked tokens.{"code":"resource_not_found","error":"resource_not_found: can't find user node"}`, output[0])
		})
		t.Run("Multiple locked tokens", func(t *testing.T) {
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
