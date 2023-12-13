package model

import (
	"time"

	"github.com/0chain/gosdk/core/common"
)

type Provider int

type Timestamp int64

const (
	ProviderMiner Provider = iota + 1
	ProviderSharder
	ProviderBlobber
	ProviderValidator
	ProviderAuthorizer
)

var providerString = []string{"unknown", "miner", "sharder", "blobber", "validator", "authorizer"}

func (p Provider) String() string {
	return providerString[p]
}

type PoolStatus int

const (
	Active PoolStatus = iota
	Pending
	Inactive
	Unstaking
	Deleting
	Deleted
)

var poolString = []string{"active", "pending", "inactive", "unstaking", "deleting"}

func (p PoolStatus) String() string {
	return poolString[p]
}

type Wallet struct {
	ClientID            string `json:"client_id"`
	ClientPublicKey     string `json:"client_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}

type Allocation struct {
	ID             string    `json:"id"`
	Tx             string    `json:"tx"`
	Name           string    `json:"name"`
	ExpirationDate int64     `json:"expiration_date"`
	DataShards     int       `json:"data_shards"`
	ParityShards   int       `json:"parity_shards"`
	Size           int64     `json:"size"`
	Owner          string    `json:"owner_id"`
	OwnerPublicKey string    `json:"owner_public_key"`
	Payer          string    `json:"payer_id"`
	Blobbers       []Blobber `json:"blobbers"`
	// Stats          *AllocationStats          `json:"stats"`
	TimeUnit    time.Duration `json:"time_unit"`
	IsImmutable bool          `json:"is_immutable"`

	WritePool int64 `json:"write_pool"`

	// BlobberDetails contains real terms used for the allocation.
	// If the allocation has updated, then terms calculated using
	// weighted average values.
	BlobberDetails []*BlobberAllocation `json:"blobber_details"`

	// ReadPriceRange is requested reading prices range.
	ReadPriceRange PriceRange `json:"read_price_range"`

	// WritePriceRange is requested writing prices range.
	WritePriceRange PriceRange `json:"write_price_range"`

	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`

	StartTime            int64  `json:"start_time"`
	Finalized            bool   `json:"finalized,omitempty"`
	Canceled             bool   `json:"canceled,omitempty"`
	MovedToChallenge     int64  `json:"moved_to_challenge,omitempty"`
	MovedBack            int64  `json:"moved_back,omitempty"`
	MovedToValidators    int64  `json:"moved_to_validators,omitempty"`
	FileOptions          uint16 `json:"file_options"`
	ThirdPartyExtendable bool   `json:"third_party_extendable"`
}

type AllocationFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	Hash       string `json:"hash"`
	ActualSize int    `json:"actual_size"`
}
type Blobber struct {
	ID                string            `json:"id"`
	BaseURL           string            `json:"url"`
	Terms             Terms             `json:"terms"`     // terms
	Capacity          int64             `json:"capacity"`  // total blobber capacity
	Allocated         int64             `json:"allocated"` // allocated capacity
	TotalStake        int64             `json:"total_stake"`
	LastHealthCheck   int64             `json:"last_health_check"`
	PublicKey         string            `json:"-"`
	StakePoolSettings StakePoolSettings `json:"stake_pool_settings"`
}

type ReadPoolInfo struct {
	Balance int64   `json:"fmt"`
	Zcn     float64 `json:"zcn"`
	Usd     float64 `json:"usd"`
}

type RecentlyAddedRefResult struct {
	Offset int    `json:"offset"`
	Refs   []ORef `json:"refs"`
}

type ORef struct {
	SimilarField
	ID int64 `json:"id"`
}

