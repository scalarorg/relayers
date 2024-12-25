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
		ChainID: config.ID,
	}
	if config.RPCUrl != "" {
		log.Info().Msgf("Create rpcClient using RPC URL: %s", config.RPCUrl)
		clientCtx = clientCtx.WithNodeURI(config.RPCUrl)
	}
	if config.Mnemonic != "" {
		_, addr, err := CreateAccountFromMnemonic(config.Mnemonic, config.Bip44Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
		}
		clientCtx = clientCtx.WithFromAddress(addr)
	}
	clientCtx = clientCtx.WithCodec(codec.GetProtoCodec())
	clientCtx = clientCtx.WithOutputFormat("json")
	return &clientCtx, nil
}

type ClientContextOption func(*client.Context) error

func WithRpcClientCtx(rpcUrl string) ClientContextOption {
	return func(c *client.Context) error {
		rpcClient, err := client.NewClientFromNode(rpcUrl)
		if err != nil {
			return fmt.Errorf("failed to create RPC client: %w", err)
		}
		clientCtx := *c
		clientCtx = clientCtx.WithNodeURI(rpcUrl)
		clientCtx = clientCtx.WithClient(rpcClient)
		*c = clientCtx
		return nil
	}
}

func CreateClientContextWithOptions(config *CosmosNetworkConfig, opts ...ClientContextOption) (*client.Context, error) {
	clientCtx := client.Context{
		ChainID: config.ID,
	}
	for _, opt := range opts {
		err := opt(&clientCtx)
		if err != nil {
			return nil, err
		}
	}
	clientCtx = clientCtx.WithCodec(codec.GetProtoCodec())
	clientCtx = clientCtx.WithOutputFormat("json")
	return &clientCtx, nil
}
