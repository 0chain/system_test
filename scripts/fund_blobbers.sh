#!/bin/bash
# Fund and stake all blobbers for local 0Chain network
# Run this after blobbers are started but before they can register

set -e

ZWALLET="${ZWALLET:-$HOME/Code/zwalletcli/zwallet}"
ZBOX="${ZBOX:-$HOME/Code/zboxcli/zbox}"
CONFIG_DIR="${CONFIG_DIR:-$HOME/.zcn}"
CONFIG_FILE="${CONFIG_FILE:-config.yaml}"
WALLET_FILE="${WALLET_FILE:-wallet.json}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Get faucet tokens
get_faucet() {
    local count=${1:-10}
    print_status "Getting $count faucet tokens..."
    for i in $(seq 1 $count); do
        $ZWALLET faucet --methodName pour --input "{}" --tokens 100 \
            --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent 2>/dev/null || true
        sleep 1
    done
}

# Get blobber IDs from container logs
get_blobber_ids() {
    for i in 7 8 9 10 11 12; do
        if docker ps --format '{{.Names}}' | grep -q "blobber-$i"; then
            id=$(docker logs blobber-$i 2>&1 | grep "self identity" | head -1 | grep -o '"id": "[a-f0-9]*"' | cut -d'"' -f4)
            if [ -n "$id" ]; then
                echo "$id"
            fi
        fi
    done
}

# Fund blobbers
fund_blobbers() {
    print_status "Funding blobbers..."
    local ids=$(get_blobber_ids)

    for id in $ids; do
        print_status "Funding ${id:0:16}..."
        $ZWALLET send --to_client_id "$id" --tokens 1 --desc "Fund blobber" \
            --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent 2>/dev/null || {
            print_warning "Failed to fund $id (may already have balance)"
        }
        sleep 2
    done
}

# Stake on blobbers
stake_blobbers() {
    print_status "Staking on blobbers..."
    local blobber_ids=$($ZBOX ls-blobbers --configDir $CONFIG_DIR --config $CONFIG_FILE 2>/dev/null | \
        grep "^- id:" | awk -F': +' '{print $2}')

    for id in $blobber_ids; do
        print_status "Staking on ${id:0:16}..."
        $ZBOX sp-lock --blobber_id "$id" --tokens 1 \
            --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent 2>/dev/null || {
            print_warning "Failed to stake on $id"
        }
        sleep 2
    done
}

# Main
case "${1:-all}" in
    faucet)
        get_faucet ${2:-10}
        ;;
    fund)
        fund_blobbers
        ;;
    stake)
        stake_blobbers
        ;;
    all|*)
        get_faucet 15
        fund_blobbers
        echo ""
        print_status "Waiting 30s for blobbers to register..."
        sleep 30
        stake_blobbers
        echo ""
        print_status "Done! Checking blobbers..."
        $ZBOX ls-blobbers --configDir $CONFIG_DIR --config $CONFIG_FILE 2>/dev/null | grep -c "^- id:" || echo "0"
        ;;
esac
