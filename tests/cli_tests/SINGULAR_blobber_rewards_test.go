package cli_tests

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/assert"
)

func TestBlobberStorageRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	// Tests the rewards for blobbers from clients' end. In the first case:
	// 1. Client creates an allocation, does not use it at all and finalizes it. In this, 25% of the locked amount should be
	// moved to blobber's delegate wallet. 75% should be returned to the client.

	t.RunWithTimeout("Finalize Expired Allocation Should Work", 8*time.Minute, func(t *test.SystemTest) {
		// blobber delegate wallet and validator delegate wallet are same
		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}

		output, err := registerWallet(t, configPath)
		assert.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		assert.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobberDelegateWallet, err := getWalletForName(t, configPath, blobberOwnerWallet)
		assert.Nil(t, err, "error getting target wallet")

		balanceBefore := getBalanceFromSharders(t, blobberDelegateWallet.ClientID)

		// lock 0.5 tokens from wallet
		options := map[string]interface{}{
			"expire": "5s",
			"size":   "1024",
			"parity": "1",
			"lock":   "1",
			"data":   "1",
		}
		output, err = createNewAllocation(t, configPath, createParams(options))
		assert.Nil(t, err, strings.Join(output, "\n"))
		assert.True(t, len(output) > 0, "expected output length be at least 1")
		assert.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		assert.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		cliutils.Wait(t, 4*time.Minute)

		output, err = finalizeAllocation(t, configPath, allocationID, false)

		assert.Nil(t, err, "unexpected error updating allocation", strings.Join(output, "\n"))
		assert.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		matcher := regexp.MustCompile("Allocation finalized with txId .*$")
		assert.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		cliutils.Wait(t, 2*time.Minute)

		// 75% of 1 ZCN = 0.75 ZCN should return to the client
		output, err = getBalance(t, configPath)
		assert.Nil(t, err, strings.Join(output, "\n"))
		assert.Len(t, output, 1)
		// assert.Regexp(t, regexp.MustCompile(`Balance: 750.000 mZCN \(\d*\.?\d+ USD\)$`), output[0]) // 75% of 1 ZCN

		// Check blobber delegate wallet
		balanceAfter := getBalanceFromSharders(t, blobberDelegateWallet.ClientID)
		assert.Equal(t, float64(balanceAfter), float64(balanceBefore)+(0.75*tokenUnit))
	})
}
