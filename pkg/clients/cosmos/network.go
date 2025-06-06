package cosmos

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/types"
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

type AccountSequence struct {
	BlockHeight   int64
	QueryTime     *time.Time
	AccountNumber uint64
	Sequence      uint64
}
type NetworkClient struct {
	config          *CosmosNetworkConfig
	rpcClient       rpcclient.Client
	queryClient     *QueryClient
	addr            sdk.AccAddress
	privKey         *secp256k1.PrivKey
	txConfig        client.TxConfig
	txFactory       tx.Factory
	accountSequence AccountSequence
	seqMutex        sync.Mutex // Add mutex for sequence number synchronization
}

// Update account sequence for every 1 second
func (c *NetworkClient) UpdateAccountSequence(ctx context.Context) {
	if c.accountSequence.QueryTime == nil || time.Since(*c.accountSequence.QueryTime) >= time.Second {
		blockTime, err := c.GetCurrentBlock(ctx)
		if err != nil {
			log.Error().Msgf("failed to get current block: %+v", err)
		} else if blockTime != nil {
			now := time.Now()
			if blockTime.ResultBlock.Block.Height > c.accountSequence.BlockHeight {
				//Fetch new account number and sequence number from network
				resp, err := c.queryClient.QueryAccount(ctx, c.GetAddress())
				if err != nil {
					log.Error().Msgf("failed to get account: %+v", err)
				} else {
					accountSequence := AccountSequence{
						AccountNumber: resp.AccountNumber,
						Sequence:      resp.Sequence,
						BlockHeight:   blockTime.ResultBlock.Block.Height,
						QueryTime:     &now,
					}
					log.Debug().Int64("BlockHeight", blockTime.ResultBlock.Block.Height).
						Any("accountSequence", accountSequence).
						Msgf("[NetworkClient] update account sequence")
					c.SetAccountSequence(accountSequence)
				}
			}
		}
	}
}

// Add method to safely get sequence number
func (c *NetworkClient) GetAccountSequence() AccountSequence {
	c.seqMutex.Lock()
	defer c.seqMutex.Unlock()
	return c.accountSequence
}

// Add method to safely increment sequence number
func (c *NetworkClient) IncrementSequenceNumber() {
	c.seqMutex.Lock()
	defer c.seqMutex.Unlock()
	c.accountSequence.Sequence++
}

// Add method to safely set sequence number
func (c *NetworkClient) SetAccountSequence(accSequence AccountSequence) {
	c.seqMutex.Lock()
	defer c.seqMutex.Unlock()
	c.accountSequence.Sequence = accSequence.Sequence
	c.accountSequence.AccountNumber = accSequence.AccountNumber
	c.accountSequence.BlockHeight = accSequence.BlockHeight
	c.accountSequence.QueryTime = accSequence.QueryTime
}
func NewNetworkClient(config *CosmosNetworkConfig, queryClient *QueryClient, txConfig client.TxConfig) (*NetworkClient, error) {
	privKey, addr, err := CreateAccountFromMnemonic(config.Mnemonic, config.Bip44Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
	}
	log.Info().Msgf("Scalar NetworkClient created with broadcaster address: %s", addr.String())
	// var rpcClient rpcclient.Client
	// if config.RPCUrl != "" {
	// 	log.Info().Msgf("Create rpc client with url: %s", config.RPCUrl)
	// 	rpcClient, err = client.NewClientFromNode(config.RPCUrl)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	// 	}
	// }

	txFactory, err := createDefaultTxFactory(config, txConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx factory: %w", err)
	}
	networkClient := &NetworkClient{
		config:      config,
		queryClient: queryClient,
		addr:        addr,
		privKey:     privKey,
		txConfig:    txConfig,
		txFactory:   txFactory,
		accountSequence: AccountSequence{
			BlockHeight:   0,
			QueryTime:     nil,
			AccountNumber: 0,
			Sequence:      0,
		},
	}
	return networkClient, nil
}

type NetworkClientOption func(*NetworkClient)

