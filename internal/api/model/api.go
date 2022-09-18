package model

import (
	"encoding/json"
	"github.com/0chain/gosdk/core/sys"
	"log"
	"strconv"
	"time"
)

type NetworkDNSResponse struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

type NetworkHealthResources struct {
	NetworkDNSResponse
}

type ExecutionRequest struct {
	FormData map[string]string

	QueryParams map[string]string

	Headers map[string]string

	Body interface{}

	Dst interface{}

	RequiredStatusCode int
}

type Wallet struct {
	ClientID    string         `json:"client_id"`
	ClientKey   string         `json:"client_key"`
	Keys        []*sys.KeyPair `json:"keys"`
	Mnemonics   string         `json:"mnemonics"`
	Version     string         `json:"version"`
	DateCreated string         `json:"date_created"`
	Nonce       int
}

func (w *Wallet) IncNonce() {
	w.Nonce++
}

func (w *Wallet) MustGetKeyPair() *sys.KeyPair {
	if len(w.Keys) == 0 {
		log.Fatalln("wallet has no keys")
	}
	return w.Keys[0]
}

func (w *Wallet) MustConvertDateCreatedToInt() int {
	result, err := strconv.Atoi(w.DateCreated)
	if err != nil {
		log.Fatalln(err)
	}
	return result
}

func (w *Wallet) String() string {
	out, err := json.Marshal(w)
	if err != nil {
		return "failed to serialize wallet object"
	}

	return string(out)
}

