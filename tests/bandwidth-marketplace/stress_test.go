package bandwidth_marketplace

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/magma"
)

func TestStressUsing(t *testing.T) {
	if !testCfg.Cases.Stress.Enable {
		t.Skip()
	}

	// pouring consumer wallet if configured.
	pourConsumerWallet(t)

	results := make(chan stressReport)

	for i := 0; i < testCfg.Cases.Stress.UsersNum; i++ {
		i := i
		go func() {
			userID := "User_" + strconv.Itoa(i)
			sessionID := "test_session_id_" + strconv.Itoa(time.Now().Nanosecond())
			err := stressUsing(userID, sessionID)
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

func stressUsing(userID, sessionID string) error {
	var (
		userIMSI     = userID
		providerAPID = "access-point-id-1"
	)

	err := magma.SessionStart(userIMSI, providerAPID, sessionID, testCfg)
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
			providerAPID,
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
		providerAPID,
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
