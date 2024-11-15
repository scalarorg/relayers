package btc

import (
	"github.com/scalarorg/relayers/pkg/clients/evm"
)

const COMPONENT_NAME = "BtcClient"

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
	PrivateKey string  `mapstructure:"private_key,omitempty"`
	Address    *string `mapstructure:"address,omitempty"`
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

type ExecuteParams struct {
	SourceChain      string
	SourceAddress    string
	ContractAddress  string
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex uint64
}

// Todo: When xchains core user separated module for handling btc execution data,
// we need to define new types here
type DecodedExecuteData = evm.DecodedExecuteData

var DecodeExecuteData = evm.DecodeExecuteData
