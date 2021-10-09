#!/bin/bash
set -o pipefail
CONFIG_PATH=./zbox_config.yaml go test ./... -short -json -count=1 -timeout=30m | sed -r "/(=== (CONT|RUN|PAUSE).*)|(--- FAIL:.*)|(\"Test\":\".*\/[pP]arallel\")/d"