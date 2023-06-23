package api_tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/gocolly/colly"
	"github.com/stretchr/testify/require"
	"gopkg.in/errgo.v2/errors"
)

func TestClientSendNonceGreaterThanFutureNonceLimit(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	wallet1, mnemonic := apiClient.CreateWalletWithMnemonicsInReturnValue(t)
	zcncore.SetWallet(*wallet1.ToZCNCryptoWallet(mnemonic), false)
	faucetAmount := float64(10)
	apiClient.ExecuteFaucetWithTokens(t, wallet1, faucetAmount, client.TxSuccessfulStatus)
	balResp := apiClient.GetWalletBalance(t, wallet1, client.HttpOkStatus)
	require.EqualValues(t, zcncore.ConvertToValue(faucetAmount), balResp.Balance)
	wallet2 := apiClient.CreateWallet(t)
	fmt.Printf("%+v\n", wallet2)
	futureNonce := GetFutureNonceConfig(t)
	currentNonce := balResp.Nonce

	tokens := float64(1)
	value := int64(zcncore.ConvertToValue(tokens))

	// Add transactions with nonce + future nonce
	_, resp, err := apiClient.V1TransactionPutWithNonceAndServiceProviders(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet1,
			ToClientID: wallet2.Id,
			Value:      &value,
			TxnType:    client.SendTxType,
		},
		client.HttpBadRequestStatus,
		int(currentNonce)+futureNonce+1,
		nil,
	)

	// Expect error in transaction put
	require.NoError(t, err)
	require.Contains(t, string(resp.Body()), "invalid future transaction")
}

func TestClientSendSameNonceForDifferentTransactions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	wallet1, mnemonic := apiClient.CreateWalletWithMnemonicsInReturnValue(t)
	zcncore.SetWallet(*wallet1.ToZCNCryptoWallet(mnemonic), false)

	faucetAmount := float64(10)
	apiClient.ExecuteFaucetWithTokens(t, wallet1, faucetAmount, client.TxSuccessfulStatus)
	balResp := apiClient.GetWalletBalance(t, wallet1, client.HttpOkStatus)
	require.EqualValues(t, zcncore.ConvertToValue(faucetAmount), balResp.Balance)
	currentNonce := balResp.Nonce

	sameNonce := currentNonce + 2

	numSameTxns := 5
	wallets := make([]*model.Wallet, numSameTxns)
	hashes := make(map[string]string, numSameTxns) // clientID:hash
	transactions := make(map[string]struct{}, numSameTxns)
	value := int64(1)
	for i := 0; i < numSameTxns; i++ {
		wallets[i] = apiClient.CreateWallet(t)

		txnResp, _, err := apiClient.V1TransactionPutWithNonceAndServiceProviders(
			t,
			model.InternalTransactionPutRequest{
				Wallet:     wallet1,
				ToClientID: wallets[i].Id,
				Value:      &value,
				TxnType:    client.SendTxType,
			},
			client.HttpOkStatus,
			int(sameNonce),
			nil,
		)

		require.NoError(t, err)
		hashes[wallets[i].Id] = txnResp.Request.Hash
		transactions[txnResp.Request.Hash] = struct{}{}
	}

	require.GreaterOrEqual(t, len(apiClient.Miners), 1)
	// verify transactions are in txn pool
	txnsMap := GetTransactionsFromTxnPool(t, apiClient.Miners)
	txnsFromMap := GetTxnsMapFromGivenMapOfSlice(txnsMap)

	for txn := range transactions {
		_, ok := txnsFromMap[txn]
		require.True(t, ok)
	}

	wallet2 := apiClient.CreateWallet(t)
	txnResp, _, err := apiClient.V1TransactionPutWithNonceAndServiceProviders(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet1,
			ToClientID: wallet2.Id,
			Value:      &value,
			TxnType:    client.SendTxType,
		},
		client.HttpOkStatus,
		int(currentNonce+1),
		nil,
	)
	require.NoError(t, err)

	var confirmationResp *model.TransactionGetConfirmationResponse
	transactionTimeOut := GetTransactionTimeOut(t)
	tm := time.NewTimer(transactionTimeOut)
