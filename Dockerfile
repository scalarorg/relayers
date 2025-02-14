FROM rust:1.82-alpine3.20 as libbuilder
RUN apk add --no-cache git libc-dev
# Build bitcoin-vault lib
# Todo: select a specific version
RUN git clone https://github.com/scalarorg/bitcoin-vault.git
WORKDIR /bitcoin-vault
RUN cargo build -p vault -p macros -p ffi --release

# Build stage
FROM golang:1.23.3-alpine3.20 AS builder

ARG OS=linux
ARG ARCH=x86_64
ARG WASM=true
ARG WASMVM_VERSION=v2.1.3

# Install build dependencies
RUN apk add --no-cache --update \
    ca-certificates \
    git \
    make \
    build-base \
    linux-headers

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Use a compatible libwasmvm
# Alpine Linux requires static linking against muslc: https://github.com/CosmWasm/wasmd/blob/v0.33.0/INTEGRATION.md#prerequisites
RUN if [[ "${WASM}" == "true" ]]; then \
    wget https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.${ARCH}.a \
    -O /lib/libwasmvm_muslc.a && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.${ARCH}.a | cut -d ' ' -f 1); \
    fi

COPY --from=libbuilder /bitcoin-vault/target/release/libbitcoin_vault_ffi.* /usr/lib/

# Copy source code
COPY . .

# Build the application
RUN make MUSLC="${WASM}" WASM="${WASM}" IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" build
# RUN CGO_ENABLED=1 GOOS=${OS} GOARCH=${ARCH} go build -o /app/relayer ./main.go
# RUN CGO_ENABLED=1 go build -o /app/relayer ./main.go

# Final stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates bash jq

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/relayer /usr/bin/relayer
# Copy config files
COPY --from=builder /app/data/example ./data/example

# Create necessary directories
RUN mkdir -p data/local data/devnet data/testnet

EXPOSE 8080

ENTRYPOINT [ "relayer"]
