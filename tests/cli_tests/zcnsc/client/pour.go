package client

import "github.com/0chain/gosdk/zcncore"

func PourTokens(amount float64) zcncore.TransactionScheme {
	return StartAndVerifyTransaction(
		"Faucet",
		"pour",
		zcncore.FaucetSmartContractAddress,
		"new wallet",
		amount,
	)
}