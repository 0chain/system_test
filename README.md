# System Tests

Black box test suite that tests the functionality of the 0Chain as an end user via the CLI tools.

## Running tests

The tests run against a full 0Chain network.

### Deploying a new 0Chain network and running tests

The [CI pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can deploy a new instance of 0Chain for you with a custom set of docker images:    

<img width="322" alt="ci-deploy" src="https://user-images.githubusercontent.com/18306778/136713487-db7ef096-cb11-4a33-9b29-302ffb5470df.png">  

Ensure that the network URL is blank, and 0Chain will automatically be deployed to dev-**n**.dev-0chain.net where **n** is a free test slot.  
This network will stay available for debugging purposes if tests fail, but it's uptime is not guaranteed as it may be redeployed to by a future test run.  
The network will be torn down when tests pass, or failing that on a nightly basis.

### Running tests against an existing 0Chain network

The [CI pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can also run tests against an existing 0Chain instance  

<img width="347" alt="ci-predeployed" src="https://user-images.githubusercontent.com/18306778/136713492-fbeadfb0-51d7-4f59-90a0-34e72e9eafcb.png">  

Ensure that the network URL is the hostname of the 0Chain network you wish to use, without a URL scheme or subdomain.

### Report

The CI pipeline will generate an HTML report after test execution.  
In this report you can view logs from any test and see failures at a glance.

<img width="900" alt="report-link" src="https://user-images.githubusercontent.com/18306778/136713954-911ddb21-64b0-4180-88f7-3724a4d24de8.png">


### Running tests locally
Build or download the [zbox](https://github.com/0chain/zboxcli/tags) and [zwallet](https://github.com/0chain/zwalletcli/tags) CLIs for your system  
Modify ```block_worker``` in ```./tests/cli_tests/config/zbox_config.yaml```
to point to the network you wish to test.   
The URL should be in the same format as the one accepted by the CI.   
Then run:

```bash
cp ./zbox ./tests/cli_tests/ # Copy zbox CLI to test folder
cp ./zwallet ./tests/cli_tests/ # Copy zwallet CLI to test folder
cd ./tests/cli_tests/
go test ./... -v
```
Debug logging can be achieved by running
```bash
DEBUG=true go test ./... -v
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.


## License
[MIT](https://choosealicense.com/licenses/mit/)