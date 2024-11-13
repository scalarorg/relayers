package scalar_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/stretchr/testify/require"
)

var (
	DefaultConfig = config.Config{
		ChainEnv: "testnet",
	}
	DefaultCosmosNetworkConfig = cosmos.CosmosNetworkConfig{
		ChainID:  "scalar-testnet-1",
		Denom:    "uatom",
		RPCUrl:   "http://localhost:26657",
		GasPrice: "0.001",
		LCDUrl:   "http://localhost:2317",
		WS:       "ws://localhost:26657/websocket",
		Mnemonic: "vibrant inject indoor parent sunny file warfare scissors cube wheat detail way ship come stem fitness trust thought broken tennis exclude glance insane lens",
	}
)

func TestCreateTransaction(t *testing.T) {
	// Configure the address prefix for Axelar
	sdk.GetConfig().SetBech32PrefixForAccount("axelar", "axelarvaloper")

	// Setup test client
	dbAdapter, err := db.NewDatabaseAdapter(&DefaultConfig)
	require.NoError(t, err)
	eventBusConfig := config.EventBusConfig{}
	eventBus := events.NewEventBus(&eventBusConfig)
	client, err := scalar.NewClientFromConfig(&DefaultCosmosNetworkConfig, dbAdapter, eventBus)
	require.NoError(t, err)
	require.NotNil(t, client)

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
