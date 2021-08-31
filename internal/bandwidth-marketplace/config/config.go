package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	// Config represents configs stores in Path.
	Config struct {
		ServerChain ServerChain `yaml:"server_chain"`

		BuildingWaitTime       int `yaml:"building_wait_time"`        // in seconds
		DefaultUserGRPCTimeout int `yaml:"default_user_grpc_timeout"` // in seconds

		Cases Cases `yaml:"cases"`

		Log Log `yaml:"log"`

		Consumer *Node `yaml:"consumer"`
		Provider *Node `yaml:"provider"`
		Magma    *Node `yaml:"magma"`
	}

	// Cases contains all test cases.
	Cases struct {
		Session SessionCases `yaml:"session"`
		Stress  StressCase   `yaml:"stress"`
	}

	// SessionCases represents test cases with session.
	SessionCases struct {
		StartUpdateStopOK          bool `yaml:"start_update_stop_ok"`
		StartStopOK                bool `yaml:"start_stop_ok"`
		UpdateNonExistingSessionOK bool `yaml:"update_non_existing_session_ok"`
		StopNonExistingSessionOK   bool `yaml:"stop_non_existing_session_ok"`
	}

	// StressCase represents test case with big number of users connections.
	StressCase struct {
		Enable   bool `yaml:"enable"`
		UsersNum int  `yaml:"users_num"`
	}

	Log struct {
		Enable bool `yaml:"enable"`
	}

	// ServerChain represents config options described in "server_chain" section of the config yaml file.
	// ServerChain must be a field of Config struct
	ServerChain struct {
		ID              string `yaml:"id"`
		OwnerID         string `yaml:"owner_id"`
		BlockWorker     string `yaml:"block_worker"`
		SignatureScheme string `yaml:"signature_scheme"`
	}
)

const (
	// Path is a constant stores path to config file from root application directory.
	Path = "./src/test/config.yaml"
)

// Read reads configs from config file existing in Path.
func Read() (*Config, error) {
	f, err := os.Open(Path)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	cfg := new(Config)
	if err = yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetDefaultUserGRPCTimeout returns default grpc timeout for user converted in time.Duration.
func (cfg *Config) GetDefaultUserGRPCTimeout() time.Duration {
	return time.Duration(cfg.DefaultUserGRPCTimeout) * time.Second
}