L1:
	for {
		select {
		case <-tm.C:
			break L1
		default:
		}
		confirmationResp, _, err = apiClient.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: txnResp.Request.Hash,
			},
			client.HttpOkStatus)
		if err == nil {
			break
		}
	}

	require.NoError(t, err)
	require.NotNil(t, confirmationResp)
	require.Equal(t, txnResp.Request.Hash, confirmationResp.Transaction.Hash)

	var putError []error

	for txn := range transactions {
		_, _, err := apiClient.V1TransactionGetConfirmation(
			t,
			model.TransactionGetConfirmationRequest{
				Hash: txn,
			},
			client.HttpOkStatus)
		if err != nil {
			putError = append(putError, err)
		}
	}

	require.Len(t, putError, len(transactions)-1)

	txnsMap = GetTransactionsFromTxnPool(t, nil)
	txnsFromMap = GetTxnsMapFromGivenMapOfSlice(txnsMap)
	for txn := range transactions {
		_, ok := txnsFromMap[txn]
		require.False(t, ok)
	}
}

func TestClientSendTransactionToOnlyOneMiner(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	wallet1, mnemonic := apiClient.CreateWalletWithMnemonicsInReturnValue(t)
	zcncore.SetWallet(*wallet1.ToZCNCryptoWallet(mnemonic), false)

	faucetAmount := 10
	apiClient.ExecuteFaucetWithTokens(t, wallet1, float64(faucetAmount), client.TxSuccessfulStatus)
	balResp := apiClient.GetWalletBalance(t, wallet1, client.HttpOkStatus)
	require.EqualValues(t, zcncore.ConvertToValue(float64(faucetAmount)), balResp.Balance)
	require.GreaterOrEqual(t, len(apiClient.Miners), 1)

	wallet2 := apiClient.CreateWallet(t)
	value := int64(1)
	miner := apiClient.Miners[0]
	txnResp, _, err := apiClient.V1TransactionPutWithNonceAndServiceProviders(
		t,
		model.InternalTransactionPutRequest{
			Wallet:     wallet1,
			ToClientID: wallet2.Id,
			Value:      &value,
			TxnType:    client.SendTxType,
		},
		client.HttpOkStatus,
		0,
		[]string{miner},
	)

	require.NoError(t, err)
	time.Sleep(time.Second * 10) // Wait little optimistic time for transaction to get into pool/get-confirmation
	txnsMap := GetTransactionsFromTxnPool(t, []string{miner})
	txnsFromMap := GetTxnsMapFromGivenMapOfSlice(txnsMap)
	var foundTransaction bool
	for hash := range txnsFromMap {
		if hash == txnResp.Request.Hash {
			foundTransaction = true
			break
		}
	}

	ch := make(chan struct{}, 1)
	if !foundTransaction {
		// We should be able to confirm the transaction
		// if it is not in transaction pool
		ch <- struct{}{}
	}

	transactionTimeOut := GetTransactionTimeOut(t)
	tm := time.NewTimer(transactionTimeOut)

	select {
	case <-ch:
	case <-tm.C:
	}

	confResp, _, err := apiClient.V1TransactionGetConfirmation(
		t,
		model.TransactionGetConfirmationRequest{
			Hash: txnResp.Request.Hash,
		},
		client.HttpOkStatus,
	)
	require.NoError(t, err)
	require.NotNil(t, confResp)
	require.Equal(t, txnResp.Request.Hash, confResp.Transaction.Hash)
}

