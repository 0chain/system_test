package bandwidth_marketplace

import (
	"testing"
)

type (
	// rewardPoolsConfigurator determines configuration properties of pools.
	rewardPoolsConfigurator struct {
		provider    pool
		accessPoint pool
		all         pool
	}

	pool struct {
		enabled bool
		name    string
	}
)

func (r *rewardPoolsConfigurator) countEnabled() int64 {
	var res int64
	if r.provider.enabled {
		res++
	}
	if r.accessPoint.enabled {
		res++
	}
	if r.all.enabled {
		res++
	}
	return res
}

type (
	action func(t *testing.T)
)

func emptyAction() action {
	return func(t *testing.T) {}
}
