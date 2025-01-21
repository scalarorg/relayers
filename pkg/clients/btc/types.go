package btc

import (
	"github.com/scalarorg/bitcoin-vault/go-utils/types"
	"github.com/scalarorg/relayers/pkg/clients/evm"
)

const COMPONENT_NAME = "BtcClient"

type SigningType int

const (
	CUSTODIAL_ONLY SigningType = iota
	USER_PROTOCOL
	USER_CUSTODIAL
	PROTOCOL_CUSTODIAL
)

type BtcNetworkConfig struct {
	Network    string  `mapstructure:"network"`
	ID         string  `mapstructure:"id"`
	ChainID    uint64  `mapstructure:"chain_id"`
	Name       string  `mapstructure:"name"`
	Type       string  `mapstructure:"type"`
	Host       string  `mapstructure:"host"`
	Port       int     `mapstructure:"port"`
	User       string  `mapstructure:"user"`
	Password   string  `mapstructure:"password"`
	SSL        *bool   `mapstructure:"ssl,omitempty"`
	MempoolUrl string  `mapstructure:"mempool_url,omitempty"`
	PrivateKey string  `mapstructure:"private_key,omitempty"`
	Address    *string `mapstructure:"address,omitempty"` //Taproot address. Todo: set it as parameter from scalar note
	MaxFeeRate float64 `mapstructure:"max_fee_rate,omitempty"`
}

func (c *BtcNetworkConfig) GetChainId() uint64 {
	return c.ChainID
}
func (c *BtcNetworkConfig) GetId() string {
	return c.ID
}
func (c *BtcNetworkConfig) GetName() string {
	return c.Name
}
func (c *BtcNetworkConfig) GetFamily() string {
	return types.ChainTypeBitcoin.String()
}

// Todo: When xchains core user separated module for handling btc execution data,
// we need to define new types here
type DecodedExecuteData = evm.DecodedExecuteData

var DecodeExecuteData = evm.DecodeExecuteData

type Utxo struct {
	Txid   string `json:"txid"`
	Vout   uint32 `json:"vout"`
	Status struct {
		Confirmed   bool   `json:"confirmed"`
		BlockHeight uint64 `json:"block_height"`
		BlockHash   string `json:"block_hash"`
		BlockTime   uint64 `json:"block_time"`
	} `json:"status"`
	Value uint64 `json:"value"`
}

type CommandOutPoint struct {
	BTCFeeOpts types.BTCFeeOpts
	RBF        bool
	OutPoint   types.UnstakingOutput
}
