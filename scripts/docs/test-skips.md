# Test Skips Documentation

This document catalogues every `t.Skip()` call across the test suites. For each skip, the category is one of:

- **LEGIT-FEATURE-REMOVED** — Feature removed from gosdk/chain; test can never pass on current codebase
- **LEGIT-INFRA** — Requires infrastructure not available in local/standard environment
- **LEGIT-FLAKY** — Skip inserted at runtime when a timing-sensitive prerequisite is not met (challenge protocol, download speed, etc.)
- **LEGIT-KILL-TESTS** — Destructive test guarded by `ENABLE_KILL_TESTS=1` env var
- **LEGIT-PATCHED-SC** — Chain uses a patched SC that relaxes a constraint the test exercises
- **DISABLED-NEEDS-RECHECK** — Permanently disabled with a "needs to be re-enabled" message; test body is implemented but was disabled without a specific feature reason

---

## API Tests

| File | Test / Subtest | Skip Reason | Category | Action |
|------|----------------|-------------|----------|--------|
| `tests/api_tests/flaky___broken_scenarios_test.go` | `Test___BrokenScenariosRegisterWallet` (entire suite) | `t.Skip()` with no message — "Tests in here are skipped until the feature has been fixed" | LEGIT-FEATURE-REMOVED | Keep skip; the subtests exercise behaviours (invalid client_id, invalid public_key) that may rely on stricter miner validation that was removed. The file comment says "skipped until feature has been fixed". |
| `tests/api_tests/register_blobber_test.go` | `Write price lower than min_write_price should not allow register` | `min_write_price=0` on test chain — any non-negative write price is valid | LEGIT-PATCHED-SC | Keep skip. As long as the test chain has `min_write_price=0`, this test can never demonstrate a failure. Update message is already accurate. |
| `tests/api_tests/zvault_operations_test.go` | `Store invalid private key with correct mnemonic` | "Implement when store validation is refactored" | DISABLED-NEEDS-RECHECK | The test body is implemented. The store endpoint may now validate the private key format. Should be investigated and re-enabled or removed. |

---

## CLI Tests

### Auth / Bridge Tests (Tenderly)

All Tenderly/bridge tests are permanently skipped because the Tenderly authorizer infrastructure is not deployed in the local test environment. These are all **LEGIT-INFRA**.

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/0_tenderly_zcnbridge_verify_ethereum_test.go` | `TestVerifyEthereumAddress` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_ethereum-account-register_test.go` | `TestEthereumAccountRegister` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_auth_replace_burn_mint_test.go` | `TestAuthReplaceBurnMint` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_authorizer_rewards_test.go` | `TestAuthorizerRewards` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_list_authorizers_test.go` | `TestListAuthorizers` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_burn_test.go` | `TestZcnBridge/Burn_ZCN_tokens` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_mint_test.go` | `TestZcnBridge/Mint_ZCN_tokens` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_settings_global_test.go` | `TestZcnBridgeSettingsGlobal` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_validator_config_update_test.go` | `TestValidatorConfigUpdate` | Not deployed in local env | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_tenderly_zcnbridge_add_and_rm_authorizer_test.go` | `TestAddAndRmAuthorizer` (both subtests) | Not deployed in local env | LEGIT-INFRA | Keep |

### Migration Tests (S3 / Dropbox / GDrive)

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/0_s3mgrt_migrate_test.go` | Migration test | `s3SecretKey or s3AccessKey was missing` | LEGIT-INFRA | Keep — credentials injected via env |
| `tests/cli_tests/0_s3mgrt_migrate_alternate_test.go` | Migration test | `s3SecretKey or s3AccessKey was missing` + `S3 Bucket operation is not working properly` | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_s3mgrt_migrate_alternate2_test.go` | Migration test | `s3SecretKey or s3AccessKey was missing` + `S3 Bucket operation is not working properly` | LEGIT-INFRA | Keep |
| `tests/cli_tests/0_dropboxmgrt_migrate_test.go` | Migration test | `dropbox Access Token was missing` | LEGIT-INFRA | Keep — credentials injected via env |
| `tests/cli_tests/0_gdrivemgrt_migrate_test.go` | Migration test | `Gdrive Access Token was missing` | LEGIT-INFRA | Keep — credentials injected via env |

