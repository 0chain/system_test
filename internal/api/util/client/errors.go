package client

import "errors"

var (
	ErrNetworkHealth = errors.New("network seems to be unhealthy")

	ErrNoMinersHealth  = errors.New("all miners seem to be unhealthy")
	ErrNoShadersHealth = errors.New("all shaders seem to be unhealthy")

	ErrGetFromResource = errors.New("error happened during GET request")

	ErrExecutionConsensus = errors.New("execution consensus is not reached")
)
