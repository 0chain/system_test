package model

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/herumi/bls-go-binary/bls"
	"gorm.io/gorm"
)

type HealthyServiceProviders struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
	Blobbers []string
}

type ExecutionRequest struct {
	FormData map[string]string

	QueryParams map[string]string

	Headers map[string]string

	Body interface{}

	Dst interface{}

	RequiredStatusCode int
}

type Wallet struct {
	Id           string   `json:"id"`
	Version      string   `json:"version"`
	CreationDate *int     `json:"creation_date"`
	PublicKey    string   `json:"public_key"`
	Nonce        int      `json:"nonce"`
	Keys         *KeyPair `json:"-"`
}

func (w *Wallet) FromSdkWallet(sdkWallet SdkWallet, keys *KeyPair) {
	w.Id = sdkWallet.ClientID
	w.PublicKey = sdkWallet.ClientKey
	w.Version = sdkWallet.Version

	w.Keys = keys

	w.Nonce = 3

	var creationDate int
	w.CreationDate = &creationDate
}

func (w *Wallet) IncNonce() {
	w.Nonce++
}

type SdkWallet struct {
	ClientID    string        `json:"client_id"`
	ClientKey   string        `json:"client_key"`
	Keys        []*SdkKeyPair `json:"keys"`
	Mnemonics   string        `json:"mnemonics"`
	Version     string        `json:"version"`
	DateCreated string        `json:"date_created"`
}

func (s *SdkWallet) UnmarshalFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(file); err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal(buf.Bytes(), s); err != nil {
		log.Fatalln(err)
	}
}

type SdkKeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type KeyPair struct {
	PublicKey  bls.PublicKey
	PrivateKey bls.SecretKey
}

func (w *SdkWallet) String() (string, error) {
	out, err := json.Marshal(w)
	if err != nil {
		return "", err
	}

	return string(out), nil
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

func NewCollectRewardTransactionData(providerID string, providerType int) TransactionData {
	var input = map[string]interface{}{
		"provider_id":   providerID,
		"provider_type": providerType,
	}

	return TransactionData{
		Name:  "collect_reward",
		Input: input,
	}
}

func NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse *SCRestGetAllocationBlobbersResponse) TransactionData {
	return TransactionData{
		Name:  "new_allocation_request",
		Input: *scRestGetAllocationBlobbersResponse,
	}
}

func NewCreateStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "stake_pool_lock",
		Input: &createStakePoolRequest,
	}
}

func NewDeleteStackPoolTransactionData(deleteStakePoolRequest DeleteStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "stake_pool_unlock",
		Input: &deleteStakePoolRequest,
	}
}

func NewCreateWritePoolTransactionData(createWritePoolRequest CreateWritePoolRequest) TransactionData {
	return TransactionData{
		Name:  "write_pool_lock",
		Input: &createWritePoolRequest,
	}
}

func NewDeleteWritePoolTransactionData(deleteWritePoolRequest DeleteWritePoolRequest) TransactionData {
	return TransactionData{
		Name:  "write_pool_unlock",
		Input: &deleteWritePoolRequest,
	}
}

func NewCreateReadPoolTransactionData() TransactionData {
	return TransactionData{
		Name: "read_pool_lock",
	}
}

func NewDeleteReadPoolTransactionData() TransactionData {
	return TransactionData{
		Name: "read_pool_unlock",
	}
}

