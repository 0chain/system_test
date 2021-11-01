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
go test ./... -v -short
```
Debug logging can be achieved by running
```bash
DEBUG=true go test ./... -v -short
```
Include tests for broken features as part of your test run by removing the '-short' flag
```bash
go test ./... -v
```
PS: Test suite execution may be slower when running locally vs the system tests pipeline.   
Output will also be less clear vs the system tests pipeline.   
Therefore, we recommend using an IDE such as [GoLand/Intellij IDEA](https://www.jetbrains.com/go/) to run/debug individual tests locally

## Handling test failures
The test suite/pipeline should pass when ran against a healthy network.   
If some tests fail, it is likely that a code issue has been introduced.  
Try running the same tests against another network to rule out environmental issues.  
If the failure persists, and you believe this to be a false positive, [contact the system tests team](https://0chain.slack.com/archives/C02AV6MKT36).

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.


## License
[MIT](https://choosealicense.com/licenses/mit/)

## Bandwidth marketplace

### Presetting

Nodes used in tests are running in docker containers, so before starting tests, you need to start docker.

See [docs](https://docs.docker.com/) for more info.

### Configuration

#### Test configuration

Main configuration of tests stores in `system_test/tests/bandwidth-marketplace/src/test/config.yaml` file.

If you want to change the docker image, you must configure `${node}.ref` section of the config file.

Also, you can manage the running of the tests cases. It defined by `cases` section.

#### Nodes configuration

Nodes are configured as usual. You can see config files in
`system_test/tests/bandwidth-marketplace/src/${node}/config` directory
and keys file in `system_test/tests/bandwidth-marketplace/src/${node}/keys` directory

### Run

Running is simple, just run inside the `tests/bandwidth-marketplace` package:

```shell
go test -tags bn256
```