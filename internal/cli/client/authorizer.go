package client

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/zcncore"
	"github.com/go-resty/resty/v2"
	"sync"
	"time"
)

const (
	burnTicketPath = "/v1/0chain/burnticket/get"
)

func SendTicketsToEthereumBridge() {
	fmt.Println("Sending tickets to Ethereum bridge")
}

func GetBurnProofTickets(authorizers []string, hash string) {
	// Create a Resty Client
	httpClient := resty.New()
	httpClient.SetTimeout(time.Second * 3)

	ch := make(chan *ProofOfBurn)
	done := make(chan bool)
	var tickets []*ProofOfBurn

	wg := sync.WaitGroup{}

	for _, url := range authorizers {
		wg.Add(1)
		authUrl := url + burnTicketPath
		go getTicketFromAuthorizer(hash, authUrl, ch, httpClient.R(), &wg)
	}

	go func(done chan<- bool) {
		for ticket := range ch {
			fmt.Printf("Found burn ticket: %v\n", ticket)
			tickets = append(tickets, ticket)
		}
		done <- true
	}(done)

	wg.Wait()
	close(ch)
	<-done

	fmt.Printf("Received %d tickets from authorizers", len(tickets))
}

func getTicketFromAuthorizer(hash string, url string, ch chan *ProofOfBurn, rest *resty.Request, wg *sync.WaitGroup) {
	defer wg.Done()
	proof := &ProofOfBurn{}

	resp, err := rest.
		SetFormData(map[string]string{
			"hash": hash,
		}).
		Post(url)

	if err == nil {
		body := resp.Body()
		err = json.Unmarshal(body, proof)
		ch <- proof
		if err != nil {
			fmt.Printf("Error to unmarhall body: %v\n", body)
		}
	}
}

// RegisterAuthorizer Test function. Missing wallet initialization prior to register the authorizer
func RegisterAuthorizer(url string) zcncore.TransactionScheme {
	fmt.Println("---------------------------")
	fmt.Println("Started Registering an authorizer...")
	status := NewZCNStatus()
	txn, err := zcncore.NewTransaction(status, 0)
	if err != nil {
		ExitWithError(err)
	}

	payload := &AuthorizerNode{
		PublicKey: "public key",
		URL:       url,
	}

	buffer, _ := json.Marshal(payload)

	fmt.Printf("Payload: AuthorizerNode: %s\n", buffer)

	status.Begin()
	err = txn.ExecuteSmartContract(
		ZcnscAddress,
		AddAuthorizerMethod,
		string(buffer),
		zcncore.ConvertToValue(3),
	)

	if err != nil {
		fmt.Printf("Transaction failed with error: '%s'\n", err.Error())
		return nil
	}

	status.Wait()
	fmt.Printf("Executed smart contract ZCNSC:AddAuthorizer with TX = '%s'\n", txn.GetTransactionHash())

	VerifyTransaction(txn, status)

	return txn
}