package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSystemTestWrapper(testSetup *testing.T) {
	t := &test.SystemTest{T: testSetup}

	t.Skip("Temporarily added to test system tests wrapper")

	t.Parallel()

	t.Run("Test fail", func(t *test.SystemTest) {
		t.Parallel()

		require.Nil(t, "not nil")
	})

	t.Run("Test success", func(t *test.SystemTest) {
		t.Parallel()

		require.Nil(t, nil)
	})

	t.Run("Test timeout", func(t *test.SystemTest) {
		t.Parallel()

		time.Sleep(30 * time.Second)
	})

	t.Run("Test panic", func(t *test.SystemTest) {
		t.Parallel()

		panic("panic!")
	})
}
