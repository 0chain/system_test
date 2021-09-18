package cli_model

type Allocation struct {
	ID             string `json:"id"`
	ExpirationDate int64  `json:"expiration_date"`
	DataShards     int    `json:"data_shards"`
	ParityShards   int    `json:"parity_shards"`
	Size           int64  `json:"size"`
}
