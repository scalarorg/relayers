#!/bin/bash

source .env
go run ./main.go --env ${1:-""}