package btc

import (
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
	PrivateKey string  `mapstructure:"private_key,omitempty"`
	Address    *string `mapstructure:"address,omitempty"`
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

// Todo: When xchains core user separated module for handling btc execution data,
// we need to define new types here
type DecodedExecuteData = evm.DecodedExecuteData

var DecodeExecuteData = evm.DecodeExecuteData
