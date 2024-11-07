package scalar

import (
	"context"
	"errors"
	"fmt"

	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	client_tx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	grpc "google.golang.org/grpc"
)

func createTxFactory(config *Config, txConfig client.TxConfig) client_tx.Factory {
	factory := client_tx.Factory{}
	factory.WithTxConfig(txConfig)
	factory.WithChainID(config.ChainID)
	factory.WithGas(200000) // Adjust as needed
	//Direct Sign mode with single signer
	factory.WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	factory.WithMemo("") // Optional memo
	factory.WithSequence(0)
	factory.WithAccountNumber(0)
	return factory
}

type NetworkClient struct {
	config      *Config
	rpcEndpoint string
	rpcClient   rpcclient.Client
	addr        sdk.AccAddress
	privKey     *secp256k1.PrivKey
	txConfig    client.TxConfig
	txFactory   client_tx.Factory
}

func NewNetworkClient(config *Config, txConfig client.TxConfig) (*NetworkClient, error) {
	privKey, addr, err := CreateAccountFromMnemonic(config.Mnemonic)
	if err != nil {
		return nil, err
	}
	var rpcClient rpcclient.Client
	if config.RpcEndpoint != "" {
		rpcClient, err = client.NewClientFromNode(config.RpcEndpoint)
		if err != nil {
			return nil, err
		}
	}
	// wsClient, err := rpchttp.New(config.WsEndpoint, "/websocket")
	// if err != nil {
	// 	return nil, err
	// }
	txFactory := createTxFactory(config, txConfig)
	networkClient := &NetworkClient{
		config:    config,
		rpcClient: rpcClient,
		addr:      addr,
		privKey:   privKey,
		txConfig:  txConfig,
		txFactory: txFactory,
	}
	return networkClient, nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/tx/tx.go#L31
func (c *NetworkClient) ConfirmEvmTx(ctx context.Context, msg *emvtypes.ConfirmGatewayTxsRequest) (*sdk.TxResponse, error) {
	//1. Build unsigned transaction using txFactory
	txf := c.txFactory
	// Every required params are set in the txFactory
	txBuilder, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		return nil, err
	}
	txBuilder.SetFeeGranter(c.addr)
	err = c.signTx(ctx, txBuilder, true)

	if err != nil {
		return nil, err
	}

	//2. Encode the transaction for Broadcasting
	txBytes, err := c.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	result, err := c.BroadcastTx(ctx, txBytes)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *NetworkClient) signTx(ctx context.Context, txBuilder client.TxBuilder, overwriteSig bool) error {
	txf := c.txFactory
	//2. Sign the transaction
	signerData := authsigning.SignerData{
		ChainID:       txf.ChainID(),
		AccountNumber: txf.AccountNumber(),
		Sequence:      txf.Sequence(),
	}
	// For SIGN_MODE_DIRECT, calling SetSignatures calls setSignerInfos on
	// TxBuilder under the hood, and SignerInfos is needed to generated the
	// sign bytes. This is the reason for setting SetSignatures here, with a
	// nil signature.
	//
	// Note: this line is not needed for SIGN_MODE_LEGACY_AMINO, but putting it
	// also doesn't affect its generated sign bytes, so for code's simplicity
	// sake, we put it here.
	sigData := signing.SingleSignatureData{
		SignMode:  txf.SignMode(),
		Signature: nil,
	}
	sigV2 := signing.SignatureV2{
		PubKey:   c.privKey.PubKey(),
		Data:     &sigData,
		Sequence: txf.Sequence(),
	}
	var prevSignatures []signing.SignatureV2
	var err error
	if !overwriteSig {
		prevSignatures, err = txBuilder.GetTx().GetSignaturesV2()
		if err != nil {
			return err
		}
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return err
	}
	// Generate the bytes to be signed.
	bytesToSign, err := c.txConfig.SignModeHandler().GetSignBytes(txf.SignMode(), signerData, txBuilder.GetTx())
	if err != nil {
		return err
	}

	// Sign those bytes
	sigBytes, err := c.privKey.Sign(bytesToSign)
	if err != nil {
		return err
	}

	// Construct the SignatureV2 struct
	sigData = signing.SingleSignatureData{
		SignMode:  txf.SignMode(),
		Signature: sigBytes,
	}
	sigV2 = signing.SignatureV2{
		PubKey:   c.privKey.PubKey(),
		Data:     &sigData,
		Sequence: txf.Sequence(),
	}

	if overwriteSig {
		return txBuilder.SetSignatures(sigV2)
	}

	// Sign the transaction
	// sigV2, err := client_tx.SignWithPrivKey(
	// 	signing.SignMode_SIGN_MODE_DIRECT,
	// 	signerData,
	// 	txBuilder,
	// 	c.privKey,
	// 	c.txConfig,
	// 	c.txFactory.Sequence(),
	// )
	// if err != nil {
	// 	return err
	// }
	prevSignatures = append(prevSignatures, sigV2)
	return txBuilder.SetSignatures(prevSignatures...)
}

