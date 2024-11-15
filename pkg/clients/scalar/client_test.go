package scalar_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/rs/zerolog/log"

	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

const (
	cosmosAddress = "axelar17a5k3uxm2cj8te80yyaykgmqvfd7hh8rgjz7hk"
	scalarRpcUrl  = "http://localhost:26657"
)

var (
	protoCodec   = encoding.GetCodec(proto.Name)
	scalarConfig = &cosmos.CosmosNetworkConfig{
		ChainID: "scalar-testnet-1",
		RPCUrl:  scalarRpcUrl,
	}
	err        error
	clientCtx  *client.Context
	accAddress sdk.AccAddress
)

func TestMain(m *testing.M) {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("axelar", "axelarvaloper")
	clientCtx, err = scalar.CreateClientContext(scalarConfig)
	if err != nil {
		log.Error().Msgf("failed to create client context: %+v", err)
	}
	m.Run()
}
func TestAccountAddress(t *testing.T) {
	accAddress, err := sdk.AccAddressFromBech32(cosmosAddress)
	if err != nil {
		log.Error().Msgf("failed to get accAddress: %+v", err)
	}
	log.Info().Msgf("accAddress: %+v, string value %s", accAddress, accAddress.String())
	assert.Equal(t, accAddress.String(), cosmosAddress)
}

func TestCosmosGrpcClient(t *testing.T) {
	fmt.Println("TestCosmosGrpcClient")
	assert.NotNil(t, clientCtx)
	authClient := auth.NewQueryClient(clientCtx)
	assert.NotNil(t, authClient)
	assert.NotNil(t, protoCodec)
	resp, err := authClient.Account(context.Background(), &auth.QueryAccountRequest{Address: cosmosAddress})
	assert.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to query account: %+v", err)
		log.Error().Msgf("failed to query account: %+v", err)
	}
	buf := &bytes.Buffer{}
	ctx := clientCtx.WithOutput(buf)
	err = ctx.PrintProto(resp.Account)
	assert.NoError(t, err)
	log.Info().Msgf("resp: %s", buf.String())
	var account auth.BaseAccount
	err = ctx.Codec.UnpackAny(resp.Account, &account)
	assert.NoError(t, err)
	log.Info().Msgf("Base Account: %+v", &account)
}
