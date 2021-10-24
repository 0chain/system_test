package zcnsc

import (
	client2 "github.com/0chain/system_test/tests/cli_tests/zcnsc/client"
)

// ToWzcn converts from ZCN to WZNC
// Flow:
// 1. User burns token
// 2. Authorizer sends the user proof-of-Burn ticket
// 3. User gathers tickets from authorizers
// 4. User sends tickets to ETH bridge
func ToWzcn(amount float64, nonce int64) {
	transaction := client2.Burn(amount, nonce)
	client2.CheckBalance()

	authorizersFromChain := client2.GetAuthorizersFromChain()

	client2.GetBurnProofTickets(authorizersFromChain, transaction.GetTransactionHash())
	client2.SendTicketsToEthereumBridge()
}
