# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/relayer ./cmd/relayer/main.go

# Final stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/relayer /usr/bin/relayer
# Copy config files
COPY --from=builder /app/data/example-env ./data/example-env

# Create necessary directories
RUN mkdir -p data/local data/devnet data/testnet

EXPOSE 8080

ENTRYPOINT [ "relayer"]
