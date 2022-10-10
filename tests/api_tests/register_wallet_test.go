package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"testing"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		apiClient.RegisterWalletWithAssertions(t, "", "", nil, true, client.HttpOkStatus)
	})
}
