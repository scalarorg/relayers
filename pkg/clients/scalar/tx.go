package scalar_clients

import (
	"context"

	"cosmossdk.io/math"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	client_tx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codec_types "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
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
