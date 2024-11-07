# relayers

Relayer for cross chain liquidity

## How to run locally

Duplicate and edit:

- From `.env.example` into `.env.local`
- From `data/example-env` into `data/local`
- From `example-runtime/chains` into `runtime/chains`

Then run:

```
go mod tidy
source .env.local && go run cmd/relayer/main.go
```

## How to use as a docker container

Prepare the files mentioned above, and review `compose.yml`. Then build and run the docker contrainer as following:

```
docker build -t scalarorg/xchains-relayer-go .
docker compose up -d
```

## About smart contract abi files

This repository already come with smart contract abi files.

Generate the contracts using abigen is stated below. This action only needed to do when there are changes in smart contracts. Otherwise, this can be skipped at default.

```
chmod +x generate_axelar_bindings.sh
./generate_axelar_bindings.sh
```