func GetGlobalConfig(t *test.SystemTest) map[string]interface{} {
	cb := GlobalCB{
		Globals: make(map[string]interface{}),
		doneCh:  make(chan struct{}),
		errCh:   make(chan error),
	}

	err := zcncore.GetMinerSCGlobals(&cb)
	require.NoError(t, err)

	dur := time.Minute
	tm := time.NewTimer(dur)
	select {
	case <-cb.doneCh:
	case err := <-cb.errCh:
		t.Error(err)
	case <-tm.C:
		t.Errorf("Timeout occurred while waiting for global config after %v", dur)
	}

	return cb.Globals
}

func GetTransactionTimeOut(t *test.SystemTest) time.Duration {
	globalConfig := GetGlobalConfig(t)
	tmKey := "server_chain.transaction.timeout"
	i, ok := globalConfig[tmKey]
	require.True(t, ok)
	s := i.(string)
	tm, err := strconv.Atoi(s)
	require.NoError(t, err)
	return time.Duration(tm) * time.Second
}

func GetFutureNonceConfig(t *test.SystemTest) int {
	globalConfig := GetGlobalConfig(t)
	fnKey := "server_chain.transaction.future_nonce" // future nonce key
	i, ok := globalConfig[fnKey]
	require.True(t, ok)
	s := i.(string)
	n, err := strconv.Atoi(s)
	require.NoError(t, err)
	return n
}

type GlobalCB struct {
	Globals map[string]interface{}
	doneCh  chan struct{}
	errCh   chan error
}

func (g *GlobalCB) OnInfoAvailable(op int, status int, info string, err string) {
	if status == zcncore.StatusError {
		g.errCh <- errors.New(err)
		return
	}

	m := make(map[string]interface{})
	if e := json.Unmarshal([]byte(info), &m); e != nil {
		g.errCh <- e
		return
	}

	g.Globals = m["fields"].(map[string]interface{})
	g.doneCh <- struct{}{}
}

var (
	/* fields provide which fields of `td` tag to consider for getting data. So for example if table has element
	<tr>
		<td>Data1</td>
		<td>Data2</td>
		<td>Data3</td>
		<td>Data4</td>
	</tr>

	And if we only need Data1 and Data4 i.e. we only need 0th and 3rd field so fields will be:
	fields = map[int]string{0:"tag1", 3:"tag2"}
	where tag is struct tag which can be used to de-serialize data into.
	*/
	fields = map[int]string{
		0: "hash",
		1: "client_id",
	}
	diagnosticUrl = "_diagnostics/txns_in_pool"
)

func GetTransactionsFromTxnPool(t *test.SystemTest, miners []string) map[string][]map[string]string {
	// This should be made nil after its value is copied in returnMap as below.
	txns := make([]map[string]string, 0)

	c := colly.NewCollector()
	c.OnHTML("table.menu tbody", func(h *colly.HTMLElement) { // on html calls the defined function when it finds `tabl.menu body` element
		h.ForEach("tr", func(i int, h *colly.HTMLElement) {
			if i < 2 {
				return
			}

			txnMap := make(map[string]string)

			h.ForEach("td", func(i int, h *colly.HTMLElement) {
				field, ok := fields[i]
				if !ok {
					return
				}
				txnMap[field] = h.Text
				t.Logf("Extracting %s: %s", field, txnMap[field])
			})

			txns = append(txns, txnMap)
		})
	})

	rMap := make(map[string][]map[string]string) // return map
	for _, m := range miners {
		u := client.NewURLBuilder().SetPath(diagnosticUrl)
		err := u.MustShiftParse(m)
		require.NoError(t, err)
		txnPoolUrl := u.String()
		t.Log("Getting transactions from ", txnPoolUrl)
		err = c.Visit(txnPoolUrl)
		require.NoError(t, err)

		rMap[m] = make([]map[string]string, len(txns))
		copy(rMap[m], txns)
		txns = nil
	}

	return rMap
}

func GetTxnsMapFromGivenMapOfSlice(m map[string][]map[string]string) map[string]struct{} {
	txns := make(map[string]struct{})
	for _, s := range m {
		for _, txnMap := range s {
			txns[txnMap["hash"]] = struct{}{}
		}
	}
	return txns
}
