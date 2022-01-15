package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestListAuthorizers(t *testing.T) {
	t.Parallel()

	const (
		Help = "Getting  list of authorizers"
	)

	var zwallet = func(cmd string) ([]string, error) {
		t.Logf(Help)
		run := fmt.Sprintf("./zwallet %s", cmd)
		return cliutils.RunCommandWithoutRetry(run)
	}

	t.Run("List of authorizers", func(t *testing.T) {
		t.Parallel()

		output, err := zwallet("bridge-list-auth")

		require.Nil(t, err, "error trying to get the list of authorizers", strings.Join(output, "\n"))
		t.Log(output)
	})
}
