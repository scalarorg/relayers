package btc

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
)

var BtcClients []*BtcClient

type BtcClient struct {
	client *rpcclient.Client
	config *config.BtcNetworkConfig
}

type BtcClientInterface interface {
	SendTx(tx *wire.MsgTx, maxFeeRate *float64) (*chainhash.Hash, error)
	TestMempoolAccept(txs []*wire.MsgTx, maxFeeRatePerKb float64) ([]*btcjson.TestMempoolAcceptResult, error)
}

func NewBtcClients(configs []config.BtcNetworkConfig) error {
	for _, cfg := range configs {
		// Create a new instance of config for each client
		clientConfig := cfg

		// Configure connection
		connCfg := &rpcclient.ConnConfig{
			Host:         fmt.Sprintf("%s:%d", clientConfig.Host, clientConfig.Port),
			User:         clientConfig.User,
			Pass:         clientConfig.Password,
			HTTPPostMode: true,
			DisableTLS:   clientConfig.SSL == nil || !*clientConfig.SSL,
		}

		// Create new client
		client, err := rpcclient.New(connCfg, nil)
		if err != nil {
			return fmt.Errorf("failed to create BTC client for network %s: %w", clientConfig.Network, err)
		}

		BtcClients = append(BtcClients, &BtcClient{
			client: client,
			config: &clientConfig,
		})
	}

	if len(BtcClients) == 0 {
		return fmt.Errorf("no BTC client was initialized")
	}
	return nil
}

// GetTransaction retrieves detailed information about a transaction given its ID
// func (c *BtcClient) GetTransaction(txID string) (*btcjson.GetTransactionResult, error) {
// 	result, err := c.client.RawRequest("gettransaction", []json.RawMessage{[]byte(`"` + txID + `"`)})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get transaction info: %w", err)
// 	}

// 	var tx btcjson.GetTransactionResult
// 	if err := json.Unmarshal(result, &tx); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
// 	}
// 	return &tx, nil
// }

func (c *BtcClient) Config() *config.BtcNetworkConfig {
	return c.config
}

func (c *BtcClient) SendTx(tx *wire.MsgTx, maxFeeRate *float64) (*chainhash.Hash, error) {
	// If testnet4, create Command then call c.RpcClient.SendCmd(cmd)
	if c.config.Network == "testnet4" {
		rawTx, err := CreateRawTx(tx)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Send rawTx: %s\n", rawTx)
		cmd := c.creatSendRawTransactionCmd(rawTx, maxFeeRate)
		res := c.client.SendCmd(cmd)
		// Cast the response to FutureTestMempoolAcceptResult and call Receive
		future := rpcclient.FutureSendRawTransactionResult(res)
		return future.Receive()
	} else {
		// Otherwise, call c.RpcClient.SendRawTransaction(tx, true)x
		return c.client.SendRawTransaction(tx, true)
	}
}

// if maxFeeRate is not nil, set the feeSetting parameter
// otherwise, don't set the feeSetting parameter use default value which is set by bitcoind 0.10
func (c *BtcClient) creatSendRawTransactionCmd(rawTxHex string, maxFeeRate *float64) *btcjson.SendRawTransactionCmd {
	if maxFeeRate != nil {
		return btcjson.NewBitcoindSendRawTransactionCmd(rawTxHex, *maxFeeRate)
	}
	return &btcjson.SendRawTransactionCmd{
		HexTx:      rawTxHex,
		FeeSetting: nil,
	}
}
func (c *BtcClient) TestMempoolAccept(txs []*wire.MsgTx, maxFeeRatePerKb float64) ([]*btcjson.TestMempoolAcceptResult, error) {
	if c.config.Network == "testnet4" {
		// Add some checks to make sure the txs are valid
		rawTxns, err := CreateRawTxs(txs)
		if err != nil {
			return nil, err
		}
		res := c.client.SendCmd(btcjson.NewTestMempoolAcceptCmd(rawTxns, maxFeeRatePerKb))
		// Cast the response to FutureTestMempoolAcceptResult and call Receive
		future := rpcclient.FutureTestMempoolAcceptResult(res)
		return future.Receive()
	} else {
		return c.client.TestMempoolAccept(txs, maxFeeRatePerKb)
	}
}

func CreateRawTx(tx *wire.MsgTx) (string, error) {
	// Serialize the transaction and convert to hex string.
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	// TODO(yy): add similar checks found in `BtcDecode` to
	// `BtcEncode` - atm it just serializes bytes without any
	// bitcoin-specific checks.
	if err := tx.Serialize(buf); err != nil {
		return "", err
	}
	// Sanity check the provided tx is valid, which can be removed
	// once we have similar checks added in `BtcEncode`.
	//
	// NOTE: must be performed after buf.Bytes is copied above.
	//
	// TODO(yy): remove it once the above TODO is addressed.
	// if err := tx.Deserialize(buf); err != nil {
	// 	err = fmt.Errorf("%w: %v", rpcclient.ErrInvalidParam, err)
	// 	return "", err
	// }
	return hex.EncodeToString(buf.Bytes()), nil
}

func CreateRawTxs(txns []*wire.MsgTx) ([]string, error) {
	// Iterate all the transactions and turn them into hex strings.
	rawTxns := make([]string, 0, len(txns))
	for _, tx := range txns {
		rawTx, err := CreateRawTx(tx)
		if err != nil {
			return nil, err
		}
		rawTxns = append(rawTxns, rawTx)

	}

	return rawTxns, nil
}
