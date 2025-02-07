package scalar

import (
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	switch event.EventType {
	case events.EVENT_ELECTRS_VAULT_TRANSACTION:
		//Broadcast from electrs.handleVaultTransaction
		return c.requestConfirmBtcTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_EVM_TOKEN_SENT, events.EVENT_EVM_CONTRACT_CALL, events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN:
		return c.requestConfirmEvmTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_BTC_PSBT_SIGN_REQUEST:
		return c.requestPsbtSign(event.Data.(types.SignPsbtsRequest))
	}

	return nil
}

func (c *Client) requestConfirmBtcTxs(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		err := c.broadcaster.ConfirmBtcTxs(sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [requestConfirmBtcTxs] enqueue confirming bitcoin txs failed")
			return err
		}
	} else {
		log.Error().Msgf("[ScalarClient] [requestConfirmBtcTxs] no valid txs to confirm")
	}
	return nil
}

func (c *Client) requestConfirmEvmTxs(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		err := c.broadcaster.ConfirmEvmTxs(sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [requestConfirmEvmTxs] enqueue confirm evm txs failed")
			return err
		} else {
			log.Debug().Str("sourceChain", sourceChain).Msgf("[ScalarClient] [requestConfirmEvmTxs] successfully enqueue confirm txs %v", txHashes)
		}
	} else {
		log.Debug().Str("sourceChain", sourceChain).Msg("[ScalarClient] [requestConfirmEvmTxs] no valid txs to confirm")
	}
	return nil
}

// Add psbts to pendingChainPsbtCommands
func (c *Client) requestPsbtSign(psbt types.SignPsbtsRequest) error {
	log.Debug().Str("ChainName", psbt.ChainName).Int("psbtCount", len(psbt.Psbts)).Msgf("[ScalarClient] [requestPsbtSign] Set psbts to pendingChainPsbtCommands")
	c.pendingCommands.StorePsbts(psbt.ChainName, psbt.Psbts)
	return nil
}

func (c *Client) extractValidConfirmTxs(confirmRequest events.ConfirmTxsRequest) (string, []string) {
	txHashes := make([]string, 0)
	for txHash := range confirmRequest.TxHashs {
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
