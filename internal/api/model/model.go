package model

import (
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"io"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
)

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
	Terms           climodel.Terms         `json:"terms"`     // terms
	Capacity        int64                  `json:"capacity"`  // total blobber capacity
	Allocated       int64                  `json:"allocated"` // allocated capacity
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

//
//type KeyPair struct {
//	PublicKey  bls.PublicKey
//	PrivateKey bls.SecretKey
//}

//type Confirmation struct {
//	Version               string          `json:"version"`
//	Hash                  string          `json:"hash"`
//	BlockHash             string          `json:"block_hash"`
//	PreviousBlockHash     string          `json:"previous_block_hash"`
//	Transaction           *Transaction    `json:"txn,omitempty"`
//	CreationDate          int64           `json:"creation_date,omitempty"`
//	MinerID               string          `json:"miner_id"`
//	Round                 int64           `json:"round"`
//	Status                int             `json:"transaction_status"`
//	RoundRandomSeed       int64           `json:"round_random_seed"`
//	StateChangesCount     int             `json:"state_changes_count"`
//	MerkleTreeRoot        string          `json:"merkle_tree_root"`
//	MerkleTreePath        *MerkleTreePath `json:"merkle_tree_path"`
//	ReceiptMerkleTreeRoot string          `json:"receipt_merkle_tree_root"`
//	ReceiptMerkleTreePath *MerkleTreePath `json:"receipt_merkle_tree_path"`
//}

type MerkleTreePath struct {
	Nodes     []string `json:"nodes"`
	LeafIndex int      `json:"leaf_index"`
}

type Allocation struct {
	sdk.Allocation
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

type GetBlobberResponse struct {
	ID                string            `json:"id"`
	BaseURL           string            `json:"url"`
	Terms             Terms             `json:"terms"`
	Capacity          int64             `json:"capacity"`
	Allocated         int64             `json:"allocated"`
	LastHealthCheck   int64             `json:"last_health_check"`
	PublicKey         string            `json:"-"`
	StakePoolSettings StakePoolSettings `json:"stake_pool_settings"`
	TotalStake        int64             `json:"total_stake"`
}

type StakePoolSettings struct {
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int     `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
}

type Terms struct {
	ReadPrice        int64         `json:"read_price"`
	WritePrice       int64         `json:"write_price"`
	MinLockDemand    float64       `json:"min_lock_demand"`
	MaxOfferDuration time.Duration `json:"max_offer_duration"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type ClientPutWalletRequest struct {
	Id        string `json:"id"`
	PublicKey string `json:"public_key"`
	//CreationDate *int   `json:"creation_date"`
}

type ClientPutWalletResponse struct {
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

type BlobberUploadFileMeta struct {
	ConnectionID string `json:"connection_id" validation:"required"`
	FileName     string `json:"filename" validation:"required"`
	FilePath     string `json:"filepath" validation:"required"`
	ActualHash   string `json:"actual_hash,omitempty" validation:"required"`
	ContentHash  string `json:"content_hash" validation:"required"`
	MimeType     string `json:"mimetype" validation:"required"`
	ActualSize   int64  `json:"actual_size,omitempty" validation:"required"`
	IsFinal      bool   `json:"is_final" validation:"required"`
}

type BlobberUploadFileRequest struct {
	URL, ClientID, ClientKey, ClientSignature, AllocationID string
	File                                                    io.Reader
	Meta                                                    BlobberUploadFileMeta
}

type BlobberUploadFileResponse struct {
}

type BlobberListFilesRequest struct {
	sys.KeyPair
	URL, ClientID, ClientKey, ClientSignature, AllocationID, PathHash, Path string
}

type BlobberListFilesResponse struct {
}

type BlobberCommitConnectionWriteMarker struct {
	AllocationRoot string `json:"allocation_root"`
	AllocationID   string `json:"allocation_id"`
	BlobberID      string `json:"blobber_id"`
	ClientID       string `json:"client_id"`
	Signature      string `json:"signature"`
	Name           string `json:"name"`
	ContentHash    string `json:"content_hash"`
	LookupHash     string `json:"lookup_hash"`
	Timestamp      int64  `json:"timestamp"`
	Size           int64  `json:"size"`
}

type BlobberCommitConnectionRequest struct {
	URL, ConnectionID, ClientKey string
	WriteMarker                  BlobberCommitConnectionWriteMarker
}

type BlobberCommitConnectionResponse struct{}

type BlobberGetFileReferencePathRequest struct {
	URL, ClientID, ClientKey, ClientSignature, AllocationID string
}

type BlobberGetFileReferencePathResponse struct {
	sdk.ReferencePathResult
}

type CreateStakePoolRequest struct {
	BlobberID string `json:"blobber_id"`
}

type BlobberDownloadFileReadMarker struct {
	ClientID     string `json:"client_id"`
	ClientKey    string `json:"client_public_key"`
	BlobberID    string `json:"blobber_id"`
	AllocationID string `json:"allocation_id"`
	OwnerID      string `json:"owner_id"`
	Signature    string `json:"signature"`
	Timestamp    int64  `json:"timestamp"`
	Counter      int64  `json:"counter"`
}

type BlobberDownloadFileRequest struct {
	ReadMarker BlobberDownloadFileReadMarker
	URL        string
	PathHash   string
	BlockNum   string
	NumBlocks  string
}

type BlobberDownloadFileResponse struct {
}