type SimilarField struct {
	Type                string           `json:"type"`
	AllocationID        string           `json:"allocation_id"`
	LookupHash          string           `json:"lookup_hash"`
	Name                string           `json:"name"`
	Path                string           `json:"path"`
	PathHash            string           `json:"path_hash"`
	ParentPath          string           `json:"parent_path"`
	PathLevel           int              `json:"level"`
	Size                int64            `json:"size"`
	ActualFileSize      int64            `json:"actual_file_size"`
	ActualFileHash      string           `json:"actual_file_hash"`
	MimeType            string           `json:"mimetype"`
	ActualThumbnailSize int64            `json:"actual_thumbnail_size"`
	ActualThumbnailHash string           `json:"actual_thumbnail_hash"`
	CreatedAt           common.Timestamp `json:"created_at"`
	UpdatedAt           common.Timestamp `json:"updated_at"`
}

type ListFileResult struct {
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	Type            string    `json:"type"`
	Size            int64     `json:"size"`
	Hash            string    `json:"hash"`
	Mimetype        string    `json:"mimetype"`
	NumBlocks       int       `json:"num_blocks"`
	LookupHash      string    `json:"lookup_hash"`
	EncryptionKey   string    `json:"encryption_key"`
	ActualSize      int64     `json:"actual_size"`
	ActualNumBlocks int       `json:"actual_num_blocks"`
	CreatedAt       Timestamp `json:"created_at"`
	UpdatedAt       Timestamp `json:"updated_at"`
}

type Terms struct {
	ReadPrice  int64 `json:"read_price"`
	WritePrice int64 `json:"write_price"`
}

type Settings struct {
	Delegate_wallet string  `json:"delegate_wallet"`
	Num_delegates   int     `json:"num_delegates"`
	Service_charge  float64 `json:"service_charge"`
}

type BlobberInfo struct {
	Id                       string            `json:"id"`
	Url                      string            `json:"url"`
	Terms                    Terms             `json:"terms"`
	Capacity                 int64             `json:"capacity"`
	Allocated                int64             `json:"allocated"`
	LastHealthCheck          int64             `json:"last_health_check"`
	StakePoolSettings        StakePoolSettings `json:"stake_pool_settings"`
	TotalStake               int64             `json:"total_stake"`
	UsedAllocation           int64             `json:"used_allocation"`
	TotalOffers              int64             `json:"total_offers"`
	TotalServiceCharge       int64             `json:"total_service_charge"`
	UncollectedServiceCharge int64             `json:"uncollected_service_charge"`
	IsKilled                 bool              `json:"is_killed"`
	IsShutdown               bool              `json:"is_shutdown"`
}

type ChallengePoolInfo struct {
	Id         string `json:"id"`
	Balance    int64  `json:"balance"`
	StartTime  int64  `json:"start_time"`
	Expiration int64  `json:"expiration"`
	Finalized  bool   `json:"finalized"`
}

type FileMetaResult struct {
	Name            string          `json:"Name"`
	Path            string          `json:"Path"`
	Type            string          `json:"Type"`
	Size            int64           `json:"Size"`
	ActualFileSize  int64           `json:"ActualFileSize"`
	LookupHash      string          `json:"LookupHash"`
	Hash            string          `json:"Hash"`
	MimeType        string          `json:"MimeType"`
	ActualNumBlocks int             `json:"ActualNumBlocks"`
	EncryptedKey    string          `json:"EncryptedKey"`
	CommitMetaTxns  []CommitMetaTxn `json:"CommitMetaTxns"`
	Collaborators   []Collaborator  `json:"Collaborators"`
}

type CommitMetaTxn struct {
	RefID     int64  `json:"ref_id"`
	TxnID     string `json:"txn_id"`
	CreatedAt string `json:"created_at"`
}

type Collaborator struct {
	RefID     int64  `json:"ref_id"`
	ClientID  string `json:"client_id"`
	CreatedAt string `json:"created_at"`
}

