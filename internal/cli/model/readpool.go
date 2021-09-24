package cli_model

type Blobber struct {
	BlobberID string `json:"blobber_id"`
	Balance   int64  `json:"balance"`
}

type ReadPoolInfo struct {
	Id           string    `json:"id"`
	Balance      int64     `json:"balance"`
	ExpireAt     int64     `json:"expire_at"`
	AllocationId string    `json:"allocation_id"`
	Blobber      []Blobber `json:"blobbers"`
	Locked       bool      `json:"locked"`
}
