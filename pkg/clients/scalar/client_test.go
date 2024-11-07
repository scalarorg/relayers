package scalar_test

import (
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type RelayerTx struct {
	txConfig client.TxConfig
}

func NewRelayerTx(txConfig client.TxConfig) *RelayerTx {
	return &RelayerTx{txConfig: txConfig}
}
func (r *RelayerTx) CreateConfirmTxPayload(sender sdk.AccAddress, chain string, txId string) (client.TxBuilder, error) {
	txHash := emvtypes.Hash(common.HexToHash(txId))

	msg := emvtypes.NewConfirmGatewayTxRequest(sender, chain, txHash)

	// Create a new transaction builder
	txBuilder := r.txConfig.NewTxBuilder()
	// Set the message
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}
	txBuilder = r.SetExtraParams(txBuilder)
	return txBuilder, nil
}
func (r *RelayerTx) SetExtraParams(txBuilder client.TxBuilder) client.TxBuilder {
	txBuilder.SetGasLimit(200000) // Adjust as needed
	txBuilder.SetFeeAmount(types.NewCoins(types.NewCoin("stake", types.NewInt(1000))))
	txBuilder.SetMemo("") // Optional memo
	return txBuilder
}

// func (r *RelayerTx) CreateTransaction(ctx context.Context, msg types.Msg) (client.TxBuilder, error) {
// 	// Create a new transaction builder
// 	txBuilder := r.txConfig.NewTxBuilder()

// 	// Set the message
// 	err := txBuilder.SetMsgs(msg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	txBuilder = r.SetExtraParams(txBuilder)

// 	// Sign the transaction
// 	signerData := signingtypes.SignerData{
// 		ChainID:       "your-chain-id",
// 		AccountNumber: 0, // Get from account query
// 		Sequence:      0, // Get from account query
// 	}
// 	// Sign the transaction
// 	sigV2, err := client_tx.SignWithPrivKey(
// 		ctx,
// 		signing.SignMode_SIGN_MODE_DIRECT,
// 		signerData,
// 		txBuilder,
// 		privKey,
// 		c.txConfig,
// 		uint64(0),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = txBuilder.SetSignatures(sigV2)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return txBuilder, nil
// }
