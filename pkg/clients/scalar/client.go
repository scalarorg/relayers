package scalar

import (
	"context"
	"fmt"

	"github.com/axelarnetwork/axelar-core/utils"
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	nexus "github.com/axelarnetwork/axelar-core/x/nexus/exported"

	//tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codec_types "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Client struct {
	txConfig client.TxConfig
	network  *NetworkClient
	// Add other necessary fields like chain ID, gas prices, etc.
}

func NewClient(config *Config) (*Client, error) {
	txConfig := tx.NewTxConfig(codec.NewProtoCodec(codec_types.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	networkClient, err := NewNetworkClient(config, txConfig)
	if err != nil {
		return nil, err
	}
	client := &Client{
		network:  networkClient,
		txConfig: tx.NewTxConfig(codec.NewProtoCodec(codec_types.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
	}
	return client, nil
}

// Todo: Seperate the function for BTC and EVM
func (c *Client) ConfirmBtcTx(ctx context.Context, chainName string, txId string) (*sdk.TxResponse, error) {
	return c.ConfirmEvmTx(ctx, chainName, txId)
}

// Relayer call this function for request Scalar network to confirm the transaction on the source chain
func (c *Client) ConfirmEvmTx(ctx context.Context, chainName string, txId string) (*sdk.TxResponse, error) {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	txHash := emvtypes.Hash(common.HexToHash(txId))
	msg := emvtypes.NewConfirmGatewayTxsRequest(c.network.getAddress(), nexusChain, []emvtypes.Hash{txHash})

	//2. Sign and broadcast the payload using the network client, which has the private key
	confirmTx, err := c.network.ConfirmEvmTx(ctx, msg)
	if err != nil {
		return nil, err
	}

	return confirmTx, nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L159
func (c *Client) Subscribe(ctx context.Context, subscriber string, event Event, callback func(data tmtypes.TMEventData, err error) error) error {
	eventCh, err := c.network.Subscribe(ctx, subscriber, event.TopicId)
	if err != nil {
		return fmt.Errorf("failed to subscribe to Event: %v, %w", event, err)
	}
	defer c.network.UnSubscribeAll(context.Background(), subscriber) //nolint:errcheck // ignore
	select {
	case evt := <-eventCh:
		if evt.Query != event.TopicId {
			callback(nil, fmt.Errorf("event query is not match"))
		} else {
			callback(evt.Data, nil)
		}
	case <-ctx.Done():
		return errors.ErrLogic.Wrapf("timed out waiting for event, the transaction could have already been included or wasn't yet included")
	}
	return nil
}