func WithRpcClient(rpcClient rpcclient.Client) NetworkClientOption {
	return func(c *NetworkClient) {
		c.rpcClient = rpcClient
	}
}

func WithQueryClient(queryClient *QueryClient) NetworkClientOption {
	return func(c *NetworkClient) {
		c.queryClient = queryClient
	}
}

func WithAccount(privKey *secp256k1.PrivKey, addr sdk.AccAddress) NetworkClientOption {
	return func(c *NetworkClient) {
		c.privKey = privKey
		c.addr = addr
	}
}

func WithTxConfig(txConfig client.TxConfig) NetworkClientOption {
	return func(c *NetworkClient) {
		c.txConfig = txConfig
	}
}

func WithTxFactory(txFactory tx.Factory) NetworkClientOption {
	return func(c *NetworkClient) {
		c.txFactory = txFactory
	}
}

func (c *NetworkClient) SetTxFactory(txFactory tx.Factory) {
	c.txFactory = txFactory
}

func NewNetworkClientWithOptions(config *CosmosNetworkConfig, queryClient *QueryClient, txConfig client.TxConfig, opts ...NetworkClientOption) (*NetworkClient, error) {
	networkClient := &NetworkClient{
		config: config,
	}
	for _, opt := range opts {
		opt(networkClient)
	}

	log.Info().Msgf("Scalar NetworkClient created with broadcaster address: %s", networkClient.addr.String())
	if config.RPCUrl != "" {
		log.Info().Msgf("Create rpc client with url: %s", config.RPCUrl)
		rpcClient, err := client.NewClientFromNode(config.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client: %w", err)
		}
		networkClient.rpcClient = rpcClient
	}

	if networkClient.txFactory.AccountNumber() == 0 {
		txFactory, err := createDefaultTxFactory(config, txConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create tx factory: %w", err)
		}
		networkClient.txFactory = txFactory
	}
	return networkClient, nil
}

