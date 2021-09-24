package cli_model

type ChallengePoolInfo struct {
	Id         string `json:"id"`
	Balance    int64  `json:"balance"`
	StartTime  int64  `json:"start_time"`
	Expiration int64  `json:"expiration"`
	Finalized  bool   `json:"finalized"`
}
