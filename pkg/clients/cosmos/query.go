package cosmos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/rs/zerolog/log"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	protypes "github.com/scalarorg/scalar-core/x/protocol/types"
	scalartypes "github.com/scalarorg/scalar-core/x/scalarnet/types"
	//gogogrpc "github.com/cosmos/gogoproto/grpc"
	//pbgrpc "github.com/gogo/protobuf/grpc"
)

type QueryClient struct {
	clientCtx *client.Context
}

func NewQueryClient(clientCtx *client.Context) *QueryClient {
	return &QueryClient{
		clientCtx: clientCtx,
	}
}

func (c *QueryClient) GetClientCtx() (*client.Context, error) {
	//Lazily create clientCtx based on Client pointer
	if c.clientCtx.NodeURI != "" && c.clientCtx.Client == nil {
		rpcClient, err := client.NewClientFromNode(c.clientCtx.NodeURI)
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client: %w", err)
		}
		ctx := c.clientCtx.WithClient(rpcClient)
		c.clientCtx = &ctx
	}
	return c.clientCtx, nil
}
func (c *QueryClient) GetAuthQueryClient() (auth.QueryClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return auth.NewQueryClient(clientCtx), nil
}

func (c *QueryClient) GetChainQueryServiceClient() (chainstypes.QueryServiceClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return chainstypes.NewQueryServiceClient(clientCtx), nil
}

func (c *QueryClient) GetProtocolQueryClient() (protypes.QueryClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return protypes.NewQueryClient(clientCtx), nil
}

func (c *QueryClient) GetScalarnetQueryClient() (scalartypes.QueryServiceClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return scalartypes.NewQueryServiceClient(clientCtx), nil
}

func (c *QueryClient) GetMsgServiceClient() (scalartypes.MsgServiceClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return scalartypes.NewMsgServiceClient(clientCtx), nil
}

func (c *QueryClient) GetTxServiceClient() (tx.ServiceClient, error) {
	clientCtx, err := c.GetClientCtx()
	if err != nil {
		return nil, err
	}
	return tx.NewServiceClient(clientCtx), nil
}

func (c *QueryClient) QueryBatchedCommands(ctx context.Context, destinationChain string, batchedCommandId string) (*chainstypes.BatchedCommandsResponse, error) {
	req := &chainstypes.BatchedCommandsRequest{
		Chain: destinationChain,
		Id:    batchedCommandId,
	}
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.BatchedCommands(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("query batched commands with id %s: %w", batchedCommandId, err)
	}
	return resp, nil
}

func (c *QueryClient) QueryCommand(ctx context.Context, destinationChain string, commandId string) (*chainstypes.CommandResponse, error) {
	req := &chainstypes.CommandRequest{
		Chain: destinationChain,
		ID:    commandId,
	}
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.Command(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query command with id %s: %w", commandId, err)
	}
	return resp, nil
}

func (c *QueryClient) QueryPendingCommands(ctx context.Context, destinationChain string) ([]chainstypes.QueryCommandResponse, error) {
	req := &chainstypes.PendingCommandsRequest{
		Chain: destinationChain,
	}
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.PendingCommands(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commands: %w", err)
	}

	return resp.Commands, nil
}

func (c *QueryClient) QueryRouteMessageRequest(ctx context.Context, sender sdk.AccAddress, feegranter sdk.AccAddress, id string, payload string) (*scalartypes.RouteMessageResponse, error) {
	req := &scalartypes.RouteMessageRequest{
		Sender:     sender,
		ID:         id,
		Payload:    []byte(payload),
		Feegranter: feegranter,
	}
	client, err := c.GetMsgServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.RouteMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query route message request: %w", err)
	}
	return resp, nil
}
func (c *QueryClient) QueryAccount(ctx context.Context, address sdk.AccAddress) (*auth.BaseAccount, error) {
	req := &auth.QueryAccountRequest{Address: address.String()}
	client, err := c.GetAuthQueryClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.Account(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}
	if resp.Account == nil {
		return nil, fmt.Errorf("account value is nil")
	}
	var account auth.BaseAccount
	err = c.UnmarshalAccount(resp, &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}
	return &account, nil
}

// Todo: Add code for more correct unmarshal
func (c *QueryClient) UnmarshalAccount(resp *auth.QueryAccountResponse, account *auth.BaseAccount) error {
	// err = account.Unmarshal(resp.Account.Value)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	// }
	buf := &bytes.Buffer{}
	clientCtx := c.clientCtx.WithOutput(buf)
	err := clientCtx.PrintProto(resp.Account)
	if err != nil {
		return fmt.Errorf("failed to print proto: %w", err)
	}
	var accountMap map[string]any
	err = json.Unmarshal(buf.Bytes(), &accountMap)
	if err != nil {
		return fmt.Errorf("failed to unmarshal account: %w", err)
	}
	log.Debug().Any("accountMap", accountMap).Msgf("[QueryClient] [UnmarshalAccount]")
	account.Address = accountMap["address"].(string)
	account.AccountNumber, err = strconv.ParseUint(accountMap["account_number"].(string), 10, 64)
	if err != nil {
		log.Error().Msgf("failed to parse account number: %+v", err)
	}
	account.Sequence, err = strconv.ParseUint(accountMap["sequence"].(string), 10, 64)
	if err != nil {
		log.Error().Msgf("failed to parse sequence: %+v", err)
	}
	//pubKey := secp256k1.PubKey{}
	//pubKey.Key = accountMap["public_key"].(map[string]any)["key"].(string)
	//account.PubKey = &pubKey
	return nil
}

func (c *QueryClient) QueryTx(ctx context.Context, txHash string) (*sdk.TxResponse, error) {
	// Query by hash
	req := &tx.GetTxRequest{
		Hash: txHash,
	}
	client, err := c.GetTxServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	res, err := client.GetTx(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query tx: %w", err)
	}
	return res.GetTxResponse(), nil
}

func (c *QueryClient) QueryActivedChains(ctx context.Context) ([]string, error) {
	req := &chainstypes.ChainsRequest{
		Status: chainstypes.Activated,
	}
	client, err := c.GetChainQueryServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}
	resp, err := client.Chains(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query chains: %w", err)
	}
	chains := make([]string, len(resp.Chains))
	for ind, chain := range resp.Chains {
		chains[ind] = chain.String()
	}
	return chains, nil
}

// func (c *NetworkClient) QueryBalance(ctx context.Context, addr sdk.AccAddress) (*sdk.Coins, error) {
// 	// Create gRPC connection
// 	grpcConn, err := grpc.Dial(
// 		// c.rpcEndpoint,
// 		"localhost:9090",
// 		grpc.WithInsecure(),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
// 	}
// 	defer grpcConn.Close()

// 	// Create bank query client
// 	bankClient := banktypes.NewQueryClient(grpcConn)

// 	// Query all balances
// 	balanceResp, err := bankClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
// 		Address: addr.String(),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query balance: %w", err)
// 	}

// 	return &balanceResp.Balances, nil
// }
