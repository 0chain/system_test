package api_tests

import (
	"testing"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register valid wallet", func(t *testing.T) {
		t.Parallel()

		r, _ := zeroChain.getFromMiners(t, "/v1/chain/get/stats")
		println("RANDOM RESPONSE: " + r.String())

	})
}
