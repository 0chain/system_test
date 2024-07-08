package model

import (
	"time"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/gosdk/core/common"
)

type Blobber struct {
	ID                  string           `gorm:"primaryKey" json:"id"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	DelegateWallet      string           `json:"delegate_wallet"`
	NumDelegates        int              `json:"num_delegates"`
	ServiceCharge       float64          `json:"service_charge"`
	TotalStake          currency.Coin    `json:"total_stake"`
	Downtime            uint64           `json:"downtime"`
	LastHealthCheck     common.Timestamp `json:"last_health_check"`
	IsKilled            bool             `json:"is_killed"`
	IsShutdown          bool             `json:"is_shutdown"`
	BaseURL             string           `json:"base_url" gorm:"uniqueIndex"`
	ReadPrice           currency.Coin    `json:"read_price"`
	WritePrice          currency.Coin    `json:"write_price"`
	Capacity            int64            `json:"capacity"`
	Allocated           int64            `json:"allocated"`
	SavedData           int64            `json:"saved_data"`
	ReadData            int64            `json:"read_data"`
	NotAvailable        bool             `json:"not_available"`
	IsRestricted        bool             `json:"is_restricted"`
	OffersTotal         currency.Coin    `json:"offers_total"`
	TotalServiceCharge  currency.Coin    `json:"total_service_charge"`
	ChallengesPassed    uint64           `json:"challenges_passed"`
	ChallengesCompleted uint64           `json:"challenges_completed"`
	OpenChallenges      uint64           `json:"open_challenges"`
	RankMetric          float64          `json:"rank_metric"`
	TotalBlockRewards   currency.Coin    `json:"total_block_rewards"`
	TotalStorageIncome  currency.Coin    `json:"total_storage_income"`
	TotalReadIncome     currency.Coin    `json:"total_read_income"`
	TotalSlashedStake   currency.Coin    `json:"total_slashed_stake"`
	CreationRound       int64            `json:"creation_round"`
}
type Miner struct {
	ID              string        `json:"id"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	DelegateWallet  string        `json:"delegate_wallet"`
	NumDelegates    int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
	TotalStake      int64         `json:"total_stake"`
	Downtime        int64         `json:"downtime"`
	LastHealthCheck int64         `json:"last_health_check"`
	IsKilled        bool          `json:"is_killed"`
	IsShutdown      bool          `json:"is_shutdown"`
	N2NHost         string        `json:"n2n_host"`
	Host            string        `json:"host"`
	Port            int64         `json:"port"`
	Path            string        `json:"path"`
	PublicKey       string        `json:"public_key"`
	ShortName       string        `json:"short_name"`
	BuildTag        string        `json:"build_tag"`
	Delete          bool          `json:"delete"`
	Fees            currency.Coin `json:"fees"`
	Active          bool          `json:"active"`
	BlocksFinalised int64         `json:"blocks_finalised"`
	CreationRound   int64         `json:"creation_round" gorm:"index:idx_miner_creation_round"`
}

type Sharder struct {
	ID             string  `gorm:"primaryKey" json:"id"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DelegateWallet string  `json:"delegate_wallet"`
	NumDelegates   string  `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
	TotalStake     string  `json:"total_stake"`
	// Rewards         ProviderRewards  `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`

	N2NHost       string `gorm:"column:n2n_host"`
	Host          string
	Port          int
	Path          string
	PublicKey     string
	ShortName     string
	BuildTag      string
	Delete        bool
	Fees          currency.Coin
	Active        bool
	CreationRound int64 `json:"creation_round" gorm:"index:idx_sharder_creation_round"`
}

