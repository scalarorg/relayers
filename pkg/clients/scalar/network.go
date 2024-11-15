package scalar

import (
	"context"
	"errors"
	"fmt"

	axltypes "github.com/axelarnetwork/axelar-core/x/axelarnet/types"
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func createDefaultTxFactory(config *cosmos.CosmosNetworkConfig, txConfig client.TxConfig) (tx.Factory, error) {
	factory := tx.Factory{}
	factory = factory.WithTxConfig(txConfig)
	if config.ChainID == "" {
		return factory, fmt.Errorf("chain ID is required")
	}
	factory = factory.WithChainID(config.ChainID)
	factory = factory.WithGas(200000) // Adjust as needed
	//Direct Sign mode with single signer
	factory = factory.WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	factory = factory.WithMemo("") // Optional memo
	factory = factory.WithFees(sdk.NewCoin("uaxl", sdk.NewInt(20000)).String())
	return factory, nil
}

type NetworkClient struct {
	config         *cosmos.CosmosNetworkConfig
	rpcEndpoint    string
	rpcClient      rpcclient.Client
	queryClient    *QueryClient
	addr           sdk.AccAddress
	privKey        *secp256k1.PrivKey
	txConfig       client.TxConfig
	txFactory      tx.Factory
	sequenceNumber uint64
}

func NewNetworkClient(config *cosmos.CosmosNetworkConfig, queryClient *QueryClient, txConfig client.TxConfig) (*NetworkClient, error) {
	privKey, addr, err := CreateAccountFromMnemonic(config.Mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
	}
	log.Info().Msgf("Scalar NetworkClient created with broadcaster address: %s", addr.String())
	var rpcClient rpcclient.Client
	if config.RPCUrl != "" {
		log.Info().Msgf("Create rpc client with url: %s", config.RPCUrl)
		rpcClient, err = client.NewClientFromNode(config.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client: %w", err)
		}
	}

	// wsClient, err := rpchttp.New(config.WsEndpoint, "/websocket")
	// if err != nil {
	// 	return nil, err
	// }
	resp, err := queryClient.QueryAccount(context.Background(), addr)
	if err != nil {
		return nil, err
	}
	txFactory, err := createDefaultTxFactory(config, txConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx factory: %w", err)
	}
	//Get account sequence number from network
	sequenceNumber := resp.Sequence
	networkClient := &NetworkClient{
		config:         config,
		rpcClient:      rpcClient,
		queryClient:    queryClient,
		addr:           addr,
		privKey:        privKey,
		txConfig:       txConfig,
		txFactory:      txFactory,
		sequenceNumber: sequenceNumber,
	}
	return networkClient, nil
}

// Start connections: rpc, websocket...
func (c *NetworkClient) Start() error {
	rpcClient, err := c.GetRpcClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	return rpcClient.Start()
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/tx/tx.go#L31
func (c *NetworkClient) ConfirmEvmTx(ctx context.Context, msg *emvtypes.ConfirmGatewayTxsRequest) (*sdk.TxResponse, error) {
	return c.SignAndBroadcastMsgs(ctx, msg)
}

func (c *NetworkClient) SignCommandsRequest(ctx context.Context, destinationChain string) (*sdk.TxResponse, error) {
	req := emvtypes.NewSignCommandsRequest(
		c.getAddress(),
		destinationChain)

	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign commands request: %w", err)
	}
	return txRes, nil
}
func (c *NetworkClient) SendRouteMessageRequest(ctx context.Context, id string, payload string) (*sdk.TxResponse, error) {
	payloadBytes := []byte(payload)
	req := axltypes.NewRouteMessage(
		c.getAddress(),
		c.getFeegranter(),
		id,
		payloadBytes,
	)
	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign commands request: %w", err)
	}
	return txRes, nil
}
func (c *NetworkClient) createTxFactory(ctx context.Context) tx.Factory {
	txf := c.txFactory
	resp, err := c.queryClient.QueryAccount(ctx, c.getAddress())
	if err != nil {
		log.Error().Msgf("failed to get account: %+v", err)
	} else {
		log.Debug().Msgf("[ScalarClient] [NetworkClient] account: %v", resp)
		txf = txf.WithAccountNumber(resp.AccountNumber)
		txf = txf.WithSequence(resp.Sequence)
	}
	return txf
}
func (c *NetworkClient) SignAndBroadcastMsgs(ctx context.Context, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	//1. Build unsigned transaction using txFactory
	txf := c.createTxFactory(ctx)
	// Every required params are set in the txFactory
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}
	txBuilder.SetFeeGranter(c.addr)
	err = c.signTx(txBuilder, true)

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
	if result != nil && result.Code == 0 {
		log.Debug().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] success broadcast tx with tx hash: %s", result.TxHash)
		//Update sequence and account number
		c.txFactory = c.txFactory.WithSequence(c.txFactory.Sequence() + 1)
	} else {
		log.Error().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] failed to broadcast tx: %+v", result)
	}
	return result, nil
}
func (c *NetworkClient) signTx(txBuilder sdkclient.TxBuilder, overwriteSig bool) error {
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
	rpcClient, err := c.GetRpcClient()
	res, err := rpcClient.BroadcastTxSync(context.Background(), txBytes)
	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}
func (c *NetworkClient) BroadcastTxAsync(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	rpcClient, err := c.GetRpcClient()
	if err != nil {
		return nil, err
	}

	res, err := rpcClient.BroadcastTxAsync(context.Background(), txBytes)
	if errRes := sdkclient.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}

	return sdk.NewResponseFormatBroadcastTx(res), err
}
func (c *NetworkClient) BroadcastTxCommit(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	node, err := c.GetRpcClient()
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
	client, err := c.GetRpcClient()
	if err != nil {
		return nil, err
	}
	return client.Subscribe(ctx, subscriber, query)
}

func (c *NetworkClient) UnSubscribe(ctx context.Context, subscriber string, query string) error {
	client, err := c.GetRpcClient()
	if err != nil {
		return err
	}
	return client.Unsubscribe(ctx, subscriber, query)
}

func (c *NetworkClient) UnSubscribeAll(ctx context.Context, subscriber string) error {
	client, err := c.GetRpcClient()
	if err != nil {
		return err
	}
	return client.UnsubscribeAll(ctx, subscriber)
}

// Get Broadcast Address from config (privatekey or mnemonic)
func (c *NetworkClient) getAddress() sdk.AccAddress {
	return c.addr
}
func (c *NetworkClient) getFeegranter() sdk.AccAddress {
	return c.addr
}
func (c *NetworkClient) GetRpcClient() (rpcclient.Client, error) {
	if c.rpcClient == nil {
		return nil, errors.New("no RPC client is defined in offline mode")
	}

	return c.rpcClient, nil
}
