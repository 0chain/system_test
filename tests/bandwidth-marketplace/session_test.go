package bandwidth_marketplace

import (
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/ap"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/magma"
)

func TestSession(t *testing.T) {
	var (
		userIMSI = "user_id"

		accessPoint *magmasc.AccessPoint
	)
	var err error
	accessPoint, err = ap.RegisterAndStake(testCfg.AccessPoint.KeysFile, testCfg.AccessPoint.NodeDir, testCfg)
	require.NoError(t, err)

	pourProviderWalletAndStake(t)

	t.Run("Start->Update->Stop_OK", func(t *testing.T) {
		if !testCfg.Cases.Session.StartUpdateStopOK {
			t.Skip()
		}

		// pouring consumer wallet if it is configured
		pourConsumerWallet(t)

		var (
			sessionID               = "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())
			sessionTimeDelay uint32 = 5 // in seconds

			startingConsumerBal, startingProviderBal, startingAccPointBal        = getConsumerBalance(t), getProviderBalance(t), getAccessPointBalance(t)
			octetsIn, octetsOut                                           uint64 = 3000000, 3000000
		)

		// checks for usual using

		err := magma.SessionStart(userIMSI, accessPoint.Id, sessionID, testCfg)
		require.NoError(t, err)

		// request session and check balances
		checkLockedTokens(t, sessionID, startingConsumerBal)

		err = magma.SessionUpdate(userIMSI,
			accessPoint.Id,
			sessionID,
			sessionTimeDelay,
			octetsIn,
			octetsOut,
			testCfg,
		)
		require.NoError(t, err)

		err = magma.SessionStop(
			userIMSI,
			accessPoint.Id,
			sessionID,
			2*sessionTimeDelay,
			octetsIn,
			octetsOut,
			testCfg,
		)
		require.NoError(t, err)

		// request billing and check balances
		payed := checkBillingPayments(t, sessionID, startingConsumerBal, startingProviderBal, startingAccPointBal)

		// we dont need min cost payment, so check it
		sess, err := magmasc.RequestSession(sessionID)
		require.NoError(t, err)
		require.True(
			t,
			payed != sess.AccessPoint.TermsGetMinCost(),
			"Billing amount should not be equal to the MinCost",
		)
	})

	t.Run("Start->Stop_OK", func(t *testing.T) {
		if !testCfg.Cases.Session.StartStopOK {
			t.Skip()
		}

		// pouring consumer wallet if needed
		pourConsumerWallet(t)

		var (
			sessionTimeDelay uint32 = 5 // in seconds
			sessionID               = "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())

			startingConsumerBal, startingProviderBal, startingAccPointBal = getConsumerBalance(t), getProviderBalance(t), getAccessPointBalance(t)
		)

		err := magma.SessionStart(
			userIMSI,
			accessPoint.Id,
			sessionID,
			testCfg,
		)
		require.NoError(t, err)

		// request session and check balances
		checkLockedTokens(t, sessionID, startingConsumerBal)

		err = magma.SessionStop(
			userIMSI,
			accessPoint.Id,
			sessionID,
			sessionTimeDelay,
			0,
			0,
			testCfg,
		)
		require.NoError(t, err)

		// request billing and check balances
		payed := checkBillingPayments(t, sessionID, startingConsumerBal, startingProviderBal, startingAccPointBal)

		// we need min cost payment, so check it
		sess, err := magmasc.RequestSession(sessionID)
		require.NoError(t, err)
		require.True(
			t,
			payed == sess.AccessPoint.TermsGetMinCost(),
			"Billing amount should be equal to the MinCost",
		)
	})

	t.Run("Update_Non_Existing_Session_ERR", func(t *testing.T) {
		if !testCfg.Cases.Session.UpdateNonExistingSessionOK {
			t.Skip()
		}

		var (
			startTime = time.Now().Unix()
			sessionID = "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())

			startingConsumerBalance, startingProviderBalance = getConsumerBalance(t), getProviderBalance(t)
		)

		sessTime := uint32(time.Now().Unix() - startTime)
		err := magma.SessionUpdate(
			userIMSI,
			accessPoint.Id,
			sessionID,
			sessTime,
			0,
			0,
			testCfg)
		require.Error(t, err)

		// checks for balances that must be unchanged
		require.Equal(t, startingConsumerBalance, getConsumerBalance(t))
		require.Equal(t, startingProviderBalance, getProviderBalance(t))
	})

	t.Run("Stop_Non_Existing_Session_ERR", func(t *testing.T) {
		if !testCfg.Cases.Session.StopNonExistingSessionOK {
			t.Skip()
		}

		var (
			startTime = time.Now().Unix()
			sessionID = "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())

			startingConsumerBalance, startingProviderBalance = getConsumerBalance(t), getProviderBalance(t)
		)

		sessTime := uint32(time.Now().Unix() - startTime)
		err := magma.SessionStop(
			userIMSI,
			accessPoint.Id,
			sessionID,
			sessTime,
			0,
			0,
			testCfg,
		)
		require.Error(t, err)

		// checks for balances that must be unchanged
		require.Equal(t, startingConsumerBalance, getConsumerBalance(t))
		require.Equal(t, startingProviderBalance, getProviderBalance(t))
	})
}
