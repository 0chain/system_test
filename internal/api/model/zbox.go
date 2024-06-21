package model

import (
	"time"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	resty "github.com/go-resty/resty/v2"
)

const GB = float64(1024 * 1024 * 1024)

type CSRFToken struct {
	CSRFToken string `json:"csrf_token"`
}

type ZboxMessageResponse struct {
	Message string `json:"message"`
}

type ZboxMessageDataShareinfoResponse struct {
	Message string          `json:"message"`
	Data    []ZboxShareInfo `json:"data"`
}

type ZboxOwner struct {
	UserID      string `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	UserName    string `json:"username"`
	Email       string `json:"email"`
	Biography   string `json:"biography"`
}

type ZboxWallet struct {
	ClientID    string   `json:"client_id"`
	WalletId    int      `json:"wallet_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Mnemonic    string   `json:"mnemonic"`
	AppType     []string `json:"app_type"`
	PublicKey   string   `json:"public_key"`
	LastUpdate  string   `json:"last_update"`
}

type ZboxWalletList struct {
	Message string       `json:"message"`
	Data    []ZboxWallet `json:"data"`
}

type ZboxAllocation struct {
	ID             string `json:"id"`
	WalletID       int64  `json:"wallet_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	AllocationType string `json:"allocation_type"`
	AppType        string `json:"app_type"`
	UpdateAt       string `json:"last_update"`
}

type ZboxFundingResponse struct {
	Id                int64  `json:"id"`
	Amount            int    `json:"amount"`
	Description       string `json:"description"`
	Funded            bool   `json:"funded"`
	TransactionStatus bool   `json:"tx_done"`
}
type ZboxFreeStorage struct {
	Marker     string `json:"marker"`
	FundidngId int    `json:"funding_id"`
}
type ZboxFreeStorageMarker struct {
	Assigner   string           `json:"assigner"`
	Recipient  string           `json:"recipient"`
	FreeTokens float64          `json:"free_tokens"`
	Timestamp  common.Timestamp `json:"timestamp"`
	Signature  string           `json:"signature"`
	Blobbers   []string         `json:"blobbers"`
}
type ZboxFreeStorageMarkerResponse struct {
	Marker             string `json:"marker"`
	RecipientPublicKey string `json:"recipient_public_key"`
}
type ZboxResourceExist struct {
	Exists bool   `json:"exists"`
	Field  string `json:"field"`
}

type ZboxFCMResponse struct {
	FCMToken   string `json:"fcm_token"`
	DeviceType string `json:"device_type"`
	Message    string `json:"message"`
}

type ZboxAllocationList struct {
	WalletName string           `json:"wallet_name"`
	Allocs     []ZboxAllocation `json:"allocs"`
}

type DexState struct {
	TxHash    string `form:"tx_hash" json:"tx_hash" binding:"-"`
	Stage     string `form:"stage" json:"stage" binding:"required"`
	Reference string `form:"reference" json:"reference" binding:"required"`
}

type ZboxUsername struct {
	Username string `json:"username"`
}

type ZboxImage struct {
	SmallLoc string `json:"small_loc"`
	MedLoc   string `json:"medium_loc"`
	LargeLoc string `json:"large_loc"`
}

type ZboxUserInfo struct {
	Username        string    `json:"user_name"` // FIXME: THIS IS INCONSISTENT WITH THE SPELLING OF "USERNAME"
	Biography       string    `json:"biography"`
	CreatedAt       string    `json:"created_at"`
	Avatar          ZboxImage `json:"avatar"`
	BackgroundImage ZboxImage `json:"bg_img"`
}

type ZboxGraphRequest struct {
	From       string `json:"from"`
	To         string `json:"to"`
	DataPoints string `json:"data_points"`
}

type ZboxGraphInt64Response []int64
type ZboxTotalInt64Response int64

type ZboxGraphChallengesResponse struct {
	TotalChallenges      ZboxGraphInt64Response `json:"total"`
	SuccessfulChallenges ZboxGraphInt64Response `json:"successful"`
}

type ZboxShareInfo struct {
	AuthTicket string `json:"auth_ticket"` // FIXME: THIS IS INCONSISTENT WITH THE SPELLING OF "USERNAME"
	Message    string `json:"message"`
	ClientID   string `json:"client_id"`
	Receiver   string `json:"receiver_client_id"`
	LookUpHash string `json:"lookup_hash"`
	CreatedAt  string `json:"CreatedAt"`
	UpdatedAt  string `json:"UpdatedAt"`
}

type ZboxNftCollection struct {
	AllocationId           string  `json:"allocation_id"`
	CollectionId           string  `json:"collection_id"`
	AuthTicket             string  `json:"auth_ticket"`
	CreatedBy              string  `json:"created_by"`
	CollectionName         string  `json:"collection_name"`
	TotalNfts              int64   `json:"total_nfts"`
	CollectionType         string  `json:"collection_type"`
	Symbol                 string  `json:"symbol"`
	BaseUrl                string  `json:"base_url"`
	LastUpdate             string  `json:"last_update"`
	CreatedAtDate          string  `json:"created_at_date"`
	CollectionImage        string  `json:"collection_image"`
	ColleectionBannerImage string  `json:"collection_banner_image"`
	CreatorName            string  `json:"creator_name"`
	PricePerPack           float64 `json:"price_per_back"`
	MaxMints               int64   `json:"max_mints"`
	CurrMints              int64   `json:"curr_mints"`
	BatchSize              int64   `json:"batch_size"`
}

type ZboxNftCollectionList struct {
	ZboxNftCollection  []ZboxNftCollection `json:"collections"`
	NftCollectionCount int64               `json:"total"`
}

type ZboxNft struct {
	Id              int64  `json:"id"`
	AllocationId    string `json:"allocation_id"`
	ClientId        string `json:"client_id"`
	CollectionId    string `json:"collection_id"`
	OwnedBy         string `json:"owned_by"`
	Stage           string `json:"stage"`
	Reference       string `json:"reference"`
	NftActivity     string `json:"nft_activity"`
	MetaData        string `json:"meta_data"`
	NftImage        string `json:"nft_image"`
	IsMinted        bool   `json:"is_minted"`
	AuthTicket      string `json:"auth_ticket"`
	RemotePath      string `json:"remote_path"`
	CreatedBy       string `json:"created_by"`
	CreatorName     string `json:"creator_name"`
	ContractAddress string `json:"contract_address"`
	TokenId         string `json:"token_id"`
	TokenStandard   string `json:"token_standard"`
	TxHash          string `json:"tx_hash"`
	CreatedAtDate   string `json:"created_at_date"`
	LastUpdate      string `json:"last_update"`
}

type ZboxNftList struct {
	NftList  []ZboxNft `json:"all_nfts"`
	NftCount int64     `json:"total"`
}

type ReferralCodeOfUser struct {
	ReferrerCode string `json:"referral_code"`
	ReferrerLink string `json:"referral_link"`
}

type ReferralCount struct {
	ReferralCount int64  `json:"referral_count"`
	RewardPoints  int64  `json:"reward_points"`
	TotalRewards  uint64 `json:"total_rewards"`
}

type TopReferrer struct {
	Referrer     string `json:"referrer"`
	ReferrerName string `json:"referrer_name"`
	Count        int    `json:"count"`
	Avatar       []byte `json:"avatar"`
}

type TopReferrerResponse struct {
	TopUsers []TopReferrer `json:"top_users"`
}

type ReferralRankOfUser struct {
	UserRank   int64 `json:"rank"`
	UserCount  int64 `json:"count"`
	ReferrerID int64 `json:"referrer_id"`
}

type ProviderRewards struct {
	ID                            uint          `json:"id" gorm:"primarykey"`
	CreatedAt                     time.Time     `json:"created_at"`
	UpdatedAt                     time.Time     `json:"updated_at"`
	ProviderID                    string        `json:"provider_id" gorm:"uniqueIndex"`
	Rewards                       currency.Coin `json:"rewards"`
	TotalRewards                  currency.Coin `json:"total_rewards"`
	RoundServiceChargeLastUpdated int64         `json:"round_service_charge_last_updated"`
}

type ZboxGraphEndpoint func(*test.SystemTest, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)
type ZboxGraphBlobberEndpoint func(*test.SystemTest, string, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)

type ZboxBlobber struct {
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
type ZboxProviderBase struct {
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
	City            *string          `json:"city"`
	BrandID         int64            `json:"brand_id"`
	GeoLocation     *string          `json:"geo_location"`
	UserName        *string          `json:"user_name"`
	CustomIcon      *string          `json:"custom_icon"`
	Description     *string          `json:"description"`
}
type ZboxMiner struct {
	ZboxProviderBase

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

type ZboxSharder struct {
	ZboxProviderBase

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

type ZboxValidator struct {
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
type ZboxAuthorizer struct {
	AuthorizerID    string        `json:"id"`
	URL             string        `json:"url"`
	Fee             currency.Coin `json:"fee"`
	LastHealthCheck int64         `json:"last_health_check"`
	DelegateWallet  string        `json:"delegate_wallet"`
	NumDelegates    int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}

type ZboxSharderList struct {
	Sharders []ZboxSharder `json:"sharders"`
}

type ZboxMinerList struct {
	Miners []ZboxMiner `json:"miners"`
}

type ZboxValidatorList struct {
	Validators []ZboxValidator `json:"validators"`
}

type ZboxAuthorizerList struct {
	Authorizers []ZboxAuthorizer `json:"authorizers"`
}
