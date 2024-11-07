package scalar_clients

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	client_tx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codec_types "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	grpc "google.golang.org/grpc"
	evm_proto_types "github.com/scalarorg/xchains-core/x/evm/types"
)

type Client struct {
	rpcEndpoint string
	txConfig    client.TxConfig
	// Add other necessary fields like chain ID, gas prices, etc.
}

func NewClient(rpcEndpoint string) *Client {
	return &Client{
		rpcEndpoint: rpcEndpoint,
		txConfig:    tx.NewTxConfig(codec.NewProtoCodec(codec_types.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
	}
}

func (c *Client) CreateTransaction(ctx context.Context, msg types.Msg, privKey cryptotypes.PrivKey) (client.TxBuilder, error) {
	// Create a new transaction builder
	txBuilder := c.txConfig.NewTxBuilder()

	// Set the message
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}

	// Set other transaction parameters
	txBuilder.SetGasLimit(200000) // Adjust as needed
	txBuilder.SetFeeAmount(types.NewCoins(types.NewCoin("stake", math.NewInt(1000))))
	txBuilder.SetMemo("") // Optional memo

	// Sign the transaction
	signerData := signingtypes.SignerData{
		ChainID:       "your-chain-id",
		AccountNumber: 0, // Get from account query
		Sequence:      0, // Get from account query
	}

	// Sign the transaction
	sigV2, err := client_tx.SignWithPrivKey(
		ctx,
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		privKey,
		c.txConfig,
		uint64(0),
	)
	if err != nil {
		return nil, err
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	return txBuilder, nil
}

func (c *Client) BroadcastTx(ctx context.Context, txBuilder client.TxBuilder) (*ctypes.ResultBroadcastTx, error) {
	txBytes, err := c.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	// Create HTTP client and broadcast the transaction
	rpcClient, err := rpchttp.New(c.rpcEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}

	result, err := rpcClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) QueryTx(ctx context.Context, hash []byte) (*ctypes.ResultTx, error) {
	node, err := rpchttp.New(c.rpcEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}

	// Query by hash
	res, err := node.Tx(ctx, hash, false)
	if err != nil {
		return nil, err
	}

	return &ctypes.ResultTx{
		Hash:     hash,
		Height:   res.Height,
		Index:    res.Index,
		TxResult: res.TxResult,
		Tx:       res.Tx,
	}, nil
}

func (c *Client) QueryBalance(ctx context.Context, addr types.AccAddress) (*types.Coins, error) {
	// Create gRPC connection
	grpcConn, err := grpc.Dial(
		// c.rpcEndpoint,
		"localhost:9090",
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}
	defer grpcConn.Close()

	// Create bank query client
	bankClient := banktypes.NewQueryClient(grpcConn)

	// Query all balances
	balanceResp, err := bankClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
		Address: addr.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query balance: %w", err)
	}

	return &balanceResp.Balances, nil
}

func CreateAccountFromMnemonic(mnemonic string) (types.AccAddress, *secp256k1.PrivKey, error) {
	// Derive the seed from mnemonic
	seed := bip39.NewSeed(mnemonic, "")

	// Create master key and derive the private key
	// Using "m/44'/118'/0'/0/0" for Cosmos
	master, ch := hd.ComputeMastersFromSeed(seed)
	privKeyBytes, err := hd.DerivePrivateKeyForPath(master, ch, "m/44'/118'/0'/0/0")
	if err != nil {
		return nil, nil, err
	}

	// Create private key and get address
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	addr := types.AccAddress(privKey.PubKey().Address())

	return addr, privKey, nil
}

// Add this new type definition
type ConfirmGatewayTxRequest struct {
	Sender string
	Chain  string
	TxId   []byte
}

const (
	EvmProtobufPackage = "axelar.evm.v1beta1"
)

// Add this method to create the Any-wrapped message
func CreateConfirmGatewayTxMsg(sender string, chain string, txId []byte) (*codec_types.Any, error) {
	msg := &ConfirmGatewayTxRequest{
		Sender: sender,
		Chain:  chain,
		TxId:   txId,
	}

	// Create Any type with the correct typeUrl
	return codec_types.NewAnyWithValue(msg)
}

// Alternative method if you need to create the Any message manually
func CreateConfirmGatewayTxMsgManual(sender string, chain string, txId []byte) *codec_types.Any {
	return &codec_types.Any{
		TypeUrl: "/" + EvmProtobufPackage + ".ConfirmGatewayTxRequest",
		Value: ConfirmGatewayTxRequest{
			Sender: sender,
			Chain:  chain,
			TxId:   txId,
		}.Marshal(), // You'll need to implement Marshal() or use protobuf generated code
	}
}
