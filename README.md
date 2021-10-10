# System Tests

A black/grey box suite that tests the functionality of the 0Chain network as an end user via the CLI tools.

## Running tests

The tests require a full 0Chain network to be deployed and running in a healthy state.

### Deploying a new 0Chain network and running tests

The [System Tests Pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can deploy a new instance of 0Chain with a custom set of docker images and execute tests:    
<details>
  <summary><b>[Click to show screenshot]</b></summary>
<img width="322" alt="ci-deploy" src="https://user-images.githubusercontent.com/18306778/136713487-db7ef096-cb11-4a33-9b29-302ffb5470df.png">  
</details>

In this mode, keep the network url field blank  
0Chain will automatically be deployed to a free test slot at ```dev-[1-5].dev-0chain.net```.  
If tests fail, the network will stay available for debugging purposes, however uptime is not guaranteed as the network may be overridden by another test run.

### Running tests against an existing 0Chain network

The [System Tests Pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can also run tests against an existing 0Chain network  
<details>
  <summary><b>[Click to show screenshot]</b></summary>
<img width="347" alt="ci-predeployed" src="https://user-images.githubusercontent.com/18306778/136713492-fbeadfb0-51d7-4f59-90a0-34e72e9eafcb.png">  
</details>

In this mode, the docker image input fields will be ignored  
The network URL is address of the 0Chain network you wish to test, without a URL scheme or subdomain.  
eg. beta.0chain.net

### Report

The CI pipeline will generate an HTML report after test execution.  
In this report you can view logs from any test and see failures at a glance.

<img width="900" alt="report-link" src="https://user-images.githubusercontent.com/18306778/136713954-911ddb21-64b0-4180-88f7-3724a4d24de8.png">


### Running tests locally
Build or download the [zbox](https://github.com/0chain/zboxcli/tags) and [zwallet](https://github.com/0chain/zwalletcli/tags) CLIs, ensuring they are compatible with the network you wish to test.  
Modify the ```block_worker``` field in ```./tests/cli_tests/config/zbox_config.yaml``` to point to the network.   
The URL should be in the same format as the one accepted by the pipeline.   

To run the entire test suite (minus tests for known broken features) run:

```bash
cp ./zbox ./tests/cli_tests/ # Copy zbox CLI to test folder
cp ./zwallet ./tests/cli_tests/ # Copy zwallet CLI to test folder
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

## Reporting Failures
The test suite/pipeline should pass against a healthy network.   
If the tests begin to fail, it is likely that a code issue has been introduced.  
Try running the same tests against another network.  
If the failures persist and you believe them to be a false positive, raise an issue or [contact the system tests team](https://0chain.slack.com/archives/C02AV6MKT36).

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.


## License
[MIT](https://choosealicense.com/licenses/mit/)