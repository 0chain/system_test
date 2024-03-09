package api_tests

import (
	"fmt"
	"testing"
	"encoding/base64"
	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestSetWalletInfo(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Check if setWalletInfo is setting splitkey wallet or not")

	t.Run("Check if setWalletInfo is setting splitkey wallet or not", func(t *test.SystemTest) {
		wallet := createWallet(t)
		privateKey := wallet.Keys.PrivateKey
		serializedPrivateKey := privateKey.Serialize()
		stringPrivateKey := base64.StdEncoding.EncodeToString(serializedPrivateKey)
		// this represents number of split keys made from private key
		numSplit := 2
		splitKeyWallet, err := zcncore.SplitKeys(stringPrivateKey, numSplit)
		if err != nil {
			fmt.Println("Error while spliting keys:", err)
			return
		}
		// setwallet info takes a json string wallet (marshalled wallet)
		err = zcncore.SetWalletInfo(splitKeyWallet, true)
		require.Nil(t, err)
		require.NotNil(t, splitKeyWallet)
	})
}
