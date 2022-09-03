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

const storageScAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"

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

func (ch *ChainHistory) DumpTransactions() {
	for _, block := range ch.blocks {
		for _, tx := range block.Transactions {
			fmt.Println("tx", "round", tx.Round, "fees", tx.Fee, "data", tx.TransactionData, "miner id", block.MinerID)
		}
	}
}

func (ch *ChainHistory) TimesWonBestMiner(minerId string, start, end int64) int64 {
	var won int64
	for _, block := range ch.blocks {
		if block.Round < start || block.Round >= end {
			continue
		}
		if minerId == block.MinerID {
			won++
		}
	}
	return won
}

func (ch *ChainHistory) TotalFees(start, end int64) int64 {
	var fees int64
	for _, block := range ch.blocks {
		if block.Round < start || block.Round >= end {
			continue
		}
		fees += ch.TotalBlockFees(block)
	}
	return fees
}

func (ch *ChainHistory) TotalMinerFees(minerId string, start, end int64) int64 {
	var fees int64
	for _, block := range ch.blocks {
		if block.Round < start || block.Round >= end {
			continue
		}
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

// localhost/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks?content=full&end=10&start=1
func apiGetBlocks(start, end int64, sharderBaseURL string) (*http.Response, error) {
	url := fmt.Sprintf(sharderBaseURL+"/v1/screst/"+storageScAddress+
		"/get_blocks?content=full&start=%d&end=%d&limit=%d", start, end, end-start)
	fmt.Println("url", url)
	return http.Get(url)
}

func getBlocks(t *testing.T, from, to int64, sharderBaseUrl string) []model.EventDbBlock {
	res, err := apiGetBlocks(from, to, sharderBaseUrl)
	require.NoError(t, err, "retrieving blocks %d to %d", from, to)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"gailed API request to get blocks %d to %d, status code: %d", from, to, res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var blocks []model.EventDbBlock
	err = json.Unmarshal(resBody, &blocks)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)

	return blocks
}

func (ch *ChainHistory) ReadBlocks(t *testing.T, sharderBaseUrl string) {
	for current := ch.from; current < ch.to; {
		blocks := getBlocks(t, current, ch.to, sharderBaseUrl)
		ch.blocks = append(ch.blocks, blocks...)
		current = ch.blocks[len(ch.blocks)-1].Round
		current++
	}
}
