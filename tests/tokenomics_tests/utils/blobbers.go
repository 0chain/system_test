package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// DefaultAuthRoundExpiry is a far-future round number used for enterprise blobber auth tickets.
// When the hermes hardfork is active, the blobber signs Hash(clientID_round) and the smart contract
// verifies it against auth_round_expiry. This must be > current chain round.
const DefaultAuthRoundExpiry = int64(999999999)

var zboxTeamWalletFile *climodel.WalletFile

func ListBlobbers(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func ListBlobbersWithWallet(t *test.SystemTest, walletName, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, walletName, cliConfigFilename), 3, time.Second*2)
}

// scBlobberNode mirrors the SC REST API response for a blobber node (includes is_enterprise).
type scBlobberNode struct {
	ID              string `json:"id"`
	BaseURL         string `json:"url"`
	IsEnterprise    bool   `json:"is_enterprise"`
	LastHealthCheck int64  `json:"last_health_check"`
}

type scBlobberList struct {
	Nodes []scBlobberNode `json:"Nodes"`
}

// GetEnterpriseBlobbers queries the SC REST API to get only enterprise blobbers.
// The CLI's ls-blobbers doesn't include is_enterprise, so we query the SC directly.
// This function is safe to call before skip checks: it returns an empty slice (not a
// test failure) when the sharder is unreachable or returns unexpected data.
func GetEnterpriseBlobbers(t *test.SystemTest) []climodel.Blobber {
	sharderUrl := GetSharderUrlSafe(t)
	if sharderUrl == "" {
		t.Logf("Warning: no sharder URL available, cannot query enterprise blobbers")
		return nil
	}
	url := sharderUrl + "/v1/screst/" + StorageSmartContractAddress + "/getblobbers"

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		t.Logf("Warning: failed to query SC for blobbers: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Warning: failed to read SC blobber response: %v", err)
		return nil
	}

	var blobberList scBlobberList
	err = json.Unmarshal(body, &blobberList)
	if err != nil {
		t.Logf("Warning: failed to parse SC blobber response: %v", err)
		return nil
	}

	// Only include enterprise blobbers with a recent health check (within the last hour).
	// Blobbers with stale health checks are considered inactive by the chain.
	recentThreshold := time.Now().Unix() - 3600
	var enterprise []climodel.Blobber
	for _, b := range blobberList.Nodes {
		if b.IsEnterprise && b.LastHealthCheck > recentThreshold {
			enterprise = append(enterprise, climodel.Blobber{
				ID:      b.ID,
				BaseURL: b.BaseURL,
			})
		}
	}
	t.Logf("Found %d active enterprise blobbers out of %d total", len(enterprise), len(blobberList.Nodes))
	return enterprise
}

// GetAllEnterpriseBlobberIDs returns a set of all blobber IDs with is_enterprise=true,
// regardless of last_health_check time. Use this for building exclusion maps only;
// for selecting active blobbers, use GetEnterpriseBlobbers which filters by health check.
func GetAllEnterpriseBlobberIDs(t *test.SystemTest) map[string]bool {
	sharderUrl := GetSharderUrlSafe(t)
	if sharderUrl == "" {
		t.Logf("Warning: no sharder URL available, cannot query enterprise blobber IDs")
		return nil
	}
	url := sharderUrl + "/v1/screst/" + StorageSmartContractAddress + "/getblobbers"
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		t.Logf("Warning: failed to query SC for enterprise blobber IDs: %v", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Warning: failed to read SC blobber response: %v", err)
		return nil
	}
	var blobberList scBlobberList
	err = json.Unmarshal(body, &blobberList)
	if err != nil {
		t.Logf("Warning: failed to parse SC blobber response: %v", err)
		return nil
	}
	ids := make(map[string]bool)
	for _, b := range blobberList.Nodes {
		if b.IsEnterprise {
			ids[b.ID] = true
		}
	}
	t.Logf("Found %d enterprise blobber IDs (regardless of health check) out of %d total", len(ids), len(blobberList.Nodes))
	return ids
}

