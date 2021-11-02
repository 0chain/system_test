package bandwidth_marketplace

import (
	"crypto/rand"
	"math/big"
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

	if err := magma.SessionStart(userIMSI, apID, sessionID, testCfg); err != nil {
		return err
	}

	var (
		sessionStartTime    = time.Now()
		octetsIn, octetsOut uint64
		sessTime            uint32
	)
	numUpdates, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return err
	}
	for i := 0; i < int(numUpdates.Int64()); i++ {
		rndIn, rndOut, err := randOctets(10000)
		if err != nil {
			return err
		}
		sessTime = uint32(time.Since(sessionStartTime).Seconds())

		err = magma.SessionUpdate(userIMSI,
			apID,
			sessionID,
			sessTime,
			octetsIn+rndIn,
			octetsOut+rndOut,
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

func randOctets(max int64) (in, out uint64, err error) {
	inBI, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, 0, err
	}
	outBI, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, 0, err
	}
	return inBI.Uint64(), outBI.Uint64(), nil
}
