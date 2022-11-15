package cliutils

import (
	"fmt"
	"testing"

	"github.com/0chain/system_test/internal/cli/model"
)

const (
	MaxQueryLimit    = 20
	StorageScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

type ChainHistory struct {
	from, to int64
	blocks   []model.EventDBBlock
}

func NewHistory(from, to int64) *ChainHistory {
	return &ChainHistory{
		from: from,
		to:   to,
	}
}

func (ch *ChainHistory) TimesWonBestMiner(minerId string) int64 {
	var won int64
	for i := range ch.blocks {
		if minerId == ch.blocks[i].MinerID {
			won++
		}
	}
	return won
}

func (ch *ChainHistory) TotalFees() int64 {
	var fees int64
	for i := range ch.blocks {
		fees += ch.TotalBlockFees(&ch.blocks[i])
	}
	return fees
}

func (ch *ChainHistory) TotalMinerFees(minerId string) int64 {
	var fees int64
	for i := range ch.blocks {
		if ch.blocks[i].MinerID == minerId {
			fees += ch.TotalBlockFees(&ch.blocks[i])
		}
	}
	return fees
}

func (ch *ChainHistory) TotalBlockFees(block *model.EventDBBlock) int64 {
	var fees int64
	for i := range block.Transactions {
		fees += block.Transactions[i].Fee
	}
	return fees
}

func (ch *ChainHistory) ReadBlocks(t *testing.T, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/get_blocks")
	params := map[string]string{
		"contents": "full",
	}
	ch.blocks = ApiGetList[model.EventDBBlock](t, url, params, ch.from, ch.to)
}

// debug dumps

func (ch *ChainHistory) DumpTransactions() {
	for i := range ch.blocks {
		for j := range ch.blocks[i].Transactions {
			tx := &ch.blocks[i].Transactions[j]
			_, _ = fmt.Println("tx", "round", tx.Round, "fees", tx.Fee, "data", tx.TransactionData, "miner id", ch.blocks[i].MinerID)
		}
	}
}

func (ch *ChainHistory) AccountingMiner(id string) {
	_, _ = fmt.Println("-------------", "accounts for", id, "-------------")
	for i := range ch.blocks {
		if id == ch.blocks[i].MinerID {
			ch.AccountingMinerBlock(id, &ch.blocks[i])
		}
	}
}

func (ch *ChainHistory) AccountingMinerBlock(id string, block *model.EventDBBlock) {
	if id != block.MinerID {
		return
	}
	for i := range block.Transactions {
		tx := &block.Transactions[i]
		if tx.Fee > 0 {
			_, _ = fmt.Println("round", block.Round, "fee", tx.Fee, "data", tx.TransactionData)
		}
	}
}
