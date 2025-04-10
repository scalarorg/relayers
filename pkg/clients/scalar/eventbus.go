package scalar

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	switch event.EventType {
	case events.EVENT_ELECTRS_VAULT_TRANSACTION:
		//Broadcast from electrs.handleVaultTransaction
		return c.requestConfirmBtcTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_ELECTRS_REDEEM_TRANSACTION:
		//Broadcast from electrs.handleRedeemTransaction
		return c.broadcaster.ConfirmRedeemTxRequest(event.Data.(events.ConfirmRedeemTxRequest))
	case events.EVENT_ELECTRS_NEW_BLOCK:
		return c.handleElectrsEventNewBlock(event.Data.(events.ChainBlockHeight))
	case events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL,
		events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		events.EVENT_EVM_REDEEM_TOKEN:
		return c.requestConfirmEvmTxs(event.Data.(events.ConfirmTxsRequest))
	case events.EVENT_EVM_TOKEN_DEPLOYED:
		return c.requestConfirmTokenDeployed(event.Data.(*chains.TokenDeployed))
	case events.EVENT_EVM_SWITCHED_PHASE:
		return c.requestConfirmSwitchedPhase(event.Data.(*chains.SwitchedPhase))
		// case events.EVENT_BTC_PSBT_SIGN_REQUEST:
		// 	return c.requestPsbtSign(event.Data.(types.SignPsbtsRequest))
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
func (c *Client) handleElectrsEventNewBlock(blockHeight events.ChainBlockHeight) error {
	log.Debug().Uint64("blockHeight", blockHeight.Height).Msg("[ScalarClient] [handleElectrsEventNewBlock] Received new block event")
	// Todo: init utxo for first time
	if !c.initUtxo {
		c.broadcaster.InitializeUtxoRequest(blockHeight.Chain, blockHeight.Height)
		c.initUtxo = true
	}
	return nil
}

// Add psbts to pendingChainPsbtCommands
// func (c *Client) requestPsbtSign(psbt types.SignPsbtsRequest) error {
// 	log.Debug().Str("ChainName", psbt.ChainName).Int("psbtCount", len(psbt.Psbts)).Msgf("[ScalarClient] [requestPsbtSign] Set psbts to pendingChainPsbtCommands")
// 	c.pendingCommands.StorePsbts(psbt.ChainName, psbt.Psbts)
// 	return nil
// }

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

func (c *Client) requestConfirmTokenDeployed(tokenDeployed *chains.TokenDeployed) error {
	//Try to query to the scalar network, if token is not confirms then broadcast confirm request
	tokenInfo, err := c.GetTokenInfo(context.Background(), tokenDeployed.Chain, tokenDeployed.Symbol)
	if err != nil || tokenInfo == nil {
		return fmt.Errorf("failed to get token info: %w", err)
	}
	if tokenInfo.Confirmed {
		log.Debug().
			Str("TxHash", tokenDeployed.TxHash).
			Str("TokenAddress", tokenInfo.Address).
			Str("Symbol", tokenDeployed.Symbol).
			Msgf("[ScalarClient] [requestConfirmTokenDeployed] Token is confirmed")
		tokenDeployed.TokenAddress = tokenInfo.Address
	} else {
		log.Debug().
			Str("TxHash", tokenDeployed.TxHash).
			Str("TokenAddress", tokenDeployed.TokenAddress).
			Str("Symbol", tokenDeployed.Symbol).
			Msgf("[ScalarClient] [requestConfirmTokenDeployed] Confirm token deployed")
		c.broadcaster.ConfirmTokenDeployed(tokenDeployed)
	}
	return nil
}

func (c *Client) requestConfirmSwitchedPhase(switchedPhase *chains.SwitchedPhase) error {
	log.Debug().
		Str("TxHash", switchedPhase.TxHash).
		Uint64("SessionSequence", switchedPhase.SessionSequence).
		Uint8("From", switchedPhase.From).
		Uint8("To", switchedPhase.To).
		Msgf("[ScalarClient] [requestConfirmSwitchedPhase] Confirm switched phase")
	c.broadcaster.ConfirmSwitchedPhase(switchedPhase)
	return nil
}