// GetBlobberReadPrice queries the SC REST API and returns the blobber's current read_price in SAS.
// Returns -1 on error.
func GetBlobberReadPrice(t *test.SystemTest, blobberID string) int64 {
	sharderURL := GetSharderUrlSafe(t)
	if sharderURL == "" {
		return -1
	}
	url := sharderURL + "/v1/screst/" + StorageSmartContractAddress + "/getBlobber?blobber_id=" + blobberID
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		t.Logf("GetBlobberReadPrice: HTTP error: %v", err)
		return -1
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1
	}
	var result struct {
		Terms struct {
			ReadPrice int64 `json:"read_price"`
		} `json:"terms"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("GetBlobberReadPrice: JSON error: %v", err)
		return -1
	}
	return result.Terms.ReadPrice
}

func GetBlobberDetails(t *test.SystemTest, cliConfigFilename, blobberId string) (*climodel.Blobber, error) {
	return GetBlobberDetailsWithWallet(t, EscapedTestName(t), cliConfigFilename, blobberId)
}

func GetBlobberDetailsWithWallet(t *test.SystemTest, walletName, cliConfigFilename, blobberId string) (*climodel.Blobber, error) {
	output, err := ListBlobbersWithWallet(t, walletName, cliConfigFilename, "--json")
	require.Nil(t, err, "Unable to get blobbers list", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of blobber list output")

	var blobberList []climodel.Blobber
	err = json.NewDecoder(strings.NewReader(output[len(output)-1])).Decode(&blobberList)
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
	nonce, err := GetNonceForWallet(t, cliConfigFilename, wallet)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update --silent --wallet %s_wallet.json --configDir ./config --config %s %s%s", wallet, cliConfigFilename, params, nonceParam), 3, time.Second*2)
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
	// Get only enterprise blobbers from the SC REST API
	blobbersList := GetEnterpriseBlobbers(t)
	if len(blobbersList) == 0 {
		t.Skip("No active enterprise blobbers (last_health_check > 1h stale); skipping enterprise test")
		return
	}

	// Get auth tickets for enterprise blobbers only, skipping unhealthy ones
	var tickets []string
	var ids []string
	wallet, err := GetWalletForName(t, configFileName, walletName)
	require.Nil(t, err, "could not get wallet")

	for i := range blobbersList {
		blobber := blobbersList[i]
		authTicket, err := getBlobberAuthTicket(t, configFileName, blobber.ID, blobber.BaseURL, zboxTeamWalletName, wallet.ClientID)
		if err != nil || authTicket == "" {
			t.Logf("Skipping unhealthy enterprise blobber %s (err: %v)", blobber.ID, err)
			continue
		}
		tickets = append(tickets, authTicket)
		ids = append(ids, blobber.ID)
	}
	if len(tickets) == 0 {
		t.Skip("No enterprise blobber could provide an auth ticket; skipping enterprise test")
		return
	}

	return strings.Join(tickets, ","), strings.Join(ids, ",")
}

// GetEnterpriseBlobberIdAndUrlNotPartOfAllocation returns an enterprise blobber
// that is not currently part of the given allocation. This ensures that enterprise
// allocation add/replace operations pick an enterprise blobber (not a regular one).
func GetEnterpriseBlobberIdAndUrlNotPartOfAllocation(t *test.SystemTest, configPath, allocationID string) (blobberId, blobberUrl string, err error) {
	enterpriseBlobbers := GetEnterpriseBlobbers(t)
	if len(enterpriseBlobbers) == 0 {
		t.Skip("No active enterprise blobbers (last_health_check > 1h stale); skipping enterprise test")
		return
	}
	alloc := GetAllocation(t, allocationID)

	allocBlobberMap := map[string]bool{}
	for _, b := range alloc.BlobberDetails {
		allocBlobberMap[b.BlobberID] = true
	}

	for _, b := range enterpriseBlobbers {
		if !allocBlobberMap[b.ID] {
			return b.ID, b.BaseURL, nil
		}
	}

	return "", "", fmt.Errorf("no enterprise blobber found outside allocation")
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
	if zboxTeamWalletFile == nil {
		// Read the wallet file directly to get the private key (Wallet struct doesn't include keys)
		walletPath := "./config/" + zboxTeamWalletName + "_wallet.json"
		walletData, err := os.ReadFile(walletPath)
		require.Nil(t, err, "could not read zbox team wallet file: "+walletPath)

		var wf climodel.WalletFile
		err = json.Unmarshal(walletData, &wf)
		require.Nil(t, err, "could not parse zbox team wallet file")
		require.NotEmpty(t, wf.Keys, "zbox team wallet has no keys")

		zboxTeamWalletFile = &wf
	}

	var authTicket string
	signatureScheme := zcncrypto.NewSignatureScheme("bls0chain")
	_ = signatureScheme.SetPrivateKey(zboxTeamWalletFile.Keys[0].PrivateKey)
	_ = signatureScheme.SetPublicKey(zboxTeamWalletFile.Keys[0].PublicKey)

	signature, err := signatureScheme.Sign(hex.EncodeToString([]byte(zboxTeamWalletFile.Keys[0].PublicKey)))
	if err != nil {
		return authTicket, err
	}

	url := blobberUrl + "/v1/auth/generate?client_id=" + clientID + "&round=" + fmt.Sprintf("%d", DefaultAuthRoundExpiry)
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