type CommitResponse struct {
	//FIXME: POSSIBLE ISSUE: json-tags are not available for commit response

	TxnID    string `json:"TxnID"`
	MetaData struct {
		Name            string          `json:"Name"`
		Type            string          `json:"Type"`
		Path            string          `json:"Path"`
		LookupHash      string          `json:"LookupHash"`
		Hash            string          `json:"Hash"`
		MimeType        string          `json:"MimeType"`
		EncryptedKey    string          `json:"EncryptedKey"`
		Size            int64           `json:"Size"`
		ActualFileSize  int64           `json:"ActualFileSize"`
		ActualNumBlocks int             `json:"ActualNumBlocks"`
		CommitMetaTxns  []CommitMetaTxn `json:"CommitMetaTxns"`
		Collaborators   []Collaborator  `json:"Collaborators"`
	} `json:"MetaData"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type BlobberAllocation struct {
	BlobberID       string `json:"blobber_id"`
	Size            int64  `json:"size"`
	Terms           Terms  `json:"terms"`
	Spent           int64  `json:"spent"`
	Penalty         int64  `json:"penalty"`
	ReadReward      int64  `json:"read_reward"`
	Returned        int64  `json:"returned"`
	ChallengeReward int64  `json:"challenge_reward"`
	FinalReward     int64  `json:"final_reward"`
}

type StakePoolInfo struct {
	ID          string                      `json:"pool_id"`      // pool ID
	Balance     int64                       `json:"balance"`      // total balance
	Unstake     int64                       `json:"unstake"`      // total unstake amount
	Free        int64                       `json:"free"`         // free staked space
	Capacity    int64                       `json:"capacity"`     // blobber bid
	WritePrice  int64                       `json:"write_price"`  // its write price
	OffersTotal int64                       `json:"offers_total"` //
	Delegate    []StakePoolDelegatePoolInfo `json:"delegate"`
	Penalty     int64                       `json:"penalty"` // total for all
	Rewards     int64                       `json:"rewards"`
	Settings    StakePoolSettings           `json:"settings"`
}

type StakePoolDelegatePoolInfo struct {
	ID         string `json:"id"`          // blobber ID
	Balance    int64  `json:"balance"`     // current balance
	DelegateID string `json:"delegate_id"` // wallet
	Rewards    int64  `json:"rewards"`     // current
	UnStake    bool   `json:"unstake"`     // want to unstake

	TotalReward  int64  `json:"total_reward"`
	TotalPenalty int64  `json:"total_penalty"`
	Status       string `json:"status"`
	RoundCreated int64  `json:"round_created"`
}

type StakePoolSettings struct {
	// DelegateWallet for pool owner.
	DelegateWallet string `json:"delegate_wallet"`
	// MaxNumDelegates maximum allowed.
	MaxNumDelegates int `json:"num_delegates"`
	// ServiceCharge is blobber service charge.
	ServiceCharge float64 `json:"service_charge"`
}

type NodeList struct {
	Nodes []Node `json:"Nodes"`
}

type DelegatePool struct {
	Balance              int64  `json:"balance"`
	Reward               int64  `json:"reward"`
	Status               int    `json:"status"`
	RoundCreated         int64  `json:"round_created"` // used for cool down
	DelegateID           string `json:"delegate_id"`
	RoundPoolLastUpdated int64  `json:"round_pool_last_updated"`
}

type StakePool struct {
	Pools    map[string]*DelegatePool `json:"pools"`
	Reward   int64                    `json:"rewards"`
	Settings StakePoolSettings        `json:"settings"`
	Minter   int                      `json:"minter"`
}

type Node struct {
	SimpleNode  `json:"simple_miner"`
	StakePool   `json:"stake_pool"`
	TotalReward int64 `json:"total_reward"`
}

type SimpleNode struct {
	ID                            string      `json:"id"`
	N2NHost                       string      `json:"n2n_host"`
	Host                          string      `json:"host"`
	Port                          int         `json:"port"`
	PublicKey                     string      `json:"public_key"`
	ShortName                     string      `json:"short_name"`
	BuildTag                      string      `json:"build_tag"`
	TotalStake                    int64       `json:"total_stake"`
	Stat                          interface{} `json:"stat"`
	RoundServiceChargeLastUpdated int64       `json:"round_service_charge_last_updated"`
	IsKilled                      bool        `json:"is_killed"`
}

type Sharder struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	CreationDate int64  `json:"creation_date"`
	PublicKey    string `json:"public_key"`
	N2NHost      string `json:"n2n_host"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Path         string `json:"path"`
	Type         int    `json:"type"`
	Description  string `json:"description"`
	SetIndex     int    `json:"set_index"`
	Status       int    `json:"status"`
	Info         struct {
		BuildTag                string `json:"build_tag"`
		StateMissingNodes       int    `json:"state_missing_nodes"`
		MinersMedianNetworkTime int64  `json:"miners_median_network_time"`
		AvgBlockTxns            int    `json:"avg_block_txns"`
	} `json:"info"`
}

