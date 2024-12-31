#!/bin/bash

# Create output directory if it doesn't exist
mkdir -p pkg/contracts/generated

# Generate Go bindings for IScalarGateway
abigen --abi=pkg/clients/evm/contracts/abi/IScalarGateway.json \
       --pkg=contracts \
       --out=pkg/clients/evm/contracts/generated/gateway.go \
       --type=IScalarGateway

# Generate Go bindings for IScalarExecutable
abigen --abi=pkg/clients/evm/contracts/abi/IScalarExecutable.json \
       --pkg=contracts \
       --out=pkg/clients/evm/contracts/generated/executable.go \
       --type=IScalarExecutable

# Generate Go bindings for IScalarERC20CrossChain
abigen --abi=pkg/clients/evm/contracts/abi/IScalarERC20CrossChain.json \
       --pkg=contracts \
       --out=pkg/clients/evm/contracts/generated/erc20crosschain.go \
       --type=IScalarERC20CrossChain


echo "Go contract bindings generated successfully!"
