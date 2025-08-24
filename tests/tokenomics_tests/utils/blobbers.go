package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk/core/zcncrypto"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const zboxTeamWalletName = "wallets/zbox_team"

var zboxTeamWallet *climodel.Wallet

func ListBlobbers(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func ListBlobbersWithWallet(t *test.SystemTest, walletName, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, walletName, cliConfigFilename), 3, time.Second*2)
}

func GetBlobberDetails(t *test.SystemTest, cliConfigFilename, blobberId string) (*climodel.Blobber, error) {
	return GetBlobberDetailsWithWallet(t, EscapedTestName(t), cliConfigFilename, blobberId)
}

func GetBlobberDetailsWithWallet(t *test.SystemTest, walletName, cliConfigFilename, blobberId string) (*climodel.Blobber, error) {
	output, err := ListBlobbersWithWallet(t, walletName, cliConfigFilename, "--json")
	require.Nil(t, err, "Unable to get blobbers list", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error invalid json data for list blobbers")

	var blobberList []climodel.Blobber
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
	require.Nil(t, err, "Error parsing blobbers list")

	for idx := range blobberList {
		if blobberList[idx].ID == blobberId {
			return &blobberList[idx], nil
		}
	}

	return nil, nil
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
func GenerateBlobberAuthTickets(t *test.SystemTest, configFileName string) (blobberAuthTickets, blobberIds string) {
	return GenerateBlobberAuthTicketsWithWallet(t, EscapedTestName(t), configFileName)
}

func GenerateBlobberAuthTicketsWithWallet(t *test.SystemTest, walletName, configFileName string) (blobberAuthTicket, blobberIds string) {
	var blobbersList []climodel.Blobber
	output, err := ListBlobbersWithWallet(t, walletName, configFileName, "--json")
	require.Nil(t, err, "Failed to get blobbers list", strings.Join(output, "\n"))
	require.NotNil(t, output[0], "Empty list blobbers json response")

	err = json.NewDecoder(strings.NewReader(strings.Join(output, "\n"))).Decode(&blobbersList)
	require.Nil(t, err, "Error parsing the blobbers list", strings.Join(output, "\n"))
	require.NotNil(t, blobbersList, "Blobbers list is empty")

	// Get auth tickets for all blobbers
	var blobberAuthTickets string
	var blobbersIds string
	wallet, err := GetWalletForName(t, configFileName, walletName)
	require.Nil(t, err, "could not get wallet")

	for i := range blobbersList {
		blobber := blobbersList[i]
		authTicket, err := getBlobberAuthTicket(t, configFileName, blobber.ID, blobber.BaseURL, zboxTeamWalletName, wallet.ClientID)
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

func GetBlobberAuthTicketWithId(t *test.SystemTest, cliConfigFileName, blobberID, blobberUrl string) (string, error) {
	return getBlobberAuthTicketForIdWithWallet(t, EscapedTestName(t), cliConfigFileName, blobberID, blobberUrl)
}
func getBlobberAuthTicketForIdWithWallet(t *test.SystemTest, walletName, cliConfigFileName, blobberId, blobberUrl string) (string, error) {
	userWallet, err := GetWalletForName(t, cliConfigFileName, walletName)
	if err != nil {
		return "", err
	}

	return getBlobberAuthTicket(t, cliConfigFileName, blobberId, blobberUrl, zboxTeamWalletName, userWallet.ClientID)
}

func getBlobberAuthTicket(t *test.SystemTest, cliConfigFileName, blobberID, blobberUrl, zboxTeamWalletName, clientID string) (string, error) {
	if zboxTeamWallet == nil {
		zboxWallet, err := GetWalletForName(t, cliConfigFileName, zboxTeamWalletName)
		require.Nil(t, err, "could not get zbox wallet")

		zboxTeamWallet = zboxWallet
	}

	var authTicket string
	signatureScheme := zcncrypto.NewSignatureScheme("bls0chain")
	_ = signatureScheme.SetPrivateKey("85e2119f494cd40ca524f6342e8bdb7bef2af03fe9a08c8d9c1d9f14d6c64f14")
	_ = signatureScheme.SetPublicKey(zboxTeamWallet.ClientPublicKey)

	signature, err := signatureScheme.Sign(hex.EncodeToString([]byte(zboxTeamWallet.ClientPublicKey)))
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
	defer resp.Body.Close() //nolint:errcheck
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
