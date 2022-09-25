package wait

import (
	"time"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(duration time.Duration, callback func() bool) {
	ticker := time.NewTicker(time.Millisecond * 500)

	defer ticker.Stop()

	after := time.After(duration)
	postAfter := time.After(time.Second * 15)

primary:
	for range ticker.C {
		select {
		case <-after:
			break primary
		default:
			if callback() {
				break primary
			}
		}
	}

	for range ticker.C {
		select {
		case <-postAfter:
			return
		default:
		}
	}
}
