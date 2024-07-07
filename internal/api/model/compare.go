package model

import (
	"time"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/gosdk/core/common"
)

type Blobber struct {
	ID             string `gorm:"primaryKey" json:"id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	DelegateWallet string `json:"delegate_wallet"`
	NumDelegates   string `json:"num_delegates"`
	ServiceCharge  string `json:"service_charge"`
	TotalStake     string `json:"total_stake"`
	// Rewards         ProviderRewards  `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime        uint64 `json:"downtime"`
	LastHealthCheck string `json:"last_health_check"`
	IsKilled        bool   `json:"is_killed"`
	IsShutdown      bool   `json:"is_shutdown"`
	BaseURL         string `json:"base_url"`
	ReadPrice       string `json:"read_price"`
	WritePrice      string `json:"write_price"`
	Capacity        int64  `json:"capacity"`
	Allocated       int64  `json:"allocated"`
}
type Miner struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DelegateWallet string    `json:"delegate_wallet"`

	TotalStake      int64         `json:"total_stake"`
	Downtime        int64         `json:"downtime"`
	LastHealthCheck int64         `json:"last_health_check"`
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
