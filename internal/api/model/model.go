package model

import (
	"encoding/json"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/herumi/bls-go-binary/bls"
)

type Balance struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
}

type TransactionResponse struct {
	Async  bool        `json:"async"`
	Entity Transaction `json:"entity"`
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
	TransactionNonce  int    `json:"transaction_nonce"`
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

type LatestFinalizedBlock struct {
	CreationDate      int64  `json:"creation_date"`
	Hash              string `json:"hash,omitempty"`
	StateHash         string `json:"state_hash"`
	MinerId           string `json:"miner_id"`
	Round             int64  `json:"round"`
	StateChangesCount int    `json:"state_changes_count"`
	NumTxns           int    `json:"num_txns"`
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
	LastHealthCheck int64                  `json:"last_health_check"`
	PublicKey       string                 `json:"-"`
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings climodel.StakePoolSettings `json:"stake_pool_settings"`
}

type ValidationTicket struct {
	ChallengeID  string `json:"challenge_id"`
	BlobberID    string `json:"blobber_id"`
	ValidatorID  string `json:"validator_id"`
	ValidatorKey string `json:"validator_key"`
	Result       bool   `json:"success"`
	Message      string `json:"message"`
	MessageCode  string `json:"message_code"`
	Timestamp    int64  `json:"timestamp"`
	Signature    string `json:"signature"`
}

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type StorageChallenge struct {
	Created         int64  `json:"created"`
	ID              string `json:"id"`
	TotalValidators int    `json:"total_validators"`
	AllocationID    string `json:"allocation_id"`
	BlobberID       string `json:"blobber_id"`
	Responded       bool   `json:"responded"`
}

type BlobberChallenge struct {
	BlobberID  string       `json:"blobber_id"`
	Challenges []Challenges `json:"challenges"`
}

type Challenges struct {
	ID             string            `json:"id"`
	Created        int64             `json:"created"`
	Validators     []*ValidationNode `json:"validators"`
	RandomNumber   int64             `json:"seed"`
	AllocationID   string            `json:"allocation_id"`
	AllocationRoot string            `json:"allocation_root"`
	BlobberID      string            `json:"blobber_id"`
	Responded      bool              `json:"responded"`
}

type KeyPair struct {
	PublicKey  bls.PublicKey
	PrivateKey bls.SecretKey
}

type Confirmation struct {
	Version               string          `json:"version"`
	Hash                  string          `json:"hash"`
	BlockHash             string          `json:"block_hash"`
	PreviousBlockHash     string          `json:"previous_block_hash"`
	Transaction           *Transaction    `json:"txn,omitempty"`
	CreationDate          int64           `json:"creation_date,omitempty"`
	MinerID               string          `json:"miner_id"`
	Round                 int64           `json:"round"`
	Status                int             `json:"transaction_status"`
	RoundRandomSeed       int64           `json:"round_random_seed"`
	StateChangesCount     int             `json:"state_changes_count"`
	MerkleTreeRoot        string          `json:"merkle_tree_root"`
	MerkleTreePath        *MerkleTreePath `json:"merkle_tree_path"`
	ReceiptMerkleTreeRoot string          `json:"receipt_merkle_tree_root"`
	ReceiptMerkleTreePath *MerkleTreePath `json:"receipt_merkle_tree_path"`
}

type MerkleTreePath struct {
	Nodes     []string `json:"nodes"`
	LeafIndex int      `json:"leaf_index"`
}

type Allocation struct {
	ID             string           `json:"id"`
	Tx             string           `json:"tx"`
	DataShards     int              `json:"data_shards"`
	ParityShards   int              `json:"parity_shards"`
	Size           int64            `json:"size"`
	Expiration     int64            `json:"expiration_date"`
	Owner          string           `json:"owner_id"`
	OwnerPublicKey string           `json:"owner_public_key"`
	Payer          string           `json:"payer_id"`
	Blobbers       []*StorageNode   `json:"blobbers"`
	Stats          *AllocationStats `json:"stats"`
	TimeUnit       time.Duration    `json:"time_unit"`
	IsImmutable    bool             `json:"is_immutable"`

	BlobberDetails []*BlobberAllocation `json:"blobber_details"`

	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`

	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
	StartTime               int64         `json:"start_time"`
	Finalized               bool          `json:"finalized,omitempty"`
	Canceled                bool          `json:"canceled,omitempty"`
	MovedToChallenge        int64         `json:"moved_to_challenge,omitempty"`
	MovedBack               int64         `json:"moved_back,omitempty"`
	MovedToValidators       int64         `json:"moved_to_validators,omitempty"`
	Curators                []string      `json:"curators"`
}

type AllocationStats struct {
	UsedSize                  int64  `json:"used_size"`
	NumWrites                 int64  `json:"num_of_writes"`
	NumReads                  int64  `json:"num_of_reads"`
	TotalChallenges           int64  `json:"total_challenges"`
	OpenChallenges            int64  `json:"num_open_challenges"`
	SuccessChallenges         int64  `json:"num_success_challenges"`
	FailedChallenges          int64  `json:"num_failed_challenges"`
	LastestClosedChallengeTxn string `json:"latest_closed_challenge"`
}

type BlobberAllocation struct {
	BlobberID       string `json:"blobber_id"`
	Size            int64  `json:"size"`
	Terms           Terms  `json:"terms"`
	MinLockDemand   int64  `json:"min_lock_demand"`
	Spent           int64  `json:"spent"`
	Penalty         int64  `json:"penalty"`
	ReadReward      int64  `json:"read_reward"`
	Returned        int64  `json:"returned"`
	ChallengeReward int64  `json:"challenge_reward"`
	FinalReward     int64  `json:"final_reward"`
}

type Terms struct {
	ReadPrice               int64         `json:"read_price"`
	WritePrice              int64         `json:"write_price"`
	MinLockDemand           float64       `json:"min_lock_demand"`
	MaxOfferDuration        time.Duration `json:"max_offer_duration"`
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type BlobberRequirements struct {
	Blobbers                   *[]string  `json:"blobbers"`
	DataShards                 int64      `json:"data_shards"`
	ParityShards               int64      `json:"parity_shards"`
	Size                       int64      `json:"size"`
	OwnerId                    string     `json:"owner_id"`
	OwnerPublicKey             string     `json:"owner_public_key"`
	ExpirationDate             int64      `json:"expiration_date"`
	ReadPriceRange             PriceRange `json:"read_price_range"`
	WritePriceRange            PriceRange `json:"write_price_range"`
	MaxChallengeCompletionTime int64      `json:"max_challenge_completion_time"`
}

type Wallet struct {
	Id           string `json:"id"`
	Version      string `json:"version"`
	CreationDate *int   `json:"creation_date"`
	PublicKey    string `json:"public_key"`
	Nonce        int
}

type ChallengeEntity struct {
	ChallengeID             string   `json:"id"`
	PrevChallengeID         string   `json:"prev_id"`
	RandomNumber            int64    `json:"seed"`
	AllocationID            string   `json:"allocation_id"`
	AllocationRoot          string   `json:"allocation_root"`
	RespondedAllocationRoot string   `json:"responded_allocation_root"`
	Status                  int      `json:"status"`
	Result                  int      `json:"result"`
	StatusMessage           string   `json:"status_message"`
	CommitTxnID             string   `json:"commit_txn_id"`
	BlockNum                int64    `json:"block_num"`
	RefID                   int64    `json:"-"`
	LastCommitTxnIDs        []string `json:"last_commit_txn_ids" gorm:"-"`
}

type BCChallengeResponse struct {
	BlobberID  string             `json:"blobber_id"`
	Challenges []*ChallengeEntity `json:"challenges"`
}

func (w Wallet) String() string {
	out, err := json.Marshal(w)
	if err != nil {
		return "failed to serialize wallet object"
	}

	return string(out)
}
