package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type BtcClient struct {
	globalConfig *config.Config
	btcConfig    *BtcNetworkConfig
	client       *rpcclient.Client
	scalarClient *scalar.Client
	dbAdapter    *db.DatabaseAdapter
	eventBus     *events.EventBus
}

type BtcClientInterface interface {
	SendTx(tx *wire.MsgTx, maxFeeRate *float64) (*chainhash.Hash, error)
	TestMempoolAccept(txs []*wire.MsgTx, maxFeeRatePerKb float64) ([]*btcjson.TestMempoolAcceptResult, error)
}

func NewBtcClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) ([]*BtcClient, error) {
	// Read Scalar config from JSON file
	if globalConfig == nil || globalConfig.ConfigPath == "" {
		return nil, fmt.Errorf("btc config path is not set")
	}
	btcCfgPath := fmt.Sprintf("%s/btc.json", globalConfig.ConfigPath)
	configs, err := config.ReadJsonArrayConfig[BtcNetworkConfig](btcCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read btc config from file: %s, %w", btcCfgPath, err)
	}
	btcClients := make([]*BtcClient, 0, len(configs))
	for _, btcConfig := range configs {
		if btcConfig.MempoolUrl == "" {
			panic(fmt.Sprintf("mempool url is not set for %s", btcConfig.Name))
		}
		if btcConfig.PrivateKey == "" {
			if config.GlobalConfig.BtcPrivateKey == "" {
				log.Warn().Msgf("btc private key is not set for %s", btcConfig.Name)
			} else {
				btcConfig.PrivateKey = config.GlobalConfig.BtcPrivateKey
			}
		}
		// Set max fee rate to 0.10 if not set
		if btcConfig.MaxFeeRate == 0 {
			btcConfig.MaxFeeRate = 0.10
		}
		client, err := NewBtcClientFromConfig(globalConfig, &btcConfig, dbAdapter, eventBus, scalarClient)
		if err != nil {
			log.Warn().Msgf("Failed to create btc client for %s: %v", btcConfig.Name, err)
			continue
		}
		globalConfig.AddChainConfig(config.IChainConfig(&btcConfig))
		btcClients = append(btcClients, client)
	}
	return btcClients, nil
}

func NewBtcClientFromConfig(globalConfig *config.Config, btcConfig *BtcNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) (*BtcClient, error) {
	// Configure connection
	connCfg := &rpcclient.ConnConfig{
		Host:         fmt.Sprintf("%s:%d", btcConfig.Host, btcConfig.Port),
		User:         btcConfig.User,
		Pass:         btcConfig.Password,
		HTTPPostMode: true,
		DisableTLS:   btcConfig.SSL == nil || !*btcConfig.SSL,
	}

	// Create new client
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create BTC client for network %s: %w", btcConfig.Network, err)
	}
	btcClient := &BtcClient{
		globalConfig: globalConfig,
		btcConfig:    btcConfig,
		client:       client,
		scalarClient: scalarClient,
		dbAdapter:    dbAdapter,
		eventBus:     eventBus,
	}
	return btcClient, nil
}

func (c *BtcClient) Start(ctx context.Context) error {
	//Subscribe to the event bus by string identity
	receiver := c.eventBus.Subscribe(c.btcConfig.GetId())
	for event := range receiver {
		go func() {
			err := c.handleEventBusMessage(event)
			if err != nil {
				log.Error().Msgf("failed to handle event bus message: %v", err)
			}
		}()
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

func (c *BtcClient) Config() *BtcNetworkConfig {
	return c.btcConfig
}

func (c *BtcClient) BroadcastTx(tx *wire.MsgTx, maxFeeRate *float64) (*chainhash.Hash, error) {
	// If testnet4, create Command then call c.RpcClient.SendCmd(cmd)
	if c.btcConfig.Network == "testnet4" {
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
func (c *BtcClient) BroadcastRawTx(signedPsbtHex string) (*chainhash.Hash, error) {
	cmd := c.creatSendRawTransactionCmd(signedPsbtHex, &c.btcConfig.MaxFeeRate)
	res := c.client.SendCmd(cmd)
	// Cast the response to FutureTestMempoolAcceptResult and call Receive
	future := rpcclient.FutureSendRawTransactionResult(res)
	return future.Receive()
}

func (c *BtcClient) TestMempoolRawTx(signedPsbtHex string) (*chainhash.Hash, error) {
	// First decode the hex string to MsgTx
	msgTx := wire.NewMsgTx(wire.TxVersion)
	txBytes, err := hex.DecodeString(signedPsbtHex)
	if err != nil {
		return nil, err
	}
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, err
	}

	// Test mempool accept
	results, err := c.TestMempoolAccept([]*wire.MsgTx{msgTx}, c.btcConfig.MaxFeeRate)
	if err != nil {
		return nil, err
	}

	// If accepted, return the transaction hash
	if len(results) > 0 && results[0].Allowed {
		hash := msgTx.TxHash()
		return &hash, nil
	}

	return nil, fmt.Errorf("transaction rejected: %v", results[0].RejectReason)
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
	if c.btcConfig.Network == "testnet4" {
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
