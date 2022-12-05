package cliutils

import (
	"fmt"
	"strconv"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/cli/model"
)

const (
	MaxQueryLimit    = 20
	StorageScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	MinerScAddress   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
)

type ChainHistory struct {
	from, to        int64
	blocks          []model.EventDBBlock
	delegateRewards []model.RewardDelegate
	providerRewards []model.RewardProvider
	roundHistories  map[int64]RoundHistory
}

type RoundHistory struct {
	Block           model.EventDBBlock
	DelegateRewards []model.RewardDelegate
	ProviderRewards []model.RewardProvider
}

func NewHistory(from, to int64) *ChainHistory {
	return &ChainHistory{
		from: from,
		to:   to,
	}
}

func (ch *ChainHistory) RoundHistory(t *test.SystemTest, round int64) RoundHistory {
	require.NotNil(t, ch.roundHistories, "requestingB round history")
	require.Len(t, ch.roundHistories, int(round), "requested round in histories")
	return ch.roundHistories[round]
}

func (ch *ChainHistory) From() int64 {
	return ch.from
}

func (ch *ChainHistory) To() int64 {
	return ch.to
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

func (ch *ChainHistory) Read(t *test.SystemTest, sharderBaseUrl string) {
	ch.readBlocks(t, sharderBaseUrl)
	ch.readDelegateRewards(t, sharderBaseUrl)
	ch.readProviderRewards(t, sharderBaseUrl)
	ch.organise(t)
}

func (ch *ChainHistory) readBlocks(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/get_blocks")
	params := map[string]string{
		"contents": "full",
	}
	ch.blocks = ApiGetList[model.EventDBBlock](t, url, params, ch.from, ch.to)
}

func (ch *ChainHistory) readDelegateRewards(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + MinerScAddress + "/delegate-rewards")
	params := map[string]string{
		"start": strconv.FormatInt(ch.from, 10),
		"end":   strconv.FormatInt(ch.to, 10),
	}
	ch.delegateRewards = ApiGetList[model.RewardDelegate](t, url, params, ch.from, ch.to)
}

func (ch *ChainHistory) readProviderRewards(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + MinerScAddress + "/provider-rewards")
	params := map[string]string{
		"start": strconv.FormatInt(ch.from, 10),
		"end":   strconv.FormatInt(ch.to, 10),
	}
	ch.providerRewards = ApiGetList[model.RewardProvider](t, url, params, ch.from, ch.to)
}

func (ch *ChainHistory) organise(t *test.SystemTest) {
	ch.roundHistories = make(map[int64]RoundHistory, ch.to-ch.from)

	for _, bk := range ch.blocks {
		ch.roundHistories[bk.Round] = RoundHistory{
			Block: bk,
		}
	}

	var currentRound int64
	var currentHistory RoundHistory
	for _, pr := range ch.providerRewards {
		require.True(t, pr.BlockNumber >= currentRound, "provider rewards out of order")
		if currentRound < pr.BlockNumber {
			if currentRound > 0 {
				ch.roundHistories[currentRound] = currentHistory
			}
			var ok bool
			currentHistory, ok = ch.roundHistories[pr.BlockNumber]
			require.True(t, ok, "should have block information for provider rewards")
			currentRound = pr.BlockNumber
		}
		currentHistory.ProviderRewards = append(currentHistory.ProviderRewards, pr)
	}
	if currentRound > 0 {
		ch.roundHistories[currentRound] = currentHistory
	}

	currentRound = 0
	currentHistory = RoundHistory{}
	for _, dr := range ch.delegateRewards {
		require.GreaterOrEqual(t, dr.BlockNumber, currentRound, "delegate rewards out of order")
		if currentRound < dr.BlockNumber {
			if currentRound > 0 {
				ch.roundHistories[currentRound] = currentHistory
			}
			var ok bool
			currentHistory, ok = ch.roundHistories[dr.BlockNumber]
			require.True(t, ok, "should have block information for provider rewards")
			currentRound = dr.BlockNumber
		}
		currentHistory.DelegateRewards = append(currentHistory.DelegateRewards, dr)
	}
	if currentRound > 0 {
		ch.roundHistories[currentRound] = currentHistory
	}

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
