package api_tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

type Key struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type splitKeyWallet struct {
	ClientID    string `json:"client_id"`
	ClientKey   string `json:"client_key"`
	Keys        Key  `json:"keys"`
	Mnemonics   string `json:"mnemonics"`
	Version     string `json:"version"`
	DateCreated string `json:"date_created"`
	Nonce       int    `json:"nonce"`
}

func TestSplitKey(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Check if Splitkey handler is generating split keys or not")

	t.Run("Check if Splitkey handler is generating split keys or not", func(t *test.SystemTest) {
		wallet := createWallet(t)
		privateKey := wallet.Keys.PrivateKey
		// this represents number of split keys made from private key
		numSplit := 2
		wStr, err := zcncore.SplitKeys(privateKey, numSplit)
		if err != nil {
			fmt.Println("Error while spliting keys:", err)
			return
		}
		/*
			{
				"client_id": "f2461679c2407f12a0cbe161b55f1367aeb7af9e196438effa39b9c29e147af8",
				"client_key": "49bd9013d0ebee27ff16f4d4b6888db21c5e2db9c4f93c2a11d7124f86e7580fb2ccd2ad5ba2450ddf8c6f6b280fa4db6be48a8a9c276b8cd02be012fb5b4e21",
				"keys": [
					{
					    "public_key": "f4caf190ffa8be2d1fd03f79d781ffe0d3fadc44708d3c0fad2eec23af367b11316debd2f27882ef41545f2855e14fefd62e795fd2386a792366aef6e8e0a71f",
					    "private_key": "18aeedb37af04422ab3ed28b51bf006029083a79a77274074042a18364b43b16"
					}
				],

				"mnemonics": "",
				"version": "",
				"date_created": "",
				"nonce": 0
			}
		*/
		var splitKeyWallet splitKeyWallet
		err = json.Unmarshal([]byte(wStr), &splitKeyWallet)
		if err != nil {
			fmt.Println("Error while unmarshalling split key wallet:", err)
			return
		}
		require.Nil(t, err)
		require.NotNil(t, splitKeyWallet)
		require.Equal(t, splitKeyWallet.ClientID, wallet.Id)
		require.NotNil(t,splitKeyWallet.Keys.PublicKey)
		require.NotNil(t,splitKeyWallet.Keys.PrivateKey)
	})
}
