package model

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

type ZboxWalletAlt struct { // FIXME THIS IS INCONSISTENT AND SHOULD BE FIXED SEE https://github.com/0chain/0box/issues/375
	WalletId    string   `json:"wallet_id"`
	PhoneNumber string   `json:"phone_number"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Mnemonic    string   `json:"mnemonic"`
	AppType     []string `json:"app_type"`
	Allocation  string   `json:"allocation"`
	LastUpdate  string   `json:"last_update"`
}

type ZboxAllocation struct {
	Id             string   `json:"id"`
	WalletId       int      `json:"wallet_id"`
	Name           string   `json:"name"`
	Description    string   `json:"random_description"`
	AllocationType string   `json:"allocation_type"`
	AppType        string   `json:"app_type"`
	NftState       NftState `json:"nft_state"`
	LastUpdate     string   `json:"last_update"`
}

type NftState struct {
	AllocationId string `json:"allocation_id"`
	CollectionId string `json:"collection_id"`
	OwnedBy      string `json:"owned_by"`
	Stage        string `json:"stage"`
	Reference    string `json:"reference"`
	NftActivity  string `json:"nft_activity"`
	Metadata     string `json:"meta_data"`
	NftImage     string `json:"nft_image"`
}

type ZboxWallet struct {
	ClientId          string   `json:"client_id"`
	WalletId          int      `json:"wallet_id"`
	WalletName        string   `json:"wallet_name"`
	WalletDescription string   `json:"wallet_description"`
	AppId             []string `json:"app_id"`
	PublicKey         string   `json:"public_key"`
	LastUpdate        string   `json:"last_update"`
}

type MessageContainer struct {
	Message string `json:"message"`
}

type ZboxWalletList struct {
	MessageContainer
	Data []ZboxWalletAlt `json:"data"`
}

type ZboxAllocationList struct {
	WalletName string           `json:"wallet_name"`
	Allocs     []ZboxAllocation `json:"allocs"`
}

type ZboxWalletKeys []struct {
	ZboxWallet
}

type ZboxSuccess struct {
	Success string `json:"success"`
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

type ZboxShareInfo struct {
	AuthTicket string `json:"auth_ticket"` // FIXME: THIS IS INCONSISTENT WITH THE SPELLING OF "USERNAME"
	Message    string `json:"message"`
	FromInfo   string `json:"from_info"`
	ClientID   string `json:"client_id"`
	Receiver   string `json:"Receiver"`
	LookUpHash string `json:"lookup_hash"`
	CreatedAt  string `json:"CreatedAt"`
	UpdatedAt  string `json:"UpdatedAt"`
	AppType    int    `json:"app_type"`
	// [FIXME] need string for app type. Sanchit is working o this.
}

type ZboxShareInfoList struct {
	MessageContainer
	Data []ZboxShareInfo `json:"data"`
}

type ZboxNftCollection struct {
	AllocationId           string `json:"allocation_id"`
	CollectionId           string `json:"collection_id"`
	CreatedBy              string `json:"created_by"`
	CollectionName         string `json:"collection_name"`
	TotalNfts              int    `json:"total_nfts"`
	CollectionType         string `json:"collection_type"`
	Symbol                 string `json:"symbol"`
	BaseUrl                string `json:"base_url"`
	LastUpdate             string `json:"last_update"`
	CreatedAtDate          string `json:"created_at_date"`
	CollectionImage        string `json:"collection_image"`
	ColleectionBannerImage string `json:"collection_banner_image"`
	CreatorName            string `json:"creator_name"`
	PricePerPack           int    `json:"price_per_back"`
	MaxMints               int    `json:"max_mints"`
	CurrMints              int    `json:"curr_mints"`
	BatchSize              int    `json:"batch_size"`
}

type ZboxNft struct {
	Id              string `json:"id"`
	AllocationId    string `json:"allocation_id"`
	ClientId        string `json:"client_id"`
	CollectionId    string `json:"collection_id"`
	OwnedBy         string `json:"owned_by"`
	Stage           string `json:"stage"`
	Reference       string `json:"reference"`
	NftActivity     string `json:"nft_activity"`
	MetaData        string `json:"meta_data"`
	NftImage        string `json:"nft_image"`
	IsMinted        string `json:"is_minted"`
	AuthTicket      string `json:"auth_ticket"`
	RemotePath      string `json:"remote_path"`
	CreatedBy       string `json:"created_by"`
	CreatorName     string `json:"creator_name"`
	ContractAddress string `json:"contract_address"`
	TokenId         string `json:"token_id"`
	TokenStandard   string `json:"token_standard"`
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

type ZboxNftListByCollectionId struct {
	NftList []ZboxNft `json:"collections"`
}
