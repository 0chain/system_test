package bandwidth_marketplace

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/ap"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/magma"
)

func TestStressUsing(t *testing.T) {
	if !testCfg.Cases.Stress.Enable {
		t.Skip()
	}
	var (
		accessPoint *magmasc.AccessPoint
		err         error
	)
	accessPoint, err = ap.RegisterAndStake(testCfg.AccessPoint.KeysFile, testCfg.AccessPoint.NodeDir, testCfg)
	require.NoError(t, err)

	// pouring consumer wallet if configured.
	pourConsumerWallet(t)

	pourProviderWalletAndStake(t)

	results := make(chan stressReport)

	for i := 0; i < testCfg.Cases.Stress.UsersNum; i++ {
		i := i
		go func() {
			userID := "User_" + strconv.Itoa(i)
			sessionID := "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())
			err := stressUsing(userID, sessionID, accessPoint.Id)
			if err != nil {
				results <- stressReport{err: err, sessionID: sessionID}
				return
			}
			results <- stressReport{sessionID: sessionID}
		}()
	}

	var (
		tries int
	)
	for tries < testCfg.Cases.Stress.UsersNum {
		rep := <-results
		if rep.err != nil {
			log.Logger.Debug("Success tries", zap.Int("tries", tries))
			t.Fatalf("got error for session: %s, err: %v", rep.sessionID, rep.err)
		}
		tries++
	}
}

type stressReport struct {
	err       error
	sessionID string
}

func stressUsing(userID, sessionID, apID string) error {
	var (
		userIMSI = userID
	)

	err := magma.SessionStart(userIMSI, apID, sessionID, testCfg)
	if err != nil {
		return err
	}

	var (
		sessionStartTime    = time.Now()
		numUpdates          = rand.Intn(10)
		octetsIn, octetsOut uint64
		sessTime            uint32
	)
	for i := 0; i < numUpdates; i++ {
		octetsIn += uint64(rand.Intn(10000))
		octetsOut += uint64(rand.Intn(10000))
		sessTime = uint32(time.Since(sessionStartTime).Seconds())

		err = magma.SessionUpdate(userIMSI,
			apID,
			sessionID,
			sessTime,
			octetsIn,
			octetsOut,
			testCfg,
		)
		if err != nil {
			return err
		}
	}

	err = magma.SessionStop(
		userIMSI,
		apID,
		sessionID,
		sessTime,
		octetsIn,
		octetsOut,
		testCfg,
	)
	if err != nil {
		return err
	}

	return nil
}
