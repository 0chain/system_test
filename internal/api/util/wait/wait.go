package wait

import (
	"testing"
	"time"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(t *testing.T, duration time.Duration, callback func() bool) {
	ticker := time.NewTicker(time.Millisecond * 500)

	defer ticker.Stop()

	after := time.After(duration)

	for range ticker.C {
		select {
		case <-after:
			t.Fatal("Wait pool received a timeout")
		default:
			if callback() {
				t.Log("Wait pool callback has succeed")
				return
			}
			t.Log("Wait pool callback has failed, continue...")
		}
	}
}
