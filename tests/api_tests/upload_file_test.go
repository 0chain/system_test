package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/endpoint"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFileUpload(t *testing.T) {
	t.Run("File upload API call should be successful given a valid request", func(t *testing.T) {
		t.Parallel()

		registeredWallet, keyPair := registerWallet(t)
		executeFaucet(t, registeredWallet, keyPair)
		blobbers, blobberRequirements := getBlobbersMatchingRequirements(t, registeredWallet, keyPair, 10000, 1, 1, time.Minute*20)
		blobberRequirements.Blobbers = blobbers
		transactionResponse, confirmation := createAllocation(t, registeredWallet, keyPair, blobberRequirements)
		require.Equal(t, endpoint.TxSuccessfulStatus, confirmation.Status, confirmation.Transaction.TransactionOutput)
		allocation := getAllocation(t, transactionResponse.Entity.Hash)
		require.NotNil(t, allocation)

	})
}
