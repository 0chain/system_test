package cli_tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// cmd: bridge-client-init
func TestBridgeClientInit(t *testing.T) {
	t.Parallel()

	t.Run("Init bridge client config", func(t *testing.T) {
		t.Parallel()

		output, err := bridgeClientInit(t,
			"password",
			"0x860FA46F170a87dF44D7bB867AA4a5D2813127c1",
			"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
			"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
			"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
			0.75,
			300000,
			0,
		)

		require.Nil(t, err, "error trying to create an initial client bridge config", strings.Join(output, "\n"))
		require.Equal(t, "config written to bridge.yaml", output[len(output)-1])
	})
}

// cmd: bridge-owner-init
func TestBridgeOwnerInit(t *testing.T) {
	t.Parallel()

	t.Run("Init bridge owner config", func(t *testing.T) {
		t.Parallel()

		output, err := bridgeOwnerInit(
			t,
			"password",
			"0x860FA46F170a87dF44D7bB867AA4a5D2813127c1",
			"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
			"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
			"0xFE20Ce9fBe514397427d20C91CB657a4478A0FFa",
			"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
			300000,
			0,
		)

		require.Nil(t, err, "error trying to create an initial owner bridge config", strings.Join(output, "\n"))
		require.Equal(t, "config written to owner.yaml", output[len(output)-1])
	})
}

// cmd: bridge-owner-init
func bridgeOwnerInit(
	t *testing.T,
	password, ethereumaddress, bridgeaddress, wzcnaddress, authorizersaddress, ethereumnodeurl string,
	gaslimit, value int64,
) ([]string, error) {
	t.Logf("Init bridge owner config (owner.yaml) in HOME (~/.zcn) folder")

	cmd := "./zwallet bridge-owner-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --authorizersaddress " + authorizersaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// cmd: bridge-client-init
func bridgeClientInit(
	t *testing.T,
	password, ethereumaddress, bridgeaddress, wzcnaddress, ethereumnodeurl string,
	consensusthreshold float64,
	gaslimit, value int64,
) ([]string, error) {
	t.Logf("Init bridge client config (bridge.yaml) in HOME (~/.zcn) folder")

	cmd := "./zwallet bridge-client-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --consensusthreshold " + fmt.Sprintf("%.4f", consensusthreshold) +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}
