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
	DelegateRewards []model.RewardDelegate
	providerRewards []model.RewardProvider
	transactions    []model.EventDBTransaction
	roundHistories  map[int64]RoundHistory
}

type RoundHistory struct {
	Block           *model.EventDBBlock
	DelegateRewards []model.RewardDelegate
	ProviderRewards []model.RewardProvider
	Transactions    []model.EventDBTransaction
}

func NewHistory(from, to int64) *ChainHistory {
	return &ChainHistory{
		from: from,
		to:   to,
	}
}

func (ch *ChainHistory) RoundHistory(t *test.SystemTest, round int64) RoundHistory {
	require.NotNil(t, ch.roundHistories, "round histories' nil, expected to be not nil"+
		" histories for round %v not found", round)

	rh, found := ch.roundHistories[round]
	if !found {
		require.True(t, found, "requested round %d in histories from %d to %d", round, ch.from, ch.to)
	}
	return rh
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

func (ch *ChainHistory) FeesForRound(t *test.SystemTest, round int64) int64 {
	roundHistory, ok := ch.roundHistories[round]
	require.Truef(t, ok, "cannot find history for round %d", round)
	var fee int64
	for i := 0; i < len(roundHistory.Transactions); i++ {
		fee += roundHistory.Transactions[i].Fee
	}
	return fee
}

func (ch *ChainHistory) Read(t *test.SystemTest, sharderBaseUrl string, includeTransactions bool) {
	ch.readBlocks(t, sharderBaseUrl)
	ch.readDelegateRewards(t, sharderBaseUrl)
	ch.readProviderRewards(t, sharderBaseUrl)
	if includeTransactions {
		ch.readTransaction(t, sharderBaseUrl)
	}
	ch.setup(t)
}

func (ch *ChainHistory) readBlocks(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/get_blocks")
	params := map[string]string{
		"contents": "full",
	}
	ch.blocks = ApiGetList[model.EventDBBlock](t, url, params, ch.from, ch.to+1)
}

func (ch *ChainHistory) readDelegateRewards(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + MinerScAddress + "/delegate-rewards")
	params := map[string]string{
		"start": strconv.FormatInt(ch.from, 10),
		"end":   strconv.FormatInt(ch.to+1, 10),
	}
	ch.DelegateRewards = ApiGetList[model.RewardDelegate](t, url, params, ch.from, ch.to+1)
}

func (ch *ChainHistory) readProviderRewards(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + MinerScAddress + "/provider-rewards")
	params := map[string]string{
		"start": strconv.FormatInt(ch.from, 10),
		"end":   strconv.FormatInt(ch.to+1, 10),
	}
	ch.providerRewards = ApiGetList[model.RewardProvider](t, url, params, ch.from, ch.to+1)
}

func (ch *ChainHistory) readTransaction(t *test.SystemTest, sharderBaseUrl string) {
	url := fmt.Sprintf(sharderBaseUrl + "/v1/screst/" + StorageScAddress + "/transactions")
	params := map[string]string{}
	ch.transactions = ApiGetList[model.EventDBTransaction](t, url, params, ch.from, ch.to+1)
}

func (ch *ChainHistory) setup(t *test.SystemTest) { // nolint:
	ch.roundHistories = make(map[int64]RoundHistory, ch.to-ch.from+1)

	for i := range ch.blocks {
		ch.roundHistories[ch.blocks[i].Round] = RoundHistory{
			Block: &ch.blocks[i],
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
	for _, dr := range ch.DelegateRewards {
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
	ch.setupTransactions(t)

	require.Equalf(t, int(ch.to-ch.from+1), len(ch.roundHistories),
		"mismatched round count recorded, from %d, to %d", ch.to, ch.from)
}

func (ch *ChainHistory) setupTransactions(t *test.SystemTest) {
	var currentRound int64 = 0
	var currentHistory RoundHistory
	for i := 0; i < len(ch.transactions); i++ {
		require.GreaterOrEqual(t, ch.transactions[i].Round, currentRound, "delegate rewards out of order")
		if currentRound < ch.transactions[i].Round {
			if currentRound > 0 {
				ch.roundHistories[currentRound] = currentHistory
			}
			var ok bool
			currentHistory, ok = ch.roundHistories[ch.transactions[i].Round]
			require.True(t, ok, "should have block information for every round")
			currentRound = ch.transactions[i].Round
		}
		currentHistory.Transactions = append(currentHistory.Transactions, ch.transactions[i])
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
