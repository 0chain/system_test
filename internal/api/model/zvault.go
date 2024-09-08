package model

// SplitWallet represents both split wallet and split key after generation operation.
type SplitWallet struct {
	ClientID      string    `json:"client_id"`
	ClientKey     string    `json:"client_key"`
	PeerPublicKey string    `json:"peer_public_key"`
	Keys          []KeyPair `json:"keys"`
	Mnemonic      string    `json:"mnemonics"`
	Version       string    `json:"version"`
	DateCreated   string    `json:"date_created"`
	Nonce         int64     `json:"nonce"`
	IsSplit       bool      `json:"is_split"`
}

// GetKeyResponse represents retrieved set of split keys.
type GetKeyResponse struct {
	Keys []*SplitKey `json:"keys"`
}

// SplitKey represents retrieved split key.
type SplitKey struct {
	UserID        string `json:"user_id"`
	ClientID      string `json:"client_id"`
	ClientKey     string `json:"client_key"`
	PrivateKey    string `gorm:"unique" json:"private_key"`
	PublicKey     string `gorm:"unique" json:"public_key"`
	PeerPublicKey string `gorm:"unique" json:"peer_public_key"`
	Mnemonic      string `json:"mnemonics"`
	SharedTo      string `json:"shared_to"`
	IsRevoked     bool   `json:"is_revoked"`
	CreatedAt     int64  `json:"created_at"`
	ExpiresAt     int64  `json:"expires_at"`
}

// StoreRequest represents store request payload.
type StoreRequest struct {
	Mnemonic   string `json:"mnemonic"`
	PrivateKey string `json:"private_key"`
}

// ShareWalletRequest represents share wallet request payload.
type ShareWalletRequest struct {
	PublicKey    string `json:"public_key"`
	TargetUserID string `json:"target_user_id"`
}