type FileStats struct {
	Name                string    `json:"name"`
	Size                int64     `json:"size"`
	PathHash            string    `json:"path_hash"`
	Path                string    `json:"path"`
	NumOfBlocks         int64     `json:"num_of_blocks"`
	NumOfUpdates        int64     `json:"num_of_updates"`
	NumOfBlockDownloads int64     `json:"num_of_block_downloads"`
	NumOfChallenges     int64     `json:"num_of_failed_challenges"`
	LastChallengeTxn    string    `json:"last_challenge_txn"`
	WriteMarkerTxn      string    `json:"write_marker_txn"`
	BlobberID           string    `json:"blobber_id"`
	BlobberURL          string    `json:"blobber_url"`
	BlockchainAware     bool      `json:"blockchain_aware"`
	CreatedAt           time.Time `json:"CreatedAt"`
}

type BlobberDetails struct {
	ID                string            `json:"id"`
	BaseURL           string            `json:"url"`
	Terms             Terms             `json:"terms"`
	Capacity          int64             `json:"capacity"`
	Allocated         int64             `json:"allocated"`
	LastHealthCheck   int64             `json:"last_health_check"`
	PublicKey         string            `json:"-"`
	StakePoolSettings StakePoolSettings `json:"stake_pool_settings"`
	IsKilled          bool              `json:"is_killed"`
	IsShutdown        bool              `json:"is_shutdown"`
	NotAvailable      bool              `json:"not_available"`
}

