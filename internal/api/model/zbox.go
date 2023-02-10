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

type DexState struct {
	TxHash    string `form:"tx_hash" json:"tx_hash" binding:"-"`
	Stage     string `form:"stage" json:"stage" binding:"required"`
	Reference string `form:"reference" json:"reference" binding:"required"`
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
