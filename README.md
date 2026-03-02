# 0Chain System Tests

Integration test suite for the [0Chain](https://github.com/0chain) decentralized storage network. Tests validate storage, wallets, allocations, tokenomics, and service integrations as an end user.

## Test Suites

| Suite | Path | Description |
|-------|------|-------------|
| **API** | `tests/api_tests/` | REST API endpoint tests (allocations, blobbers, 0box, zauth, zvault, zs3) |
| **CLI** | `tests/cli_tests/` | CLI tool tests for `zbox` and `zwallet` (file ops, allocations, staking) |
| **SDK** | `tests/sdk_tests/` | Go SDK integration tests (multi-operation, wallet creation) |
| **Tokenomics** | `tests/tokenomics_tests/` | Financial tests (rewards, penalties, enterprise blobbers) |

## Prerequisites

- Go 1.22.0+
- A running 0Chain network (local or remote)
- For CLI tests: `zbox` and `zwallet` binaries in `tests/cli_tests/`
- For zs3/mc tests: `mc`, `warp`, `minio` binaries in `tests/cli_tests/`

## Quick Start

### Option 1: Local Deployment with Deploy Script (Recommended)

The deploy script automates the entire setup: chain, blobbers, services, wallets, and test configuration.

```bash
# Full deployment from scratch
cd scripts/
./deploy_local.sh all

# Or step by step:
./deploy_local.sh start-chain     # Start miners, sharders, 0dns
./deploy_local.sh chain           # Initialize chain config (hardforks, SC settings)
./deploy_local.sh blobbers        # Build, start, fund, stake blobbers
./deploy_local.sh services        # Start 0box, zauth, zvault, Kafka, Elasticsearch
./deploy_local.sh test-setup      # Setup test wallets and configs
./deploy_local.sh verify          # Verify everything is healthy
```

See [`scripts/README.md`](scripts/README.md) for full deployment documentation including:
- Configuration via `scripts/deploy_config.yaml`
- Branch management for multi-repo development
- Enterprise blobber setup
- Nginx reverse proxy with SSL
- Monitoring and chaos testing
- Web-apps and rclone-zus deployment

### Option 2: CI/CD Pipeline

The [System Tests Pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can:
- Deploy a fresh 0Chain network with custom docker images
- Run tests against an existing network
- Generate HTML test reports

## Running Tests

```bash
# Run all tests in a suite
cd tests/api_tests && go test ./... -v -timeout 30m
cd tests/cli_tests && go test ./... -v -timeout 60m
cd tests/sdk_tests && go test ./... -v -timeout 30m
cd tests/tokenomics_tests && go test ./... -v -timeout 30m

# Run a specific test by name
go test -run "^TestCreateAllocation$" ./... -v

# Run smoke tests only (fast subset)
SMOKE_TEST_MODE=true go test ./... -v

# Run with debug logging
DEBUG=true go test ./... -v

# Exclude known-broken tests (CLI)
go test -run "^Test[^___]*$" ./... -v

# Run all suites with auto-retry via deploy script
cd scripts/ && ./deploy_local.sh test
```

### CLI Test Setup

CLI tests require `zbox` and `zwallet` binaries:

```bash
# Build from source
cd ~/Code/zboxcli && make install && cp ./zbox ../system_test/tests/cli_tests/
cd ~/Code/zwalletcli && make zwallet && cp ./zwallet ../system_test/tests/cli_tests/

# Or use the deploy script (builds and copies automatically)
./scripts/deploy_local.sh test-setup
```

### Configuration

Each test suite has its own config directory:

| Suite | Config | Key Settings |
|-------|--------|--------------|
| API | `tests/api_tests/config/api_tests_config.yaml` | `block_worker`, sharder URLs |
| CLI | `tests/cli_tests/config/zbox_config.yaml` | `block_worker`, wallet paths |
| SDK | `tests/sdk_tests/config/sdk_tests_config.yaml` | `block_worker` |
| Tokenomics | `tests/tokenomics_tests/config/tokenomics_tests_config.yaml` | `block_worker` |

The `block_worker` URL should point to the 0dns service (e.g., `http://198.18.0.100:9091` for Docker, `http://localhost:9091` for host access).

## Architecture

```
system_test/
├── internal/                    # Shared test libraries
│   ├── api/model/               # Data models (Wallet, Allocation, Blobber, etc.)
│   ├── api/util/client/         # HTTP clients (APIClient, SDKClient, ZboxClient)
│   ├── api/util/test/           # SystemTest framework (timeouts, logging)
│   ├── api/util/crypto/         # BLS signatures
│   └── cli/                     # CLI-specific utilities
├── tests/
│   ├── api_tests/               # REST API tests
│   ├── cli_tests/               # CLI tool tests
│   │   ├── mc_tests/            # MinIO client (mc) tests
│   │   └── zs3server_tests/     # S3 gateway tests
│   ├── sdk_tests/               # Go SDK tests
│   └── tokenomics_tests/        # Financial/tokenomics tests
├── scripts/
│   ├── deploy_local.sh          # Full local deployment automation
│   ├── deploy_config.yaml       # Deployment configuration
│   ├── run_tests.sh             # Test runner with retry logic
│   └── README.md                # Deployment documentation
└── .github/workflows/ci.yml     # CI/CD pipeline
```

## Services

The full test environment includes:

| Service | Port | Description |
|---------|------|-------------|
| 0dns | 9091 | DNS/discovery service |
| Miners (4) | 7071-7074 | Block producers |
| Sharders (2) | 7171-7172 | Block storage |
| Blobbers (6-12) | 5051-5065 | Storage providers |
| Validators (6-12) | 5061-5069 | Challenge validators |
| Enterprise Blobbers (3) | 5071-5073 | Enterprise storage |
| 0box | 9081 | Aggregation/metadata API |
| zauth | 8080 | Authentication service |
| zvault | 8090 | Key vault service |
| zs3server | 9000 | S3 gateway (MinIO-compatible) |
| Elasticsearch | 9200 | Search backend for 0box |
| Kafka | 9092 | Event streaming (sharder to 0box) |

## Multi-Repo Development

When PRs span multiple repos (e.g., `0chain` + `blobber` + `gosdk`):

1. Create feature branches in each repo
2. Use the deploy script with branch overrides:
   ```bash
   ./deploy_local.sh --branch 0chain=fix/my-feature --branch blobber=fix/my-feature all
   ```
3. Or run the CI pipeline manually from any repo's Actions tab with custom branches

## Updating Services, Clients, and Web Apps

The deploy script provides a `swap-image` command to update any service, CLI client, or web app without a full redeployment.

```bash
# Usage
./scripts/deploy_local.sh swap-image <repo> [branch] [--gosdk-branch <branch>]
```

### Update a Service (e.g., 0box, blobber, 0chain)

```bash
# Update 0box to a specific branch
./scripts/deploy_local.sh swap-image 0box staging

# Update blobber (uses branch from deploy_config.yaml)
./scripts/deploy_local.sh swap-image blobber

# Update 0chain (rebuilds both miner + sharder images, restarts all)
./scripts/deploy_local.sh swap-image 0chain fix/my-feature

# Update a service with a specific gosdk dependency
./scripts/deploy_local.sh swap-image 0box staging --gosdk-branch enterprise-blobber
```

This will: `git checkout branch` → `build Docker image` → `restart containers`.

### Update CLI Clients (zboxcli / zwalletcli)

```bash
# Update zboxcli to a branch (rebuilds binary, copies to test dirs)
./scripts/deploy_local.sh swap-image zboxcli fix/my-branch

# To use a specific gosdk branch with a CLI client:
cd /path/to/zboxcli
go get github.com/0chain/gosdk@<branch-or-commit>
go mod tidy
./scripts/deploy_local.sh swap-image zboxcli
```

Binaries are built and copied to both `tests/cli_tests/` and `tests/tokenomics_tests/`.

### Update Web Apps with a Specific gosdk (WASM)

```bash
# Rebuild web-apps with a specific gosdk branch for zcn.wasm
./scripts/deploy_local.sh swap-image web-apps master --gosdk-branch enterprise-blobber
```

This will: `checkout gosdk branch` → `build zcn.wasm` → `copy to all packages/*/public/` → `rebuild web-apps`.

### Available Repos

| Repo | What It Builds | Containers Affected |
|------|---------------|-------------------|
| `0chain` | Miner + Sharder images | miner-1..4, sharder-1..2 |
| `blobber` | Blobber + Validator images | blobber-1..12, validator-1..12 |
| `eblobber` | Enterprise blobber image | blobber-13..15 |
| `0box` | 0box image | 0box container |
| `zauth-server` | zauth image | zauth container |
| `zvault` | zvault image | zvault container |
| `zs3server` | MinIO S3 gateway | zs3server container |
| `crawler` | Crawler image | crawler container |
| `web-apps` | Web frontends (Vult, Bolt, Blimp, etc.) | PM2 processes |
| `zboxcli` | `zbox` binary | Copies to test dirs (no container) |
| `zwalletcli` | `zwallet` binary | Copies to test dirs (no container) |

### Run Tests After Update

```bash
./scripts/deploy_local.sh test          # Run all test suites
./scripts/deploy_local.sh test api      # Run only API tests
./scripts/deploy_local.sh test cli      # Run only CLI tests
./scripts/deploy_local.sh test token    # Run only tokenomics tests
```

## Handling Test Failures

- Tests should pass against a healthy network
- If tests fail, check chain health first: `./scripts/deploy_local.sh verify`
- Check blobber balances (blobbers need ZCN to redeem write markers)
- Try running the specific failing test in isolation
- For transient failures, re-run the test (some tests are timing-sensitive)

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

[MIT](https://choosealicense.com/licenses/mit/)
