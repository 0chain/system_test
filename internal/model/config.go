package model

type Config struct {
	BlockWorker             string `yaml:"block_worker"`
	SignatureScheme         string `yaml:"signature_scheme"`
	MinSubmit               int    `yaml:"min_submit"`
	MinConfirmation         int    `yaml:"min_confirmation"`
	ConfirmationChainLength int    `yaml:"confirmation_chain_length"`
	MaxTxnQuery             int    `yaml:"max_txn_query"`
	QuerySleepTime          int    `yaml:"query_sleep_time"`
}
