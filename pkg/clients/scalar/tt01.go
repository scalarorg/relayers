package scalar_clients

import (
	"context"

	axelartypes "github.com/axelarnetwork/axelar-core/x/axelarnet/types"
	"github.com/cosmos/cosmos-sdk/client"
	"google.golang.org/grpc"
)

// QueryClient is similar to AxelarQueryClientType
type AxelarQueryClient struct {
	axelartypes.QueryClient
	ctx context.Context
}

// Create a new query client
func NewAxelarQueryClient(grpcConn *grpc.ClientConn) *AxelarQueryClient {
	return &AxelarQueryClient{
		QueryClient: axelartypes.NewQueryClient(grpcConn),
		ctx:         context.Background(),
	}
}

// For signing transactions (similar to AxelarSigningClient)
type AxelarTxClient struct {
	clientCtx client.Context
}

func NewAxelarTxClient(clientCtx client.Context) *AxelarTxClient {
	return &AxelarTxClient{
		clientCtx: clientCtx,
	}
}
