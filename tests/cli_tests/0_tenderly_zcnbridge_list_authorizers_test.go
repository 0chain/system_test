package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0TenderlyListAuthorizers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("List authorizers should work")

	t.Parallel()

	time.Sleep(20 * time.Second)

	t.Run("List authorizers should work", func(t *test.SystemTest) {
		output, err := getAuthorizersList(t, true)

		require.Nil(t, err, "error trying to get the list of authorizers", strings.Join(output, "\n"))
	})
}

// nolint
func getAuthorizersList(t *test.SystemTest, retry bool) ([]string, error) {
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
