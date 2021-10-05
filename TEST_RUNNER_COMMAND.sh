#!/bin/bash
[[ $# -eq 0 ]] && args="./..." || args=$@
set -o pipefail
CONFIG_PATH=./zbox_config.yaml go test $args -json -count=1 | sed -r "/(=== (CONT|RUN|PAUSE).*)|(--- FAIL:.*)|(\"Test\":\".*\/[pP]arallel\")/d"