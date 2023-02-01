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
}

type ZboxWalletList struct {
	MessageContainer
	Data []ZboxWalletAlt `json:"data"`
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

type PostNftInfo struct {
	AllocationId    string `form:"allocation_id" json:"allocation_id"`
	CreatedBy       string `form:"created_by" json:"created_by"`
	ContractAddress string `form:"contract_address" json:"contract_address"`
	TokenID         string `form:"token_id" json:"token_id"`
	TokenStandard   string `form:"token_standard" json:"token_standard"`
}

type PutNftInfo struct {
	AllocationId    string `form:"allocation_id" json:"allocation_id"`
	CreatedBy       string `form:"created_by" json:"created_by"`
	ContractAddress string `form:"contract_address" json:"contract_address"`
	TokenID         string `form:"token_id" json:"token_id"`
	TokenStandard   string `form:"token_standard" json:"token_standard"`
}

type PutNftState struct {
	Stage        string `form:"stage" json:"stage"`
	Reference    string `form:"reference" json:"reference"`
	CollectionId string `form:"collection_id" json:"collection_id"`
	OwnedBy      string `form:"owned_by" json:"owned_by"`
	NftActivity  string `form:"nft_activity" json:"nft_activity"`
	Metadata     string `form:"meta_data" json:"meta_data"`
	AllocationId string `form:"allocation_id" json:"allocation_id"`
}