### Kill Tests (Destructive)

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/zzwalletcli_kill_sharder_test.go` | `TestKillSharder` | Guarded by `ENABLE_KILL_TESTS=1`; also skips if all sharders already killed | LEGIT-KILL-TESTS | Keep |
| `tests/cli_tests/zzzwalletcli_kill_miner_test.go` | `TestKillMiner` | Guarded by `ENABLE_KILL_TESTS=1`; also skips if all miners already killed | LEGIT-KILL-TESTS | Keep |
| `tests/cli_tests/zboxcli_kill_blobber_test.go` | `TestKillBlobber` | Guarded by `ENABLE_KILL_TESTS=1` | LEGIT-KILL-TESTS | Keep |
| `tests/tokenomics_tests/blobber_slash_penalty_test.go` | `TestBlobberSlashPenalty` | Kills blobber on-chain; guarded by `ENABLE_KILL_TESTS=1` | LEGIT-KILL-TESTS | Keep |

### Block/Fee Reward Tests (Debug Event Database)

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/miner_block_rewards_test.go` | `TestMinerBlockRewards` | Requires debug event database | LEGIT-INFRA | Keep |
| `tests/cli_tests/sharder_block_rewards_test.go` | `TestSharderBlockRewards` | Requires debug event database | LEGIT-INFRA | Keep |
| `tests/cli_tests/miner_fee_rewards_test.go` | `TestMinerFeeRewards` | Requires debug event database | LEGIT-INFRA | Keep |
| `tests/cli_tests/sharder_fee_rewards_test.go` | `TestSharderFeeRewards` | Requires debug event database | LEGIT-INFRA | Keep |

### Feature Removed from gosdk/Chain

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_common_user_functions_test.go` | `TestCommonUserFunctions` | `Create Allocation - Blobbers must lock appropriate amount of tokens in stake pool` | `sp-info API no longer returns per-allocation offer breakdown (Offers field removed in gosdk readpool cleanup Sep 2024)` | LEGIT-FEATURE-REMOVED | Keep |
| `tests/cli_tests/zboxcli_common_user_functions_test.go` | `TestCommonUserFunctions` | `Update Allocation - Blobbers' lock in stake pool must increase according to updated size` | Same as above | LEGIT-FEATURE-REMOVED | Keep |
| `tests/cli_tests/zboxcli_file_download_test.go` | `TestDownload` | `Download Shared File without Paying Should Not Work` | Read pool per-user payment model removed; auth-ticket downloads are free when `read_price=0` | LEGIT-FEATURE-REMOVED | Keep |

### Patched SC / Chain Configuration

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/zwalletcli_miner_update_settings_test.go` | `TestMinerUpdateSettings` | (specific subtest) | Miner settings update from non-delegate wallet succeeded — chain may not enforce delegate wallet restrictions (patched SC) | LEGIT-PATCHED-SC | Keep — runtime guard, only skips if chain doesn't enforce |
| `tests/cli_tests/zwalletcli_sharder_update_settings_test.go` | `TestSharderUpdateSettings` | (specific subtest) | Sharder settings update from non-delegate wallet succeeded — patched SC | LEGIT-PATCHED-SC | Keep — runtime guard |

### Runtime / Flaky Guards (Challenge Protocol Not Settling)

These skips are inserted at runtime when a prerequisite condition is not met. They are not permanently disabled.

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_file_update_test.go` | `TestFileUpdate` | Upload cost subtest | `MovedToChallenge=0` after polling | LEGIT-FLAKY | Keep — correct guard |
| `tests/cli_tests/zboxcli_file_update_test.go` | `TestFileUpdate` | Update cost subtest | `MovedToChallenge` did not increase after update | LEGIT-FLAKY | Keep — correct guard |
| `tests/cli_tests/zboxcli_file_upload_test.go` | `TestUpload` | Upload cost subtest | `MovedToChallenge=0` after polling | LEGIT-FLAKY | Keep — correct guard |
| `tests/cli_tests/zboxcli_file_copy_test.go` | `TestFileCopy` | Copy cost (first check) | `MovedToChallenge=0` after polling | LEGIT-FLAKY | Keep — correct guard |
| `tests/cli_tests/zboxcli_file_copy_test.go` | `TestFileCopy` | Copy cost (after copy) | `MovedToChallenge` did not increase after copy | LEGIT-FLAKY | Keep — correct guard |
| `tests/cli_tests/0_challenge_protocol_test.go` | `TestChallengeProtocol` | Multiple subtests | Sharder endpoint unavailable / no challenges generated / chain stalled / challenge_allocations.txt not found | LEGIT-FLAKY + LEGIT-INFRA | Keep — CI-setup files not present locally |
| `tests/tokenomics_tests/allocation_test.go` | `TestAllocation` | Multiple subtests | `MovedToChallenge=0` after polling, no challenge rewards, no blobber available | LEGIT-FLAKY | Keep — correct guard |