func NewUpdateAllocationTransactionData(updateAllocationRequest *UpdateAllocationRequest) TransactionData {
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

func NewCancelAllocationTransactionData(allocationID string) TransactionData {
	var input = map[string]interface{}{
		"allocation_id": allocationID,
	}

	return TransactionData{
		Name:  "cancel_allocation",
		Input: input,
	}
}

type InternalTransactionPutRequest struct {
	TransactionData
	ToClientID string
	Wallet     *Wallet
	Value      *int64
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
	Nonce   int64  `json:"nonce"`
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
	BlobberRequirements
	Blobbers []string `json:"blobbers"`
}

type SCRestGetAllocationRequest struct {
	AllocationID string
}

type SCRestGetAllocationBlobbersRequest struct {
	BlobberRequirements
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
	ProviderType int    `json:"provider_type,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
}

type DeleteStakePoolRequest struct {
	ProviderType int    `json:"provider_type,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
	PoolID       string `json:"pool_id,omitempty"`
}

type CreateWritePoolRequest struct {
	AllocationID string `json:"allocation_id"`
}

type DeleteWritePoolRequest struct {
	AllocationID string `json:"allocation_id"`
}

type CreateReadPoolRequest struct {
}

type DeleteReadPoolRequest struct {
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
	WritePool       int64            `json:"write_pool"`
	ReadPriceRange  PriceRange       `json:"read_price_range"`
	WritePriceRange PriceRange       `json:"write_price_range"`
}

type StorageNodes struct {
	Nodes []*StorageNode `json:"Nodes"`
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

type BlobberGetFileRefsRequest struct {
	URL, ClientID, ClientKey, ClientSignature, AllocationID, RefType, RemotePath string
}

type BlobberFileRefPathRequest struct {
	URL, Path, AllocationID, ClientID, ClientKey, ClientSignature string
}

type BlobberObjectTreeRequest struct {
	URL, Path, AllocationID, ClientID, ClientKey, ClientSignature string
}

type RefsData struct {
	ID             int    `json:"id"`
	Type           string `json:"type"`
	AllocationId   string `json:"allocation_id"`
	LookupHash     string `json:"lookup_hash"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	Hash           string `json:"hash"`
	NumBlocks      int    `json:"num_blocks"`
	PathHash       string `json:"path_hash"`
	ParentPath     string `json:"parent_path"`
	Level          int    `json:"level"`
	ContentHash    string `json:"content_hash"`
	Size           int    `json:"size"`
	MerkleRoot     string `json:"merkle_root"`
	ActualFileSize int    `json:"actual_file_size"`
	ActualFileHash string `json:"actual_file_hash"`
	WriteMarker    string `json:"write_marker"`
	CreatedAt      int    `json:"created_at"`
	UpdatedAt      int    `json:"updated_at"`
	ChunkSize      int    `json:"chunk_size"`
}

type LatestWriteMarker struct {
	AllocationRoot     string `json:"allocation_root"`
	PrevAllocationRoot string `json:"prev_allocation_root"`
	AllocationId       string `json:"allocation_id"`
	Size               int    `json:"size"`
	BlobberId          string `json:"blobber_id"`
	Timestamp          int    `json:"timestamp"`
	ClientId           string `json:"client_id"`
	Signature          string `json:"signature"`
	LookupHash         string `json:"lookup_hash"`
	Name               string `json:"name"`
	ContentHash        string `json:"content_hash"`
}

type BlobberGetFileRefsResponse struct {
	TotalPages        int                `json:"total_pages"`
	OffsetPath        string             `json:"offset_path"`
	Refs              []*RefsData        `json:"refs"`
	LatestWriteMarker *LatestWriteMarker `json:"latest_write_marker"`
}

type CommitMetaTxn struct {
	RefID     int64     `gorm:"ref_id;not null" json:"ref_id"`
	TxnID     string    `gorm:"txn_id;size:64;not null" json:"txn_id"`
	CreatedAt time.Time `gorm:"created_at;timestamp without time zone;not null;default:current_timestamp" json:"created_at"`
}

type Ref struct {
	ID                  int64  `gorm:"column:id;primaryKey"`
	Type                string `gorm:"column:type;size:1" dirlist:"type" filelist:"type"`
	AllocationID        string `gorm:"column:allocation_id;size:64;not null;index:idx_path_alloc,priority:1;index:idx_lookup_hash_alloc,priority:1" dirlist:"allocation_id" filelist:"allocation_id"`
	LookupHash          string `gorm:"column:lookup_hash;size:64;not null;index:idx_lookup_hash_alloc,priority:2" dirlist:"lookup_hash" filelist:"lookup_hash"`
	Name                string `gorm:"column:name;size:100;not null" dirlist:"name" filelist:"name"`
	Path                string `gorm:"column:path;size:1000;not null;index:idx_path_alloc,priority:2;index:path_idx" dirlist:"path" filelist:"path"`
	Hash                string `gorm:"column:hash;size:64;not null" dirlist:"hash" filelist:"hash"`
	NumBlocks           int64  `gorm:"column:num_of_blocks;not null;default:0" dirlist:"num_of_blocks" filelist:"num_of_blocks"`
	PathHash            string `gorm:"column:path_hash;size:64;not null" dirlist:"path_hash" filelist:"path_hash"`
	ParentPath          string `gorm:"column:parent_path;size:999"`
	PathLevel           int    `gorm:"column:level;not null;default:0"`
	CustomMeta          string `gorm:"column:custom_meta;not null" filelist:"custom_meta"`
	ContentHash         string `gorm:"column:content_hash;size:64;not null" filelist:"content_hash"`
	Size                int64  `gorm:"column:size;not null;default:0" dirlist:"size" filelist:"size"`
	MerkleRoot          string `gorm:"column:merkle_root;size:64;not null" filelist:"merkle_root"`
	ActualFileSize      int64  `gorm:"column:actual_file_size;not null;default:0" filelist:"actual_file_size"`
	ActualFileHash      string `gorm:"column:actual_file_hash;size:64;not null" filelist:"actual_file_hash"`
	MimeType            string `gorm:"column:mimetype;size:64;not null" filelist:"mimetype"`
	WriteMarker         string `gorm:"column:write_marker;size:64;not null"`
	ThumbnailSize       int64  `gorm:"column:thumbnail_size;not null;default:0" filelist:"thumbnail_size"`
	ThumbnailHash       string `gorm:"column:thumbnail_hash;size:64;not null" filelist:"thumbnail_hash"`
	ActualThumbnailSize int64  `gorm:"column:actual_thumbnail_size;not null;default:0" filelist:"actual_thumbnail_size"`
	ActualThumbnailHash string `gorm:"column:actual_thumbnail_hash;size:64;not null" filelist:"actual_thumbnail_hash"`
	EncryptedKey        string `gorm:"column:encrypted_key;size:64" filelist:"encrypted_key"`
	Children            []*Ref `gorm:"-"`
	childrenLoaded      bool   //nolint
	OnCloud             bool   `gorm:"column:on_cloud;default:false" filelist:"on_cloud"`

	CommitMetaTxns []CommitMetaTxn `gorm:"foreignkey:ref_id" filelist:"commit_meta_txns"`
	CreatedAt      int64           `gorm:"column:created_at;index:idx_created_at,sort:desc" dirlist:"created_at" filelist:"created_at"`
	UpdatedAt      int64           `gorm:"column:updated_at;index:idx_updated_at,sort:desc;" dirlist:"updated_at" filelist:"updated_at"`

	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"` // soft deletion

	ChunkSize        int64 `gorm:"column:chunk_size;not null;default:65536" dirlist:"chunk_size" filelist:"chunk_size"`
	HashToBeComputed bool  `gorm:"-"`
}

type BlobberFileRefPathResponse struct {
	Meta map[string]interface{}        `json:"meta_data"`
	List []*BlobberFileRefPathResponse `json:"list,omitempty"`
	Ref  *Ref
}

type WriteMarker struct {
	AllocationRoot         string `gorm:"column:allocation_root;size:64;primaryKey" json:"allocation_root"`
	PreviousAllocationRoot string `gorm:"column:prev_allocation_root;size:64" json:"prev_allocation_root"`
	AllocationID           string `gorm:"column:allocation_id;size:64;index:idx_seq,unique,priority:1" json:"allocation_id"`
	Size                   int64  `gorm:"column:size" json:"size"`
	BlobberID              string `gorm:"column:blobber_id;size:64" json:"blobber_id"`
	Timestamp              int64  `gorm:"column:timestamp" json:"timestamp"`
	ClientID               string `gorm:"column:client_id;size:64" json:"client_id"`
	Signature              string `gorm:"column:signature;size:64" json:"signature"`

	LookupHash  string `gorm:"column:lookup_hash;size:64;" json:"lookup_hash"`
	Name        string `gorm:"column:name;size:100;" json:"name"`
	ContentHash string `gorm:"column:content_hash;size:64;" json:"content_hash"`
}

type BlobberObjectTreePathResponse struct {
	*BlobberFileRefPathResponse
	LatestWM *WriteMarker `json:"latest_write_marker"`
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
	KeyPair
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

// type BlobberGetFileReferencePathRequest struct {
//	URL, ClientID, ClientKey, ClientSignature, AllocationID string
// }

// type BlobberGetFileReferencePathResponse struct {
//	sdk.ReferencePathResult
// }

type BlobberGetHashnodeRequest struct {
	URL, ClientId, ClientKey, ClientSignature, AllocationID string
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

type SCRestGetStakePoolStatRequest struct {
	ProviderType string
	ProviderID   string
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

type AllocationBlobbers interface{}

func ConvertInterfaceStringArray(blobbers AllocationBlobbers) []string {
	var blobbersInterfaceArray = blobbers.([]interface{})
	var blobbersStringArray []string
	for _, v := range blobbersInterfaceArray {
		blobbersStringArray = append(blobbersStringArray, v.(string))
	}
	return blobbersStringArray
}

type BlobberGetHashnodeResponse struct {
	// hash data
	AllocationID   string `json:"allocation_id,omitempty"`
	Type           string `json:"type,omitempty"`
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	ContentHash    string `json:"content_hash,omitempty"`
	MerkleRoot     string `json:"merkle_root,omitempty"`
	ActualFileHash string `json:"actual_file_hash,omitempty"`
	ChunkSize      int64  `json:"chunk_size,omitempty"`
	Size           int64  `json:"size,omitempty"`
	ActualFileSize int64  `json:"actual_file_size,omitempty"`

	// other data
	ParentPath string                        `json:"-"`
	Children   []*BlobberGetHashnodeResponse `json:"children,omitempty"`
}

type GetTotalCloudSizeResponse int

type GetTotalBlobberCapacityResponse int

type GetAverageWritePriceResponse int

type GetTotalMintedResponse int

type GetTotalTotalChallengesResponse int

type GetTotalSuccessfulChallengesResponse int

type GetTotalAllocatedStorage int

type GetTotalStakedResponse int

type GetGraphBlobberInactiveRoundsRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberInactiveRoundsResponse []int

type GetGraphBlobberChallengesCompletedRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberChallengesCompletedResponse []int

type GetGraphBlobberChallengesPassedRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberChallengesPassedResponse []int

type GetGraphBlobberServiceChargeRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberServiceChargeResponse []int

type GetGraphBlobberWritePriceRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberWritePriceResponse []int

type GetGraphBlobberCapacityRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberCapacityResponse []int

type GetGraphBlobberAllocatedRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberAllocatedResponse []int

type GetGraphBlobberSavedDataRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberSavedDataResponse []int

type GetGraphBlobberReadDataRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberReadDataResponse []int

type GetGraphBlobberOffersTotalRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberOffersTotalResponse []int

type GetGraphBlobberUnstakeTotalRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberUnstakeTotalResponse []int

type GetGraphBlobberTotalStakeRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberTotalStakeResponse []int

type GetGraphBlobberChallengesOpenedRequest struct {
	DataPoints    int
	BlobberID, To string
}

type GetGraphBlobberChallengesOpenedResponse []int

type GetGraphTotalLockedRequest struct {
	DataPoints int
	To         string
}

type GetGraphTotalLockedResponse []int

type GetGraphTokenSupplyRequest struct {
	DataPoints int
	To         string
}

type GetGraphTokenSupplyResponse []int

type GetCurrentRoundResponse struct {
	CurrentRound int `json:"current_round"`
}

func (cr *GetCurrentRoundResponse) CurrentRoundTwiceToString() string {
	return strconv.Itoa(cr.CurrentRound * 2)
}
