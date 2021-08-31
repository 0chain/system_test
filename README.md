
# System tests

## Table of Contents

* [Bandwidth-marketplace](#bandwidth-marketplace)
  * [Presetting](#presetting)
  * [Configuration](#configuration)
    * [Test configuration](#test-configuration)
    * [Nodes configuration](#nodes-configuration)
  * [Run](#run)  

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