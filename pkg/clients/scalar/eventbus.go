package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().Msgf("[ScalarClient] [handleEventBusMessage]: %v", event)
	switch event.EventType {
	case events.EVENT_ELECTRS_VAULT_TRANSACTION:
		//Broadcast from electrs.handleVaultTransaction
		return c.handleElectrsVaultTransaction(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_EVM_CONTRACT_CALL:
		//Broadcast from evm.handleContractCall
		return c.handleEvmContractCall(event.Data.(events.ConfirmTxsRequest))
	}
	return nil
}

func (c *Client) handleElectrsVaultTransaction(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		_, err := c.ConfirmTxs(context.Background(), sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [handleElectrsVaultTransaction] error confirming txs")
			return err
		}
	} else {
		log.Error().Msgf("[ScalarClient] [handleElectrsVaultTransaction] no valid txs to confirm")
	}
	return nil
}

func (c *Client) handleEvmContractCall(confirmRequest events.ConfirmTxsRequest) error {
	sourceChain, txHashes := c.extractValidConfirmTxs(confirmRequest)
	if len(txHashes) > 0 {
		_, err := c.ConfirmTxs(context.Background(), sourceChain, txHashes)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [handleEvmContractCall] error confirming txs")
			return err
		}
	} else {
		log.Debug().Str("sourceChain", sourceChain).Msg("[ScalarClient] [handleEvmContractCall] no valid txs to confirm")
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
