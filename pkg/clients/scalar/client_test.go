package scalar_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/utils"
	covenanttypes "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

const (
	cosmosAddress        = "axelar17a5k3uxm2cj8te80yyaykgmqvfd7hh8rgjz7hk"
	scalarRpcUrl         = "http://localhost:26657"
	chainNameBtcTestnet4 = "bitcoin-testnet4"
)

var (
	protoCodec          = encoding.GetCodec(proto.Name)
	DefaultGlobalConfig = config.Config{}
	ScalarNetworkConfig = cosmos.CosmosNetworkConfig{
		ChainID:       73475,
		ID:            "cosmos|73475",
		Name:          "scalar-testnet-1",
		Denom:         "scalar",
		RPCUrl:        "http://localhost:26657",
		GasPrice:      0.001,
		LCDUrl:        "http://localhost:2317",
		WSUrl:         "ws://localhost:26657/websocket",
		MaxRetries:    3,
		RetryInterval: int64(1000),
		BroadcastMode: "sync",
		Mnemonic:      "",
	}
	err           error
	clientCtx     *client.Context
	queryClient   *cosmos.QueryClient
	accAddress    sdk.AccAddress
	networkClient *cosmos.NetworkClient
)

func TestMain(m *testing.M) {
	config := types.GetConfig()
	err := godotenv.Load("../../../.env.test")
	if err != nil {
		log.Error().Err(err).Msg("Error loading .env.test file: %v")
	}
	ScalarNetworkConfig.Mnemonic = os.Getenv("SCALAR_MNEMONIC")
	config.SetBech32PrefixForAccount("scalar", "scalarvaloper")
	clientCtx, err = cosmos.CreateClientContext(&ScalarNetworkConfig)
	if err != nil {
		log.Error().Msgf("failed to create client context: %+v", err)
	}
	queryClient = cosmos.NewQueryClient(clientCtx)
	txConfig := tx.NewTxConfig(codec.GetProtoCodec(), tx.DefaultSignModes)
	networkClient, err = cosmos.NewNetworkClient(&ScalarNetworkConfig, queryClient, txConfig)
	if err != nil {
		log.Error().Msgf("failed to create network client: %+v", err)
	}
	m.Run()
}

func TestQueryRedeemSession(t *testing.T) {
	groupUid := "bffb71bf819ae4cb65188905ac54763a09144bc3a0629808d7142dd5dbd98693"
	groupBytes32, err := utils.DecodeGroupUid(groupUid)
	require.NoError(t, err)
	clientCtx, err := queryClient.GetClientCtx()
	require.NoError(t, err)
	queryClient := covenanttypes.NewQueryServiceClient(clientCtx)
	session, err := queryClient.RedeemSession(context.Background(), &covenanttypes.RedeemSessionRequest{
		UID: groupBytes32[:],
	})
	require.NoError(t, err)
	t.Logf("Session %++v", session.Session.String())

}
func TestConfirmRedeemTx(t *testing.T) {
	broadcaster := scalar.NewBroadcaster(networkClient,
		scalar.NewPendingCommands(),
		time.Second*10,
		10)
	broadcaster.Start(context.Background())
	redeemTxs := []string{"0x8d9eb539206db1e3f4dc4c26419616fb90723a667d5b1c126e2fc4f9c4a6fc98"}
	broadcaster.ConfirmEvmTxs("evm|11155111", redeemTxs)
	select {}

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
	log.Info().Msgf("Account: %+v", account)
}

func TestLongtimeListing(t *testing.T) {
	retryInterval := time.Second * 5
	deadCount := 0
	ctx := context.Background()
	//cancelCtx, cancelFunc := context.WithCancel(ctx)
	//Start rpc client
	log.Debug().Msg("[ScalarClient] [Start] Try to start scalar connection")
	tmclient, err := networkClient.Start()
	if err != nil {
		deadCount += 1
		if deadCount >= 10 {
			log.Debug().Msgf("[ScalarClient] [Start] Connect to the scalar network failed, sleep for %ds then retry", int64(retryInterval.Seconds()))
		}
		networkClient.RemoveRpcClient()
		time.Sleep(retryInterval)
	}
	log.Info().Msgf("[ScalarClient] [Start] Start rpc client success. Subscribing for events...")
	go func() {
		// handler := func(ctx context.Context, events []scalar.IBCEvent[scalar.ScalarMessage]) error {
		// 	log.Info().Msgf("[ScalarClient] [subscribeAllEvent] events: %+v", events)
		// 	return nil
		// }
		eventCh, err := networkClient.Subscribe(ctx, scalar.AllNewBlockEvent.Type, scalar.AllNewBlockEvent.TopicId)
		require.NoError(t, err)
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msgf("[ScalarClient] [Subscribe] timed out waiting for event, the transaction could have already been included or wasn't yet included")
				networkClient.UnSubscribeAll(context.Background(), scalar.AllNewBlockEvent.Type) //nolint:errcheck // ignore
				log.Info().Msgf("[ScalarClient] [Subscribe] Unsubscribe all event. Context done")
			case evt := <-eventCh:
				if evt.Query != scalar.AllNewBlockEvent.TopicId {
					log.Debug().Msgf("[ScalarClient] [Subscribe] Event query is not match query: %v, topicId: %s", evt.Query, scalar.AllNewBlockEvent.TopicId)
				} else {
					//Extract the data from the event
					log.Debug().Str("Topic", evt.Query).Any("Events", evt.Events).Msg("[ScalarClient] [Subscribe] Received new event")
				}
			}
		}
	}()
	//HeatBeat
	aliveCount := 0
	for {
		_, err := tmclient.Health(ctx)
		if err != nil {
			// clean all subscriber then retry
			log.Info().Msgf("[ScalarClient] ScalarNode is dead. Perform reconnecting")
			networkClient.RemoveRpcClient()
			break
		} else {
			aliveCount += 1
			if aliveCount >= 100 {
				log.Debug().Msgf("[ScalarClient] ScalarNode is alive")
				aliveCount = 0
			}
		}
		time.Sleep(retryInterval)
	}
	//cancelFunc()
}
