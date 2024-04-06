package model

import (
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/util/test"
	resty "github.com/go-resty/resty/v2"
)

const GB = float64(1024 * 1024 * 1024)

type FirebaseSession struct {
	SessionInfo string `json:"sessionInfo"`
}
type FirebaseToken struct {
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalId      string `json:"localId"`
	IsNewUser    bool   `json:"isNewUser"`
	PhoneNumber  string `json:"phoneNumber"`
}

type CSRFToken struct {
	CSRFToken string `json:"csrf_token"`
}

type ZboxOwner struct {
	PhoneNumber string `json:"phone_number"`
	UserName    string `json:"username"`
}

type ZboxWallet struct {
	ClientID    string           `json:"client_id"`
	WalletId    int              `json:"wallet_id"`
	PhoneNumber string           `json:"phone_number"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Mnemonic    string           `json:"mnemonic"`
	AppType     []string         `json:"app_type"`
	Allocation  []ZboxAllocation `json:"allocation"`
	PublicKey   string           `json:"public_key"`
	LastUpdate  string           `json:"last_update"`
}

type ZboxFundingResponse struct {
	Id                int    `json:"id"`
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

type ZboxAllocation struct {
	Id             string `json:"id"`
	WalletId       int    `json:"wallet_id"`
	Name           string `json:"name"`
	Description    string `json:"random_description"`
	AllocationType string `json:"allocation_type"`
	AppType        string `json:"app_type"`
	LastUpdate     string `json:"last_update"`
}

type MessageContainer struct {
	Message string `json:"message"`
}

type ZboxWalletList struct {
	MessageContainer
	Data []ZboxWallet `json:"data"`
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

type ZboxMessageResponse struct {
	Message string `json:"message"`
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

type ZboxShareInfoList struct {
	MessageContainer
	Data []ZboxShareInfo `json:"data"`
}

type ZboxNftCollection struct {
	AllocationId           string `json:"allocation_id"`
	CollectionId           string `json:"collection_id"`
	AuthTicket             string `json:"auth_ticket"`
	CreatedBy              string `json:"created_by"`
	CollectionName         string `json:"collection_name"`
	TotalNfts              string `json:"total_nfts"`
	CollectionType         string `json:"collection_type"`
	Symbol                 string `json:"symbol"`
	BaseUrl                string `json:"base_url"`
	LastUpdate             string `json:"last_update"`
	CreatedAtDate          string `json:"created_at_date"`
	CollectionImage        string `json:"collection_image"`
	ColleectionBannerImage string `json:"collection_banner_image"`
	CreatorName            string `json:"creator_name"`
	PricePerPack           int    `json:"price_per_back"`
	MaxMints               string `json:"max_mints"`
	CurrMints              string `json:"curr_mints"`
	BatchSize              string `json:"batch_size"`
}

type ZboxNftCollectionById struct {
	NftCollection ZboxNftCollection `json:"collections"`
}
type ZboxNftCollectionList struct {
	ZboxNftCollection  []ZboxNftCollection `json:"collections"`
	NftCollectionCount int                 `json:"total"`
}

type ReferralCodeOfUser struct {
	ReferrerCode  string `json:"referral_code"`
	ReferrerLink  string `json:"referral_link"`
	ReferrerPhone string `json:"referer_mail"`
}

type ReferralCountOfUser struct {
	ReferralCount int64 `json:"referral_count"`
	RewardPoints  int64 `json:"reward_points"`
}

type TopUser struct {
	Score int64 `json:"score"`
	Rank  int64 `json:"rank"`
}

type ReferralLeaderBoard struct {
	Users []TopUser `json:"users"`
	Total int64     `json:"total"`
}

type ReferralRankOfUser struct {
	UserRank   int64 `json:"rank"`
	UserCount  int64 `json:"count"`
	ReferrerID int64 `json:"referrer_id"`
}

type ZboxNft struct {
	Id              int    `json:"id"`
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
	NftCount int       `json:"total"`
}

type ZboxNftListByWalletID struct {
	NftList  []ZboxNft `json:"nfts_by_client_id"`
	NftCount int       `json:"total"`
}

type ZboxNftListByCollection struct {
	NftList  []ZboxNft `json:"nfts_by_collection_id"`
	NftCount int       `json:"total"`
}
type ZboxGraphEndpoint func(*test.SystemTest, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)
type ZboxGraphBlobberEndpoint func(*test.SystemTest, string, *ZboxGraphRequest) (*ZboxGraphInt64Response, *resty.Response, error)
