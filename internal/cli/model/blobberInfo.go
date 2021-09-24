package cli_model

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
