# Scalar Relayer

A cross-chain relayer service that enables seamless token transfers between **Bitcoin**, **EVM chains**, or **non-EVM chains**, securely verified by Scalar Network. The relayer acts as a bridge by:

1. Monitoring source chains for transfer events
2. Requesting verification from Scalar Network
3. Creating and submitting execution payloads to destination chains

## 🌟 Key Features

- Bitcoin to EVM/non-EVM token transfers
- Cross-chain message verification
- Automated transaction processing and relay
- Support for multiple destination networks
- Secure verification through Scalar Network

## 🌟 How It Works

1. The relayer monitors source chains (e.g., Bitcoin) for specific transactions we call staking transactions
2. When an event is detected, it requests verification from Scalar Network
3. Upon successful verification, the relayer creates an execution payload on the payload chain
4. The payload is submitted to the destination network (EVM or non-EVM)
5. Transaction status and history are maintained in a PostgreSQL database

## 🌟 Architecture Components

### Electrum Client

- Listens to Bitcoin transactions related to Scalar vault using Electrum BTC indexer

### Bitcoin Client

- Handles connections with Bitcoin network
- Broadcasts staking and unstaking transactions to the Bitcoin network

### EVM Client

- Handles connections with EVM-compatible networks
- Manages smart contract interactions
- Processes cross-chain events and transactions

### Scalar Client

- Manages communication with Scalar Network
- Handles cross-chain message verification
- Processes state synchronization

### Database

- PostgreSQL database for persistent storage
- Stores transaction history, events, and relay status
- Maintains cross-chain state synchronization data

## 🌟 How to run locally

Duplicate and edit:

- From `data/example` into `data/local`

Copy `.env.example` into `.env` and edit the following variables:

```
APP_NAME=scalar-relayer
CONFIG_PATH=./data/local
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/relayer
SCALAR_MNEMONIC="ENTER YOUR SCALAR MNEMONIC HERE"
EVM_PRIVATE_KEY="ENTER YOUR EVM PRIVATE KEY HERE"
```

Build dependencies **bitcoin-vault** in repos: <https://github.com/scalarorg/bitcoin-vault>

Then copy the built binaries into `./lib` folder:

```
cp {BITCOIN_VAULT_FOLDER_PATH}/target/release/libbitcoin_vault_ffi.* ./lib/
```

Then run:

```bash
source .env
CGO_LDFLAGS="-L./lib -lbitcoin_vault_ffi" CGO_CFLAGS="-I./lib" go run ./main.go
```

## 🌟 How to use as a docker container

Prepare the files mentioned above, and review `compose.yml`. Then build and run the docker container as following:

```bash
make docker-image
docker compose up -d
```

## 🌟 Smart Contract Integration

This repository includes smart contract ABI files. Generate the contracts using abigen when there are changes in smart contracts:

```bash
./generate_bindings.sh
```
