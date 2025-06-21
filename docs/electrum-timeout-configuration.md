# Electrum Client Timeout Configuration

## Overview

The Electrum client now supports configurable timeouts to handle slow server responses and prevent "context deadline exceeded" errors.

## Configuration Options

### Timeout Fields

Add these fields to your `electrum.json` configuration file:

- `dial_timeout`: Connection timeout for dialing the electrum server (default: 10s)
- `method_timeout`: Method timeout for electrum RPC calls (default: 30s)  
- `ping_interval`: Ping interval for keeping the connection alive (default: -1s, disabled)

### Example Configuration

```json
[
  {
    "host": "your-electrum-server.com",
    "port": 50001,
    "user": "your-username",
    "password": "your-password",
    "source_chain": "bitcoin|4",
    "batch_size": 1,
    "confirmations": 1,
    "last_vault_tx": "",
    "dial_timeout": "10s",
    "method_timeout": "30s",
    "ping_interval": "-1s"
  }
]
```

## Timeout Values

### Duration Format

Timeouts are specified using Go's duration format:

- `"10s"` - 10 seconds
- `"30s"` - 30 seconds  
- `"1m"` - 1 minute
- `"5m"` - 5 minutes
- `"-1s"` - Disabled (for ping_interval)

### Recommended Values

- **dial_timeout**: 10-30 seconds (depends on network latency)
- **method_timeout**: 30-60 seconds (depends on server performance)
- **ping_interval**: -1 (disabled) or 30s-60s if needed

## Troubleshooting

### Context Deadline Exceeded Errors

If you see errors like:

```
[ElectrumClient] [BlockchainHeaderHandler] Failed to receive block chain header error="context deadline exceeded"
[ElectrumClient] [HandleValueBlockWithVaultTxs] Failed to receive vault transaction: context deadline exceeded
```

**Solution**: Increase the `method_timeout` value in your configuration.

### Connection Timeout Errors

If you see connection timeout errors:

```
Failed to connect to electrum server at host:port
```

**Solution**: Increase the `dial_timeout` value in your configuration.

## Migration

If you don't specify timeout values, the client will use these defaults:

- `dial_timeout`: 10 seconds
- `method_timeout`: 30 seconds  
- `ping_interval`: -1 (disabled)

These defaults should resolve most timeout issues that were occurring with the previous 1-second timeout.
