#!/bin/bash
set -o pipefail
CONFIG_PATH=./zbox_config.yaml go test ./... -json -count=1 | sed -r "/(=== (CONT|RUN|PAUSE).*)|(--- FAIL:.*)|(\"Test\":\".*\/[pP]arallel\")/d"