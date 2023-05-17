#!/bin/bash
set -o pipefail
[ -z "$1" ] && TESTS_TO_RUN="-run ^\QTestAllocation\E$/^\QCreate_+_Upload_+_Cancel,_equal_read_price_0.1\E$ ./..." || TESTS_TO_RUN=$1
CONFIG_PATH=./zbox_config.yaml go test -timeout=60m $TESTS_TO_RUN -json -count=1 | sed -r "/(=== (CONT|RUN|PAUSE).*)|(--- FAIL:.*)|(\"Test\":\".*\/[pP]arallel\")/d"