package zcnsc

import (
	"github.com/0chain/system_test/internal/cli/client"
)

// ToWzcn converts from ZCN to WZNC
// Flow:
// 1. User burns token
// 2. Authorizer sends the user proof-of-Burn ticket
// 3. User gathers tickets from authorizers
// 4. User sends tickets to ETH bridge
func ToWzcn(amount float64, nonce int64) {
	transaction := client.Burn(amount, nonce)
	client.CheckBalance()

	authorizersFromChain := client.GetAuthorizersFromChain()

	client.GetBurnProofTickets(authorizersFromChain, transaction.GetTransactionHash())
	client.SendTicketsToEthereumBridge()
}
