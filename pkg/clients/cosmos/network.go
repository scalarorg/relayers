package cosmos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func createDefaultTxFactory(config *CosmosNetworkConfig, txConfig client.TxConfig) (tx.Factory, error) {
	factory := tx.Factory{}
	factory = factory.WithTxConfig(txConfig)
	if config.ID == "" {
		return factory, fmt.Errorf("chain ID is required")
	}
	factory = factory.WithChainID(config.ID)
	//Todo: estimate gas each time broadcast tx
	factory = factory.WithGas(0) // Adjust in estimateGas()
	factory = factory.WithGasAdjustment(DEFAULT_GAS_ADJUSTMENT)
	//factory = factory.WithGasPrices(config.GasPrice)
	// factory = factory.WithFees(sdk.NewCoin("uaxl", sdk.NewInt(20000)).String())

	//Direct Sign mode with single signer
	factory = factory.WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	factory = factory.WithMemo("") // Optional memo

	return factory, nil
}

type NetworkClient struct {
	config         *CosmosNetworkConfig
	rpcEndpoint    string
	rpcClient      rpcclient.Client
	queryClient    *QueryClient
	addr           sdk.AccAddress
	privKey        *secp256k1.PrivKey
	txConfig       client.TxConfig
	txFactory      tx.Factory
	sequenceNumber uint64
}

func NewNetworkClient(config *CosmosNetworkConfig, queryClient *QueryClient, txConfig client.TxConfig) (*NetworkClient, error) {
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

	account, err := queryClient.QueryAccount(context.Background(), addr)
	if err != nil {
		return nil, err
	}
	txFactory, err := createDefaultTxFactory(config, txConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx factory: %w", err)
	}
	networkClient := &NetworkClient{
		config:         config,
		rpcClient:      rpcClient,
		queryClient:    queryClient,
		addr:           addr,
		privKey:        privKey,
		txConfig:       txConfig,
		txFactory:      txFactory,
		sequenceNumber: account.Sequence,
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
		c.GetAddress(),
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
		c.GetAddress(),
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

// Inject account number and sequence number into txFactory for signing
func (c *NetworkClient) createTxFactory(ctx context.Context) tx.Factory {
	txf := c.txFactory
	resp, err := c.queryClient.QueryAccount(ctx, c.GetAddress())
	if err != nil {
		log.Error().Msgf("failed to get account: %+v", err)
	} else {
		log.Debug().Msgf("[ScalarClient] [NetworkClient] account number: %v, sequence number: %v", resp.AccountNumber, resp.Sequence)
		txf = txf.WithAccountNumber(resp.AccountNumber)
		//If sequence number is greater than current sequence number, update the sequence number
		//This is to avoid the situation where the transaction is not included in the next block
		//Then account sequence number is not updated on the server side
		if resp.Sequence >= txf.Sequence() {
			txf = txf.WithSequence(resp.Sequence)
		}
	}
	return txf
}
func (c *NetworkClient) SignAndBroadcastMsgs(ctx context.Context, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	//1. Build unsigned transaction using txFactory
	txf := c.createTxFactory(ctx)
	//Estimate fees
	simRes, adjusted, err := tx.CalculateGas(c.queryClient.GetClientCtx(), txf, msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate gas: %w", err)
	}
	fees := int64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed) * c.config.GasPrice)
	txf = txf.WithGas(adjusted)
	txf = txf.WithFees(sdk.NewCoin(c.config.Denom, sdk.NewInt(fees)).String())
	log.Debug().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] estimated gas: %v, gasPrice: %v, fees: %v", adjusted, c.config.GasPrice, txf.Fees())
	// Every required params are set in the txFactory
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeGranter(c.addr)
	//Try to sign and broadcast the transaction until success or reach max retry
	result, err := c.trySignAndBroadcastMsgs(ctx, txBuilder)
	if err != nil {
		return nil, err
	}
	if result != nil && result.Code == 0 {
		//log.Debug().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] success broadcast tx with tx hash: %s", result.TxHash)
		//Update sequence and account number
		c.txFactory = c.txFactory.WithSequence(c.txFactory.Sequence() + 1)
	} else {
		log.Error().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] failed to broadcast tx: %+v", result)
	}
	return result, nil
}

func (c *NetworkClient) trySignAndBroadcastMsgs(ctx context.Context, txBuilder sdkclient.TxBuilder) (*sdk.TxResponse, error) {
	var err error
	var result *sdk.TxResponse
	for i := 0; i < c.config.MaxRetries; i++ {
		txf := c.createTxFactory(ctx)
		log.Debug().Msgf("[ScalarNetworkClient] [trySignAndBroadcastMsgs] account sequence: %d", txf.Sequence())
		c.txFactory = txf
		err = c.signTx(txf, txBuilder, true)

		if err != nil {
			return nil, err
		}

		//2. Encode the transaction for Broadcasting
		var txBytes []byte
		txBytes, err = c.txConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return nil, err
		}

		result, err = c.BroadcastTx(ctx, txBytes)
		//Return if success
		//Or error it not nil
		if result != nil && result.Code == 0 || err != nil {
			return result, err
		}

		//Sleep for a while if error is nil
		//Or error "account sequence mismatch"
		if result != nil && result.Code > 0 && strings.Contains(result.RawLog, "account sequence mismatch") {
			log.Debug().Msgf("[ScalarNetworkClient] [trySignAndBroadcast] sleep for %d milliseconds due to error: %s", c.config.RetryInterval, result.RawLog)
			time.Sleep(time.Duration(c.config.RetryInterval) * time.Millisecond)
		} else {
			return result, nil
		}
	}
	log.Error().Msgf("[ScalarNetworkClient] [trySignAndBroadcast] failed to broadcast tx after %d retries", c.config.MaxRetries)
	return result, err
}
func (c *NetworkClient) signTx(txf tx.Factory, txBuilder sdkclient.TxBuilder, overwriteSig bool) error {
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
	if err != nil {
		return nil, fmt.Errorf("failed to get rpc client: %w", err)
	}
	res, err := rpcClient.BroadcastTxSync(context.Background(), txBytes)
	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
		return errRes, nil
	}
	txResponse := sdk.NewResponseFormatBroadcastTx(res)
	return txResponse, err
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
	txResponse := sdk.NewResponseFormatBroadcastTx(res)
	return txResponse, err
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
	txResponse := sdk.NewResponseFormatBroadcastTxCommit(res)
	return txResponse, err
}

func (c *NetworkClient) Subscribe(ctx context.Context, subscriber string, query string) (<-chan ctypes.ResultEvent, error) {
	client, err := c.GetRpcClient()
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("[ScalarNetworkClient] [Subscribe] query: %s", query)
	res, err := client.Subscribe(ctx, subscriber, query)
	if err != nil {
		log.Error().Msgf("[ScalarNetworkClient] [Subscribe] error: %v", err)
	}
	return res, err
}

func (c *NetworkClient) UnSubscribe(ctx context.Context, subscriber string, query string) error {
	client, err := c.GetRpcClient()
	if err != nil {
		return err
	}
	log.Debug().Msgf("[ScalarNetworkClient] [UnSubscribe] query: %s", query)
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
func (c *NetworkClient) GetAddress() sdk.AccAddress {
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