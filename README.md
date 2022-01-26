# System Tests

A black/grey box suite that tests the functionality of the 0Chain network as an end user via the CLI tools.

## Running tests

The tests require a full 0Chain network to be deployed and running in a healthy state.

### Deploy a new 0Chain network and run tests with the system tests pipeline (RECOMMENDED)

The [System Tests Pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can deploy a new 0Chain network with a custom set of docker images then run tests:    
<details>
  <summary><b>[Click to show screenshot]</b></summary>
<img width="322" alt="ci-deploy" src="https://user-images.githubusercontent.com/18306778/136713487-db7ef096-cb11-4a33-9b29-302ffb5470df.png">  
</details>

**In this mode, do not supply the network URL. Supply the docker images you wish to deploy**  

You can view a list available 0chain docker images at [Docker Hub](https://hub.docker.com/search?q=0chain&type=image), or build your own by running the docker build pipeline in the repo of your feature branch.  

0Chain will automatically be deployed to a free test slot at ```dev-[1-5].devnet-0chain.net```.  
You can view the network URL of deployment by checking the "VIEW TEST CONFIGURATION" step of the pipeline.   
<details>
  <summary><b>[Click to show screenshot]</b></summary>
<img width="1200" alt="ci-config" src="https://user-images.githubusercontent.com/18306778/137035204-4feffd1e-1692-4021-bc06-e97b7925f5a9.png">  
</details>

If tests fail, the network will stay available for debugging purposes, however uptime is not guaranteed as the network may be overridden by another test run.

### Run tests against an existing 0Chain network with the system tests pipeline

The [System Tests Pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can also run tests against an existing 0Chain network  
<details>
  <summary><b>[Click to show screenshot]</b></summary>
<img width="347" alt="ci-predeployed" src="https://user-images.githubusercontent.com/18306778/136713492-fbeadfb0-51d7-4f59-90a0-34e72e9eafcb.png">  
</details>

**In this mode, supply the network URL. Docker image input fields will be ignored**  

Set the network URL field to the 0Chain network you wish to test, without the URL scheme or subdomain.  
eg. beta.0chain.net

### Report

The CI pipeline will generate an HTML report after test execution.  
In this report you can view logs from any test and see failures at a glance.

<img width="900" alt="report-link" src="https://user-images.githubusercontent.com/18306778/136713954-911ddb21-64b0-4180-88f7-3724a4d24de8.png">


### Run tests against an existing 0Chain network locally
Requires BASH shell (UNIX, macOS, WSL) and [go](https://golang.org/dl/)  

Build or download the [zbox](https://github.com/0chain/zboxcli/tags) and [zwallet](https://github.com/0chain/zwalletcli/tags) CLIs, ensuring they are compatible with the network you wish to test.  
Modify the ```block_worker``` field in ```./tests/cli_tests/config/zbox_config.yaml``` to point to the network.   

To run the entire test suite (minus tests for known broken features) run:

```bash
cp $ZBOX_LOCATION ./tests/cli_tests/ # Copy zbox CLI to test folder
cp $ZWALLET_LOCATION ./tests/cli_tests/ # Copy zwallet CLI to test folder
cd ./tests/cli_tests/
go test -run "^Test[^___]*$" ./... -v
```
Debug logging can be achieved by running
```bash
DEBUG=true go test -run "^Test[^___]*$" ./... -v
```
Include tests for broken features as part of your test run by running
```bash
go test ./... -v
```
PS: Test suite execution will be slower when running locally vs the system tests pipeline.   
Output will also be less clear vs the system tests pipeline.   
Therefore, we recommend using an IDE such as [GoLand/Intellij IDEA](https://www.jetbrains.com/go/) to run/debug individual tests locally

## Run individual tests against local 0chain network

For developing new system tests for code still in developer branches, tests can be run against a locally running chain.
Typically, for a 0chain change you will have a PR for several modules that need to work
together. For example, `0chain`, `blobber`, `GoSDK`, `zboxclie` and `zwalletclie`.

The first step requires setting up a running chain using the GitHub branches from the PRs.
Use the instructions for building a [local chain 0chain](https://github.com/0chain/0chain#setup-network),
[add a few blobbers](https://github.com/0chain/blobber#building-and-starting-the-nodes).
Make sure you [stake the blobbers](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/storagesc/README.md#order).

For `zboxcli` and `zwalletcle` changes you need to first build the executable and copy into local
system test directory. For example:
```bash
cd zboxcli
make install
cp zbox ../system_test/tests/cli_tests/zbox
```

Make sure you have the correct system test branch. Now you need to edit `system_test/tests/cli_tests/config/zbox_config.yaml`
Edit the line `block_worker: https://dev.0chain.net/dns` to the appropriate setting for you, something like
```yaml
block_worker: http://192.168.1.100:9091
```
Finally, you need to add a `system_test/tests/cli_tests/config/sc_owner_wallet.json` file with the
following data
```json
{"client_id":"1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}
```

Now open the system_test project in [GoLand](https://www.jetbrains.com/go/),
you should now be able to run any of the `cli_tests` in debug.

## Handling test failures
The test suite/pipeline should pass when ran against a healthy network.   
If some tests fail, it is likely that a code issue has been introduced.  
Try running the same tests against another network to rule out environmental issues.  
If the failure persists, and you believe this to be a false positive, [contact the system tests team](https://0chain.slack.com/archives/C02AV6MKT36).

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.


## License
[MIT](https://choosealicense.com/licenses/mit/)
