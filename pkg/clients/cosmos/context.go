package cosmos

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/internal/codec"
)

// var _ grpc.ClientConn = &client.Context{}

func CreateClientContext(config *CosmosNetworkConfig) (*client.Context, error) {
	clientCtx := client.Context{
		ChainID: config.ChainID,
	}
	if config.RPCUrl != "" {
		log.Info().Msgf("Create rpcClient using RPC URL: %s", config.RPCUrl)
		clientCtx = clientCtx.WithNodeURI(config.RPCUrl)
		rpcClient, err := client.NewClientFromNode(config.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client: %w", err)
		}
		clientCtx = clientCtx.WithClient(rpcClient)
	}
	if config.Mnemonic != "" {
		_, addr, err := CreateAccountFromMnemonic(config.Mnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
		}
		clientCtx = clientCtx.WithFromAddress(addr)
	}
	clientCtx = clientCtx.WithCodec(codec.GetProtoCodec())
	clientCtx = clientCtx.WithOutputFormat("json")
	return &clientCtx, nil
}
