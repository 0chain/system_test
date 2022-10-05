package model

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/core/transaction"
	"github.com/0chain/gosdk/zboxcore/sdk"

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

type EventDbTransaction struct {
	Hash              string `json:"hash" `
	BlockHash         string `json:"block_hash"`
	Round             int64  `json:"round"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id" `
	ToClientId        string `json:"to_client_id" `
	TransactionData   string `json:"transaction_data"`
	Value             int64  `json:"value"`
	Signature         string `json:"signature"`
	CreationDate      int64  `json:"creation_date"  `
	Fee               int64  `json:"fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output"`
	OutputHash        string `json:"output_hash"`
	Status            int    `json:"status"`
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

type EventDbBlock struct {
	Hash                  string               `json:"hash"`
	Version               string               `json:"version"`
	CreationDate          int64                `json:"creation_date" `
	Round                 int64                `json:"round" `
	MinerID               string               `json:"miner_id"`
	RoundRandomSeed       int64                `json:"round_random_seed"`
	MerkleTreeRoot        string               `json:"merkle_tree_root"`
	StateHash             string               `json:"state_hash"`
	ReceiptMerkleTreeRoot string               `json:"receipt_merkle_tree_root"`
	NumTxns               int                  `json:"num_txns"`
	MagicBlockHash        string               `json:"magic_block_hash"`
	PrevHash              string               `json:"prev_hash"`
	Signature             string               `json:"signature"`
	ChainId               string               `json:"chain_id"`
	RunningTxnCount       string               `json:"running_txn_count"`
	RoundTimeoutCount     int                  `json:"round_timeout_count"`
	CreatedAt             time.Time            `json:"created_at"`
	Transactions          []EventDbTransaction `json:"transactions"`
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
	sdk.Allocation
}

type Timestamp int64

type AllocationUpdate struct {
	ID                   string    `json:"id"`              // allocation id
	Name                 string    `json:"name"`            // allocation name
	OwnerID              string    `json:"owner_id"`        // Owner of the allocation
	Size                 int64     `json:"size"`            // difference
	Expiration           Timestamp `json:"expiration_date"` // difference
	SetImmutable         bool      `json:"set_immutable"`
	UpdateTerms          bool      `json:"update_terms"`
	AddBlobberId         string    `json:"add_blobber_id"`
	RemoveBlobberId      string    `json:"remove_blobber_id"`
	ThirdPartyExtendable bool      `json:"third_party_extendable"`
	FileOptions          uint8     `json:"file_options"`
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

type BlobberRequirements struct {
	Blobbers        *[]string  `json:"blobbers"`
	DataShards      int64      `json:"data_shards"`
	ParityShards    int64      `json:"parity_shards"`
	Size            int64      `json:"size"`
	OwnerId         string     `json:"owner_id"`
	OwnerPublicKey  string     `json:"owner_public_key"`
	ExpirationDate  int64      `json:"expiration_date"`
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`
}

type ClientPutWalletRequest struct {
	Id           string `json:"id"`
	PublicKey    string `json:"public_key"`
	CreationDate *int   `json:"creation_date"`
}

type ClientPutWalletResponse struct {
	Id           string `json:"id"`
	Version      string `json:"version"`
	CreationDate *int   `json:"creation_date"`
	PublicKey    string `json:"public_key"`
	Nonce        int
}

type Wallet struct {
	ClientID    string         `json:"client_id"`
	ClientKey   string         `json:"client_key"`
	Keys        []*sys.KeyPair `json:"keys"`
	Mnemonics   string         `json:"mnemonics"`
	Version     string         `json:"version"`
	DateCreated string         `json:"date_created"`
	Nonce       int
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

type MinerStats struct {
	BlockFinality      float64                  `json:"block_finality"`
	LastFinalizedRound int64                    `json:"last_finalized_round"`
	BlocksFinalized    int64                    `json:"blocks_finalized"`
	StateHealth        int64                    `json:"state_health"`
	CurrentRound       int64                    `json:"current_round"`
	RoundTimeout       int64                    `json:"round_timeout"`
	Timeouts           int64                    `json:"timeouts"`
	AverageBlockSize   int                      `json:"average_block_size"`
	NetworkTime        map[string]time.Duration `json:"network_times"`
}

type SharderStats struct {
	LastFinalizedRound     int64   `json:"last_finalized_round"`
	StateHealth            int64   `json:"state_health"`
	AverageBlockSize       int     `json:"average_block_size"`
	PrevInvocationCount    uint64  `json:"previous_invocation_count"`
	PrevInvocationScanTime string  `json:"previous_incovcation_scan_time"`
	MeanScanBlockStatsTime float64 `json:"mean_scan_block_stats_time"`
}

type SharderSCStateResponse struct {
	ID        string    `json:"ID"`
	StartTime time.Time `json:"StartTime"`
	Used      float64   `json:"Used"`
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

type BlobberDeleteConnectionRequest struct {
	URL, ConnectionId, ClientKey, ClientSignature, remotePath string
	// WriteMarket                  BlobberCommitConnectionWriteMarker
}
type BlobberCommitConnectionResponse struct{}

type BlobberGetFileReferencePathRequest struct {
	URL, ClientID, ClientKey, ClientSignature, AllocationID string
}

type BlobberGetFileReferencePathResponse struct {
	sdk.ReferencePathResult
}

func (w *Wallet) MustConvertDateCreatedToInt() int {
	result, err := strconv.Atoi(w.DateCreated)
	if err != nil {
		log.Fatalln(err)
	}
	return result
}

func (w *Wallet) String() string {
	out, err := json.Marshal(w)
	if err != nil {
		return "failed to serialize wallet object"
	}

	return string(out)
}

func (s *StubStatusBar) Started(allocationId, filePath string, op int, totalBytes int) {
}
func (s *StubStatusBar) InProgress(allocationId, filePath string, op int, completedBytes int, data []byte) {
}

func (s *StubStatusBar) Completed(allocationId, filePath string, filename string, mimetype string, size int, op int) {
}

func (s *StubStatusBar) Error(allocationID string, filePath string, op int, err error) {
}

func (s *StubStatusBar) CommitMetaCompleted(request, response string, txn *transaction.Transaction, err error) {
	fmt.Println(response, err)
}

func (s *StubStatusBar) RepairCompleted(filesRepaired int) {
}

type StubStatusBar struct {
}

type CreateStakePoolRequest struct {
	BlobberID string `json:"blobber_id"`
	PoolID    string `json:"pool_id"`
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
