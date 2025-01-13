package electrs

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) BlockchainHeaderHandler(header *types.BlockchainHeader, err error) error {
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [BlockchainHeaderHandler] Failed to receive block chain header")
		return fmt.Errorf("failed to parse block chain header: %w", err)
	}
	if header == nil {
		log.Debug().Msg("[ElectrumClient] [BlockchainHeaderHandler] No blockchain header received")
		return nil
	}
	c.currentHeight = header.Height
	// Check pending vault transactions in the relayer db, if the confirmation is enough, send to the event bus
	lastConfirmedBlockNumber := header.Height - c.electrumConfig.Confirmations + 1
	tokenSents, err := c.dbAdapter.FindPendingBtcTokenSent(c.electrumConfig.SourceChain, lastConfirmedBlockNumber)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [BlockchainHeaderHandler] Failed to get pending vault transactions from db")
		return fmt.Errorf("failed to get pending vault transactions from db: %w", err)
	}
	if len(tokenSents) == 0 {
		log.Debug().Msgf("[ElectrumClient] [BlockchainHeaderHandler] No pending vault transactions with %d confirmations found", c.electrumConfig.Confirmations)
		return nil
	}
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: c.electrumConfig.SourceChain,
		TxHashs:   make(map[string]string),
	}

	for _, tokenSent := range tokenSents {
		tokenSent.Status = chains.TokenSentStatusVerifying
		confirmTxs.TxHashs[tokenSent.TxHash] = tokenSent.DestinationChain
	}
	c.dbAdapter.SaveValuesWithCheckpoint(tokenSents, nil)
	if c.eventBus != nil {
		log.Debug().Msgf("[ElectrumClient] [BlockchainHeaderHandler] Broadcasting confirm tx request: %v", confirmTxs)
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[ElectrumClient] [BlockchainHeaderHandler] event bus is undefined")
	}
	return nil
}

// Handle vault messages
// Todo: Add some logging, metric and error handling if needed
func (c *Client) vaultTxMessageHandler(vaultTxs []types.VaultTransaction, err error) error {
	if err != nil {
		log.Warn().Msgf("[ElectrumClient] [vaultTxMessageHandler] Failed to receive vault transaction: %v", err)
		return fmt.Errorf("failed to receive vault transaction: %w", err)
	}
	if len(vaultTxs) == 0 {
		log.Debug().Msg("[ElectrumClient] [vaultTxMessageHandler] No vault transactions received")
		return nil
	}
	c.PreProcessVaultsMessages(vaultTxs)
	//1. parse vault transactions to relay data
	tokenSents, err := c.CreateTokenSents(vaultTxs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert vault transaction to relay data")
		return fmt.Errorf("failed to convert vault transaction to relay data: %w", err)
	} else {
		log.Debug().Msgf("Successfully stored %d vault transactions to relay data", len(tokenSents))
	}
	//2. update last checkpoint
	lastCheckpoint := c.getLastCheckpoint()
	for _, tx := range vaultTxs {
		if uint64(tx.Height) > lastCheckpoint.BlockNumber ||
			(uint64(tx.Height) == lastCheckpoint.BlockNumber && uint(tx.TxPosition) > lastCheckpoint.LogIndex) {
			lastCheckpoint.BlockNumber = uint64(tx.Height)
			lastCheckpoint.TxHash = tx.TxHash
			lastCheckpoint.LogIndex = uint(tx.TxPosition)
			lastCheckpoint.EventKey = tx.Key
		}
	}
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: c.electrumConfig.SourceChain,
		TxHashs:   make(map[string]string),
	}
	//If confirmations is 1, send to the event bus with destination chain is scalar for confirmation
	//If confirmations is greater than 1, wait for the next blocks to get more confirmations before broadcasting to the scalar network
	for _, tokenSent := range tokenSents {
		if c.electrumConfig.Confirmations <= 1 || c.currentHeight-int(tokenSent.BlockNumber) >= c.electrumConfig.Confirmations {
			tokenSent.Status = chains.TokenSentStatusVerifying
			confirmTxs.TxHashs[tokenSent.TxHash] = tokenSent.DestinationChain
		}
	}

	//3. store relay data to the db, update last checkpoint
	err = c.dbAdapter.SaveValuesWithCheckpoint(tokenSents, lastCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store relay data to the db")
		return fmt.Errorf("failed to store relay data to the db: %w", err)
	}

	//4. Send to the event bus with destination chain is scalar for confirmation
	if len(confirmTxs.TxHashs) > 0 {
		if c.eventBus != nil {
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data:             confirmTxs,
			})
		} else {
			log.Warn().Msg("[ElectrumClient] [vaultTxMessageHandler] event bus is undefined")
		}
	}

	return nil
}

// Todo: Log and validate incomming message
func (c *Client) PreProcessVaultsMessages(vaultTxs []types.VaultTransaction) error {
	log.Info().Msgf("Received %d vault transactions", len(vaultTxs))
	for _, vaultTx := range vaultTxs {
		log.Debug().Msgf("Received vaultTx with key=>%v; stakerAddress=>%v; stakerPubkey=>%v", vaultTx.Key, vaultTx.StakerAddress, vaultTx.StakerPubkey)
	}
	return nil
}
