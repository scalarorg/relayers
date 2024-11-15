#!/bin/bash

start() {
    source .env
    go run ./main.go --env ${1:-""}
}
 # example ./run.sh test clients/scalar
 # example ./run.sh test clients/scalar/client_test.go
test() {
    if [ -n "$1" ]; then
        go test -timeout 10m ./pkg/${1} -v -count=1
    else
        go test -timeout 10m -v -count=1
    fi
}
$@
