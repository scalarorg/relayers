package cosmos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/axelarnetwork/axelar-core/utils"
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	nexus "github.com/axelarnetwork/axelar-core/x/nexus/exported"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
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
		ChainID:       "scalar-testnet-1",
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
func TestCreateTransaction(t *testing.T) {
	// Configure the address prefix for Axelar
	sdk.GetConfig().SetBech32PrefixForAccount("axelar", "axelarvaloper")

	// Setup test client
	// dbAdapter, err := db.NewDatabaseAdapter(&DefaultGlobalConfig)
	// require.NoError(t, err)
	// eventBusConfig := config.EventBusConfig{}
	// eventBus := events.NewEventBus(&eventBusConfig)
	// client, err := scalar.NewClientFromConfig(&DefaultGlobalConfig, &DefaultCosmosNetworkConfig, dbAdapter, eventBus)
	// require.NoError(t, err)
	// require.NotNil(t, client)

	// Create test wallet/account
	//privKey := secp256k1.GenPrivKey()
	//addr := sdktypes.AccAddress(privKey.PubKey().Address())

	// Create test message
	// msg := banktypes.NewMsgSend(
	// 	addr,
	// 	addr, // sending to self for test
	// 	sdktypes.NewCoins(sdktypes.NewCoin("stake", sdktypes.NewInt(100))),
	// )

	// Build and sign transaction
	// tx, err := client.CreateTransaction(context.Background(), msg, privKey)
	// require.NoError(t, err)
	// require.NotNil(t, tx)

	// // Broadcast transaction
	// resp, err := client.BroadcastTx(context.Background(), tx)
	// require.NoError(t, err)
	// require.NotNil(t, resp)

	// fmt.Printf("resp: %+v\n", resp)

	// // Add verification steps
	// require.Equal(t, resp.Code, uint32(0), "Transaction failed with code: %d, log: %s", resp.Code, resp.Log)

	// // Wait for transaction to be included in a block
	// time.Sleep(6 * time.Second) // Adjust based on your chain's block time

	// // Query the transaction using the hash
	// txResp, err := client.QueryTx(context.Background(), resp.Hash)
	// require.NoError(t, err)
	// require.NotNil(t, txResp)
	// require.Equal(t, resp.Hash, txResp.Hash)
}

// func TestConfirmGatewayTxRequest(t *testing.T) {
// 	// Configure the address prefix for Axelar
// 	config := types.GetConfig()
// 	config.SetBech32PrefixForAccount("axelar", "axelarvaloper")

// 	// Setup test client
// 	client := scalar_clients.NewClient("http://localhost:26657")

// 	// Create test wallet/account
// 	privKey := secp256k1.GenPrivKey()
// 	addr := types.AccAddress(privKey.PubKey().Address())

// 	// Create test ConfirmGatewayTxRequest
// 	txHash, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Example tx hash
// 	msg_payload := MsgConfirmGatewayTx{
// 		Sender: addr.String(),
// 		Chain:  "ethereum",
// 		TxId:   txHash,
// 	}

// 	// Build and sign transaction
// 	tx, err := client.CreateTransaction(context.Background(), msg, privKey)
// 	require.NoError(t, err)
// 	require.NotNil(t, tx)

// 	// Broadcast transaction
// 	resp, err := client.BroadcastTx(context.Background(), tx)
// 	require.NoError(t, err)
// 	require.NotNil(t, resp)

// 	fmt.Printf("resp: %+v\n", resp)

// 	// Add verification steps
// 	require.Equal(t, uint32(0), resp.Code, "Transaction failed with code: %d, log: %s", resp.Code, resp.Log)
// }

// func TestCheckAccountBalance(t *testing.T) {
// 	// Configure the address prefix for Axelar
// 	config := types.GetConfig()
// 	config.SetBech32PrefixForAccount("axelar", "axelarvaloper")

// 	// Setup test client
// 	client := scalar_clients.NewClient("http://localhost:26657")

// 	// Create test wallet/account
// 	// privKey := secp256k1.GenPrivKey()
// 	addr, privKey, err := scalar_clients.CreateAccountFromMnemonic("vibrant inject indoor parent sunny file warfare scissors cube wheat detail way ship come stem fitness trust thought broken tennis exclude glance insane lens")
// 	require.NoError(t, err)
// 	require.NotNil(t, addr)
// 	require.NotNil(t, privKey)

// 	// Check account balance
// 	balanceResp, err := client.QueryBalance(context.Background(), addr)

// 	fmt.Printf("balanceResp: %+v\n", balanceResp)

// 	require.NoError(t, err)
// 	require.NotNil(t, balanceResp)
// 	require.Zero(t, balanceResp.AmountOf("stake").Int64(), "New account should have zero balance")
// }