type Validator struct {
	ValidatorID              string           `json:"validator_id"`
	BaseUrl                  string           `json:"url"`
	StakeTotal               currency.Coin    `json:"stake_total"`
	PublicKey                string           `json:"public_key"`
	LastHealthCheck          common.Timestamp `json:"last_health_check"`
	IsKilled                 bool             `json:"is_killed"`
	IsShutdown               bool             `json:"is_shutdown"`
	DelegateWallet           string           `json:"delegate_wallet"`
	NumDelegates             int              `json:"num_delegates"`
	ServiceCharge            float64          `json:"service_charge"`
	TotalServiceCharge       currency.Coin    `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin    `json:"uncollected_service_charge"`
}
type Authorizer struct {
	AuthorizerID    string        `json:"id"`
	URL             string        `json:"url"`
	Fee             currency.Coin `json:"fee"`
	LastHealthCheck int64         `json:"last_health_check"`
	DelegateWallet  string        `json:"delegate_wallet"`
	NumDelegates    int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}
type User struct {
	ID        uint          `json:"id" gorm:"primarykey"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	UserID    string        `json:"user_id" gorm:"uniqueIndex"`
	TxnHash   string        `json:"txn_hash"`
	Balance   currency.Coin `json:"balance"`
	Round     int64         `json:"round"`
	Nonce     int64         `json:"nonce"`
	MintNonce int64         `json:"mint_nonce"`
}
type UserSnapshot struct {
	UserID          string `json:"user_id" gorm:"uniqueIndex"`
	Round           int64  `json:"round"`
	TotalReward     int64  `json:"total_reward"`
	CollectedReward int64  `json:"collected_reward"`
	TotalStake      int64  `json:"total_stake"`
	ReadPoolTotal   int64  `json:"read_pool_total"`
	WritePoolTotal  int64  `json:"write_pool_total"`
	PayedFees       int64  `json:"payed_fees"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
type BlobberSnapshot struct {
	BlobberID           string        `json:"id" gorm:"uniquIndex"`
	URL                 string        `json:"url"`
	Round               int64         `json:"round"`
	WritePrice          currency.Coin `json:"write_price"`
	Capacity            int64         `json:"capacity"`  // total blobber capacity
	Allocated           int64         `json:"allocated"` // allocated capacity
	SavedData           int64         `json:"saved_data"`
	ReadData            int64         `json:"read_data"`
	OffersTotal         currency.Coin `json:"offers_total"`
	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	TotalRewards        currency.Coin `json:"total_rewards"`
	TotalStake          currency.Coin `json:"total_stake"`
	TotalBlockRewards   currency.Coin `json:"total_block_rewards"`
	TotalStorageIncome  currency.Coin `json:"total_storage_income"`
	TotalReadIncome     currency.Coin `json:"total_read_income"`
	TotalSlashedStake   currency.Coin `json:"total_slashed_stake"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	CreationRound       int64         `json:"creation_round"`
	IsKilled            bool          `json:"is_killed"`
	IsShutdown          bool          `json:"is_shutdown"`
}

type SharderSnapshot struct {
	SharderID string `json:"id" gorm:"uniqueIndex"`
	Round     int64  `json:"round"`
	URL       string `json:"url"`

	Fees          currency.Coin `json:"fees"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}
type AuthorizerSnapshot struct {
	AuthorizerID string `json:"id" gorm:"uniquIndex"`
	Round        int64  `json:"round"`
	URL          string `json:"url"`

	Fee           currency.Coin `json:"fee"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	TotalMint     currency.Coin `json:"total_mint"`
	TotalBurn     currency.Coin `json:"total_burn"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}
type ValidatorSnapshot struct {
	ValidatorID string `json:"id" gorm:"uniqueIndex"`
	URL         string `json:"url"`

	Round         int64         `json:"round"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}
type MinerSnapshot struct {
	MinerID         string        `json:"id" gorm:"uniqueIndex"`
	Round           int64         `json:"round"`
	URL             string        `json:"url"`
	Fees            currency.Coin `json:"fees"`
	TotalStake      currency.Coin `json:"total_stake"`
	TotalRewards    currency.Coin `json:"total_rewards"`
	ServiceCharge   float64       `json:"service_charge"`
	BlocksFinalised int64         `json:"blocks_finalised"`
	CreationRound   int64         `json:"creation_round"`
	IsKilled        bool          `json:"is_killed"`
	IsShutdown      bool          `json:"is_shutdown"`
}
