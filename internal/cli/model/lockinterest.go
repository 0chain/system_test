package climodel

import (
	"time"
)

type LockedInterestPoolStats struct {
	Stats []LockedInterestPoolStat `json:"stats"`
}

type LockedInterestPoolStat struct {
	ID           string        `json:"pool_id"`
	StartTime    int64         `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	TimeLeft     time.Duration `json:"time_left"`
	Locked       bool          `json:"locked"`
	APR          float64       `json:"apr"`
	TokensEarned int64         `json:"tokens_earned"`
	Balance      int64         `json:"balance"`
}
