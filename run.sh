#!/bin/bash

start() {
    source .env
    CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go run ./main.go --env ${1:-""}
}
 # example ./run.sh test clients/scalar
 # example ./run.sh test clients/scalar/client_test.go
test() {
    OPTIONS="-timeout 10m -v -count=1"

    if [ -n "$1" ]; then
        OPTIONS="${OPTIONS} -run ^${1}$"
        if [ -n "$2" ]; then
            OPTIONS="${OPTIONS} ./pkg/${2}"
        fi
    fi
    CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go test ${OPTIONS}
}
$@
