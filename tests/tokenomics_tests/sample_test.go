package tokenomics_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	"testing"

	"github.com/0chain/system_test/tests/cli_tests"
)

func TestBlockRewardsForBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	prevBlock := cli_tests.GetLatestFinalizedBlock(t)

	fmt.Println("prevBlock", prevBlock)

}
