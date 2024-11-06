package scalar_clients_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	scalar_clients "github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/stretchr/testify/require"
)

func TestCreateTransaction(t *testing.T) {
	// Configure the address prefix for Axelar
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("axelar", "axelarvaloper")

	// Setup test client
	client := scalar_clients.NewClient("http://localhost:26657")

	// Create test wallet/account
	privKey := secp256k1.GenPrivKey()
	addr := types.AccAddress(privKey.PubKey().Address())

	// Create test message
	msg := banktypes.NewMsgSend(
		addr,
		addr, // sending to self for test
		types.NewCoins(types.NewCoin("stake", math.NewInt(100))),
	)

	// Build and sign transaction
	tx, err := client.CreateTransaction(context.Background(), msg, privKey)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Broadcast transaction
	resp, err := client.BroadcastTx(context.Background(), tx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	fmt.Printf("resp: %+v\n", resp)

	// Add verification steps
	require.Equal(t, resp.Code, uint32(0), "Transaction failed with code: %d, log: %s", resp.Code, resp.Log)

	// // Wait for transaction to be included in a block
	// time.Sleep(6 * time.Second) // Adjust based on your chain's block time

	// // Query the transaction using the hash
	// txResp, err := client.QueryTx(context.Background(), resp.Hash)
	// require.NoError(t, err)
	// require.NotNil(t, txResp)
	// require.Equal(t, resp.Hash, txResp.Hash)
}

type MsgConfirmGatewayTx struct {
	Sender string
	Chain  string
	TxId   []byte
}

func TestConfirmGatewayTxRequest(t *testing.T) {
	// Configure the address prefix for Axelar
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("axelar", "axelarvaloper")

	// Setup test client
	client := scalar_clients.NewClient("http://localhost:26657")

	// Create test wallet/account
	privKey := secp256k1.GenPrivKey()
	addr := types.AccAddress(privKey.PubKey().Address())

	// Create test ConfirmGatewayTxRequest
	txHash, _ := hex.DecodeString("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Example tx hash
	msg_payload := MsgConfirmGatewayTx{
		Sender: addr.String(),
		Chain:  "ethereum",
		TxId:   txHash,
	}

	// Build and sign transaction
	tx, err := client.CreateTransaction(context.Background(), msg, privKey)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Broadcast transaction
	resp, err := client.BroadcastTx(context.Background(), tx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	fmt.Printf("resp: %+v\n", resp)

	// Add verification steps
	require.Equal(t, uint32(0), resp.Code, "Transaction failed with code: %d, log: %s", resp.Code, resp.Log)
}
