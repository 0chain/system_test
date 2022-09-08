package wait

import (
	"time"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(duration time.Duration, callback func() bool) {
	ticker := time.NewTicker(time.Second)
	after := time.After(duration)

	for range ticker.C {
		select {
		case <-after:
			return
		default:
			if callback() {
				return
			}
		}
	}
}
