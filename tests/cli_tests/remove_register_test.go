package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"
)

func TestRemoveRegister(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	registerWalletForName(t, configPath, "randomWalletName")

	balance, err := getBalanceForWallet(t, configPath, "randomWalletName")
	fmt.Println(balance, err)

	//getWalletForName(t, configPath, "randomWalletName")

	executeFaucetWithTokensForWallet(t, "randomWalletName", configPath, 5)

	balance, err = getBalanceForWallet(t, configPath, "randomWalletName")
	fmt.Println(balance, err)

}