### ZS3 / Warp Analysis

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/zs3server_tests/warp_analysis_test.go` | `TestWarpAnalysis` | warp binary not available / no CSV files found | LEGIT-INFRA | Keep |

### Restricted Blobber (Runtime Chain Enforcement Check)

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_restricted_blobber_test.go` | `TestRestrictedBlobber` | Multiple | Chain does not enforce restricted blobber auth tickets / blobber auth not configured | LEGIT-INFRA | Keep — runtime check |

### Staking / Delegate Pool Capacity (Runtime)

These skips fire at runtime when delegate pools are full. They are infrastructure guards, not permanent disables.

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_stake_unstake_token_test.go` | `TestStakeUnstakeToken` | `max_delegates reached` | LEGIT-INFRA | Keep — runtime guard |
| `tests/cli_tests/zwalletcli_sharder_stake_test.go` | `TestSharderStake` | `max_delegates reached` | LEGIT-INFRA | Keep — runtime guard |
| `tests/cli_tests/0_blobber_staked_capacity_test.go` | `TestBlobberStakedCapacity` | `max_delegates reached` | LEGIT-INFRA | Keep — runtime guard |

### Livestream / Audio-Video (CI Runner Limitation)

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_livestream_test.go` | `TestLivestream` (3 subtests) | `github runner has no any audio/camera device` | LEGIT-INFRA | Keep |
| `tests/cli_tests/zboxcli_download_livestream_test.go` | `TestDownloadLivestream` | Same | LEGIT-INFRA | Keep |

### Download Resume (Too-Fast Download)

| File | Test | Skip Reason | Category | Action |
|------|------|-------------|----------|--------|
| `tests/cli_tests/zboxcli_file_download_resume_test.go` | `TestFileDownloadResume` | Could not capture partial download state — download completed too fast | LEGIT-FLAKY | Keep — correct guard |

