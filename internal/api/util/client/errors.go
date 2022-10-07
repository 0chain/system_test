package client

import "errors"

// Contains errors used for API client
var (
	ErrNetworkHealthy = errors.New("network seems to be unhealthy")

	ErrNoMinersHealthy   = errors.New("all miners seem to be unhealthy")
	ErrNoShadersHealthy  = errors.New("all shaders seem to be unhealthy")
	ErrNoBlobbersHealthy = errors.New("all blobbers seem to be unhealthy")

	ErrGetFromResource = errors.New("error happened during request")

	ErrExecutionConsensus = errors.New("execution consensus is not reached")
)

// Contains errors used for SDK client
var (
	ErrInitStorageSDK = errors.New("error happened during SDK storage initialization")
)
