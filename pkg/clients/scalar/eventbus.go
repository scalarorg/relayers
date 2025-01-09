package scalar

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	switch event.EventType {
	case events.EVENT_ELECTRS_VAULT_TRANSACTION:
		//Broadcast from electrs.handleVaultTransaction
		return c.requestConfirmBtcTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_EVM_TOKEN_SENT, events.EVENT_EVM_CONTRACT_CALL, events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN:
		return c.requestConfirmEvmTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_BTC_PSBT_SIGN_REQUEST:
		return c.requestPsbtSign(event.DestinationChain, event.Data.(covtypes.Psbt))
	}

	return nil
}

func (c *Client) requestConfirmBtcTxs(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		_, err := c.ConfirmBtcTxs(context.Background(), sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [handleElectrsVaultTransaction] error confirming txs")
			return err
		}
	} else {
		log.Error().Msgf("[ScalarClient] [handleElectrsVaultTransaction] no valid txs to confirm")
	}
	return nil
}

func (c *Client) requestConfirmEvmTxs(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		_, err := c.ConfirmEvmTxs(context.Background(), sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [handleEvmContractCall] error confirming txs")
			return err
		}
	} else {
		log.Debug().Str("sourceChain", sourceChain).Msg("[ScalarClient] [handleEvmContractCall] no valid txs to confirm")
	}
	return nil
}

func (c *Client) requestPsbtSign(destinationChain string, psbt covtypes.Psbt) error {
	signRes, err := c.network.SignBtcCommandsRequest(context.Background(), destinationChain, psbt)
	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
		return fmt.Errorf("[ScalarClient] [requestPsbtSign] failed to sign psbt request: %v, %w", signRes, err)
	}
	return nil
}

func (c *Client) extractValidConfirmTxs(confirmRequest events.ConfirmTxsRequest) (string, []string) {
	txHashes := make([]string, 0)
	for txHash, _ := range confirmRequest.TxHashs {
		txHashes = append(txHashes, txHash)
		// if c.globalConfig.ActiveChains[destChain] {
		// 	txHashes = append(txHashes, txHash)
		// } else {
		// 	log.Warn().
		// 		Str("sourceChain", confirmRequest.ChainName).
		// 		Str("txHash", txHash).
		// 		Str("destinationChain", destChain).
		// 		Msg("[ScalarClient] [extractValidConfirmTxs] invalid destination chain")
		// }
	}
	return confirmRequest.ChainName, txHashes
}
