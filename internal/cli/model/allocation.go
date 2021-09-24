package cli_model

type Allocation struct {
	ID             string `json:"id"`
	ExpirationDate int64  `json:"expiration_date"`
	DataShards     int    `json:"data_shards"`
	ParityShards   int    `json:"parity_shards"`
	Size           int64  `json:"size"`
}

type AllocationFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
	Hash string `json:"hash"`
}
