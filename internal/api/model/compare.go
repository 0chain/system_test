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
	ID              string           `json:"id"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
	DelegateWallet  string           `json:"delegate_wallet"`
	NumDelegates    string           `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	TotalStake      string           `json:"total_stake"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
	N2NHost         string           `json:"n2n_host"`
	Host            string           `json:"host"`
	Port            int              `json:"port"`
	Path            string           `json:"path"`
	PublicKey       string           `json:"public_key"`
	ShortName       string           `json:"short_name"`
	BuildTag        string           `json:"build_tag"`
	Delete          bool             `json:"delete"`
	Fees            currency.Coin    `json:"fees"`
	Active          bool             `json:"active"`
	CreationRound   int64            `json:"creation_round" gorm:"index:idx_sharder_creation_round"`
}

type Validator struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DelegateWallet  string           `json:"delegate_wallet"`
	NumDelegates    int              `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	TotalStake      currency.Coin    `json:"total_stake"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
	Url             string           `json:"base_url"`
	PublicKey       string           `json:"public_key"`
	CreationRound   int64            `json:"creation_round"`
}
type Authorizer struct {
	ID              string           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DelegateWallet  string           `json:"delegate_wallet"`
	NumDelegates    int              `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	TotalStake      currency.Coin    `json:"total_stake"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
	URL             string           `json:"url"`
	Fee             currency.Coin    `json:"fee"`
	TotalMint       currency.Coin    `json:"total_mint"`
	TotalBurn       currency.Coin    `json:"total_burn"`
	CreationRound   int64            `json:"creation_round"`
}
type User struct {
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	UserID    string        `json:"user_id" gorm:"uniqueIndex"`
	TxnHash   string        `json:"txn_hash"`
	Balance   currency.Coin `json:"balance"`
	Round     int64         `json:"round"`
	Nonce     int64         `json:"nonce"`
	MintNonce int64         `json:"mint_nonce"`
}
type ProviderRewards struct {
	ID                            uint          `json:"id"`
	CreatedAt                     time.Time     `json:"created_at"`
	UpdatedAt                     time.Time     `json:"updated_at"`
	ProviderID                    string        `json:"provider_id"`
	Rewards                       currency.Coin `json:"rewards"`
	TotalRewards                  currency.Coin `json:"total_rewards"`
	RoundServiceChargeLastUpdated int64         `json:"round_service_charge_last_updated"`
}
