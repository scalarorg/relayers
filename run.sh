#!/bin/bash

start() {
    source .env
    CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go run ./main.go --env ${1:-""}
}
 # example ./run.sh test clients/scalar
 # example ./run.sh test TestEvmClientWatchTokenSent clients/evm/client_test.go
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

getbtctx() {
    curl --user scalar --data-binary '{"jsonrpc": "1.0", "id": "curltest", "method": "getrawtransaction", "params": ["9640d60c9f53bdca7fe0520a276e5d7f7d33bd07773a2d7c8c462ac64480b5a8", true]}' -H 'content-type: text/plain;' http://testnet4.btc.scalar.org
}
$@