// Start connections: rpc, websocket...
func (c *NetworkClient) Start() (rpcclient.Client, error) {
	rpcClient, err := c.GetRpcClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return rpcClient, rpcClient.Start()
}
func (c *NetworkClient) AdjustAccountSequence(err error) error {
	expected, got, err := ExtractCurrentSequence(err)
	if err != nil {
		log.Debug().Err(err).Msgf("[NetworkClient] [AdjustAccountSequence] failed to extract current sequence from error message")
		return err
	}
	log.Debug().Int("expected", expected).Int("got", got).Msgf("[NetworkClient] [AdjustAccountSequence] Addjust account sequence")
	accountSequence := c.GetAccountSequence()
	accountSequence.Sequence = uint64(expected)
	c.SetAccountSequence(accountSequence)
	return nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/tx/tx.go#L31
// func (c *NetworkClient) ConfirmEvmTx(ctx context.Context, msg *chainstypes.ConfirmSourceTxsRequest) (*sdk.TxResponse, error) {
// 	return c.SignAndBroadcastMsgs(ctx, msg)
// }

// func (c *NetworkClient) SignPsbtCommandsRequest(ctx context.Context, destinationChain string, psbt covtypes.Psbt) (*sdk.TxResponse, error) {
// 	req := chainstypes.NewSignPsbtCommandRequest(
// 		c.GetAddress(),
// 		destinationChain,
// 		psbt)

// 	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [SignPsbtCommandsRequest] %w", err)
// 	}
// 	return txRes, nil
// }

// func (c *NetworkClient) SignBtcCommandsRequest(ctx context.Context, destinationChain string) (*sdk.TxResponse, error) {
// 	req := chainstypes.NewSignBtcCommandsRequest(
// 		c.GetAddress(),
// 		destinationChain)

// 	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [SignBtcCommandsRequest] %w", err)
// 	}
// 	return txRes, nil
// }

// func (c *NetworkClient) SignEvmCommandsRequest(ctx context.Context, destinationChain string) (*sdk.TxResponse, error) {
// 	req := chainstypes.NewSignCommandsRequest(
// 		c.GetAddress(),
// 		destinationChain)

// 	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [SignCommandsRequest] %w", err)
// 	}
// 	return txRes, nil
// }
// func (c *NetworkClient) SignCommandsRequests(ctx context.Context, destinationChains []string) (*sdk.TxResponse, error) {
// 	requests := []sdk.Msg{}
// 	for _, chain := range destinationChains {
// 		chainName := exported.ChainName(chain)
// 		if chainstypes.IsEvmChain(chainName) {
// 			requests = append(requests, chainstypes.NewSignCommandsRequest(c.GetAddress(), chain))
// 		} else if chainstypes.IsBitcoinChain(chainName) {
// 			requests = append(requests, chainstypes.NewSignBtcCommandsRequest(c.GetAddress(), chain))
// 		}
// 	}
// 	txRes, err := c.SignAndBroadcastMsgs(ctx, requests...)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [SignCommandsRequest] %w", err)
// 	}
// 	return txRes, nil
// }

// func (c *NetworkClient) CreatePendingTransfersRequest(ctx context.Context, chain string) (*sdk.TxResponse, error) {
// 	req := chainstypes.CreatePendingTransfersRequest{
// 		Sender: c.GetAddress(),
// 		Chain:  exported.ChainName(chain),
// 	}
// 	txRes, err := c.SignAndBroadcastMsgs(ctx, &req)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [CreatePendingTransderRequest] %w", err)
// 	}
// 	return txRes, nil
// }

// func (c *NetworkClient) SendRouteMessageRequest(ctx context.Context, id string, payload string) (*sdk.TxResponse, error) {
// 	payloadBytes := []byte(payload)
// 	req := scalarnettypes.NewRouteMessage(
// 		c.GetAddress(),
// 		c.getFeegranter(),
// 		id,
// 		payloadBytes,
// 	)
// 	txRes, err := c.SignAndBroadcastMsgs(ctx, req)
// 	if err != nil {
// 		return nil, fmt.Errorf("[NetworkClient] [SendRouteMessageRequest] %w", err)
// 	}
// 	return txRes, nil
// }

// Inject account number and sequence number into txFactory for signing
func (c *NetworkClient) CreateTxFactory(ctx context.Context) tx.Factory {
	txf := c.txFactory
	//Get current Scalar's block height for every 1 second
	accountSequence := c.GetAccountSequence()
	log.Debug().Int64("BlockHeight", accountSequence.BlockHeight).
		Any("accountSequence", accountSequence).
		Msgf("[NetworkClient] [CreateTxFactory]")
	//Set adjusted account number and sequence number to the txFactory
	txf = txf.WithAccountNumber(accountSequence.AccountNumber).WithSequence(accountSequence.Sequence)
	return txf
}

func (c *NetworkClient) CreateTxBuilder(ctx context.Context, msgs ...sdk.Msg) (client.TxBuilder, error) {
	//Estimate fees
	cliContext, err := c.queryClient.GetClientCtx()
	if err != nil {
		log.Debug().Err(err).Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] cannot create client context")
	}
	//1. Build unsigned transaction using txFactory
	var txf tx.Factory
	for {
		txf = c.CreateTxFactory(ctx)
		simRes, adjusted, err := tx.CalculateGas(cliContext, txf, msgs...)
		if err != nil {
			//Addjust account sequence number if error is "account sequence mismatch"
			//try extract current sequence from error message: error: code = Unknown desc = account sequence mismatch, expected 32, got 31:
			err := c.AdjustAccountSequence(err)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate gas: %w", err)
			}
			//Retry in the next iteration
			continue
		}
		fees := int64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed) * c.config.GasPrice)
		txf = txf.WithGas(adjusted)
		txf = txf.WithFees(sdk.NewCoin(c.config.Denom, sdk.NewInt(fees)).String())
		log.Debug().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] estimated gas: %v, gasPrice: %v, fees: %v",
			adjusted, c.config.GasPrice, txf.Fees())
		break
	}

	// Every required params are set in the txFactory
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeGranter(c.addr)
	return txBuilder, nil
}

