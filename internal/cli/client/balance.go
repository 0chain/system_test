package client

import (
	"fmt"
	"github.com/0chain/gosdk/zcncore"
)

func CheckBalance() int64 {
	fmt.Println("---------------------------")
	fmt.Println("Started Checking balance...")
	balance := NewZCNStatus()
	balance.Begin()
	err := zcncore.GetBalance(balance)
	if err == nil {
		balance.Wait()
		fmt.Printf("Client balance: %f\n", zcncore.ConvertToToken(balance.balance))
		return balance.balance
	} else {
		fmt.Println("Failed to get the balance: " + err.Error())
		return -1
	}
}
