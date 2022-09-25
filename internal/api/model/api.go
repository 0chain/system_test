package model

import (
	"encoding/json"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/gosdk/core/sys"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/herumi/bls-go-binary/bls"
	"io"
	"log"
	"strconv"
	"time"
)

type NetworkDNSResponse struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

type NetworkHealthResources struct {
	NetworkDNSResponse
}

type ExecutionRequest struct {
	FormData map[string]string

	QueryParams map[string]string

	Headers map[string]string

	Body interface{}

	Dst interface{}

	RequiredStatusCode int
}

type ClientPutRequest struct {
	ClientID     string `json:"id"`
	ClientKey    string `json:"public_key"`
	CreationDate *int   `json:"creation_date"`
}

type ClientPutResponse struct {
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
	Nonce       int            `json:"-"`
	RawKeys     *KeyPair       `json:"-"`
}

type KeyPair struct {
	PublicKey  bls.PublicKey
	PrivateKey bls.SecretKey
}

func (w *Wallet) IncNonce() {
	w.Nonce++
}

func (w *Wallet) MustGetKeyPair() *sys.KeyPair {
	if len(w.Keys) == 0 {
		log.Fatalln("wallet has no keys")
	}
	return w.Keys[0]
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

type TransactionData struct {
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

func NewFaucetTransactionData() TransactionData {
	return TransactionData{
		Name:  "pour",
		Input: "{}",
	}
}

func NewCollectRewardTransactionData(providerID, poolID string, providerType int64) TransactionData {
	var input = map[string]interface{}{
		"provider_id":   providerID,
		"provider_type": providerType,
		"pool_id":       poolID,
	}

	return TransactionData{
		Name:  "collect_reward",
		Input: input,
	}
}

func NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse SCRestGetAllocationBlobbersResponse) TransactionData {
	return TransactionData{
		Name:  "new_allocation_request",
		Input: scRestGetAllocationBlobbersResponse,
	}
}

func NewCreateStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "stake_pool_lock",
		Input: &createStakePoolRequest,
	}
}

func NewUpdateAllocationTransactionData(updateAllocationRequest UpdateAllocationRequest) TransactionData {
	return TransactionData{
		Name:  "update_allocation_request",
		Input: updateAllocationRequest,
	}
}

func NewUpdateBlobberTransactionData(scRestGetBlobberResponse *SCRestGetBlobberResponse) TransactionData {
	return TransactionData{
		Name:  "update_blobber_settings",
		Input: scRestGetBlobberResponse,
	}
}

type InternalTransactionPutRequest struct {
	TransactionData
	ToClientID, Body string
	Wallet           *Wallet
	Value            *int64
}

type TransactionPutRequest struct {
	Hash              string `json:"hash"`
	Signature         string `json:"signature"`
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionNonce  int    `json:"transaction_nonce"`
}

type TransactionEntity struct {
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionNonce  int    `json:"transaction_nonce"`
	Hash              string `json:"hash"`
	ChainId           string `json:"chain_id"`
	Signature         string `json:"signature"`
	TransactionStatus int    `json:"transaction_status"`
}

type TransactionPutResponse struct {
	Request TransactionPutRequest
	Async   bool              `json:"async"`
	Entity  TransactionEntity `json:"entity"`
}

type TransactionGetConfirmationRequest struct {
	Hash string
}

type TransactionGetConfirmationResponse struct {
	Version               string             `json:"version"`
	Hash                  string             `json:"hash"`
	BlockHash             string             `json:"block_hash"`
	PreviousBlockHash     string             `json:"previous_block_hash"`
	Transaction           *TransactionEntity `json:"txn,omitempty"`
	CreationDate          int64              `json:"creation_date,omitempty"`
	MinerID               string             `json:"miner_id"`
	Round                 int64              `json:"round"`
	Status                int                `json:"transaction_status"`
	RoundRandomSeed       int64              `json:"round_random_seed"`
	StateChangesCount     int                `json:"state_changes_count"`
	MerkleTreeRoot        string             `json:"merkle_tree_root"`
	MerkleTreePath        *MerkleTreePath    `json:"merkle_tree_path"`
	ReceiptMerkleTreeRoot string             `json:"receipt_merkle_tree_root"`
	ReceiptMerkleTreePath *MerkleTreePath    `json:"receipt_merkle_tree_path"`
}

type MerkleTreePath struct {
	Nodes     []string `json:"nodes"`
	LeafIndex int      `json:"leaf_index"`
}

type ClientGetBalanceRequest struct {
	ClientID string
}

type ClientGetBalanceResponse struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
}

type SCStateGetRequest struct {
	SCAddress, Key string
}

type SCStateGetResponse struct {
	ID        string    `json:"ID"`
	StartTime time.Time `json:"StartTime"`
	Used      float64   `json:"Used"`
}

