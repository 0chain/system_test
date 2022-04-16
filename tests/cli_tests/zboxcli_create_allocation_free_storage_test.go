package cli_tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/herumi/bls-go-binary/bls"

	"github.com/stretchr/testify/require"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
)

const (
	chainID                     = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
	storageSmartContractAddress = `6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7`
	txnTypeSmartContract        = 1000 // A smart contract transaction type

	freeTokensIndividualLimit = 10.0
	freeTokensTotalLimit      = 100.0
)

func init() {
	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		panic(err)
	}
}

func readWalletFile(t *testing.T, file string) *climodel.WalletFile {
	wallet := &climodel.WalletFile{}

	f, err := os.Open(file)
	require.Nil(t, err, "wallet file %s not found", file)

	ownerWalletBytes, err := io.ReadAll(f)
	require.Nil(t, err, "error reading wallet file %s", file)

	err = json.Unmarshal(ownerWalletBytes, wallet)
	require.Nil(t, err, "error marshaling wallet content")

	return wallet
}

func sendTxn(miners climodel.NodeList, txn *apimodel.Transaction) error {
	var (
		err error
		wg  sync.WaitGroup
	)

	for i := range miners.Nodes {
		wg.Add(1)
		go func(node climodel.Node) {
			defer wg.Done()
			_, apiErr := apiPutTransaction(getNodeBaseURL(node.Host, node.Port), txn)
			if apiErr != nil {
				err = apiErr
			}
		}(miners.Nodes[i])
	}
	wg.Wait()

	return err
}

func apiPutTransaction(minerBaseURL string, txn *apimodel.Transaction) (*http.Response, error) {
	txnData, err := json.Marshal(txn)
	if err != nil {
		return nil, err
	}

	return http.Post(minerBaseURL+"/v1/transaction/put", "application/json", bytes.NewBuffer(txnData))
}

func freeAllocationAssignerTxn(t *testing.T, from, assigner *climodel.WalletFile) *apimodel.Transaction {
	txn := &apimodel.Transaction{}
	txn.Version = "1.0"
	txn.ClientId = from.ClientID
	txn.CreationDate = time.Now().Unix()
	txn.ChainId = chainID
	txn.PublicKey = from.ClientKey
	txn.TransactionType = txnTypeSmartContract
	txn.ToClientId = storageSmartContractAddress
	txn.TransactionValue = 0

	input := map[string]interface{}{
		"name":             assigner.ClientID,
		"public_key":       assigner.ClientKey,
		"individual_limit": freeTokensIndividualLimit,
		"total_limit":      freeTokensTotalLimit,
	}

	sn := apimodel.SmartContractTxnData{Name: "add_free_storage_assigner", InputArgs: input}
	snBytes, err := json.Marshal(sn)
	require.Nil(t, err, "error marshaling smart contract data")

	txn.TransactionData = string(snBytes)
	txn.Hash = txnHash(txn)
	txn.Signature = sign(t, txn.Hash, from)

	return txn
}

func sign(t *testing.T, data string, wallet *climodel.WalletFile) string {
	rawHash, err := hex.DecodeString(data)
	require.Nil(t, err, "failed to decode hex %s", data)
	require.NotNil(t, rawHash, "failed to decode hex %s", data)

	var sk bls.SecretKey
	sk.SetByCSPRNG()
	err = sk.DeserializeHexStr(wallet.Keys[0].PrivateKey)
	require.Nil(t, err, "failed to serialize hex of private key")

	sig := sk.Sign(string(rawHash))

	return sig.SerializeToHexStr()
}

func txnHash(txn *apimodel.Transaction) string {
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", txn.CreationDate, txn.ClientId,
		txn.ToClientId, txn.TransactionValue, hash(txn.TransactionData))
	return hash(hashdata)
}

func hash(data string) string {
	h := sha3.New256()
	h.Write([]byte(data))
	var buf []byte
	return hex.EncodeToString(h.Sum(buf))
}
