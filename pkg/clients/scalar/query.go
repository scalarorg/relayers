package scalar

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covenanttypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

type TokenInfoResponse struct {
	TokenAddress string
	Asset        string
	Response     *chainstypes.TokenInfoResponse
	Error        error
}

var (
	chainTokenInfos = map[string][]*TokenInfoResponse{}
)

func (c *Client) GetChainQueryServiceClient() (chainstypes.QueryServiceClient, error) {
	clientCtx, err := c.queryClient.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return chainstypes.NewQueryServiceClient(clientCtx), nil
}

func (c *Client) GetCovenantQueryClient() covenanttypes.QueryServiceClient {
	clientCtx, err := c.queryClient.GetClientCtx()
	if err != nil {
		return nil
	}
	return covenanttypes.NewQueryServiceClient(clientCtx)
}
func (c *Client) GetSymbol(ctx context.Context, chainId string, tokenAddress string) (string, error) {
	//Try get token info from cache
	if !strings.HasPrefix(tokenAddress, "0x") {
		tokenAddress = "0x" + tokenAddress
	}
	tokenInfos, ok := chainTokenInfos[chainId]
	var tokenResponse *TokenInfoResponse = nil
	if ok {
		for _, info := range tokenInfos {
			if info.TokenAddress == tokenAddress {
				tokenResponse = info
				break
			}
		}
	}
	//Response not found, make fist query to the node
	if tokenResponse == nil {
		client, err := c.GetChainQueryServiceClient()
		if err != nil {
			log.Warn().Err(err).Msgf("[ScalarClient] [GetSymbol] cannot get chain query client")
			return "", err
		}
		tokenRequest := chainstypes.TokenInfoRequest{
			Chain: chainId,
			FindBy: &chainstypes.TokenInfoRequest_Address{
				Address: tokenAddress,
			},
		}
		response, err := client.TokenInfo(ctx, &tokenRequest)
		tokenResponse = &TokenInfoResponse{
			TokenAddress: tokenAddress,
			Response:     response,
			Error:        err,
		}
		if response != nil {
			tokenResponse.Asset = response.Asset
		}
		chainTokenInfos[chainId] = append(tokenInfos, tokenResponse)
	}
	if tokenResponse.Error != nil {
		return "", tokenResponse.Error
	}
	return tokenResponse.Response.Asset, nil
}
func (c *Client) GetTokenInfo(ctx context.Context, chainId, symbol string) (*chainstypes.TokenInfoResponse, error) {
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetSymbol] cannot get chain query client")
		return nil, err
	}
	tokenRequest := chainstypes.TokenInfoRequest{
		Chain: chainId,
		FindBy: &chainstypes.TokenInfoRequest_Symbol{
			Symbol: symbol,
		},
	}
	response, err := client.TokenInfo(ctx, &tokenRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetTokenInfo] cannot get token info from scalar-core")
		return nil, err
	}
	return response, nil
}
func (c *Client) GetTokenContractAddressFromSymbol(ctx context.Context, chainId, symbol string) string {
	//Try get token info from cache
	tokenInfos, ok := chainTokenInfos[chainId]
	if ok {
		for _, info := range tokenInfos {
			if strings.EqualFold(info.Asset, symbol) {
				return info.Response.Address
			}
		}
	}
	//If not found, query from scalar-core
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetTokenContractAddressFromSymbol] cannot get chain query client")
		return ""
	}
	tokenRequest := chainstypes.TokenInfoRequest{
		Chain: chainId,
		FindBy: &chainstypes.TokenInfoRequest_Symbol{
			Symbol: symbol,
		},
	}
	response, err := client.TokenInfo(ctx, &tokenRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetTokenContractAddressFromSymbol] cannot get token info from scalar-core")
		return ""
	}
	chainTokenInfos[chainId] = append(tokenInfos, &TokenInfoResponse{
		TokenAddress: response.Address,
		Asset:        response.Asset,
		Response:     response,
		Error:        nil,
	})
	return response.Address
}
func (c *Client) GetCommand(chainName string, commandId string) (*chainstypes.CommandResponse, error) {
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetSymbol] cannot get chain query client")
		return nil, err
	}
	commandRequest := chainstypes.CommandRequest{
		Chain: chainName,
		ID:    commandId,
	}
	response, err := client.Command(context.Background(), &commandRequest)
	if err != nil {
		log.Warn().Err(err).Msgf("[ScalarClient] [GetCommandId] cannot get command")
		return nil, err
	}
	return response, nil
}

func (c *Client) GetCovenantGroups(ctx context.Context) ([]*covExported.CustodianGroup, error) {
	client := c.GetCovenantQueryClient()
	if client == nil {
		return nil, errors.New("covenant query client is nil")
	}
	response, err := client.Groups(ctx, &covenanttypes.GroupsRequest{})
	if err != nil {
		return nil, err
	}
	return response.Groups, nil
}
