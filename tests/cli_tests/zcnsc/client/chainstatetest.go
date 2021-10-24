package client

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/zcncore"
)

const (
	StateErrorTest = "state_error_test"
	StateErrorTest2 = "state_error_test2"
	StateErrorTestDelete = "state_error_test_delete"
)

// Contains debugging method in different smart contracts

func TestStorageSc() zcncore.TransactionScheme {
	fmt.Println("----------------------------------------------")
	fmt.Println("Started executing smart contract TestStorageSc...")
	status := NewZCNStatus()
	txn, err := zcncore.NewTransaction(status, 0)
	if err != nil {
		ExitWithError(err)
	}

	status.Begin()

	err = txn.ExecuteSmartContract(StorageAddress, StateErrorTest, "", zcncore.ConvertToValue(1))
	if err != nil {
		fmt.Printf("Transaction failed with error: '%s'", err.Error())
		return nil
	}

	status.Wait()
	fmt.Printf("Executed smart contract TestStorageSc with TX = '%s'\n", txn.GetTransactionHash())

	VerifyTransaction(txn, status)

	return txn
}

func TestZcnscScWithoutPayload() zcncore.TransactionScheme {
	fmt.Println("----------------------------------------------")
	fmt.Println("Started executing smart contract TestZcnscScWithoutPayload...")
	status := NewZCNStatus()
	txn, err := zcncore.NewTransaction(status, 0)
	if err != nil {
		ExitWithError(err)
	}

	payload := &AuthorizerNode{
		PublicKey: "PublicKey",
		URL:       "localhost",
	}

	buffer, _ := json.Marshal(payload)

	status.Begin()
	err = txn.ExecuteSmartContract(ZcnscAddress, StateErrorTest, string(buffer), zcncore.ConvertToValue(1))
	if err != nil {
		fmt.Printf("Transaction failed with error: '%s'", err.Error())
		return nil
	}

	status.Wait()
	fmt.Printf("Executed smart contract TestZcnscScWithoutPayload with TX = '%s'\n", txn.GetTransactionHash())

	VerifyTransaction(txn, status)

	return txn
}

func TestZcnscScWithPayload() zcncore.TransactionScheme {
	fmt.Println("----------------------------------------------")
	fmt.Println("Started executing smart contract TestZcnscScWithPayload...")
	status := NewZCNStatus()
	txn, err := zcncore.NewTransaction(status, 0)
	if err != nil {
		ExitWithError(err)
	}

	payload := &AuthorizerNode{
		PublicKey: "PublicKey",
		URL:       "localhost",
	}

	buffer, _ := json.Marshal(payload)

	status.Begin()
	err = txn.ExecuteSmartContract(ZcnscAddress, StateErrorTest2, string(buffer), zcncore.ConvertToValue(1))
	if err != nil {
		fmt.Printf("Transaction failed with error: '%s'", err.Error())
		return nil
	}

	status.Wait()
	fmt.Printf("Executed smart contract TestZcnscScWithPayload with TX = '%s'\n", txn.GetTransactionHash())

	VerifyTransaction(txn, status)

	return txn
}

func TestDeleteAuthorizer() zcncore.TransactionScheme {
	fmt.Println("----------------------------------------------")
	fmt.Println("Started executing smart contract TestDeleteAuthorizer...")
	status := NewZCNStatus()
	txn, err := zcncore.NewTransaction(status, 0)
	if err != nil {
		ExitWithError(err)
	}

	status.Begin()
	err = txn.ExecuteSmartContract(ZcnscAddress, StateErrorTestDelete, "", zcncore.ConvertToValue(1))
	if err != nil {
		fmt.Printf("Transaction failed with error: '%s'", err.Error())
		return nil
	}

	status.Wait()
	fmt.Printf("Executed smart contract TestDeleteAuthorizer with TX = '%s'\n", txn.GetTransactionHash())

	VerifyTransaction(txn, status)

	return txn
}
