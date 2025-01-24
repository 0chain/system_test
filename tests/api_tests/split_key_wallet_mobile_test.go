package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0chain/gosdk/mobilesdk/sdk"
	"github.com/0chain/gosdk_common/core/zcncrypto"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestSplitKeyMobile(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Check if Splitkey handler is generating split keys or not")

	t.Run("Check if Splitkey handler is generating split keys or not", func(t *test.SystemTest) {
		wallet := createWallet(t)
		privateKey := wallet.Keys.PrivateKey
		serializedPrivateKey := privateKey.Serialize()
		stringPrivateKey := base64.StdEncoding.EncodeToString(serializedPrivateKey)
		// this represents number of split keys made from private key
		numSplit := 2
		signatureScheme := "bls0chain"
		wStr, err := sdk.SplitKeys(stringPrivateKey, signatureScheme, numSplit)
		if err != nil {
			fmt.Println("Error while spliting keys:", err)
			return
		}
		var splitKeyWallet zcncrypto.Wallet
		err = json.Unmarshal([]byte(wStr), &splitKeyWallet)
		if err != nil {
			fmt.Println("Error while unmarshalling split key wallet:", err)
			return
		}
		require.Nil(t, err)
		require.NotNil(t, splitKeyWallet)
		require.Equal(t, splitKeyWallet.ClientID, wallet.Id)
		require.NotNil(t, splitKeyWallet.Keys[0].PrivateKey)
		require.NotNil(t, splitKeyWallet.Keys[0].PublicKey)
	})
}