type Validator struct {
	ID             string  `json:"validator_id"`
	BaseURL        string  `json:"url"`
	PublicKey      string  `json:"-"`
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int64   `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
	TotalStake     int64   `json:"stake"`
}

type FileDiff struct {
	Op   string `json:"operation"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type FreeStorageMarker struct {
	Assigner   string  `json:"assigner,omitempty"`
	Recipient  string  `json:"recipient"`
	FreeTokens float64 `json:"free_tokens"`
	Nonce      int64   `json:"nonce"`
	Signature  string  `json:"signature,omitempty"`
}

type WalletFile struct {
	ClientID    string    `json:"client_id"`
	ClientKey   string    `json:"client_key"`
	Keys        []KeyPair `json:"keys"`
	Mnemonic    string    `json:"mnemonics"`
	Version     string    `json:"version"`
	DateCreated string    `json:"date_created"`
}

type KeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type Miner struct {
	ID                string      `json:"id"`
	N2NHost           string      `json:"n2n_host"`
	Host              string      `json:"host"`
	Port              int         `json:"port"`
	PublicKey         string      `json:"public_key"`
	ShortName         string      `json:"short_name"`
	BuildTag          string      `json:"build_tag"`
	TotalStake        int         `json:"total_stake"`
	DelegateWallet    string      `json:"delegate_wallet"`
	ServiceCharge     float64     `json:"service_charge"`
	NumberOfDelegates int         `json:"number_of_delegates"`
	MinStake          int64       `json:"min_stake"`
	MaxStake          int64       `json:"max_stake"`
	Stat              interface{} `json:"stat"`
}

type MinerSCNodes struct {
	Nodes []Node `json:"Nodes"`
}

type MinerSCDelegatePoolInfo struct {
	ID         string `json:"id"`
	Balance    int64  `json:"balance"`
	Reward     int64  `json:"reward"`      // uncollected reread
	RewardPaid int64  `json:"reward_paid"` // total reward all time
	Status     string `json:"status"`
}

type LockConfig struct {
	ID               string           `json:"ID"`
	SimpleGlobalNode SimpleGlobalNode `json:"simple_global_node"`
	MinLockPeriod    int64            `json:"min_lock_period"`
}

type SimpleGlobalNode struct {
	MaxMint     int64   `json:"max_mint"`
	TotalMinted int64   `json:"total_minted"`
	MinLock     int64   `json:"min_lock"`
	Apr         float64 `json:"apr"`
	OwnerId     string  `json:"owner_id"`
}

type MinerSCUserPoolsInfo struct {
	Pools map[string][]*MinerSCDelegatePoolInfo `json:"pools"`
}

type PoolStats struct {
	DelegateID   string `json:"delegate_id"`
	High         int64  `json:"high"` // } interests and rewards
	Low          int64  `json:"low"`  // }
	InterestPaid int64  `json:"interest_paid"`
	RewardPaid   int64  `json:"reward_paid"`
	NumRounds    int64  `json:"number_rounds"`
	Status       string `json:"status"`
}

type TokenPool struct {
	ID      string `json:"id"`
	Balance int64  `json:"balance"`
}

type ZCNLockingPool struct {
	TokenPool `json:"pool"`
}

type SendTransaction struct {
	Status string `json:"status"`
	Txn    string `json:"tx"`
	Nonce  string `json:"nonce"`
}

type Balance struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
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
		MinerId             string               `json:"miner_id"`
		Round               int64                `json:"round"`
		RoundRandomSeed     int64                `json:"round_random_seed"`
		RoundTimeoutCount   int64                `json:"round_timeout_count"`
		StateHash           string               `json:"state_hash"`
		Transactions        []*TransactionEntity `json:"transactions"`
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

type ValidationNode struct {
	ID                string            `json:"id"`
	BaseURL           string            `json:"url"`
	PublicKey         string            `json:"-"`
	StakePoolSettings StakePoolSettings `json:"stake_pool_settings"`
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
	Responded      int64             `json:"responded"`
	RoundCreatedAt int64             `json:"round_created_at"`
}

type Transaction struct {
	Hash              string `json:"hash"`
	Signature         string `json:"signature"`
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ChainId           string `json:"chain_id"`
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

type TransactionData struct {
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type EventDBTransaction struct {
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

type Reward int

const (
	BlockRewardMiner Reward = iota
	BlockRewardSharder
	BlockRewardBlobber
	FeeRewardMiner
	FeeRewardAuthorizer
	FeeRewardSharder
	ValidationReward
	FileDownloadReward
	ChallengePassReward
	ChallengeSlashPenalty
	CancellationChargeReward
	NumOfRewards
)

var rewardString = []string{
	"min lock demand",
	"block_reward_miner",
	"block_reward_sharder",
	"block_reward_blobber",
	"fees miner",
	"fees_authorizer",
	"fees sharder",
	"validation reward",
	"file download reward",
	"challenge pass reward",
	"challenge slash",
	"cancellation charge",
	"invalid",
}

func (r Reward) String() string {
	return rewardString[r]
}

func (r Reward) Int() int {
	return int(r)
}

type RewardProvider struct {
	Amount      int64  `json:"amount"`
	BlockNumber int64  `json:"block_number"`
	ProviderId  string `json:"provider_id"`
	RewardType  Reward `json:"reward_type"`
}

type RewardDelegate struct {
	Amount      int64  `json:"amount"`
	BlockNumber int64  `json:"block_number"`
	PoolID      string `json:"pool_id"`
	ProviderID  string `json:"provider_id"`
	RewardType  Reward `json:"reward_type"`
}

type EventDBBlock struct {
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
	Transactions          []EventDBTransaction `json:"transactions"`
}

type ReadMarkersCount struct {
	ReadMarkersCount int64 `json:"read_markers_count"`
}

type ReadMarker struct {
	ClientID      string  `json:"client_id"`
	BlobberID     string  `json:"blobber_id"`
	AllocationID  string  `json:"allocation_id"`
	TransactionID string  `json:"transaction_id"`
	OwnerID       string  `json:"owner_id"`
	Timestamp     int64   `json:"timestamp"`
	ReadCounter   int64   `json:"read_counter"`
	ReadSize      float64 `json:"read_size"`
	Signature     string  `json:"signature"`
	PayerID       string  `json:"payer_id"`
	AuthTicket    string  `json:"auth_ticket"`
	BlockNumber   int64   `json:"block_number"`
}

var StorageKeySettings = []string{
	"owner_id",
}

var StorageFloatSettings = []string{
	"cancellation_charge",
	"free_allocation_settings.read_pool_fraction",
	"validator_reward",
	"blobber_slash",
	"block_reward.gamma.alpha",
	"block_reward.gamma.a",
	"block_reward.gamma.b",
	"block_reward.zeta.i",
	"block_reward.zeta.k",
	"block_reward.zeta.mu",
	"stakepool.kill_slash",
	"max_charge",
}

var StorageCurrencySettigs = []string{
	"min_stake",
	"min_stake_per_delegate",
	"max_stake",
	"readpool.min_lock",
	"writepool.min_lock",
	"max_total_free_allocation",
	"max_individual_free_allocation",
	"free_allocation_settings.read_price_range.min",
	"free_allocation_settings.read_price_range.max",
	"free_allocation_settings.write_price_range.min",
	"free_allocation_settings.write_price_range.max",
	"max_read_price",
	"max_write_price",
	"min_write_price",
	"block_reward.block_reward",
	"block_reward.qualifying_stake",
}

var StorageIntSettings = []string{
	"max_challenge_completion_rounds",
	"challenge_generation_gap",
	"max_file_size",
	"min_alloc_size",
	"min_blobber_capacity",
	"free_allocation_settings.data_shards",
	"free_allocation_settings.parity_shards",
	"free_allocation_settings.size",
	"max_blobbers_per_allocation",
	"validators_per_challenge",
	"num_validators_rewarded",
	"max_blobber_select_for_challenge",
	"max_delegates",
	"cost.update_settings",
	"cost.read_redeem",
	"cost.commit_connection",
	"cost.new_allocation_request",
	"cost.update_allocation_request",
	"cost.finalize_allocation",
	"cost.cancel_allocation",
	"cost.add_free_storage_assigner",
	"cost.free_allocation_request",
	"cost.blobber_health_check",
	"cost.update_blobber_settings",
	"cost.pay_blobber_block_rewards",
	"cost.challenge_response",
	"cost.generate_challenge",
	"cost.add_validator",
	"cost.update_validator_settings",
	"cost.add_blobber",
	"cost.read_pool_lock",
	"cost.read_pool_unlock",
	"cost.write_pool_lock",
	"cost.stake_pool_lock",
	"cost.stake_pool_unlock",
	"cost.commit_settings_changes",
	"cost.collect_reward",
	"cost.kill_blobber",
	"cost.kill_validator",
	"cost.shutdown_blobber",
	"cost.shutdown_validator",
}
var StorageBoolSettings = []string{
	"challenge_enabled",
}
var StorageDurationSettings = []string{
	"time_unit",
	"stakepool.min_lock_period",
	"health_check_period",
}

var StorageSettingCount = len(StorageDurationSettings) + len(StorageFloatSettings) + len(StorageIntSettings) + len(StorageKeySettings) + len(StorageBoolSettings)
