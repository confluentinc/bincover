#!/usr/bin/env bash

set -e

: "${GO:=go}"

goversion=$($GO version)

$GO mod tidy

rc=0
diff=$(git diff --exit-code -- go.mod go.sum) || rc=$?

if [ $rc -ne 0 ]; then
    echo "Found some differences when running 'go mod tidy'"
    echo "**********"
    echo "$diff"
    echo "**********"
    echo "Please ensure you are using the correct version of go ($goversion), run 'go mod tidy', and commit the changes"
fi

exit $rc
