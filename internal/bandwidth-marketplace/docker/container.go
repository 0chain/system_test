package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
)

type (
	dContainer struct {
		cfg        *container.Config
		hostCfg    *container.HostConfig
		networkCfg *network.NetworkingConfig

		name string

		ref string
	}
)

// consumer container section

func newConsumerContainer(cfg *config.Node) (*dContainer, error) {
	mounts, err := consumerCfgMounts()
	if err != nil {
		return nil, err
	}
	hostCfg, err := hostConfig(cfg, mounts)
	if err != nil {
		return nil, err
	}
	return &dContainer{
		cfg:        consumerContainerConfig(cfg, hostCfg.PortBindings),
		hostCfg:    hostCfg,
		networkCfg: networkConfig(cfg),
		ref:        cfg.Ref,
		name:       ConsumerContainerName,
	}, nil
}

func consumerContainerConfig(cfg *config.Node, ports nat.PortMap) *container.Config {
	exposedPorts := make(map[nat.Port]struct{})
	for port := range ports {
		exposedPorts[port] = struct{}{}
	}
	return &container.Config{
		Image:        cfg.Ref,
		ExposedPorts: exposedPorts,
		Cmd: []string{
			"./bin/consumer",
			fmt.Sprintf("--port=%s", cfg.Port),
			fmt.Sprintf("--grpc_port=%s", cfg.GRPCPort),
			"--hostname=localhost",
			"--deployment_mode=0",
			"--keys_file=/consumer/config/keys.txt",
			"--logs_dir=/consumer/log",
			"--db_dir=/consumer/data",
			"--config_file=/consumer/config/config.yaml",
		},
	}
}

func consumerCfgMounts() ([]mount.Mount, error) {
	cfgSrc, err := withTestRoot("src", "consumer", "config")
	if err != nil {
		return nil, err
	}
	dataSrc, err := withTestRoot("src", "consumer", "data")
	if err != nil {
		return nil, err
	}
	logSrc, err := withTestRoot("src", "consumer", "log")
	if err != nil {
		return nil, err
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: cfgSrc,
			Target: "/consumer/config",
		},
		{
			Type:   mount.TypeBind,
			Source: dataSrc,
			Target: "/consumer/data",
		},
		{
			Type:   mount.TypeBind,
			Source: logSrc,
			Target: "/consumer/log",
		},
	}
	return mounts, nil
}

// magma container section

func newMagmaContainer(cfg *config.Node) (*dContainer, error) {
	mounts, err := magmaCfgMounts()
	if err != nil {
		return nil, err
	}
	hostCfg, err := hostConfig(cfg, mounts)
	if err != nil {
		return nil, err
	}
	return &dContainer{
		cfg:        magmaContainerConfig(cfg, hostCfg.PortBindings),
		hostCfg:    hostCfg,
		networkCfg: networkConfig(cfg),
		ref:        cfg.Ref,
		name:       MagmaContainerName,
	}, nil
}

func magmaContainerConfig(magmaCfg *config.Node, ports nat.PortMap) *container.Config {
	exposedPorts := make(map[nat.Port]struct{})
	for port := range ports {
		exposedPorts[port] = struct{}{}
	}
	return &container.Config{
		Image:        magmaCfg.Ref,
		ExposedPorts: exposedPorts,
		Cmd: []string{
			"./bin/magma",
			fmt.Sprintf("--port=%s", magmaCfg.Port),
			fmt.Sprintf("--grpc_port=%s", magmaCfg.GRPCPort),
			"--hostname=\"\"",
			"--config_file=/magma/config/config.yaml",
		},
	}
}

func magmaCfgMounts() ([]mount.Mount, error) {
	cfgSrc, err := withTestRoot("src", "magma", "config")
	if err != nil {
		return nil, err
	}
	logSrc, err := withTestRoot("src", "magma", "log")
	if err != nil {
		return nil, err
	}

	log.Logger.Info("Consumer cfg " + cfgSrc)

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: cfgSrc,
			Target: "/magma/config",
		},
		{
			Type:   mount.TypeBind,
			Source: logSrc,
			Target: "/magma/log",
		},
	}
	return mounts, nil
}

// provider container section

func newProviderContainer(cfg *config.Node) (*dContainer, error) {
	mounts, err := providerCfgMounts()
	if err != nil {
		return nil, err
	}
	hostCfg, err := hostConfig(cfg, mounts)
	if err != nil {
		return nil, err
	}
	return &dContainer{
		cfg:        providerContainerConfig(cfg, hostCfg.PortBindings),
		hostCfg:    hostCfg,
		networkCfg: networkConfig(cfg),
		ref:        cfg.Ref,
		name:       ProviderContainerName,
	}, nil
}

func providerContainerConfig(cfg *config.Node, ports nat.PortMap) *container.Config {
	exposedPorts := make(map[nat.Port]struct{})
	for port := range ports {
		exposedPorts[port] = struct{}{}
	}
	return &container.Config{
		Image:        cfg.Ref,
		ExposedPorts: exposedPorts,
		Cmd: []string{
			"./bin/provider",
			fmt.Sprintf("--port=%s", cfg.Port),
			fmt.Sprintf("--grpc_port=%s", cfg.GRPCPort),
			"--hostname=localhost",
			"--deployment_mode=0",
			"--keys_file=/provider/config/keys.txt",
			"--logs_dir=/provider/log",
			"--db_dir=/provider/data",
			"--config_file=/provider/config/config.yaml",
			"--terms_config_dir=./config/terms",
		},
	}
}

func providerCfgMounts() ([]mount.Mount, error) {
	cfgSrc, err := withTestRoot("src", "provider", "config")
	if err != nil {
		return nil, err
	}
	dataSrc, err := withTestRoot("src", "provider", "data")
	if err != nil {
		return nil, err
	}
	logSrc, err := withTestRoot("src", "provider", "log")
	if err != nil {
		return nil, err
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: cfgSrc,
			Target: "/provider/config",
		},
		{
			Type:   mount.TypeBind,
			Source: dataSrc,
			Target: "/provider/data",
		},
		{
			Type:   mount.TypeBind,
			Source: logSrc,
			Target: "/provider/log",
		},
	}
	return mounts, nil
}

// common section

func hostConfig(cfg *config.Node, mounts []mount.Mount) (*container.HostConfig, error) {
	port, err := nat.NewPort("tcp", cfg.Port)
	if err != nil {
		return nil, err
	}

	grpcPort, err := nat.NewPort("tcp", cfg.GRPCPort)
	if err != nil {
		return nil, err
	}

	return &container.HostConfig{
		PortBindings: nat.PortMap{
			port: []nat.PortBinding{
				{
					HostIP:   "localhost",
					HostPort: cfg.Port,
				},
			},
			grpcPort: []nat.PortBinding{
				{
					HostIP:   "localhost",
					HostPort: cfg.GRPCPort,
				},
			},
		},
		Mounts: mounts,
	}, nil
}

func networkConfig(cfg *config.Node) *network.NetworkingConfig {
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"testnet0": {
				IPAMConfig: &network.EndpointIPAMConfig{
					IPv4Address: cfg.IPV4,
				},
			},
		},
	}
}
