package bandwidth_marketplace

import (
	"testing"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/zsdk"
)

func checkLockedTokens(t *testing.T, sessionID string, startConsBal int64) {
	ackn, err := magmasc.RequestAcknowledgment(sessionID)
	require.NoError(t, err)

	// check that consumer balance is decreased by terms amount value.
	require.Equal(t, startConsBal-ackn.Terms.GetAmount(), getConsumerBalance(t))
}

func checkBillingPayments(t *testing.T, sessionID string, startConsBal, startProvBal int64) (payed int64) {
	ackn, err := magmasc.RequestAcknowledgment(sessionID)
	require.NoError(t, err)

	// check that billing is completed; 0 timestamp means that billing is not completed
	require.True(t, ackn.Billing.CompletedAt != 0)

	require.Equal(t, getConsumerBalance(t), startConsBal-ackn.Billing.Amount)
	require.Equal(t, getProviderBalance(t), startProvBal+ackn.Billing.Amount)

	return ackn.Billing.Amount
}

func getConsumerBalance(t *testing.T) int64 {
	bal, err := zsdk.GetConsumerBalance(testCfg)
	require.NoError(t, err)
	return bal
}

func getProviderBalance(t *testing.T) int64 {
	bal, err := zsdk.GetProviderBalance(testCfg)
	require.NoError(t, err)
	return bal
}

func pourConsumerWallet(t *testing.T) {
	if testCfg.Consumer.PourWallet {
		log.Logger.Debug("Pouring consumer wallet ...")

		// init consumer wallet for making faucet functions
		err := zsdk.Init(testCfg.Consumer.KeysFile, testCfg.Consumer.NodeDir, testCfg.Consumer.ExtID, testCfg)
		require.NoError(t, err)

		err = zsdk.Pour(5000000000)
		require.NoError(t, err)
	}
}
