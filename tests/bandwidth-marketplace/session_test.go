package bandwidth_marketplace

import (
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/magma"
)

func TestSession(t *testing.T) {
	var (
		userIMSI     = "user_id"
		providerAPID = "access-point-id-1"
	)

	t.Run("Start->Update->Stop_OK", func(t *testing.T) {
		if !testCfg.Cases.Session.StartUpdateStopOK {
			t.Skip()
		}

		// pouring consumer wallet if it is configured
		pourConsumerWallet(t)

		var (
			sessionID               = "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())
			sessionTimeDelay uint32 = 5 // in seconds

			startingConsumerBalance, startingProviderBalance        = getConsumerBalance(t), getProviderBalance(t)
			octetsIn, octetsOut                              uint64 = 3000000, 3000000
		)

		// checks for usual using

		err := magma.SessionStart(userIMSI, providerAPID, sessionID, testCfg)
		require.NoError(t, err)

		// request acknowledgment and check balances
		checkLockedTokens(t, sessionID, startingConsumerBalance)

		err = magma.SessionUpdate(userIMSI,
			providerAPID,
			sessionID,
			sessionTimeDelay,
			octetsIn,
			octetsOut,
			testCfg,
		)
		require.NoError(t, err)

		err = magma.SessionStop(
			userIMSI,
			providerAPID,
			sessionID,
			2*sessionTimeDelay,
			octetsIn,
			octetsOut,
			testCfg,
		)
		require.NoError(t, err)

		// request billing and check balances
		payed := checkBillingPayments(t, sessionID, startingConsumerBalance, startingProviderBalance)

		// we dont need min cost payment, so check it
		ackn, err := magmasc.RequestAcknowledgment(sessionID)
		require.NoError(t, err)
		require.True(t, payed != ackn.Terms.GetMinCost())
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

			startingConsumerBalance, startingProviderBalance = getConsumerBalance(t), getProviderBalance(t)
		)

		err := magma.SessionStart(
			userIMSI,
			providerAPID,
			sessionID,
			testCfg,
		)
		require.NoError(t, err)

		// request acknowledgment and check balances
		checkLockedTokens(t, sessionID, startingConsumerBalance)

		err = magma.SessionStop(
			userIMSI,
			providerAPID,
			sessionID,
			sessionTimeDelay,
			0,
			0,
			testCfg,
		)
		require.NoError(t, err)

		// request billing and check balances
		payed := checkBillingPayments(t, sessionID, startingConsumerBalance, startingProviderBalance)

		// we need min cost payment, so check it
		ackn, err := magmasc.RequestAcknowledgment(sessionID)
		require.NoError(t, err)
		require.True(t, payed == ackn.Terms.GetMinCost())
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
			providerAPID,
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
			providerAPID,
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
