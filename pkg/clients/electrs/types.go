package electrs

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a custom type that can parse time.Duration from JSON strings
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", s)
	}

	*d = Duration(duration)
	return nil
}

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// ToDuration converts Duration to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

type Config struct {
	//Electrum server host
	Host string
	//Electrum server port
	Port int
	//Electrum server user
	User string
	//Electrum server password
	Password string
	//Source chain - This must match with bridge config in the xchains core config. For example bitcoin-testnet4
	SourceChain string
	//Las Vault Tx's hash received from electrum server.
	//If this parameter is empty, server will start from the first vault tx from db.
	BatchSize int
	//Confirmations is the number of confirmations required for a vault transaction to be broadcast to the scalar for confirmation
	Confirmations int
	LastVaultTx   string
	//Connection timeout for dialing the electrum server
	DialTimeout Duration
	//Method timeout for electrum RPC calls
	MethodTimeout Duration
	//Ping interval for keeping the connection alive (-1 to disable)
	PingInterval Duration
	//Reconnection configuration
	MaxReconnectAttempts int
	ReconnectDelay       Duration
	EnableAutoReconnect  bool
}
