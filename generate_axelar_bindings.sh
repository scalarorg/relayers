#!/bin/bash

# Create output directory if it doesn't exist
mkdir -p pkg/contracts/generated

# Generate Go bindings for IAxelarGateway
abigen --abi=pkg/contracts/abi/IAxelarGateway.json \
       --pkg=contracts \
       --out=pkg/contracts/generated/iaxelargateway.go \
       --type=IAxelarGateway

# Generate Go bindings for IAxelarExecutable
abigen --abi=pkg/contracts/abi/IAxelarExecutable.json \
       --pkg=contracts \
       --out=pkg/contracts/generated/iaxelarexecutable.go \
       --type=IAxelarExecutable

echo "Go contract bindings generated successfully!"
