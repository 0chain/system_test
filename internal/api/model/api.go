package model

import (
	"encoding/json"
	"io"
	"time"

	"github.com/0chain/gosdk/core/zcncrypto"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/herumi/bls-go-binary/bls"
	"gorm.io/gorm"
)

const ZCNInt64 = int64(100000000)
const ZCNFloat64 = float64(1e9)

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

	FileName string

	FilePath string
}

type Wallet struct {
	Id           string `json:"id"`
	Version      string `json:"version"`
	CreationDate *int   `json:"creation_date"`
	PublicKey    string `json:"public_key"`
	Nonce        int
	Keys         *KeyPair `json:"-"`
}

type SdkWallet struct {
	ClientID    string        `json:"client_id"`
	ClientKey   string        `json:"client_key"`
	Keys        []*SdkKeyPair `json:"keys"`
	Mnemonics   string        `json:"mnemonics"`
	Version     string        `json:"version"`
	DateCreated string        `json:"date_created"`
}

type SdkKeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type KeyPair struct {
	PublicKey  bls.PublicKey
	PrivateKey bls.SecretKey
}

func (w *Wallet) IncNonce() {
	w.Nonce++
}

func (w *SdkWallet) String() (string, error) {
	out, err := json.Marshal(w)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (w *Wallet) ToSdkWallet(mnemonics string) *SdkWallet {
	return &SdkWallet{
		ClientID:  w.Id,
		ClientKey: w.PublicKey,
		Keys: []*SdkKeyPair{{
			PrivateKey: w.Keys.PrivateKey.SerializeToHexStr(),
			PublicKey:  w.Keys.PublicKey.SerializeToHexStr(),
		}},
		Mnemonics: mnemonics,
		Version:   w.Version,
	}
}

func (w *SdkWallet) ToCliModelWalletFile() *climodel.WalletFile {
	keys := make([]climodel.KeyPair, 0, len(w.Keys))
	for _, key := range w.Keys {
		keys = append(keys, climodel.KeyPair{
			PublicKey:  key.PublicKey,
			PrivateKey: key.PrivateKey,
		})
	}

	return &climodel.WalletFile{
		ClientID:    w.ClientID,
		ClientKey:   w.ClientKey,
		Keys:        keys,
		Mnemonic:    w.Mnemonics,
		Version:     w.Version,
		DateCreated: w.DateCreated,
	}
}

func (w *Wallet) ToZCNCryptoWallet(mnemonic string) *zcncrypto.Wallet {
	if w == nil {
		return nil
	}

	var keys []zcncrypto.KeyPair
	keys = append(keys, zcncrypto.KeyPair{
		PublicKey:  w.Keys.PublicKey.GetHexString(),
		PrivateKey: w.Keys.PrivateKey.GetHexString(),
	})
	var dateCreated string
	if w.CreationDate != nil {
		dateCreated = time.Unix(int64(*w.CreationDate), 0).Format(time.RFC3339)
	}
	return &zcncrypto.Wallet{
		ClientID:    w.Id,
		ClientKey:   w.PublicKey,
		Keys:        keys,
		Mnemonic:    mnemonic,
		Version:     w.Version,
		DateCreated: dateCreated,
	}
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

func NewCreateFreeAllocationTransactionData(scRestGetFreeAllocationBlobbersResponse *SCRestGetFreeAllocationBlobbersResponse) TransactionData {
	return TransactionData{
		Name:  "free_allocation_request",
		Input: *scRestGetFreeAllocationBlobbersResponse,
	}
}

func NewCreateWritePoolTransactionData(createWritePoolRequest CreateWritePoolRequest) TransactionData {
	return TransactionData{
		Name:  "write_pool_lock",
		Input: &createWritePoolRequest,
	}
}

func NewUnlockWritePoolTransactionData(createWritePoolRequest CreateWritePoolRequest) TransactionData {
	return TransactionData{
		Name:  "write_pool_unlock",
		Input: &createWritePoolRequest,
	}
}

func NewCreateReadPoolTransactionData() TransactionData {
	return TransactionData{
		Name: "read_pool_lock",
	}
}

func NewUnlockReadPoolTransactionData() TransactionData {
	return TransactionData{
		Name: "read_pool_unlock",
	}
}

func NewCreateStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "stake_pool_lock",
		Input: &createStakePoolRequest,
	}
}

func NewUnlockStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "stake_pool_unlock",
		Input: &createStakePoolRequest,
	}
}

func NewCreateMinerStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "addToDelegatePool",
		Input: &createStakePoolRequest,
	}
}

func NewUnlockMinerStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	return TransactionData{
		Name:  "deleteFromDelegatePool",
		Input: &createStakePoolRequest,
	}
}

func NewFreeAllocationTransactionData(freeAllocationRequest *FreeAllocationRequest) TransactionData {
	return TransactionData{
		Name:  "free_update_allocation",
		Input: freeAllocationRequest,
	}
}

func NewFreeStorageAssignerTransactionData(freeStorageAssignerRequest *FreeStorageAssignerRequest) TransactionData {
	return TransactionData{
		Name:  "add_free_storage_assigner",
		Input: freeStorageAssignerRequest,
	}
}

func NewUpdateAllocationTransactionData(updateAllocationRequest *UpdateAllocationRequest) TransactionData {
	return TransactionData{
		Name:  "update_allocation_request",
		Input: updateAllocationRequest,
	}
}

func NewCancelAllocationTransactionData(cancelAllocationRequest *CancelAllocationRequest) TransactionData {
	return TransactionData{
		Name:  "cancel_allocation",
		Input: cancelAllocationRequest,
	}
}

func NewUpdateBlobberTransactionData(scRestGetBlobberResponse *SCRestGetBlobberResponse) TransactionData {
	return TransactionData{
		Name:  "update_blobber_settings",
		Input: scRestGetBlobberResponse,
	}
}

// ----------------------------------------------------
// ZCN SC
// ----------------------------------------------------
func NewBurnZcnTransactionData(burnZcnRequest *SCRestBurnZcnRequest) TransactionData {
	return TransactionData{
		Name:  "burn",
		Input: burnZcnRequest,
	}
}

//----------------------------------------------------
// End ZCN SC
//----------------------------------------------------

