package scalar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	axltypes "github.com/axelarnetwork/axelar-core/x/axelarnet/types"
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	//gogogrpc "github.com/cosmos/gogoproto/grpc"
	//pbgrpc "github.com/gogo/protobuf/grpc"
)

type QueryClient struct {
	clientCtx          *client.Context
	evmQueryClient     emvtypes.QueryServiceClient
	msgQueryClient     axltypes.MsgServiceClient
	accountQueryClient auth.QueryClient
}

func NewQueryClient(clientCtx *client.Context) *QueryClient {
	evmQueryClient := emvtypes.NewQueryServiceClient(clientCtx)
	msgQueryClient := axltypes.NewMsgServiceClient(clientCtx)
	accountQueryClient := auth.NewQueryClient(clientCtx)
	return &QueryClient{
		clientCtx:          clientCtx,
		evmQueryClient:     evmQueryClient,
		msgQueryClient:     msgQueryClient,
		accountQueryClient: accountQueryClient,
	}
}

func (c *QueryClient) QueryBatchedCommands(ctx context.Context, destinationChain string, batchedCommandId string) (*emvtypes.BatchedCommandsResponse, error) {
	req := &emvtypes.BatchedCommandsRequest{
		Chain: destinationChain,
		Id:    batchedCommandId,
	}
	resp, err := c.evmQueryClient.BatchedCommands(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query batched commands: %w", err)
	}
	return resp, nil
}

func (c *QueryClient) QueryPendingCommand(ctx context.Context, destinationChain string) ([]emvtypes.QueryCommandResponse, error) {
	req := &emvtypes.PendingCommandsRequest{
		Chain: destinationChain,
	}
	resp, err := c.evmQueryClient.PendingCommands(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commands: %w", err)
	}

	return resp.Commands, nil
}

func (c *QueryClient) QueryRouteMessageRequest(ctx context.Context, sender sdk.AccAddress, feegranter sdk.AccAddress, id string, payload string) (*axltypes.RouteMessageResponse, error) {
	req := &axltypes.RouteMessageRequest{
		Sender:     sender,
		ID:         id,
		Payload:    []byte(payload),
		Feegranter: feegranter,
	}
	resp, err := c.msgQueryClient.RouteMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query route message request: %w", err)
	}
	return resp, nil
}
func (c *QueryClient) QueryAccount(ctx context.Context, address sdk.AccAddress) (map[string]any, error) {
	req := &auth.QueryAccountRequest{Address: address.String()}
	resp, err := c.accountQueryClient.Account(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}
	if resp.Account == nil {
		return nil, fmt.Errorf("account value is nil")
	}
	buf := &bytes.Buffer{}
	clientCtx := c.clientCtx.WithOutput(buf)
	err = clientCtx.PrintProto(resp.Account)
	// var account auth.BaseAccount
	// err = account.Unmarshal(resp.Account.Value)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	// }
	var account map[string]any
	err = json.Unmarshal(buf.Bytes(), &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}
	return account, nil
}

// func (c *NetworkClient) QueryTx(ctx context.Context, hash []byte) (*ctypes.ResultTx, error) {
// 	// Query by hash
// 	res, err := c.rpcClient.Tx(ctx, hash, false)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &ctypes.ResultTx{
// 		Hash:     hash,
// 		Height:   res.Height,
// 		Index:    res.Index,
// 		TxResult: res.TxResult,
// 		Tx:       res.Tx,
// 	}, nil
// }

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
