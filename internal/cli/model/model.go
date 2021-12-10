package climodel

import (
	"time"
)

type Wallet struct {
	ClientID            string `json:"client_id"`
	ClientPublicKey     string `json:"client_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}

type Allocation struct {
	ID             string    `json:"id"`
	Tx             string    `json:"tx"`
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

	// BlobberDetails contains real terms used for the allocation.
	// If the allocation has updated, then terms calculated using
	// weighted average values.
	BlobberDetails []*BlobberAllocation `json:"blobber_details"`

	// ReadPriceRange is requested reading prices range.
	ReadPriceRange PriceRange `json:"read_price_range"`

	// WritePriceRange is requested writing prices range.
	WritePriceRange PriceRange `json:"write_price_range"`

	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`

	StartTime         int64    `json:"start_time"`
	Finalized         bool     `json:"finalized,omitempty"`
	Canceled          bool     `json:"canceled,omitempty"`
	MovedToChallenge  int64    `json:"moved_to_challenge,omitempty"`
	MovedBack         int64    `json:"moved_back,omitempty"`
	MovedToValidators int64    `json:"moved_to_validators,omitempty"`
	Curators          []string `json:"curators"`
}

type AllocationFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
	Hash string `json:"hash"`
}
type Blobber struct {
	BlobberID string `json:"blobber_id"`
	Balance   int64  `json:"balance"`
}

type ReadPoolInfo struct {
	Id           string    `json:"id"`
	Balance      int64     `json:"balance"`
	ExpireAt     int64     `json:"expire_at"`
	AllocationId string    `json:"allocation_id"`
	Blobber      []Blobber `json:"blobbers"`
	Locked       bool      `json:"locked"`
}

type WritePoolInfo struct {
	Id           string    `json:"id"`
	Balance      int64     `json:"balance"`
	ExpireAt     int64     `json:"expire_at"`
	AllocationId string    `json:"allocation_id"`
	Blobber      []Blobber `json:"blobbers"`
	Locked       bool      `json:"locked"`
}

type Attributes struct {
	WhoPaysForReads int `json:"who_pays_for_reads,omitempty"`
}

