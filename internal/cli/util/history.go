package cliutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
)

const (
	MaxQueryLimit    = 20
	StorageScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

type ChainHistory struct {
	from, to int64
	blocks   []model.EventDbBlock
}

func NewHistory(from, to int64) *ChainHistory {
	return &ChainHistory{
		from: from,
		to:   to,
	}
}

func (ch *ChainHistory) TimesWonBestMiner(minerId string) int64 {
	var won int64
	for _, block := range ch.blocks {
		if minerId == block.MinerID {
			won++
		}
	}
	return won
}

func (ch *ChainHistory) TotalFees() int64 {
	var fees int64
	for _, block := range ch.blocks {
		fees += ch.TotalBlockFees(block)
	}
	return fees
}

func (ch *ChainHistory) TotalMinerFees(minerId string) int64 {
	var fees int64
	for _, block := range ch.blocks {
		if block.MinerID == minerId {
			fees += ch.TotalBlockFees(block)
		}
	}
	return fees
}

func (ch *ChainHistory) TotalBlockFees(block model.EventDbBlock) int64 {
	var fees int64
	for _, tx := range block.Transactions {
		fees += tx.Fee
	}
	return fees
}

func apiGetBlocks(start, end, limit, offset int64, sharderBaseURL string) (*http.Response, error) {
	url := fmt.Sprintf(sharderBaseURL+"/v1/screst/"+StorageScAddress+
		"/get_blocks?content=full&start=%d&end=%d", start, end)
	if limit > 0 || offset > 0 {
		url += fmt.Sprintf("&limit=%d&offset=%d", limit, offset)
	}
	return http.Get(url)
}

func (ch *ChainHistory) ReadBlocks(t *testing.T, sharderBaseUrl string) {
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

func getBlocks(t *testing.T, from, to, limit, offset int64, sharderBaseUrl string) []model.EventDbBlock {
	res, err := apiGetBlocks(from, to, limit, offset, sharderBaseUrl)
	require.NoError(t, err, "retrieving blocks %d to %d", from, to)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request to get blocks %d to %d, status code: %d", from, to, res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var blocks []model.EventDbBlock
	err = json.Unmarshal(resBody, &blocks)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)

	return blocks
}

// debug dumps

func (ch *ChainHistory) DumpTransactions() {
	for _, block := range ch.blocks {
		for _, tx := range block.Transactions {
			fmt.Println("tx", "round", tx.Round, "fees", tx.Fee, "data", tx.TransactionData, "miner id", block.MinerID)
		}
	}
}

func (ch *ChainHistory) AccountingMiner(id string) {
	fmt.Println("-------------", "accounts for", id, "-------------")
	for _, block := range ch.blocks {
		if id == block.MinerID {
			ch.AccountingMinerBlock(id, block)
		}
	}
}

func (ch *ChainHistory) AccountingMinerBlock(id string, block model.EventDbBlock) {
	if id != block.MinerID {
		return
	}
	for _, tx := range block.Transactions {
		if tx.Fee > 0 {
			fmt.Println("round", block.Round, "fee", tx.Fee, "data", tx.TransactionData)
		}
	}
}
