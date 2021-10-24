package client

import (
	"fmt"
	"os"
	"sync"

	"github.com/0chain/gosdk/zcncore"
	"gopkg.in/cheggaaa/pb.v1"
)

type StatusBar struct {
	b  *pb.ProgressBar
	wg *sync.WaitGroup
}

type ZCNStatus struct {
	walletString string
	wg           *sync.WaitGroup
	success      bool
	errMsg       string
	balance      int64
	wallets      []string
	clientID     string
}

func NewZCNStatus() (zcns *ZCNStatus) {
	return &ZCNStatus{wg: new(sync.WaitGroup)}
}

func (zcns *ZCNStatus) Begin() { zcns.wg.Add(1) }
func (zcns *ZCNStatus) Wait()  { zcns.wg.Wait() }

func (zcns *ZCNStatus) OnBalanceAvailable(status int, value int64, info string) {
	defer zcns.wg.Done()
	if status == zcncore.StatusSuccess {
		zcns.success = true
	} else {
		zcns.success = false
	}
	zcns.balance = value
}

func (zcns *ZCNStatus) OnTransactionComplete(t *zcncore.Transaction, status int) {
	defer zcns.wg.Done()
	if status == zcncore.StatusSuccess {
		zcns.success = true
	} else {
		zcns.errMsg = t.GetTransactionError()
	}
	// fmt.Println("Txn Hash:", t.GetTransactionHash())
}

func (zcns *ZCNStatus) OnVerifyComplete(t *zcncore.Transaction, status int) {
	defer zcns.wg.Done()
	if status == zcncore.StatusSuccess {
		zcns.success = true
	} else {
		zcns.errMsg = t.GetVerifyError()
	}
	// fmt.Println(t.GetVerifyOutput())
}

func (zcns *ZCNStatus) OnAuthComplete(t *zcncore.Transaction, status int) {
	fmt.Printf("Authorization completed on zauth with status=%d, TRX=%s\n", status, t.GetTransactionHash())
}

func (zcns *ZCNStatus) OnWalletCreateComplete(status int, wallet string, err string) {
	defer zcns.wg.Done()
	if status != zcncore.StatusSuccess {
		zcns.success = false
		zcns.errMsg = err
		zcns.walletString = ""
		return
	}
	zcns.success = true
	zcns.errMsg = ""
	zcns.walletString = wallet
}

func (zcns *ZCNStatus) OnInfoAvailable(_ int, status int, config string, err string) {
	defer zcns.wg.Done()
	if status != zcncore.StatusSuccess {
		zcns.success = false
		zcns.errMsg = err
		return
	}
	zcns.success = true
	zcns.errMsg = config
}

func (zcns *ZCNStatus) OnSetupComplete(status int, err string) {
	defer zcns.wg.Done()
}

func (zcns *ZCNStatus) OnAuthorizeSendComplete(status int, _ string, val int64, desc string, creationDate int64, signature string) {
	defer zcns.wg.Done()
	fmt.Println("Status:", status)
	fmt.Println("Timestamp:", creationDate)
	fmt.Println("Signature:", signature)
}

// OnVoteComplete callback when a multisig vote is completed
func (zcns *ZCNStatus) OnVoteComplete(status int, proposal string, err string) {
	defer zcns.wg.Done()
	if status != zcncore.StatusSuccess {
		zcns.success = false
		zcns.errMsg = err
		zcns.walletString = ""
		return
	}
	zcns.success = true
	zcns.errMsg = ""
	zcns.walletString = proposal
}

func PrintError(v ...interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, v...)
}

func ExitWithError(v ...interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}