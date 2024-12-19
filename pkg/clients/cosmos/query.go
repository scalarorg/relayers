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
	emvtypes "github.com/scalarorg/scalar-core/x/evm/types"
	scalartypes "github.com/scalarorg/scalar-core/x/scalarnet/types"
	//gogogrpc "github.com/cosmos/gogoproto/grpc"
	//pbgrpc "github.com/gogo/protobuf/grpc"
)

type QueryClient struct {
	clientCtx             *client.Context
	EvmQueryServiceClient emvtypes.QueryServiceClient
	MsgServiceClient      scalartypes.MsgServiceClient
	TxServiceClient       tx.ServiceClient
	AccountQueryClient    auth.QueryClient
}

func NewQueryClient(clientCtx *client.Context) *QueryClient {
	evmQueryServiceClient := emvtypes.NewQueryServiceClient(clientCtx)
	msgServiceClient := scalartypes.NewMsgServiceClient(clientCtx)
	accountQueryClient := auth.NewQueryClient(clientCtx)
	txServiceClient := tx.NewServiceClient(clientCtx)
	return &QueryClient{
		clientCtx:             clientCtx,
		EvmQueryServiceClient: evmQueryServiceClient,
		MsgServiceClient:      msgServiceClient,
		TxServiceClient:       txServiceClient,
		AccountQueryClient:    accountQueryClient,
	}
}

func (c *QueryClient) GetClientCtx() *client.Context {
	return c.clientCtx
}

func (c *QueryClient) QueryBatchedCommands(ctx context.Context, destinationChain string, batchedCommandId string) (*emvtypes.BatchedCommandsResponse, error) {
	req := &emvtypes.BatchedCommandsRequest{
		Chain: destinationChain,
		Id:    batchedCommandId,
	}
	resp, err := c.EvmQueryServiceClient.BatchedCommands(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query batched commands: %w", err)
	}
	return resp, nil
}

func (c *QueryClient) QueryPendingCommand(ctx context.Context, destinationChain string) ([]emvtypes.QueryCommandResponse, error) {
	req := &emvtypes.PendingCommandsRequest{
		Chain: destinationChain,
	}
	resp, err := c.EvmQueryServiceClient.PendingCommands(ctx, req)
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
	resp, err := c.MsgServiceClient.RouteMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query route message request: %w", err)
	}
	return resp, nil
}
func (c *QueryClient) QueryAccount(ctx context.Context, address sdk.AccAddress) (*auth.BaseAccount, error) {
	req := &auth.QueryAccountRequest{Address: address.String()}
	resp, err := c.AccountQueryClient.Account(ctx, req)
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
	log.Debug().Msgf("accountMap: %v", accountMap)
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
	res, err := c.TxServiceClient.GetTx(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query tx: %w", err)
	}
	return res.GetTxResponse(), nil
}

func (c *QueryClient) QueryActivedChains(ctx context.Context) ([]string, error) {
	req := &emvtypes.ChainsRequest{
		Status: emvtypes.Activated,
	}
	resp, err := c.EvmQueryServiceClient.Chains(ctx, req)
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
