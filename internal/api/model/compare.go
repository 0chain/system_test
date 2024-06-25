package model

import (
	"time"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/gosdk/core/common"
)

type Blobber struct {
	ID                       string            `json:"id" validate:"hexadecimal,len=64"`
	BaseURL                  string            `json:"url"`
	Terms                    Terms             `json:"terms"`
	Capacity                 int64             `json:"capacity"`
	Allocated                int64             `json:"allocated"`
	LastHealthCheck          common.Timestamp  `json:"last_health_check"`
	IsKilled                 bool              `json:"is_killed"`
	IsShutdown               bool              `json:"is_shutdown"`
	PublicKey                string            `json:"-"`
	SavedData                int64             `json:"saved_data"`
	DataReadLastRewardRound  float64           `json:"data_read_last_reward_round"`
	LastRewardDataReadRound  int64             `json:"last_reward_data_read_round"`
	StakePoolSettings        StakePoolSettings `json:"stake_pool_settings"`
	NotAvailable             bool              `json:"not_available"`
	ChallengesPassed         int64             `json:"challenges_passed"`
	ChallengesCompleted      int64             `json:"challenges_completed"`
	TotalStake               currency.Coin     `json:"total_stake"`
	CreationRound            int64             `json:"creation_round"`
	ReadData                 int64             `json:"read_data"`
	UsedAllocation           int64             `json:"used_allocation"`
	TotalOffers              currency.Coin     `json:"total_offers"`
	StakedCapacity           int64             `json:"staked_capacity"`
	TotalServiceCharge       currency.Coin     `json:"total_service_charge"`
	UncollectedServiceCharge currency.Coin     `json:"uncollected_service_charge"`
	CreatedAt                time.Time         `json:"created_at"`
	IsRestricted             bool              `json:"is_restricted"`
}
type ProviderBase struct {
	ID              string `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DelegateWallet  string           `json:"delegate_wallet"`
	NumDelegates    int              `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	TotalStake      currency.Coin    `json:"total_stake"`
	Rewards         ProviderRewards  `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
}
type Miner struct {
	ProviderBase

	N2NHost         string `gorm:"column:n2n_host"`
	Host            string
	Port            int
	Path            string
	PublicKey       string
	ShortName       string
	BuildTag        string
	Delete          bool
	Fees            currency.Coin
	Active          bool
	BlocksFinalised int64
	CreationRound   int64 `json:"creation_round" gorm:"index:idx_miner_creation_round"`
}

type Sharder struct {
	ProviderBase

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

type SharderList struct {
	Sharders []Sharder `json:"sharders"`
}

type MinerList struct {
	Miners []Miner `json:"miners"`
}

type ValidatorList struct {
	Validators []Validator `json:"validators"`
}

type AuthorizerList struct {
	Authorizers []Authorizer `json:"authorizers"`
}
