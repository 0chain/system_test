package apimodel

import (
	"github.com/0chain/gosdk/core/common"
	climodel "github.com/0chain/system_test/internal/cli/model"
)

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

type StorageNodeGeolocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// reserved / Accuracy float64 `mapstructure:"accuracy"`
}

type ValidationNode struct {
	ID                string                     `json:"id"`
	BaseURL           string                     `json:"url"`
	PublicKey         string                     `json:"-"`
	StakePoolSettings climodel.StakePoolSettings `json:"stake_pool_settings"`
}

type StorageNode struct {
	ID              string                 `json:"id"`
	BaseURL         string                 `json:"url"`
	Geolocation     StorageNodeGeolocation `json:"geolocation"`
	Terms           climodel.Terms         `json:"terms"`    // terms
	Capacity        int64                  `json:"capacity"` // total blobber capacity
	Used            int64                  `json:"used"`     // allocated capacity
	LastHealthCheck common.Timestamp       `json:"last_health_check"`
	PublicKey       string                 `json:"-"`
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings climodel.StakePoolSettings `json:"stake_pool_settings"`
}

type ValidationTicket struct {
	ChallengeID  string           `json:"challenge_id"`
	BlobberID    string           `json:"blobber_id"`
	ValidatorID  string           `json:"validator_id"`
	ValidatorKey string           `json:"validator_key"`
	Result       bool             `json:"success"`
	Message      string           `json:"message"`
	MessageCode  string           `json:"message_code"`
	Timestamp    common.Timestamp `json:"timestamp"`
	Signature    string           `json:"signature"`
}

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type StorageChallenge struct {
	Created        common.Timestamp   `json:"created"`
	ID             string             `json:"id"`
	PrevID         string             `json:"prev_id"`
	Validators     []*ValidationNode  `json:"validators"`
	RandomNumber   int64              `json:"seed"`
	AllocationID   string             `json:"allocation_id"`
	Blobber        *StorageNode       `json:"blobber"`
	AllocationRoot string             `json:"allocation_root"`
	Response       *ChallengeResponse `json:"challenge_response,omitempty"`
}

type BlobberChallenge struct {
	BlobberID                string                       `json:"blobber_id"`
	Challenges               []*StorageChallenge          `json:"challenges"`
	ChallengeMap             map[string]*StorageChallenge `json:"-"`
	LatestCompletedChallenge *StorageChallenge            `json:"lastest_completed_challenge"`
}
