#!/bin/bash

# Create output directory if it doesn't exist
mkdir -p pkg/contracts/generated

# Generate Go bindings for IScalarGateway
abigen --abi=pkg/contracts/abi/IScalarGateway.json \
       --pkg=contracts \
       --out=pkg/contracts/generated/iscalargateway.go \
       --type=IScalarGateway

# Generate Go bindings for IScalarExecutable
abigen --abi=pkg/contracts/abi/IScalarExecutable.json \
       --pkg=contracts \
       --out=pkg/contracts/generated/iscalarexecutable.go \
       --type=IScalarExecutable

echo "Go contract bindings generated successfully!"
