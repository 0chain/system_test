package wait

import (
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(t *test.SystemTest, duration time.Duration, predicate func() bool) {
	backoffPeriod := time.Second * 1
	ticker := time.NewTicker(backoffPeriod)

	defer ticker.Stop()

	after := time.After(duration)

	for range ticker.C {
		select {
		case <-after:
			t.Fatal("Timed out waiting for wait condition to pass")
			return
		default:
			if predicate() {
				t.Log("Wait pool callback has succeed")
				return
			}
		}
	}
}