### Send-and-Balance (Residual Balance)

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/zwalletcli_send_and_balance_test.go` | `TestWalletSendAndBalance` | Zero balance send | Wallet has residual balance from a previous run | LEGIT-FLAKY | Keep — correct guard |

### LFB Sharder Selection (Topology)

| File | Test | Subtest | Skip Reason | Category | Action |
|------|------|---------|-------------|----------|--------|
| `tests/cli_tests/lfb_sharder_selection_test.go` | `TestLFBSharderSelection` | 3-miner/1-sharder topology subtest | Skips when active sharder count != 1 | LEGIT-INFRA | Keep — topology guard |

---

## Permanently Disabled Tests (DISABLED-NEEDS-RECHECK)

These tests have `t.Skip()` with a "needs to be re-enabled" message. The test bodies are fully implemented. They should be reviewed to determine if they can be re-enabled.

| File | Test | Subtest | Disable Reason | Notes |
|------|------|---------|----------------|-------|
| `tests/cli_tests/zboxcli_create_allocation_test.go` | `TestCreateAllocation` | `Create allocation with read price range Should Work` | "read price range allocation tests need to be re-enabled" | Also has `return` statement after skip — dead code. `read_price` CLI flag may still be supported; needs investigation. |
| `tests/cli_tests/zboxcli_create_allocation_test.go` | `TestCreateAllocation` | `Create allocation with read price range 0-0 Should Fail` | "read price range allocation tests need to be re-enabled" | Same as above — also has `return` after skip. |
| `tests/cli_tests/zboxcli_rollback_test.go` | `TestRollbackAllocation` | `rollback allocation after moving a file should work` | "rollback after move is not atomic in v2 - needs to be re-enabled" | Rollback v2 atomicity issue in chain; valid skip. |
| `tests/cli_tests/zboxcli_rollback_test.go` | `TestRollbackAllocation` | `rollback allocation after renaming a file should work` | "rollback after rename is not atomic in v2 - needs to be re-enabled" | Same as above. |
| `tests/cli_tests/zboxcli_file_upload_test.go` | `TestUpload` | `Upload tests with Thumbnail with different format` | "needs performance improvements to be re-enabled" | 40-minute timeout, 10GB allocation, 50 file formats — very heavy test. Valid to keep disabled. |
| `tests/cli_tests/zwalletcli_miner_update_config_test.go` | `TestMinerUpdateConfig` | `unsuccessful update of config to out of bounds value` | "needs to be re-enabled" | Tests out-of-bounds values for `reward_rate`, `share_ratio`, `reward_decline_rate` (all set to `1`). The chain may or may not enforce upper bounds on these values currently. |

---

## Summary

### By Category

| Category | Count | Description |
|----------|-------|-------------|
| LEGIT-INFRA | ~45 | Missing credentials, wrong environment, capacity limits, device availability |
| LEGIT-FLAKY | ~20 | Runtime guards for timing-sensitive preconditions (challenge protocol, download speed) |
| LEGIT-KILL-TESTS | 4 | Destructive tests guarded by `ENABLE_KILL_TESTS=1` |
| LEGIT-FEATURE-REMOVED | 4 | gosdk/chain features removed (Offers field, read pool per-user model) |
| LEGIT-PATCHED-SC | 3 | Chain uses patched SC that relaxes constraints |
| DISABLED-NEEDS-RECHECK | 6 | Test bodies implemented; disabled with "needs to be re-enabled" |

### Tests Fixed in This Session

1. **`tests/cli_tests/zboxcli_file_stats_test.go`** — The `get file stats before and after download` subtest had `t.Skip()` added in commit `b3a390b39` ("skip tests", Feb 2025). This skip has been removed in the working tree (uncommitted change). The fix also:
   - Increases allocation size from `2048` to `1048576` (1MB) so each blobber shard exceeds the 64KB minimum chunk size required for download tracking
   - Corrects the allocation parameter from `tokens` to `lock`
   - Improves the assertion: instead of checking that exactly 1 blobber skipped the download, now verifies that at least 1 blobber served the download (more accurate for variable-shard configurations)

### Tests That Could Be Re-enabled (Investigation Required)

1. **`zboxcli_create_allocation_test.go` — read price range tests**: The `--read_price` flag may still work in zbox CLI. If the chain accepts read price ranges in allocation creation, these tests can be re-enabled. The `return` statement after each skip should also be removed.

2. **`zwalletcli_miner_update_config_test.go` — out of bounds value test**: Test checks that `reward_rate=1`, `share_ratio=1`, `reward_decline_rate=1` are rejected. If the chain enforces upper bound of `< 1` for these values, this test should pass. Needs chain-level verification.

3. **`zvault_operations_test.go` — store invalid private key**: Test body is implemented. Re-enable once store validation refactor is complete.

### Notes on `t.Skip()` Placement

Several tests in `zboxcli_create_allocation_test.go` have both a `t.Skip()` AND a `return` statement:

```go
t.Run("Create allocation with read price range Should Work", func(t *test.SystemTest) {
    t.Skip("Test disabled: read price range allocation tests need to be re-enabled")
    return          // ← redundant dead code
    _ = setupWallet(t, configPath)
    ...
})
```

The `return` is redundant because `t.Skip()` marks the test as skipped and stops execution. When re-enabling these tests, the `return` line must be removed.
