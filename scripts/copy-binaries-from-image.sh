#!/usr/bin/env bash

container_id=$(docker create scalarorg/relayer:binaries)
docker cp "$container_id":/go/src/github.com/scalarorg/relayer/bin ./
docker rm -v "$container_id"
