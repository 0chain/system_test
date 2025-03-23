package model

import (
	"github.com/0chain/gosdk_common/core/common"
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

type ZboxJwtToken struct {
	JwtToken string `json:"jwt_token"`
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

type ZboxTransactionDetails struct {
	Hash              string `json:"hash"`
	BlockHash         string `json:"block_hash"`
	Round             int64  `json:"round"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	Value             int64  `json:"value"`
	Signature         string `json:"signature"`
	CreationDate      int64  `json:"creation_date"`
	Fee               int64  `json:"fee"`
	Nonce             int    `json:"nonce"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output"`
	OutputHash        string `json:"output_hash"`
	Status            int    `json:"status"`
}

type ZboxTransactionsDataResponse struct {
	PitId        string                   `json:"pit_id"`
	Transactions []ZboxTransactionDetails `json:"transactions"`
}

type ZboxGraphEndpoint func(*test.SystemTest, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)
type ZboxGraphBlobberEndpoint func(*test.SystemTest, string, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)
