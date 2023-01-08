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

type ZboxWalletAlt struct { //FIXME THIS IS INCONSISTENT AND SHOULD BE FIXED SEE https://github.com/0chain/0box/issues/375
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
