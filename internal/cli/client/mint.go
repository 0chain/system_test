package client

import (
	"encoding/json"
	"github.com/0chain/gosdk/zcncore"
)

func Mint(amount float64, nonce int64, ethTxnID string) zcncore.TransactionScheme {
	payload := &MintPayload{
		EthereumTxnID:     ethTxnID,
		Amount:            0,
		Nonce:             0,
		Signatures:        nil,
		ReceivingClientID: "",
	}

	buffer, _ := json.Marshal(payload)

	return StartAndVerifyTransaction(
		"ZCNSC",
		"mint",
		ZcnscAddress,
		string(buffer),
		amount,
	)
}

