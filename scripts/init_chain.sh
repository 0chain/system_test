#!/bin/bash
# Initialize fresh 0Chain local network with required configuration
# Run this after the blockchain is started and 0dns is responding

set -e

# Configuration
ZWALLET=${ZWALLET:-"$HOME/Code/zwalletcli/zwallet"}
ZBOX=${ZBOX:-"$HOME/Code/zboxcli/zbox"}
CONFIG_DIR=${CONFIG_DIR:-"$HOME/.zcn"}
CONFIG_FILE=${CONFIG_FILE:-"config.yaml"}
WALLET_FILE=${WALLET_FILE:-"wallet.json"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if 0dns is responding
check_chain() {
    print_status "Checking if chain is responding..."
    if ! curl -s http://127.0.0.1:9091/network > /dev/null 2>&1; then
        print_error "Chain not responding at http://127.0.0.1:9091"
        print_error "Please start the blockchain first: cd ~/Code/0chain/docker.local && docker-compose up -d"
        exit 1
    fi
    print_status "Chain is responding!"
}

# Create wallet if it doesn't exist
create_wallet() {
    if [ ! -f "$CONFIG_DIR/$WALLET_FILE" ]; then
        print_status "Creating wallet..."
        $ZWALLET create-wallet --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent
    else
        print_status "Wallet already exists at $CONFIG_DIR/$WALLET_FILE"
    fi
}

# Get faucet tokens
get_faucet() {
    print_status "Getting faucet tokens..."
    $ZWALLET faucet --methodName pour --input "{}" --tokens 100 \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || true
    sleep 2
}

# Configure hardforks
configure_hardforks() {
    print_status "Configuring hardforks (all at round 0)..."
    $ZWALLET add-hardfork \
        --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,Jason' \
        --rounds '0,0,0,0,0,0,0,0,0,0' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "Hardfork configuration may have already been applied"
    }
    sleep 2
}

# Configure miner settings
configure_miner_settings() {
    print_status "Configuring view change rounds..."
    $ZWALLET mn-update-config \
        --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' \
        --values '10,20,10,10,20' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "View change rounds may have already been configured"
    }
    sleep 2

    print_status "Configuring k_percent and x_percent..."
    $ZWALLET mn-update-config \
        --keys 'k_percent,x_percent' \
        --values '0.6,0.6' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "k_percent/x_percent may have already been configured"
    }
    sleep 2

    print_status "Configuring min_n and min_s..."
    $ZWALLET mn-update-config \
        --keys 'min_n,min_s' \
        --values '2,1' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "min_n/min_s may have already been configured"
    }
    sleep 2
}

# Configure global settings
configure_global_settings() {
    print_status "Enabling view change..."
    $ZWALLET global-update-config \
        --keys 'server_chain.view_change' \
        --values 'true' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "View change may have already been enabled"
    }
    sleep 2

    print_status "Configuring block proposal max wait time..."
    $ZWALLET global-update-config \
        --keys 'server_chain.block.proposal.max_wait_time' \
        --values '500ms' \
        --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
        print_warning "Block proposal max wait time may have already been configured"
    }
    sleep 2
}

# Fund blobber wallets
fund_blobbers() {
    print_status "Funding blobber wallets..."

    # Get blobber IDs from the chain
    local blobber_ids=$($ZBOX ls-blobbers --all --configDir $CONFIG_DIR --config $CONFIG_FILE 2>/dev/null | grep "^- id:" | awk '{print $2}')

    if [ -z "$blobber_ids" ]; then
        print_warning "No blobbers registered yet. Skipping blobber funding."
        return
    fi

    for blobber_id in $blobber_ids; do
        print_status "Funding blobber: $blobber_id"
        $ZWALLET send \
            --to_client_id "$blobber_id" \
            --tokens 5 \
            --desc "Fund blobber" \
            --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
            print_warning "Failed to fund blobber $blobber_id (may already have funds)"
        }
        sleep 1
    done
}

# Stake on blobbers
stake_blobbers() {
    print_status "Staking on blobbers..."

    # Get blobber IDs from the chain
    local blobber_ids=$($ZBOX ls-blobbers --all --configDir $CONFIG_DIR --config $CONFIG_FILE 2>/dev/null | grep "^- id:" | awk '{print $2}')

    if [ -z "$blobber_ids" ]; then
        print_warning "No blobbers registered yet. Skipping staking."
        return
    fi

    for blobber_id in $blobber_ids; do
        print_status "Staking on blobber: $blobber_id"
        $ZBOX sp-lock \
            --blobber_id "$blobber_id" \
            --tokens 1 \
            --wallet $WALLET_FILE --configDir $CONFIG_DIR --config $CONFIG_FILE --silent || {
            print_warning "Failed to stake on blobber $blobber_id"
        }
        sleep 1
    done
}

# Main execution
main() {
    echo "============================================"
    echo "  0Chain Local Network Initialization"
    echo "============================================"
    echo ""

    check_chain
    create_wallet
    get_faucet
    configure_hardforks
    configure_miner_settings
    configure_global_settings

    echo ""
    print_status "Chain configuration complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Wait for blobbers to register (may take a few minutes)"
    echo "  2. Run: $0 --fund-blobbers  # to fund and stake blobbers"
    echo ""
}

# Parse arguments
case "${1:-}" in
    --fund-blobbers)
        check_chain
        fund_blobbers
        stake_blobbers
        print_status "Blobber funding complete!"
        ;;
    --stake-only)
        check_chain
        stake_blobbers
        print_status "Staking complete!"
        ;;
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  (none)           Run full chain initialization"
        echo "  --fund-blobbers  Fund and stake on registered blobbers"
        echo "  --stake-only     Only stake on blobbers (no funding)"
        echo "  --help           Show this help message"
        ;;
    *)
        main
        ;;
esac
