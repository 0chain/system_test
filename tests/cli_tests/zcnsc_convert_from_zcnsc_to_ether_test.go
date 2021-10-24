package cli_tests

import (
	"github.com/0chain/system_test/tests/cli_tests/zcnsc"
	"github.com/0chain/system_test/tests/cli_tests/zcnsc/client"
	"testing"
)

func TestWzcnToZcnConversion(t *testing.T) {
	// Bootstrapping

	client.InitClient()
	client.CheckBalance()
	client.PourTokens(100)
	client.CheckBalance()

	// Converting from ZCN to WZCN

	for i := 1; i < 4; i++ {
		zcnsc.ToWzcn(1, int64(i))
	}
}