type ListFileResult struct {
	Name            string     `json:"name"`
	Path            string     `json:"path"`
	Type            string     `json:"type"`
	Size            int64      `json:"size"`
	Hash            string     `json:"hash"`
	Mimetype        string     `json:"mimetype"`
	NumBlocks       int        `json:"num_blocks"`
	LookupHash      string     `json:"lookup_hash"`
	EncryptionKey   string     `json:"encryption_key"`
	ActualSize      int64      `json:"actual_size"`
	ActualNumBlocks int        `json:"actual_num_blocks"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Attribute       Attributes `json:"attributes"`
}

type Terms struct {
	Read_price                int64         `json:"read_price"`
	Write_price               int64         `json:"write_price"`
	Min_lock_demand           float64       `json:"min_lock_demand"`
	Max_offer_duration        time.Duration `json:"max_offer_duration"`
	Challenge_completion_time time.Duration `json:"challenge_completion_time"`
}

type Settings struct {
	Delegate_wallet string  `json:"delegate_wallet"`
	Min_stake       int     `json:"min_stake"`
	Max_stake       int     `json:"max_stake"`
	Num_delegates   int     `json:"num_delegates"`
	Service_charge  float64 `json:"service_charge"`
}

type BlobberInfo struct {
	Id                  string   `json:"id"`
	Url                 string   `json:"url"`
	Capacity            int      `json:"capacity"`
	Last_health_check   int      `json:"last_health_check"`
	Used                int      `json:"used"`
	Terms               Terms    `json:"terms"`
	Stake_pool_settings Settings `json:"stake_pool_settings"`
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
	Attribute       Attributes      `json:"attributes"`
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
		Size            int64           `json:"Size"`
		ActualFileSize  int64           `json:"ActualFileSize"`
		ActualNumBlocks int             `json:"ActualNumBlocks"`
		EncryptedKey    string          `json:"EncryptedKey"`
		CommitMetaTxns  []CommitMetaTxn `json:"CommitMetaTxns"`
		Collaborators   []Collaborator  `json:"Collaborators"`
		Attributes      Attributes      `json:"Attributes"`
	} `json:"MetaData"`
}

type LockedInterestPoolStats struct {
	Stats []LockedInterestPoolStat `json:"stats"`
}

type LockedInterestPoolStat struct {
	ID           string        `json:"pool_id"`
	StartTime    int64         `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	TimeLeft     time.Duration `json:"time_left"`
	Locked       bool          `json:"locked"`
	APR          float64       `json:"apr"`
	TokensEarned int64         `json:"tokens_earned"`
	Balance      int64         `json:"balance"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
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

type StakePoolInfo struct {
	ID          string                       `json:"pool_id"`
	Balance     int64                        `json:"balance"`
	Unstake     int64                        `json:"unstake"`
	Free        int64                        `json:"free"`
	Capacity    int64                        `json:"capacity"`
	WritePrice  int64                        `json:"write_price"`
	Offers      []*StakePoolOfferInfo        `json:"offers"`
	OffersTotal int64                        `json:"offers_total"`
	Delegate    []*StakePoolDelegatePoolInfo `json:"delegate"`
	Earnings    int64                        `json:"interests"`
	Penalty     int64                        `json:"penalty"`
	Rewards     StakePoolRewardsInfo         `json:"rewards"`
	Settings    StakePoolSettings            `json:"settings"`
}

type StakePoolOfferInfo struct {
	Lock         int64  `json:"lock"`
	Expire       int64  `json:"expire"`
	AllocationID string `json:"allocation_id"`
	IsExpired    bool   `json:"is_expired"`
}

// StakePoolRewardsInfo represents stake pool rewards.
type StakePoolRewardsInfo struct {
	Charge    int64 `json:"charge"`    // total for all time
	Blobber   int64 `json:"blobber"`   // total for all time
	Validator int64 `json:"validator"` // total for all time
}

type StakePoolDelegatePoolInfo struct {
	ID               string `json:"id"`                // pool ID
	Balance          int64  `json:"balance"`           // current balance
	DelegateID       string `json:"delegate_id"`       // wallet
	Rewards          int64  `json:"rewards"`           // total for all time
	Interests        int64  `json:"interests"`         // total for all time
	Penalty          int64  `json:"penalty"`           // total for all time
	PendingInterests int64  `json:"pending_interests"` // total for all time
	// Unstake > 0, then the pool wants to unstake. And the Unstake is maximal
	// time it can't be unstaked.
	Unstake int64 `json:"unstake"`
}

type StakePoolSettings struct {
	// DelegateWallet for pool owner.
	DelegateWallet string `json:"delegate_wallet"`
	// MinStake allowed.
	MinStake int64 `json:"min_stake"`
	// MaxStake allowed.
	MaxStake int64 `json:"max_stake"`
	// NumDelegates maximum allowed.
	NumDelegates int `json:"num_delegates"`
	// ServiceCharge is blobber service charge.
	ServiceCharge float64 `json:"service_charge"`
}

type NodeList struct {
	Nodes []Node `json:"Nodes"`
}

type Node struct {
	SimpleNode `json:"simple_miner"`
}

type SimpleNode struct {
	ID                string      `json:"id"`
	N2NHost           string      `json:"n2n_host"`
	Host              string      `json:"host"`
	Port              int         `json:"port"`
	PublicKey         string      `json:"public_key"`
	ShortName         string      `json:"short_name"`
	BuildTag          string      `json:"build_tag"`
	TotalStake        int64       `json:"total_stake"`
	DelegateWallet    string      `json:"delegate_wallet"`
	ServiceCharge     float64     `json:"service_charge"`
	NumberOfDelegates int         `json:"number_of_delegates"`
	MinStake          int64       `json:"min_stake"`
	MaxStake          int64       `json:"max_stake"`
	Stat              interface{} `json:"stat"`
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
	Used              int64             `json:"used"`
	LastHealthCheck   int64             `json:"last_health_check"`
	PublicKey         string            `json:"-"`
	StakePoolSettings StakePoolSettings `json:"stake_pool_settings"`
}

type FreeStorageMarker struct {
	Assigner   string  `json:"assigner,omitempty"`
	Recipient  string  `json:"recipient"`
	FreeTokens float64 `json:"free_tokens"`
	Timestamp  int64   `json:"timestamp"`
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
