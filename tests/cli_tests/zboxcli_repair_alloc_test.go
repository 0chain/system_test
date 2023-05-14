package cli_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestAllocationRepair(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	t.Run("attempt file repair on single file that needs repaired", func(t *test.SystemTest) {
	})
	t.Run("attempt file repair on multiple files that needs repaired", func(t *test.SystemTest) {
	})
	t.Run("attempt file repair on file that does need repaired with a file that does not need repaired", func(t *test.SystemTest) {
	})
	t.Run("attempt file repair on file that does not exist", func(t *test.SystemTest) {
	})
	t.Run("attempt file repair using local path that does not exist", func(t *test.SystemTest) {
	})
	t.Run("don't supply repair path", func(t *test.SystemTest) {
	})
	t.Run("don't supply root path", func(t *test.SystemTest) {
	})
}