type GetMinerStatsResponse struct {
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

type GetSharderStatsResponse struct {
	LastFinalizedRound     int64   `json:"last_finalized_round"`
	StateHealth            int64   `json:"state_health"`
	AverageBlockSize       int     `json:"average_block_size"`
	PrevInvocationCount    uint64  `json:"previous_invocation_count"`
	PrevInvocationScanTime string  `json:"previous_incovcation_scan_time"`
	MeanScanBlockStatsTime float64 `json:"mean_scan_block_stats_time"`
}

type SCRestOpenChallengeRequest struct {
	BlobberID string
}

type SCRestOpenChallengeResponse struct {
	BlobberID  string       `json:"blobber_id"`
	Challenges []*Challenge `json:"challenges"`
}

type Challenge struct {
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
	LastCommitTxnIDs        []string `json:"last_commit_txn_ids"`
}

type SCRestGetAllocationBlobbersResponse struct {
	Blobbers *[]string `json:"blobbers"`
	BlobberRequirements
}

type SCRestGetAllocationRequest struct {
	AllocationID string
}

type SCRestGetAllocationBlobbersRequest struct {
	ClientID, ClientKey string
}

type BlobberRequirements struct {
	DataShards      int64      `json:"data_shards"`
	ParityShards    int64      `json:"parity_shards"`
	Size            int64      `json:"size"`
	OwnerId         string     `json:"owner_id"`
	OwnerPublicKey  string     `json:"owner_public_key"`
	ExpirationDate  int64      `json:"expiration_date"`
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type UpdateAllocationRequest struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	OwnerID              string `json:"owner_id"`
	Size                 int64  `json:"size"`
	Expiration           int64  `json:"expiration_date"`
	SetImmutable         bool   `json:"set_immutable"`
	UpdateTerms          bool   `json:"update_terms"`
	AddBlobberId         string `json:"add_blobber_id"`
	RemoveBlobberId      string `json:"remove_blobber_id"`
	ThirdPartyExtendable bool   `json:"third_party_extendable"`
	FileOptions          uint8  `json:"file_options"`
}

type SCRestGetBlobberRequest struct {
	BlobberID string
}

type SCRestGetBlobberResponse struct {
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

type Terms struct {
	ReadPrice        int64         `json:"read_price"`
	WritePrice       int64         `json:"write_price"`
	MinLockDemand    float64       `json:"min_lock_demand"`
	MaxOfferDuration time.Duration `json:"max_offer_duration"`
}

type StakePoolSettings struct {
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int     `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
}

type CreateStakePoolRequest struct {
	ProviderType sdk.ProviderType `json:"provider_type,omitempty"`
	ProviderID   string           `json:"provider_id,omitempty"`
}

type SCRestGetAllocationResponse struct {
	ID              string           `json:"id"`
	Tx              string           `json:"tx"`
	Name            string           `json:"name"`
	DataShards      int              `json:"data_shards"`
	ParityShards    int              `json:"parity_shards"`
	Size            int64            `json:"size"`
	Expiration      int64            `json:"expiration_date"`
	Owner           string           `json:"owner_id"`
	OwnerPublicKey  string           `json:"owner_public_key"`
	Payer           string           `json:"payer_id"`
	Blobbers        []*StorageNode   `json:"blobbers"`
	Stats           *AllocationStats `json:"stats"`
	TimeUnit        time.Duration    `json:"time_unit"`
	IsImmutable     bool             `json:"is_immutable"`
	WritePool       common.Balance   `json:"write_pool"`
	ReadPriceRange  PriceRange       `json:"read_price_range"`
	WritePriceRange PriceRange       `json:"write_price_range"`
}

type StorageNode struct {
	ID                string                 `json:"id"`
	BaseURL           string                 `json:"url"`
	Geolocation       StorageNodeGeolocation `json:"geolocation"`
	Terms             Terms                  `json:"terms"`     // terms
	Capacity          int64                  `json:"capacity"`  // total blobber capacity
	Allocated         int64                  `json:"allocated"` // allocated capacity
	LastHealthCheck   int64                  `json:"last_health_check"`
	PublicKey         string                 `json:"-"`
	StakePoolSettings StakePoolSettings      `json:"stake_pool_settings"`
}

type StorageNodeGeolocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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

//type BlobberGetFileReferencePathRequest struct {
//	URL, ClientID, ClientKey, ClientSignature, AllocationID string
//}

//type BlobberGetFileReferencePathResponse struct {
//	sdk.ReferencePathResult
//}

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

type SCRestGetStakePoolStatRequest struct {
	BlobberID string
}

type SCRestGetStakePoolStatResponse struct {
	ID           string                      `json:"pool_id"`
	Balance      int64                       `json:"balance"`
	Unstake      int64                       `json:"unstake"`
	Free         int64                       `json:"free"`
	Capacity     int64                       `json:"capacity"`
	WritePrice   int64                       `json:"write_price"`
	OffersTotal  int64                       `json:"offers_total"`
	UnstakeTotal int64                       `json:"unstake_total"`
	Delegate     []StakePoolDelegatePoolInfo `json:"delegate"`
	Penalty      int64                       `json:"penalty"`
	Rewards      int64                       `json:"rewards"`
	Settings     StakePoolSettings           `json:"settings"`
}

type StakePoolDelegatePoolInfo struct {
	ID         string `json:"id"`
	Balance    int64  `json:"balance"`
	DelegateID string `json:"delegate_id"`
	Rewards    int64  `json:"rewards"`
	UnStake    bool   `json:"unstake"`

	TotalReward  int64  `json:"total_reward"`
	TotalPenalty int64  `json:"total_penalty"`
	Status       string `json:"status"`
	RoundCreated int64  `json:"round_created"`
}