type TransactionData struct {
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

func NewFaucetTransactionData() TransactionData {
	return TransactionData{
		Name:  "pour",
		Input: "{}",
	}
}

func NewCreateAllocationTransactionData(scRestGetAllocationBlobbersResponse SCRestGetAllocationBlobbersResponse) TransactionData {
	input, err := json.Marshal(scRestGetAllocationBlobbersResponse)
	if err != nil {
		log.Fatalln(err)
	}

	return TransactionData{
		Name:  "new_allocation_request",
		Input: input,
	}
}

func NewStackPoolTransactionData(createStakePoolRequest CreateStakePoolRequest) TransactionData {
	input, err := json.Marshal(createStakePoolRequest)
	if err != nil {
		log.Fatalln(err)
	}

	return TransactionData{
		Name:  "stake_pool_lock",
		Input: input,
	}
}

func NewUpdateAllocationTransactionData(updateAllocationRequest UpdateAllocationRequest) TransactionData {
	input, err := json.Marshal(updateAllocationRequest)
	if err != nil {
		log.Fatalln(err)
	}

	return TransactionData{
		Name:  "update_allocation_request",
		Input: string(input),
	}
}

type InternalTransactionPutRequest struct {
	TransactionData
	ToClientID string
	Wallet     *Wallet
}

type TransactionPutRequest struct {
	Hash              string `json:"hash"`
	Signature         string `json:"signature"`
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionNonce  int    `json:"transaction_nonce"`
}

type TransactionEntity struct {
	PublicKey         string `json:"public_key,omitempty"`
	Version           string `json:"version"`
	ClientId          string `json:"client_id"`
	ToClientId        string `json:"to_client_id"`
	TransactionData   string `json:"transaction_data"`
	TransactionValue  int64  `json:"transaction_value"`
	CreationDate      int64  `json:"creation_date"`
	TransactionFee    int64  `json:"transaction_fee"`
	TransactionType   int    `json:"transaction_type"`
	TransactionOutput string `json:"transaction_output,omitempty"`
	TxnOutputHash     string `json:"txn_output_hash"`
	TransactionNonce  int    `json:"transaction_nonce"`
	Hash              string `json:"hash"`
	ChainId           string `json:"chain_id"`
	Signature         string `json:"signature"`
	TransactionStatus int    `json:"transaction_status"`
}

type TransactionPutResponse struct {
	Request TransactionPutRequest
	Async   bool              `json:"async"`
	Entity  TransactionEntity `json:"entity"`
}

type TransactionGetConfirmationRequest struct {
	Hash string
}

type TransactionGetConfirmationResponse struct {
	Version               string             `json:"version"`
	Hash                  string             `json:"hash"`
	BlockHash             string             `json:"block_hash"`
	PreviousBlockHash     string             `json:"previous_block_hash"`
	Transaction           *TransactionEntity `json:"txn,omitempty"`
	CreationDate          int64              `json:"creation_date,omitempty"`
	MinerID               string             `json:"miner_id"`
	Round                 int64              `json:"round"`
	Status                int                `json:"transaction_status"`
	RoundRandomSeed       int64              `json:"round_random_seed"`
	StateChangesCount     int                `json:"state_changes_count"`
	MerkleTreeRoot        string             `json:"merkle_tree_root"`
	MerkleTreePath        *MerkleTreePath    `json:"merkle_tree_path"`
	ReceiptMerkleTreeRoot string             `json:"receipt_merkle_tree_root"`
	ReceiptMerkleTreePath *MerkleTreePath    `json:"receipt_merkle_tree_path"`
}

type ClientGetBalanceRequest struct {
	ClientID string
}

type ClientGetBalanceResponse struct {
	Txn     string `json:"txn"`
	Round   int64  `json:"round"`
	Balance int64  `json:"balance"`
}

type SCStateGetRequest struct {
	SCAddress, Key string
}

type SCStateGetResponse struct {
	ID        string    `json:"ID"`
	StartTime time.Time `json:"StartTime"`
	Used      float64   `json:"Used"`
}

type GetMinerStatsResponse struct {
	BlockFinality      float64                  `json:"block_finality"`
	LastFinalizedRound int64                    `json:"last_finalized_round"`
	BlocksFinalized    int64                    `json:"blocks_finalized"`
	StateHealth        int64                    `json:"state_health"`
	CurrentRound       int64                    `json:"current_round"`
	RoundTimeout       int64                    `json:"round_timeout"`
	Timeouts           int64                    `json:"timeouts"`
	AverageBlockSize   int                      `json:"average_block_size"`
	NetworkTime        map[string]time.Duration `json:"network_times"`
}

type GetSharderStatsResponse struct {
	LastFinalizedRound     int64   `json:"last_finalized_round"`
	StateHealth            int64   `json:"state_health"`
	AverageBlockSize       int     `json:"average_block_size"`
	PrevInvocationCount    uint64  `json:"previous_invocation_count"`
	PrevInvocationScanTime string  `json:"previous_incovcation_scan_time"`
	MeanScanBlockStatsTime float64 `json:"mean_scan_block_stats_time"`
}

type SCRestOpenChallengesRequest struct {
	BlobberID string
}

type SCRestGetAllocationBlobbersResponse struct {
	Blobbers *[]string `json:"blobbers"`
	BlobberRequirements
}

type SCRestGetAllocationRequest struct {
	AllocationID string
}

type SCRestGetAllocationBlobbersRequest struct {
	ClientID, ClientKey string
}

type BlobberRequirements struct {
	DataShards      int64      `json:"data_shards"`
	ParityShards    int64      `json:"parity_shards"`
	Size            int64      `json:"size"`
	OwnerId         string     `json:"owner_id"`
	OwnerPublicKey  string     `json:"owner_public_key"`
	ExpirationDate  int64      `json:"expiration_date"`
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`
}

type UpdateAllocationRequest struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	OwnerID              string `json:"owner_id"`
	Size                 int64  `json:"size"`
	Expiration           int64  `json:"expiration_date"`
	SetImmutable         bool   `json:"set_immutable"`
	UpdateTerms          bool   `json:"update_terms"`
	AddBlobberId         string `json:"add_blobber_id"`
	RemoveBlobberId      string `json:"remove_blobber_id"`
	ThirdPartyExtendable bool   `json:"third_party_extendable"`
	FileOptions          uint8  `json:"file_options"`
}
