package climodel

import "time"

type Wallet struct {
	ClientID            string `json:"client_id"`
	ClientPublicKey     string `json:"client_public_key"`
	EncryptionPublicKey string `json:"encryption_public_key"`
}

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

type WritePoolInfo struct {
	Id           string    `json:"id"`
	Balance      int64     `json:"balance"`
	ExpireAt     int64     `json:"expire_at"`
	AllocationId string    `json:"allocation_id"`
	Blobber      []Blobber `json:"blobbers"`
	Locked       bool      `json:"locked"`
}

type Attributes struct {
	WhoPaysForReads int `json:"who_pays_for_reads,omitempty"`
}

type ListFileResult struct {
	Name            string     `json:"name"`
	Path            string     `json:"path"`
	Type            string     `json:"type"`
	Size            int64      `json:"size"`
	Hash            string     `json:"hash"`
	Mimetype        string     `json:"mimetype"`
	NumBlocks       int        `json:"num_blocks"`
	LookupHash      string     `json:"lookup_hash"`
	EncryptionKey   string     `json:"encryption_key"`
	ActualSize      int64      `json:"actual_size"`
	ActualNumBlocks int        `json:"actual_num_blocks"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Attribute       Attributes `json:"attributes"`
}

type Terms struct {
	Read_price                int64   `json:"read_price"`
	Write_price               int64   `json:"write_price"`
	Min_lock_demand           float64 `json:"min_lock_demand"`
	Max_offer_duration        int64   `json:"max_offer_duration"`
	Challenge_completion_time int64   `json:"challenge_completion_time"`
}

type Settings struct {
	Delegate_wallet string  `json:"delegate_wallet"`
	Min_stake       int     `json:"min_stake"`
	Max_stake       int     `json:"max_stake"`
	Num_delegates   int     `json:"num_delegates"`
	Service_charge  float64 `json:"service_charge"`
}

type BlobberInfo struct {
	Id                  string   `json:"id"`
	Url                 string   `json:"url"`
	Capacity            int      `json:"capacity"`
	Last_health_check   int      `json:"last_health_check"`
	Used                int      `json:"used"`
	Terms               Terms    `json:"terms"`
	Stake_pool_settings Settings `json:"stake_pool_settings"`
}

type ChallengePoolInfo struct {
	Id         string `json:"id"`
	Balance    int64  `json:"balance"`
	StartTime  int64  `json:"start_time"`
	Expiration int64  `json:"expiration"`
	Finalized  bool   `json:"finalized"`
}
