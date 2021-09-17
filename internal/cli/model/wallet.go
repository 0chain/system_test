package cli_model

type Wallet struct {
	ClientId            string `json:"client_id"`
	ClientPublicKey     string `json:"client_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}
