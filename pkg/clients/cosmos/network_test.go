package cosmos_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/cmd"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/scalar-core/utils"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

const (
	chainNameBtcTestnet4 = "bitcoin-testnet4"
)

var (
	protoCodec           = encoding.GetCodec(proto.Name)
	DefaultGlobalConfig  = config.Config{}
	DefaultNetworkConfig = cosmos.CosmosNetworkConfig{
		ChainID:       73475,
		ID:            "cosmos|73475",
		Name:          "scalar-testnet-1",
		Denom:         "ascal",
		RPCUrl:        "http://localhost:26657",
		GasPrice:      0.0125,
		LCDUrl:        "http://localhost:2317",
		WSUrl:         "ws://localhost:26657/websocket",
		MaxRetries:    3,
		RetryInterval: int64(1000),
		BroadcastMode: "sync",
		Mnemonic:      "latin total dream gesture brain bunker truly stove left video cost transfer guide occur bicycle oxygen world ready witness exhibit federal salute half day",
	}
	err           error
	clientCtx     *client.Context
	accAddress    sdk.AccAddress
	txConfig      client.TxConfig
	networkClient *cosmos.NetworkClient
)

func TestMain(m *testing.M) {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount(cmd.AccountAddressPrefix, cmd.AccountPubKeyPrefix)
	config.SetBech32PrefixForValidator(cmd.ValidatorAddressPrefix, cmd.ValidatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(cmd.ConsNodeAddressPrefix, cmd.ConsNodePubKeyPrefix)
	txConfig = authTx.NewTxConfig(codec.GetProtoCodec(), authTx.DefaultSignModes)
	clientCtx, err = cosmos.CreateClientContext(&DefaultNetworkConfig)
	queryClient := cosmos.NewQueryClient(clientCtx)
	networkClient, err = cosmos.NewNetworkClient(&DefaultNetworkConfig, queryClient, txConfig)
	os.Exit(m.Run())
}
func TestSubscribeContractCallApprovedEvent(t *testing.T) {
	require.NoError(t, err)
	require.NotNil(t, networkClient)
	_, err = networkClient.Start()
	require.NoError(t, err)
	//queryNewBlockHeader := "tm.event='NewBlockHeader'"
	queryContractCallApproved := scalar.ContractCallApprovedEventTopicId
	//queryEventCompleted := "tm.event='NewBlock' AND scalar.chains.v1beta1.EVMEventCompleted.event_id EXISTS"
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
	txHashs := make([]chainstypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainstypes.Hash(common.HexToHash(txId))
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

func TestSignCommandsRequest(t *testing.T) {
	ctx := context.Background()
	destinationChain := "evm|11155111"
	req := chainstypes.NewSignCommandsRequest(
		networkClient.GetAddress(),
		destinationChain)
	txBuilder, err := networkClient.CreateTxBuilder(ctx, req)
	require.NoError(t, err)
	txf := networkClient.CreateTxFactory(ctx)
	log.Debug().Msgf("[ScalarNetworkClient] [trySignAndBroadcastMsgs] account sequence: %d", txf.Sequence())
	err = networkClient.SignTx(txf, txBuilder, true)
	require.NoError(t, err)

	//2. Encode the transaction for Broadcasting
	tx := txBuilder.GetTx()
	err = verifySignature(txf, networkClient.GetPubkey(), tx)
	require.NoError(t, err)
	// txRes, err := networkClient.SignAndBroadcastMsgs(ctx, req)
	// require.NoError(t, err)
	// fmt.Printf("%v", txRes)
}

func verifySignature(txf tx.Factory, pubKey cryptotypes.PubKey, msg authsigning.Tx) error {
	sigTx, ok := msg.(authsigning.SigVerifiableTx)
	if !ok {
		return fmt.Errorf("invalid tx %v", msg)
	}
	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return err
	}

	signerAddrs := sigTx.GetSigners()

	// check that signer length and signature length are the same
	if len(sigs) != len(signerAddrs) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "invalid number of signer;  expected: %d, got %d", len(signerAddrs), len(sigs))
	}
	fmt.Printf("Sigs %v\n", sigs)
	signerData := authsigning.SignerData{
		ChainID:       txf.ChainID(),
		AccountNumber: txf.AccountNumber(),
		Sequence:      txf.Sequence(),
	}
	handler := txConfig.SignModeHandler()
	for _, sig := range sigs {
		switch data := sig.Data.(type) {
		case *signing.SingleSignatureData:
			signBytes, err := handler.GetSignBytes(data.SignMode, signerData, msg)
			if err != nil {
				return err
			}
			if !pubKey.VerifySignature(signBytes, data.Signature) {
				return fmt.Errorf("unable to verify single signer signature")
			} else {
				fmt.Printf("Successfull verify SingleSignatureData %v", data)
			}

			return nil

		case *signing.MultiSignatureData:
			// multiPK, ok := pubKey.(multisig.PubKey)
			// if !ok {
			// 	return fmt.Errorf("expected %T, got %T", (multisig.PubKey)(nil), pubKey)
			// }
			// err := multiPK.VerifyMultisignature(func(mode signing.SignMode) ([]byte, error) {
			// 	return handler.GetSignBytes(mode, signerData, tx)
			// }, data)
			// if err != nil {
			// 	return err
			// }
			fmt.Printf("MultiSignatureData %v", data)
			return nil
		default:
			return fmt.Errorf("unexpected SignatureData %T", sig.Data)
		}
		// err = authsigning.VerifySignature(pubKey, signerData, sig.Data, svd.signModeHandler, tx)
		// 	return err
	}
	return nil
}
