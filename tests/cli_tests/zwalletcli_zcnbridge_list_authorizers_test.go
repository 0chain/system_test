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

	t.Run("List of authorizers", func(t *testing.T) {
		t.Parallel()

		output, err := zwalletListAuthCLI("bridge-list-auth")

		require.Nil(t, err, "error trying to get the list of authorizers", strings.Join(output, "\n"))
		t.Log(output)
	})
}

func zwalletListAuthCLI(cmd string) ([]string, error) {
	run := fmt.Sprintf("./zwallet %s", cmd)
	return cliutils.RunCommandWithoutRetry(run)
}
