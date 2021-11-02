package config

type (
	// Node represents all needed configs for bandwidth node (consumer/provider/magma)
	Node struct {
		// Ref represents docker image build tags that will be pulled.
		Ref string `yaml:"ref"`

		// Port for pages.
		Port string `yaml:"port"`

		// GRPCPort for interacting between nodes.
		GRPCPort string `yaml:"grpc_port"`

		// IPV4 address of the node in the docker`s network.
		IPV4 string `yaml:"ipv4"`

		// PourWallet represents an optional field,
		// used only for the consumer, because cause only consumer spends tokens.
		PourWallet bool `yaml:"pour_wallet"`

		// ExtID represents external id.
		ExtID string `yaml:"ext_id"`

		// NodeDir represents source dir of the node.
		NodeDir string `yaml:"node_dir"`

		// KeysFile represents keys file that stores keys.
		// Used for operations with transactions.
		KeysFile string `yaml:"keys_file"`
	}
)

// GRPCAddress returns full GRPC address for the user,
// that can connect to the node outside the docker container.
func (n *Node) GRPCAddress() string {
	return ":" + n.GRPCPort
}
