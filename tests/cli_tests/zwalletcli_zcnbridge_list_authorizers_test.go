package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestListAuthorizers(t *testing.T) {
	t.Parallel()

	t.Run("List authorizers should work", func(t *testing.T) {
		t.Skip("Temporarily skipping due to deployment issue")
		t.Parallel()

		output, err := getAuthorizersList(t, true)
		require.Nil(t, err, "error trying to get the list of authorizers", strings.Join(output, "\n"))
	})
}

func getAuthorizersList(t *testing.T, retry bool) ([]string, error) {
	t.Logf("Getting  list of authorizers...")
	cmd := fmt.Sprintf(
		"./zwallet bridge-list-auth --silent "+
			"--configDir ./config --config %s",
		configPath,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