func (c *NetworkClient) BroadcastTx(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	switch c.config.BroadcastMode {
	case flags.BroadcastSync:
		return c.BroadcastTxSync(ctx, txBytes)
	case flags.BroadcastAsync:
		return c.BroadcastTxAsync(ctx, txBytes)
	case flags.BroadcastBlock:
		return c.BroadcastTxCommit(ctx, txBytes)
	default:
		return nil, fmt.Errorf("unsupported return type %s; supported types: sync, async, block", c.config.BroadcastMode)
	}
}

func (c *NetworkClient) BroadcastTxSync(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	node, err := c.GetNode()
	res, err := node.BroadcastTxSync(context.Background(), txBytes)
	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}
func (c *NetworkClient) BroadcastTxAsync(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	node, err := c.GetNode()
	if err != nil {
		return nil, err
	}

	res, err := node.BroadcastTxAsync(context.Background(), txBytes)
	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}
func (c *NetworkClient) BroadcastTxCommit(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	node, err := c.GetNode()
	if err != nil {
		return nil, err
	}

	res, err := node.BroadcastTxCommit(context.Background(), txBytes)
	if err == nil {
		return sdk.NewResponseFormatBroadcastTxCommit(res), nil
	}

	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}
	return sdk.NewResponseFormatBroadcastTxCommit(res), err
}

func (c *NetworkClient) Subscribe(ctx context.Context, subscriber string, query string) (<-chan ctypes.ResultEvent, error) {
	node, err := c.GetNode()
	if err != nil {
		return nil, err
	}
	return node.Subscribe(ctx, subscriber, query)
}

func (c *NetworkClient) UnSubscribe(ctx context.Context, subscriber string, query string) error {
	node, err := c.GetNode()
	if err != nil {
		return err
	}
	return node.Unsubscribe(ctx, subscriber, query)
}

func (c *NetworkClient) UnSubscribeAll(ctx context.Context, subscriber string) error {
	node, err := c.GetNode()
	if err != nil {
		return err
	}
	return node.UnsubscribeAll(ctx, subscriber)
}

// func (c *NetworkClient) QueryTx(ctx context.Context, hash []byte) (*ctypes.ResultTx, error) {
// 	// Query by hash
// 	res, err := c.rpcClient.Tx(ctx, hash, false)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &ctypes.ResultTx{
// 		Hash:     hash,
// 		Height:   res.Height,
// 		Index:    res.Index,
// 		TxResult: res.TxResult,
// 		Tx:       res.Tx,
// 	}, nil
// }

func (c *NetworkClient) QueryBalance(ctx context.Context, addr sdk.AccAddress) (*sdk.Coins, error) {
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

// Get Broadcast Address from config (privatekey or mnemonic)
func (c *NetworkClient) getAddress() sdk.AccAddress {
	return c.addr
}
func (c *NetworkClient) GetNode() (rpcclient.Client, error) {
	if c.rpcClient == nil {
		return nil, errors.New("no RPC client is defined in offline mode")
	}

	return c.rpcClient, nil
}
