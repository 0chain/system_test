package model

// SetupWallet represents wallet used to perform set up.
type SetupWallet struct {
	UserID        string   `json:"user_id"`
	ClientID      string   `json:"client_id"`
	ClientKey     string   `json:"client_key"`
	PublicKey     string   `json:"public_key"`
	PrivateKey    string   `json:"private_key"`
	PeerPublicKey string   `json:"peer_public_key"`
	Restrictions  []string `json:"restrictions"`
	ExpiredAt     int64    `json:"expired_at"`
}

// SignMessageRequest represents message requested to be signed.
type SignMessageRequest struct {
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
	ClientID  string `json:"client_id"`
}

// SignMessageResponse represents message sign operation response.
type SignMessageResponse struct {
	Sig string `json:"sig"`
}

// KeyDetailsResponse represents split key details retrieval response.
type KeyDetailsResponse struct {
	LastUsed int64 `json:"last_used"`
}
