package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk/core/zcncrypto"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const zboxTeamWallet = "wallets/zbox_team"

func ListBlobbers(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func ListBlobbersWithWallet(t *test.SystemTest, walletName, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, walletName, cliConfigFilename), 3, time.Second*2)
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
func GenerateBlobberAuthTickets(t *test.SystemTest) (string, string) {
	return GenerateBlobberAuthTicketsWithWallet(t, EscapedTestName(t))
}

func GenerateBlobberAuthTicketsWithWallet(t *test.SystemTest, walletName string) (string, string) {
	var blobbersList []climodel.Blobber
	output, err := ListBlobbersWithWallet(t, walletName, configPath, "--json")
	require.Nil(t, err, "Failed to get blobbers list", strings.Join(output, "\n"))
	require.NotNil(t, output[0], "Empty list blobbers json response")

	err = json.NewDecoder(strings.NewReader(strings.Join(output, "\n"))).Decode(&blobbersList)
	require.Nil(t, err, "Error parsing the blobbers list", strings.Join(output, "\n"))
	require.NotNil(t, blobbersList, "Blobbers list is empty")

	// Get auth tickets for all blobbers
	var blobberAuthTickets string
	var blobbersIds string
	wallet, err := GetWalletForName(t, configPath, walletName)
	require.Nil(t, err, "could not get wallet")

	for i, blobber := range blobbersList {
		authTicket, err := getBlobberAuthTicket(t, blobber.ID, blobber.BaseURL, zboxTeamWallet, wallet.ClientID)
		require.Nil(t, err, "could not get auth ticket for blobber", blobber.ID)
		require.NotNil(t, authTicket, "could not get auth ticket for blobber %v", blobber)
		require.NotEqual(t, authTicket, "", "empty auth ticket for blobber %v", blobber)

		if i == len(blobbersList)-1 {
			blobberAuthTickets += authTicket
			blobbersIds += blobber.ID
			break
		}
		blobberAuthTickets += authTicket + ","
		blobbersIds += blobber.ID + ","
	}
	return blobberAuthTickets, blobbersIds
}

func GetBlobberAuthTicketWithId(t *test.SystemTest, blobberID, blobberUrl string) (string, error) {
	return getBlobberAuthTicketForIdWithWallet(t, EscapedTestName(t), blobberID, blobberUrl)
}
func getBlobberAuthTicketForIdWithWallet(t *test.SystemTest, walletName, blobberId, blobberUrl string) (string, error) {
	userWallet, err := GetWalletForName(t, configPath, walletName)
	if err != nil {
		return "", err
	}

	return getBlobberAuthTicket(t, blobberId, blobberUrl, zboxTeamWallet, userWallet.ClientID)
}

func getBlobberAuthTicket(t *test.SystemTest, blobberID, blobberUrl, zboxTeamWallet, clientID string) (string, error) {
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
