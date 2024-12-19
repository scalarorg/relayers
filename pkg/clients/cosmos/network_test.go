package cosmos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/scalar-core/utils"
	emvtypes "github.com/scalarorg/scalar-core/x/evm/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

const (
	chainNameBtcTestnet4 = "bitcoin-testnet4"
)

var (
	protoCodec          = encoding.GetCodec(proto.Name)
	DefaultGlobalConfig = config.Config{}
	CosmosNetworkConfig = cosmos.CosmosNetworkConfig{
		ChainID:       111,
		ID:            "scalar-testnet-1",
		Name:          "scalar-testnet-1",
		Denom:         "scalar",
		RPCUrl:        "http://localhost:26657",
		GasPrice:      0.001,
		LCDUrl:        "http://localhost:2317",
		WSUrl:         "ws://localhost:26657/websocket",
		MaxRetries:    3,
		RetryInterval: int64(1000),
		Mnemonic:      "latin total dream gesture brain bunker truly stove left video cost transfer guide occur bicycle oxygen world ready witness exhibit federal salute half day",
	}
	err        error
	clientCtx  *client.Context
	accAddress sdk.AccAddress
)

func TestSubscribeContractCallApprovedEvent(t *testing.T) {
	txConfig := tx.NewTxConfig(codec.GetProtoCodec(), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	clientCtx, err := cosmos.CreateClientContext(&CosmosNetworkConfig)
	require.NoError(t, err)
	queryClient := cosmos.NewQueryClient(clientCtx)
	networkClient, err := cosmos.NewNetworkClient(&CosmosNetworkConfig, queryClient, txConfig)
	require.NoError(t, err)
	require.NotNil(t, networkClient)
	err = networkClient.Start()
	require.NoError(t, err)
	//queryNewBlockHeader := "tm.event='NewBlockHeader'"
	queryContractCallApproved := "tm.event='NewBlock' AND axelar.evm.v1beta1.ContractCallApproved.event_id EXISTS"
	//queryEventCompleted := "tm.event='NewBlock' AND axelar.evm.v1beta1.EVMEventCompleted.event_id EXISTS"
	ch, err := networkClient.Subscribe(context.Background(), "test", queryContractCallApproved)
	require.NoError(t, err)
	require.NotNil(t, ch)
	go func() {
		for event := range ch {
			fmt.Printf("event: %+v\n", event)
		}
	}()
	//Broadcast a confirm btc network tx
	nexusChain := nexus.ChainName(utils.NormalizeString(chainNameBtcTestnet4))
	txIds := []string{"f0510bcacb2e428bd89e39e9708555265ed413b5320c5f920bf4becac9c53f56"}
	log.Debug().Msgf("[ScalarClient] [ConfirmTxs] Broadcast for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]emvtypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = emvtypes.Hash(common.HexToHash(txId))
	}
	//msg := emvtypes.NewConfirmGatewayTxsRequest(networkClient.GetAddress(), nexusChain, txHashs)
	//2. Sign and broadcast the payload using the network client, which has the private key
	// confirmTx, err := networkClient.SignAndBroadcastMsgs(context.Background(), msg)
	// if err != nil {
	// 	fmt.Printf("error from network client: %v", err)
	// 	log.Error().Msgf("[ScalarClient] [ConfirmTxs] error from network client: %v", err)
	// }
	// require.NoError(t, err)
	// require.NotNil(t, confirmTx)
	time.Sleep(1 * time.Hour)
}