type InternalTransactionPutRequest struct {
	TransactionData
	ToClientID string
	Wallet     *Wallet
	Value      *int64
	TxnType    int
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

type RewardTransactionOutput struct {
	Id           int    `json:"ID"`
	CreatedAt    string `json:"CreatedAt"`
	UpdatedAt    string `json:"UpdatedAt"`
	Amount       int64  `json:"amount"`
	BlockNumber  int    `json:"block_number"`
	ClientId     string `json:"client_id"`
	PoolID       string `json:"pool_id,omitempty"`
	ProviderType string `json:"provider_type"`
	ProviderID   string `json:"provider_id"`
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

type ClientGetReadPoolBalanceRequest struct {
	ClientID string
}

type QueryRewardsRequest struct {
	Query string
}

type BlockRewardsRequest struct {
	Start int64
	End   int64
}

type ClientGetReadPoolBalanceResponse struct {
	UserID  string `json:"user_id" gorm:"uniqueIndex"`
	Balance int64  `json:"balance"`
}

type QueryRewardsResponse struct {
	TotalProviderReward float64 `json:"total_provider_reward"`
	TotalDelegateReward float64 `json:"total_delegate_reward"`
	TotalReward         float64 `json:"total_reward"`
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
	BlobberRequirements
	ClientID, ClientKey string
}

type FreeAllocationData struct {
	RecipientPublicKey string `json:"recipient_public_key"`
	Marker             string `json:"marker"`
}

type SCRestGetClientBalanceResponse struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
	Nonce   int64  `json:"nonce"`
}

type SCRestGetFreeAllocationBlobbersResponse struct {
	FreeAllocationData
	Blobbers *[]string `json:"blobbers"`
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
	AddBlobberId         string `json:"add_blobber_id"`
	RemoveBlobberId      string `json:"remove_blobber_id"`
	ThirdPartyExtendable bool   `json:"third_party_extendable"`
	FileOptionsChanged   bool   `json:"file_options_changed"`
	FileOptions          uint16 `json:"file_options"`
}

type CancelAllocationRequest struct {
	AllocationID string `json:"allocation_id"`
}

type SCRestGetBlobberRequest struct {
	BlobberID string
}

type BlobberGetHashnodeRequest struct {
	URL, ClientId, ClientKey, ClientSignature, AllocationID string
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

type SimpleNodeGeolocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type SimpleNodeResponse struct {
	ID                            string                `json:"id" validate:"hexadecimal,len=64"`
	N2NHost                       string                `json:"n2n_host"`
	Host                          string                `json:"host"`
	Port                          int                   `json:"port"`
	Geolocation                   SimpleNodeGeolocation `json:"geolocation"`
	Path                          string                `json:"path"`
	PublicKey                     string                `json:"public_key"`
	ShortName                     string                `json:"short_name"`
	BuildTag                      string                `json:"build_tag"`
	TotalStaked                   int64                 `json:"total_stake"`
	Delete                        bool                  `json:"delete"`
	NodeType                      string                `json:"node_type,omitempty"`
	LastHealthCheck               int64                 `json:"last_health_check"`
	Status                        int                   `json:"-" msg:"-"`
	LastSettingUpdateRound        int64                 `json:"last_setting_update_round"`
	RoundServiceChargeLastUpdated int64                 `json:"round_service_charge_last_updated"`
}

type SCRestGetMinerSharderResponse struct {
	SimpleNodeResponse `json:"simple_miner"`
	StakePoolResponse  `json:"stake_pool"`
	TotalReward        int64 `json:"total_reward"`
}

type SCRestGetMinersShardersResponse struct {
	Nodes []*SCRestGetMinerSharderResponse `json:"Nodes"`
}

type SCRestGetValidatorResponse struct {
	ValidatorID     string `json:"validator_id"`
	BaseUrl         string `json:"url"`
	StakeTotal      int64  `json:"stake_total"`
	UnstakeTotal    int64  `json:"unstake_total"`
	PublicKey       string `json:"public_key"`
	LastHealthCheck int64  `json:"last_health_check"`
	IsKilled        bool   `json:"is_killed"`
	IsShutdown      bool   `json:"is_shutdown"`

	// StakePoolSettings
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int64   `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`

	TotalServiceCharge       int64 `json:"total_service_charge"`
	UncollectedServiceCharge int64 `json:"uncollected_service_charge"`
}

type SCRestGetValidatorsResponse []SCRestGetValidatorResponse

type SCRestGetBlobberResponse struct {
	ID                  string            `json:"id"`
	BaseURL             string            `json:"url"`
	Terms               Terms             `json:"terms"`
	Capacity            int64             `json:"capacity"`
	Allocated           int64             `json:"allocated"`
	LastHealthCheck     int64             `json:"last_health_check"`
	PublicKey           string            `json:"-"`
	StakePoolSettings   StakePoolSettings `json:"stake_pool_settings"`
	TotalStake          int64             `json:"total_stake"`
	SavedData           int64             `json:"saved_data"`
	ReadData            int64             `json:"read_data"`
	ChallengesPassed    int64             `json:"challenges_passed"`
	ChallengesCompleted int64             `json:"challenges_completed"`
}

type SCRestGetBlobbersResponse struct {
	Nodes []*SCRestGetBlobberResponse `json:"Nodes"`
}

type Terms struct {
	ReadPrice  int64 `json:"read_price"`
	WritePrice int64 `json:"write_price"`
}

type StakePoolSettings struct {
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int     `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
}

type CreateWritePoolRequest struct {
	AllocationID string `json:"allocation_id,omitempty"`
}

type CreateStakePoolRequest struct {
	ProviderType int    `json:"provider_type,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
}

type FreeStorageAssignerRequest struct {
	Name            string  `json:"name,omitempty"`
	PublicKey       string  `json:"public_key,omitempty"`
	IndividualLimit float64 `json:"individual_limit,omitempty"`
	TotalLimit      float64 `json:"total_limit,omitempty"`
}

type FreeAllocationRequest struct {
	AllocationID string `json:"allocation_id,omitempty"`
	Marker       string `json:"marker,omitempty"`
}

type SCRestGetAllocationResponse struct {
	ID                string           `json:"id"`
	Tx                string           `json:"tx"`
	Name              string           `json:"name"`
	DataShards        int              `json:"data_shards"`
	ParityShards      int              `json:"parity_shards"`
	Size              int64            `json:"size"`
	CreatedAt         int64            `json:"created_at"`
	Expiration        int64            `json:"expiration_date"`
	Owner             string           `json:"owner_id"`
	OwnerPublicKey    string           `json:"owner_public_key"`
	Payer             string           `json:"payer_id"`
	Blobbers          []*StorageNode   `json:"blobbers"`
	Stats             *AllocationStats `json:"stats"`
	TimeUnit          time.Duration    `json:"time_unit"`
	IsImmutable       bool             `json:"is_immutable"`
	StartTime         int64            `json:"start_time"`
	Finalized         bool             `json:"finalized"`
	Canceled          bool             `json:"canceled"`
	WritePool         int64            `json:"write_pool"`
	MovedToChallenge  int64            `json:"moved_to_challenge"`
	MovedBack         int64            `json:"moved_back"`
	MovedToValidators int64            `json:"moved_to_validators"`
	MinLockDemand     float64          `json:"min_lock_demand"`
	ReadPriceRange    PriceRange       `json:"read_price_range"`
	WritePriceRange   PriceRange       `json:"write_price_range"`
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
	TotalStake        int64                  `json:"total_stake"`
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
	ID          string                      `json:"pool_id"`
	Balance     int64                       `json:"balance"`
	Unstake     int64                       `json:"unstake"`
	Free        int64                       `json:"free"`
	Capacity    int64                       `json:"capacity"`
	WritePrice  int64                       `json:"write_price"`
	OffersTotal int64                       `json:"offers_total"`
	Delegate    []StakePoolDelegatePoolInfo `json:"delegate"`
	Penalty     int64                       `json:"penalty"`
	Rewards     int64                       `json:"rewards"`
	Settings    StakePoolSettings           `json:"settings"`
}

type SCRestGetUserStakePoolStatRequest struct {
	ClientId string
}

type SCRestGetUserStakePoolStatResponse struct {
	Pools map[string]*[]StakePoolDelegatePoolInfo `json:"pools"`
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

type StakePoolResponse struct {
	Pools    map[string]*DelegatePoolResponse `json:"pools"`
	Reward   int64                            `json:"rewards"`
	Settings StakePoolSettings                `json:"settings"`
	Minter   int                              `json:"minter"`
}

type DelegatePoolResponse struct {
	DelegatePool
	RoundPoolLastUpdated int64 `json:"round_pool_last_updated"`
}

type DelegatePool struct {
	Balance      int64  `json:"balance"`
	Reward       int64  `json:"reward"`
	Status       int    `json:"status"`
	RoundCreated int64  `json:"round_created"` // used for cool down
	DelegateID   string `json:"delegate_id"`
	StakedAt     int64  `json:"staked_at"`
}

type FreeStorageMarker struct {
	Assigner   string  `json:"assigner,omitempty"`
	Recipient  string  `json:"recipient"`
	FreeTokens float64 `json:"free_tokens"`
	Nonce      int64   `json:"nonce"`
	Signature  string  `json:"signature,omitempty"`
}

// ----------------------------------------------
// ZCN SC
// ----------------------------------------------
type SCRestBurnZcnRequest struct {
	EthereumAddress string `json:"ethereum_address"`
}

//----------------------------------------------
// End ZCN SC
//----------------------------------------------

type LatestFinalizedBlock struct {
	CreationDate      int64  `json:"creation_date"`
	Hash              string `json:"hash,omitempty"`
	StateHash         string `json:"state_hash"`
	MinerId           string `json:"miner_id"`
	Round             int64  `json:"round"`
	StateChangesCount int    `json:"state_changes_count"`
	NumTxns           int    `json:"num_txns"`
}
