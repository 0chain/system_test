package api_tests

import (
	"encoding/json"
	"encoding/base64"
	"testing"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"github.com/0chain/gosdk/core/zcncrypto"
)


func TestSplitKey(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Check if Splitkey handler is generating split keys or not")

	t.Run("Check if Splitkey handler is generating split keys or not", func(t *test.SystemTest) {
		wallet := createWallet(t)
		privateKey := wallet.Keys.PrivateKey
		serializedPrivateKey := privateKey.Serialize()
		stringPrivateKey := base64.StdEncoding.EncodeToString(serializedPrivateKey)
		// this represents number of split keys made from private key
		numSplit := 2
		wStr, err := zcncore.SplitKeys(stringPrivateKey, numSplit)
		require.NoError(t, err, "failed to create split key wallet using private key")
		var splitKeyWallet  zcncrypto.Wallet
		err = json.Unmarshal([]byte(wStr), &splitKeyWallet)
		require.NoError(t, err, "failed to deserialize wallet object", wStr)
		require.NotNil(t, splitKeyWallet)
		require.Equal(t, splitKeyWallet.ClientID, wallet.Id)
		require.NotNil(t, splitKeyWallet.Keys[0].PrivateKey)
		require.NotNil(t, splitKeyWallet.Keys[0].PublicKey)
	})
}
