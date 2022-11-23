package cliutils

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
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

func apiGetBlocks(start, end, limit, offset int64, sharderBaseURL string) (*http.Response, error) {
	baseUrl := sharderBaseURL + "/v1/screst/" + StorageScAddress + "/get_blocks"
	query := fmt.Sprintf("?content=full&start=%d&end=%d", start, end)
	if limit > 0 || offset > 0 {
		query += fmt.Sprintf("&limit=%d&offset=%d", limit, offset)
	}
	return http.Get(baseUrl + query)
}

func (ch *ChainHistory) ReadBlocks(t *test.SystemTest, sharderBaseUrl string) {
	var offset int64
	for {
		blocks := getBlocks(t, ch.from, ch.to, MaxQueryLimit, offset, sharderBaseUrl)
		offset += int64(len(blocks))
		ch.blocks = append(ch.blocks, blocks...)
		if len(blocks) < MaxQueryLimit {
			break
		}
	}
}

func getBlocks(t *test.SystemTest, from, to, limit, offset int64, sharderBaseUrl string) []model.EventDBBlock {
	res, err := apiGetBlocks(from, to, limit, offset, sharderBaseUrl)
	require.NoError(t, err, "retrieving blocks %d to %d", from, to)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request to get blocks %d to %d, status code: %d", from, to, res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var blocks []model.EventDBBlock
	err = json.Unmarshal(resBody, &blocks)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)

	return blocks
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
