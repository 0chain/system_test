package zcnsc

// ToZcn converts from WZCN to ZCN
// Flow:
// 1. User sends transaction to Ethereum bridge smart contract to burn ETH. -> TransactionID // TODO: call Ethereum SC
// 2. User sends transaction ID to authorizers // TODO: Handle EthIDTransaction in webserver
// 3. The authorizers monitor the Ethereum blockchain for WZCN burn transactions. // TODO: VerifyEthSc func
// 4. If the request is valid, the authorizer sends the client a proof-of-WZCN-burn ticket. // TODO: get Eth proofOfBurn func
// 5. Client gets enough tickets and then calls `Mint` method // TODO: test Mint func in Zcn SC
func ToZcn(amount float64, nonce int64) {
}
