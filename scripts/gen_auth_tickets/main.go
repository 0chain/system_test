// gen_auth_tickets generates auth tickets for all enterprise blobbers.
// Usage: go run ./scripts/gen_auth_tickets/ <sharder_url> <zbox_team_wallet_file> <client_id>
// Output on success: <blobber_ids>|<auth_tickets>  (one line to stdout)
// Output on error: exits non-zero with error message to stderr
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/0chain/gosdk/core/zcncrypto"
)

const storageSmartContractAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
const defaultAuthRoundExpiry = int64(999999999)

type blobberNode struct {
	ID           string `json:"id"`
	BaseURL      string `json:"url"`
	IsEnterprise bool   `json:"is_enterprise"`
}

type blobberList struct {
	Nodes []blobberNode `json:"Nodes"`
}

type walletKey struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type walletFile struct {
	ClientID string      `json:"client_id"`
	Keys     []walletKey `json:"keys"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: gen_auth_tickets <sharder_url> <zbox_team_wallet_file> <client_id>\n")
		os.Exit(1)
	}
	sharderURL := strings.TrimRight(os.Args[1], "/")
	walletPath := os.Args[2]
	clientID := os.Args[3]

	// Read zbox_team wallet
	walletData, err := os.ReadFile(walletPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading wallet file %s: %v\n", walletPath, err)
		os.Exit(1)
	}
	var wf walletFile
	if err := json.Unmarshal(walletData, &wf); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing wallet file: %v\n", err)
		os.Exit(1)
	}
	if len(wf.Keys) == 0 {
		fmt.Fprintf(os.Stderr, "Wallet file has no keys\n")
		os.Exit(1)
	}

	// Get enterprise blobbers from SC REST API
	url := sharderURL + "/v1/screst/" + storageSmartContractAddress + "/getblobbers"
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying blobbers: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var bl blobberList
	if err := json.Unmarshal(body, &bl); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing blobber list: %v\n", err)
		os.Exit(1)
	}

	// Prepare BLS signer — SetPrivateKey is sufficient; SetPublicKey must NOT be called
	// after SetPrivateKey because gosdk rejects it ("cannot set public key when there is a private key").
	scheme := zcncrypto.NewSignatureScheme("bls0chain")
	if err := scheme.SetPrivateKey(wf.Keys[0].PrivateKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting private key: %v\n", err)
		os.Exit(1)
	}
	signature, err := scheme.Sign(hex.EncodeToString([]byte(wf.Keys[0].PublicKey)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing: %v\n", err)
		os.Exit(1)
	}

	var ids []string
	var tickets []string
	for _, b := range bl.Nodes {
		if !b.IsEnterprise {
			continue
		}
		authURL := fmt.Sprintf("%s/v1/auth/generate?client_id=%s&round=%d", b.BaseURL, clientID, defaultAuthRoundExpiry)
		req, err := http.NewRequest("GET", authURL, http.NoBody)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping blobber %s: %v\n", b.ID, err)
			continue
		}
		req.Header.Set("Zbox-Signature", signature)
		client := &http.Client{}
		r, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping blobber %s (unreachable): %v\n", b.ID, err)
			continue
		}
		defer r.Body.Close()
		var result map[string]string
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "Skipping blobber %s (bad response): %v\n", b.ID, err)
			continue
		}
		ticket := result["auth_ticket"]
		if ticket == "" {
			fmt.Fprintf(os.Stderr, "Skipping blobber %s (empty ticket)\n", b.ID)
			continue
		}
		ids = append(ids, b.ID)
		tickets = append(tickets, ticket)
	}

	if len(ids) == 0 {
		fmt.Fprintf(os.Stderr, "No enterprise blobbers returned auth tickets\n")
		os.Exit(1)
	}

	// Output: blobber_ids|auth_tickets
	fmt.Printf("%s|%s\n", strings.Join(ids, ","), strings.Join(tickets, ","))
}
