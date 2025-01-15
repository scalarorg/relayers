package scalar

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
)

func (c *Client) GetSymbol(ctx context.Context, chainId string, tokenAddress string) (string, error) {
	client, err := c.queryClient.GetChainQueryServiceClient()
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetSymbol] cannot get chain query client")
		return "", err
	}
	if !strings.HasPrefix(tokenAddress, "0x") {
		tokenAddress = "0x" + tokenAddress
	}
	tokenRequest := chainstypes.TokenInfoRequest{
		Chain: chainId,
		FindBy: &chainstypes.TokenInfoRequest_Address{
			Address: tokenAddress,
		},
	}
	response, err := client.TokenInfo(ctx, &tokenRequest)
	if err != nil {
		return "", err
	}
	return response.Asset, nil
}

func (c *Client) GetCommand(sourceChain string, commandId string) (*chainstypes.CommandResponse, error) {
	client, err := c.queryClient.GetChainQueryServiceClient()
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetSymbol] cannot get chain query client")
		return nil, err
	}
	commandRequest := chainstypes.CommandRequest{
		Chain: sourceChain,
		ID:    commandId,
	}
	response, err := client.Command(context.Background(), &commandRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetCommandId] cannot get command")
		return nil, err
	}
	return response, nil
}
