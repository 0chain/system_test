package bandwidth_marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"
	"github.com/0chain/gosdk/zmagmacore/node"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/magma"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/zsdk"
)

// addition checks

func checkLockedTokens(t *testing.T, sessionID string, startConsBal int64) {
	sess, err := magmasc.RequestSession(sessionID)
	require.NoError(t, err)

	billRatio, err := magmasc.FetchBillingRatio()
	require.NoError(t, err)

	// check that consumer balance is decreased by terms amount value.
	expConsBal := startConsBal - sess.AccessPoint.TermsGetAmount()*billRatio
	require.Equal(t, expConsBal, getConsumerBalance(t))
}

func checkBillingPayments(t *testing.T, sessionID string, startProvBal, startAPBal int64) (paid int64) {
	sess, err := magmasc.RequestSession(sessionID)
	require.NoError(t, err)

	// check that billing is completed; 0 timestamp means that billing is not completed
	require.True(t, sess.Billing.CompletedAt != 0)

	actProvBal, actAPBal := getProviderBalance(t), getAccessPointBalance(t)
	require.Equal(
		t,
		startProvBal+startAPBal+sess.Billing.Amount,
		actProvBal+actAPBal,
	)

	log.Logger.Info(
		"Balances checked",
		zap.Int64("prov_start_bal", startProvBal),
		zap.Int64("prov_act_bal", actProvBal),
		zap.Int64("ap_start_bal", startAPBal),
		zap.Int64("ap_act_bal", actAPBal),
		zap.Int64("sess_bill_amount", sess.Billing.Amount),
	)

	return sess.Billing.Amount
}

// balance operations

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
		// init consumer's wallet for making faucet functions
		err := zsdk.Init(testCfg.Consumer.KeysFile, testCfg.Consumer.NodeDir, testCfg.Consumer.ExtID, testCfg)
		require.NoError(t, err)

		err = zsdk.Pour(5000000000)
		require.NoError(t, err)
	}
}

func pourProviderWalletAndStake(t *testing.T) {
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

// reward pools

func lockRewardPool(name, payeeID, ownerKeysDir string, value int64, t *testing.T) *magmasc.TokenPool {
	err := zsdk.Init(ownerKeysDir, t.TempDir(), "", testCfg)
	require.NoError(t, err)

	err = zsdk.Pour(value)
	require.NoError(t, err)

	req := &magmasc.TokenPoolReq{
		TokenPoolReq: &pb.TokenPoolReq{
			Id:      name,
			PayeeId: payeeID,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rewPool, err := magmasc.ExecuteRewardPoolLock(ctx, req, value)
	require.NoError(t, err)

	log.Logger.Info("Reward pool locked", zap.Any("rew_pool", rewPool))

	return rewPool
}

func checkAndFinalizeRewPool(startingBalance, paymentsAmount int64, name string, t *testing.T) {
	rewPool, err := magmasc.RewardPoolFetch(name)
	require.NoError(t, err)

	require.Equal(
		t,
		startingBalance-paymentsAmount,
		rewPool.Balance,
		"All payments must be performed by using reward pool tokens",
	)
}

// session activities

func simpleSessionActivity(userIMSI, accessPointID, sessionID string, afterStart, afterUpdate, afterStop action, t *testing.T) {
	err := magma.SessionStart(userIMSI, accessPointID, sessionID, testCfg)
	require.NoError(t, err)

	afterStart(t)

	var (
		sessionTimeDelay    uint32 = 5 // in seconds
		octetsIn, octetsOut uint64 = 3e9, 3e9
	)

	err = magma.SessionUpdate(userIMSI,
		accessPointID,
		sessionID,
		sessionTimeDelay,
		octetsIn,
		octetsOut,
		testCfg,
	)
	require.NoError(t, err)

	afterUpdate(t)

	err = magma.SessionStop(
		userIMSI,
		accessPointID,
		sessionID,
		2*sessionTimeDelay,
		octetsIn,
		octetsOut,
		testCfg,
	)
	require.NoError(t, err)

	afterStop(t)
}

func rewPoolSessionActivity(userIMSI, accessPointID, sessionID string, rewPools rewardPoolsConfigurator, t *testing.T) {
	rewPoolOwnerKeysDir := t.TempDir() + "keys.txt"
	err := zsdk.WriteDefaultKeysFile(testCfg.ServerChain.SignatureScheme, rewPoolOwnerKeysDir)
	require.NoError(t, err)

	pourConsumerWallet(t)

	var (
		stConsBal, stProvBal, stApBal = getConsumerBalance(t), getProviderBalance(t), getAccessPointBalance(t)

		stProvRewPoolBal, stAccessPointRewPoolBal, stAllRewPoolBal int64

		rewPoolsCount = rewPools.countEnabled()

		afterUpdate action = func(t *testing.T) {
			sess, err := magmasc.RequestSession(sessionID)
			require.NoError(t, err)

			billAmount := sess.Billing.Amount

			if rewPools.provider.enabled {
				rewPool := lockRewardPool(rewPools.provider.name, sess.Provider.Id, rewPoolOwnerKeysDir, billAmount/rewPoolsCount, t)
				stProvRewPoolBal = rewPool.GetBalance()
			}

			if rewPools.accessPoint.enabled {
				rewPool := lockRewardPool(rewPools.accessPoint.name, sess.AccessPoint.Id, rewPoolOwnerKeysDir, billAmount/rewPoolsCount, t)
				stAccessPointRewPoolBal = rewPool.GetBalance()
			}

			if rewPools.all.enabled {
				rewPool := lockRewardPool(rewPools.all.name, "", rewPoolOwnerKeysDir, billAmount/rewPoolsCount, t)
				stAllRewPoolBal = rewPool.GetBalance()
			}
		}

		afterStop action = func(t *testing.T) {
			sess, err := magmasc.RequestSession(sessionID)
			require.NoError(t, err)

			log.Logger.Info("Session completed", zap.Any("sess", sess))

			billAmount := sess.Billing.Amount

			if rewPools.provider.enabled {
				checkAndFinalizeRewPool(stProvRewPoolBal, billAmount/rewPoolsCount, rewPools.provider.name, t)
			}

			if rewPools.accessPoint.enabled {
				checkAndFinalizeRewPool(stAccessPointRewPoolBal, billAmount/rewPoolsCount, rewPools.accessPoint.name, t)
			}

			if rewPools.all.enabled {
				checkAndFinalizeRewPool(stAllRewPoolBal, billAmount/rewPoolsCount, rewPools.all.name, t)
			}

			require.Equal(
				t,
				stConsBal, getConsumerBalance(t),
				"Resulting and starting consumer balances should be equal",
			)

			checkBillingPayments(t, sessionID, stProvBal, stApBal)
		}
	)

	simpleSessionActivity(userIMSI, accessPointID, sessionID, emptyAction(), afterUpdate, afterStop, t)
}
