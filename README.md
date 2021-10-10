# 0Chain System Tests

Black box test suite that tests the functionality of the 0Chain as an end user.

## Running tests

The tests run against a full 0Chain network.

### Deploying a new 0Chain network and running tests

The [CI pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can deploy a new instance of 0Chain for you with a custom set of docker images:
<img width="322" alt="Screenshot 2021-10-10 at 22 23 04" src="https://user-images.githubusercontent.com/18306778/136713487-db7ef096-cb11-4a33-9b29-302ffb5470df.png">

### Running tests against an existing 0Chain network

The [CI pipeline](https://github.com/0chain/system_test/actions/workflows/ci.yml) can also run tests against an existing 0Chain instance  
<img width="347" alt="Screenshot 2021-10-10 at 22 25 03" src="https://user-images.githubusercontent.com/18306778/136713492-fbeadfb0-51d7-4f59-90a0-34e72e9eafcb.png">



### Running tests locally
Modify ```block_worker``` in ```./tests/cli_tests/config/zbox_config.yaml```
to point to the network you wish to test.  Then run:

```bash
cp ./zbox ./tests/cli_tests/ # Copy zbox CLI to test folder
cp ./zwallet ./tests/cli_tests/ # Copy zwallet CLI to test folder
cd tests/cli_tests/
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