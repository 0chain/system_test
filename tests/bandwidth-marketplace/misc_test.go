package bandwidth_marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/node"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/zsdk"
)

func checkLockedTokens(t *testing.T, sessionID string, startConsBal int64) {
	sess, err := magmasc.RequestSession(sessionID)
	require.NoError(t, err)

	billRatio, err := magmasc.FetchBillingRatio()
	require.NoError(t, err)

	// check that consumer balance is decreased by terms amount value.
	expConsBal := startConsBal - sess.AccessPoint.TermsGetAmount()*billRatio
	require.Equal(t, expConsBal, getConsumerBalance(t))
}

func checkBillingPayments(t *testing.T, sessionID string, startConsBal, startProvBal, startAPBal int64) (payed int64) {
	sess, err := magmasc.RequestSession(sessionID)
	require.NoError(t, err)

	// check that billing is completed; 0 timestamp means that billing is not completed
	require.True(t, sess.Billing.CompletedAt != 0)
	log.Logger.Info("Balances",
		zap.Int64("cons_start_bal", startConsBal),
		zap.Int64("cons_res_bal", getConsumerBalance(t)),
		zap.Int64("prov_start_bal", startProvBal),
		zap.Int64("prov_res_bal", getProviderBalance(t)),
		zap.Int64("ap_start_bal", startAPBal),
		zap.Int64("ap_res_bal", getAccessPointBalance(t)),
		zap.Int64("sess_amount", sess.Billing.Amount),
	)

	return sess.Billing.Amount
}

func getConsumerBalance(t *testing.T) int64 {
	bal, err := zsdk.GetBalance(testCfg.Consumer, testCfg)
	require.NoError(t, err)
	return bal
}

func getProviderBalance(t *testing.T) int64 {
	bal, err := zsdk.GetBalance(testCfg.Provider, testCfg)
	require.NoError(t, err)
	return bal
}

func getAccessPointBalance(t *testing.T) int64 {
	bal, err := zsdk.GetBalance(testCfg.AccessPoint, testCfg)
	require.NoError(t, err)
	return bal
}

func pourConsumerWallet(t *testing.T) {
	if testCfg.Consumer.PourWallet {
		log.Logger.Debug("Pouring consumer wallet ...")

		// init consumer's wallet for making faucet functions
		err := zsdk.Init(testCfg.Consumer.KeysFile, testCfg.Consumer.NodeDir, testCfg.Consumer.ExtID, testCfg)
		require.NoError(t, err)

		err = zsdk.Pour(5000000000)
		require.NoError(t, err)
	}
}

func pourProviderWalletAndStake(t *testing.T) {
	log.Logger.Debug("Pouring provider's wallet ...")

	// init provider's wallet for making faucet functions
	err := zsdk.Init(testCfg.Provider.KeysFile, testCfg.Provider.NodeDir, testCfg.Provider.ExtID, testCfg)
	require.NoError(t, err)

	minStake, err := magmasc.ProviderMinStakeFetch()
	require.NoError(t, err)

	err = zsdk.Pour(minStake)
	require.NoError(t, err)

	prov, err := magmasc.ProviderFetch(node.ExtID())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = magmasc.ExecuteProviderStake(ctx, prov)
	require.NoError(t, err)
}
