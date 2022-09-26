package wait

import (
	"log"
	"time"
)

// PoolImmediately pools passed function for a certain amount of time
func PoolImmediately(duration time.Duration, callback func() bool) {
	ticker := time.NewTicker(time.Millisecond * 500)

	defer ticker.Stop()

	after := time.After(duration)
	postAfter := time.After(time.Second * 5)

primary:
	for range ticker.C {
		select {
		case <-after:
			break primary
		default:
			if callback() {
				log.Println("Wait pool callback has succeed")
				break primary
			}
			log.Println("Wait pool callback has failed")
		}
	}

	log.Println("Wait pool received a timeout")

	for range ticker.C {
		select {
		case <-postAfter:
			return
		default:
		}
	}
}
