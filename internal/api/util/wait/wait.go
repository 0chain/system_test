package wait

import (
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(t *test.SystemTest, duration time.Duration, predicate func() bool) {
	backoffPeriod := time.Second * 2
	ticker := time.NewTicker(backoffPeriod)

	defer ticker.Stop()

	after := time.After(duration)

	for range ticker.C {
		select {
		case <-after:
			t.Fatal("Timed out waiting for wait condition to pass")
		default:
			if predicate() {
				t.Log("Wait condition has succeed")
				return
			}
			t.Logf("Wait condition failed. Waiting an additional [%v]...", backoffPeriod)
		}
	}
}
