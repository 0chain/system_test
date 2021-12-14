package apimodel

type Balance struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
}

type Transaction struct {
	Hash              string `json:"hash"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	ChainId           string `json:"chain_id"`
	PublicKey         string `json:"public_key,omitempty"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	Signature         string `json:"signature"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionStatus int    `json:"transaction_status"`
}

type SmartContractTxnData struct {
	Name      string      `json:"name"`
	InputArgs interface{} `json:"input"`
}

type Block struct {
	Block struct {
		Version                        string `json:"version"`
		CreationDate                   int64  `json:"creation_date"`
		LatestFinalizedMagicBlockHash  string `json:"latest_finalized_magic_block_hash"`
		LatestFinalizedMagicBlockRound int64  `json:"latest_finalized_magic_block_round"`
		PrevHash                       string `json:"prev_hash"`
		PrevVerificationTickets        []struct {
			VerifierId string `json:"verifier_id"`
			Signature  string `json:"signature"`
		} `json:"prev_verification_tickets"`
		MinerId             string         `json:"miner_id"`
		Round               int64          `json:"round"`
		RoundRandomSeed     int64          `json:"round_random_seed"`
		RoundTimeoutCount   int64          `json:"round_timeout_count"`
		StateHash           string         `json:"state_hash"`
		Transactions        []*Transaction `json:"transactions"`
		VerificationTickets []struct {
			VerifierId string `json:"verifier_id"`
			Signature  string `json:"signature"`
		} `json:"verification_tickets"`
		Hash            string  `json:"hash"`
		Signature       string  `json:"signature"`
		ChainId         string  `json:"chain_id"`
		ChainWeight     float64 `json:"chain_weight"`
		RunningTxnCount int     `json:"running_txn_count"`
	} `json:"block"`
}

type Transfer struct {
	Minter string `json:"minter"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}