func (c *NetworkClient) SignAndBroadcastMsgs(ctx context.Context, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	txBuilder, err := c.CreateTxBuilder(ctx, msgs...)
	if err != nil {
		return nil, err
	}
	//Try to sign and broadcast the transaction until success or reach max retry
	result, err := c.trySignAndBroadcastMsgs(ctx, txBuilder)
	if err != nil {
		log.Error().Err(err).Msg("[NetworkClient] [SignAndBroadcastMsgs] error")
		return nil, err
	}
	if result != nil && result.Code == 0 {
		//log.Debug().Msgf("[ScalarNetworkClient] [SignAndBroadcastMsgs] success broadcast tx with tx hash: %s", result.TxHash)
		//Update sequence and account number
		log.Debug().Msgf("[NetworkClient] [SignAndBroadcastMsgs] success broadcast tx with tx hash: %s", result.TxHash)
		c.IncrementSequenceNumber() // Safely increment sequence number on successful broadcast
	}
	return result, nil
}

func (c *NetworkClient) trySignAndBroadcastMsgs(ctx context.Context, txBuilder client.TxBuilder) (*sdk.TxResponse, error) {
	var err error
	var result *sdk.TxResponse
	for i := 0; i < c.config.MaxRetries; i++ {
		txf := c.CreateTxFactory(ctx)
		log.Debug().Msgf("[ScalarNetworkClient] [trySignAndBroadcastMsgs] account sequence: %d", txf.Sequence())
		c.txFactory = txf
		err = c.SignTx(txf, txBuilder, true)

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
		if err != nil {
			err := c.AdjustAccountSequence(err)
			if err != nil {
				log.Debug().Err(err).Msgf("[NetworkClient] [trySignAndBroadcastMsgs] the error is not account sequence mismatch, cannot adjust account sequence")
			}
			continue
		}
		//Return if success
		//Or error it not nil
		if result != nil && result.Code == 0 {
			return result, err
		}
	}
	log.Error().Msgf("[ScalarNetworkClient] [trySignAndBroadcast] failed to broadcast tx after %d retries", c.config.MaxRetries)
	return result, err
}
func (c *NetworkClient) SignTx(txf tx.Factory, txBuilder client.TxBuilder, overwriteSig bool) error {
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
	if errRes := client.CheckTendermintError(err, txBytes); errRes != nil {
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

func (c *NetworkClient) GetPubkey() cryptotypes.PubKey {
	return c.privKey.PubKey()
}

// Get Broadcast Address from config (privatekey or mnemonic)
func (c *NetworkClient) GetAddress() sdk.AccAddress {
	return c.addr
}
func (c *NetworkClient) getFeegranter() sdk.AccAddress {
	return c.addr
}
func (c *NetworkClient) GetRpcClient() (rpcclient.Client, error) {
	if c.rpcClient == nil && c.config.RPCUrl != "" {
		rpcClient, err := client.NewClientFromNode(c.config.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client with url %s : %w", c.config.RPCUrl, err)
		}
		c.rpcClient = rpcClient
	}

	return c.rpcClient, nil
}

// This function is call when the connection with current rpc client is lost
func (c *NetworkClient) RemoveRpcClient() {
	//Todo: Figure out how to clean up resource
	c.rpcClient = nil
}

func (c *NetworkClient) GetQueryClient() *QueryClient {
	return c.queryClient
}
func (c *NetworkClient) GetCurrentBlock(ctx context.Context) (*types.BlockTime, error) {
	client, err := c.GetRpcClient()
	if err != nil {
		return nil, err
	}
	block, err := client.Block(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &types.BlockTime{
		ResultBlock: *block,
		QueryTime:   time.Now(),
	}, nil
}
func (c *NetworkClient) GetNextBlockResults(ctx context.Context, blockHeight uint64) (*ctypes.ResultBlockResults, error) {
	client, err := c.GetRpcClient()
	if err != nil {
		return nil, err
	}
	height := int64(blockHeight)
	blockResults, err := client.BlockResults(ctx, &height)
	if err != nil {
		return nil, err
	}
	return blockResults, nil
}
