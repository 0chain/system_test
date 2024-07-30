package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/stretchr/testify/require"
	"net/http"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func ListBlobbers(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func ListValidators(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting validator list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-validators %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func UpdateBlobberInfoForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string) ([]string, error) {
	t.Log("Updating blobber info...", wallet)
	wallet = "wallets/blobber_owner"
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func StakeTokens(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return StakeTokensForWallet(t, cliConfigFilename, EscapedTestName(t), params, retry)
}

func StakePoolInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Fetching stake pool info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func GetBlobberAuthTicket(t *test.SystemTest, blobberID, blobberUrl, zboxTeamWallet, clientID string) (string, error) {
	zboxWallet, err := GetWalletForName(t, configPath, zboxTeamWallet)
	require.Nil(t, err, "could not get zbox wallet")

	var authTicket string
	signatureScheme := zcncrypto.NewSignatureScheme("bls0chain")
	_ = signatureScheme.SetPrivateKey("85e2119f494cd40ca524f6342e8bdb7bef2af03fe9a08c8d9c1d9f14d6c64f14")
	_ = signatureScheme.SetPublicKey(zboxWallet.ClientPublicKey)

	signature, err := signatureScheme.Sign(hex.EncodeToString([]byte(zboxWallet.ClientPublicKey)))
	if err != nil {
		return authTicket, err
	}

	url := blobberUrl + "/v1/auth/generate?client_id=" + clientID
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return authTicket, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zbox-Signature", signature)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return authTicket, err
	}
	defer resp.Body.Close()
	var responseMap map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseMap)
	if err != nil {
		return "", err
	}
	authTicket = responseMap["auth_ticket"]
	if authTicket == "" {
		return "", common.NewError("500", "Error getting auth ticket from blobber")
	}

	return authTicket, nil
}
