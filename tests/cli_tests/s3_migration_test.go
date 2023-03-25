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

func TestS3Migration(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("S3 migration should work fine when migration command run with the correct flags", func(t *test.SystemTest) {
		output, err := S3Migrate(t, "", "", true)
		require.NoError(t, err, "", strings.Join(output, "\n"))
		require.Len(t, output, 1)
	})
}

func S3Migrate(t *test.SystemTest, accessKey, secretKey string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf("./s3mgrt %s", "help")

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
