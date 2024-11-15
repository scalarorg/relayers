package scalar_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	ctx.OutputFormat = "json"
	err = ctx.PrintProto(resp.Account)
	assert.NoError(t, err)
	log.Info().Msgf("resp: %s", buf.String())
	var account map[string]any
	//var account auth.BaseAccount
	//account.Unmarshal(resp.Account.Value)

	//err = ctx.Codec.UnpackAny(resp.Account, &account)
	//log.Info().Msgf("UnpackAny Account: %+v", account) //-> missing
	// err = gogoproto.Unmarshal(resp.Account.Value, &account)
	//err = ctx.Codec.UnmarshalInterfaceJSON(buf.Bytes(), &account)
	// assert.NoError(t, err)
	json.Unmarshal(buf.Bytes(), &account)
	sequence, err := strconv.ParseUint(account["sequence"].(string), 10, 64)
	if err != nil {
		log.Error().Msgf("failed to parse sequence: %+v", err)
	}
	log.Info().Msgf("Sequence: %+v", sequence)
	// log.Info().Msgf("Json unmarshal: %+v", account.GetSequence())

	log.Info().Msgf("Account: %+v", account)
}
