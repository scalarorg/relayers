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

	err = c.tryConfirmTokenSents(header.Height)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [BlockchainHeaderHandler] Failed to handle token sents")
		return fmt.Errorf("failed to handle token sents: %w", err)
	}

	err = c.tryHandleRedeemsTransaction(header.Height)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [BlockchainHeaderHandler] Failed to handle redeem transaction")
		return fmt.Errorf("failed to handle redeem transaction: %w", err)
	}
	if c.eventBus != nil {
		log.Debug().Msgf("[ElectrumClient] [BlockchainHeaderHandler] Broadcasting new block event: %v", header.Height)
		blockHeight := events.ChainBlockHeight{
			Chain:  c.electrumConfig.SourceChain,
			Height: uint64(header.Height),
		}
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_NEW_BLOCK,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             blockHeight,
		})
	} else {
		log.Warn().Msg("[ElectrumClient] [BlockchainHeaderHandler] event bus is undefined")
	}
	return nil
}

func (c *Client) tryConfirmTokenSents(blockHeight int) error {
	// Check pending vault transactions in the relayer db, if the confirmation is enough, send to the event bus
	lastConfirmedBlockNumber := blockHeight - c.electrumConfig.Confirmations + 1

	tokenSents, err := c.dbAdapter.FindPendingBtcTokenSent(c.electrumConfig.SourceChain, lastConfirmedBlockNumber)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [tryConfirmTokenSents] Failed to get pending vault transactions from db")
		return fmt.Errorf("failed to get pending vault transactions from db: %w", err)
	}
	if len(tokenSents) == 0 {
		log.Debug().Msgf("[ElectrumClient] [tryConfirmTokenSents] No pending vault transactions with %d confirmations found", c.electrumConfig.Confirmations)
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
	err = c.dbAdapter.SaveTokenSents(tokenSents)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [tryConfirmTokenSents] Failed to save token sents")
		return fmt.Errorf("failed to save token sents: %w", err)
	}
	if c.eventBus != nil {
		log.Debug().Msgf("[ElectrumClient] [tryConfirmTokenSents] Broadcasting confirm tx request: %v", confirmTxs)
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[ElectrumClient] [tryConfirmTokenSents] event bus is undefined")
	}
	return nil
}
func (c *Client) tryHandleRedeemsTransaction(blockHeight int) error {
	// Check pending redeem transactions in the relayer db, if the confirmation is enough, send to the event bus
	lastConfirmedBlockNumber := blockHeight - c.electrumConfig.Confirmations + 1
	redeemTxs, err := c.dbAdapter.FindPendingRedeemsTransaction(c.electrumConfig.SourceChain, lastConfirmedBlockNumber)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [tryHandleRedeemTransaction] Failed to get pending redeem transactions from db")
		return fmt.Errorf("failed to get pending redeem transactions from db: %w", err)
	}
	if len(redeemTxs) == 0 {
		log.Debug().Msgf("[ElectrumClient] [tryHandleRedeemTransaction] No pending redeem transactions with %d confirmations found", c.electrumConfig.Confirmations)
		return nil
	}
	txHashes := make([]string, len(redeemTxs))
	for i, redeemTx := range redeemTxs {
		txHashes[i] = redeemTx.TxHash
		redeemTx.Status = string(chains.RedeemStatusVerifying)
	}
	confirmRedeemTx := events.ConfirmRedeemTxRequest{
		Chain:   c.electrumConfig.SourceChain,
		TxHashs: txHashes,
	}
	err = c.dbAdapter.SaveRedeemTxs(redeemTxs)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [tryHandleRedeemTransaction] Failed to save redeem transactions")
		return fmt.Errorf("failed to save redeem transactions: %w", err)
	}
	if c.eventBus != nil {
		log.Debug().Msgf("[ElectrumClient] [tryHandleRedeemTransaction] Broadcasting confirm redeem tx request: %v", confirmRedeemTx)
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_REDEEM_TRANSACTION,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmRedeemTx,
		})
	} else {
		log.Warn().Msg("[ElectrumClient] [tryHandleRedeemTransaction] event bus is undefined")
	}
	return nil
}

// Handle vault messages
// Todo: Add some logging, metric and error handling if needed
func (c *Client) VaultTxMessageHandler(vaultTxs []types.VaultTransaction, err error) error {
	if err != nil {
		log.Warn().Msgf("[ElectrumClient] [vaultTxMessageHandler] Failed to receive vault transaction: %v", err)
		return fmt.Errorf("failed to receive vault transaction: %w", err)
	}
	if len(vaultTxs) == 0 {
		log.Debug().Msg("[ElectrumClient] [vaultTxMessageHandler] No vault transactions received")
		return nil
	}
	c.PreProcessVaultsMessages(vaultTxs)
	//1. parse vault transactions to token sent and unstaked vault txs
	tokenSents, redeemTxs := c.CategorizeVaultTxs(vaultTxs)
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
	err = c.dbAdapter.UpdateLastEventCheckPoint(lastCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update last event checkpoint")
	}
	if len(tokenSents) == 0 {
		log.Warn().Msg("No Valid vault transactions to convert to relay data")
	} else {
		log.Debug().Int("CurrentHeight", c.currentHeight).Msgf("[ElectrumClient] [VaultTxMessageHandler] Received %d validvault transactions", len(tokenSents))
	}
	// Redeem transaction
	if len(redeemTxs) > 0 {
		err = c.handleRedeemTxs(redeemTxs)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update unstaked vault transactions")
		}
	}
	if len(tokenSents) > 0 {
		err := c.handleTokenSents(tokenSents)
		if err != nil {
			log.Error().Err(err).Msg("Failed to handle token sents")
		}
		return err
	}
	return nil
}

// Todo: Log and validate incomming message
func (c *Client) PreProcessVaultsMessages(vaultTxs []types.VaultTransaction) error {
	log.Info().Msgf("Received %d vault transactions", len(vaultTxs))
	for _, vaultTx := range vaultTxs {
		log.Debug().Msgf("Received vaultTx with key=>%v; stakerAddress=>%v; stakerPubkey=>%v, destChain=>%v; destTokenAddress=>%v, destRecipientAddress=>%v", vaultTx.Key, vaultTx.StakerAddress, vaultTx.StakerPubkey, vaultTx.DestChain, vaultTx.DestTokenAddress, vaultTx.DestRecipientAddress)
	}
	return nil
}
func (c *Client) handleTokenSents(tokenSents []*chains.TokenSent) error {
	log.Debug().Int("CurrentHeight", c.currentHeight).Msgf("[ElectrumClient] [handleTokenSents] Received %d token sent transactions", len(tokenSents))
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
		} else {
			log.Debug().Msgf("[ElectrumClient] [handleTokenSents] BridgeTx %s does not have enough %d confirmed yet",
				tokenSent.TxHash, c.electrumConfig.Confirmations)
		}
	}

	//3. store relay data to the db, update last checkpoint
	err := c.dbAdapter.SaveTokenSentsAndRemoveDupplicates(tokenSents)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store relay data to the db")
		return fmt.Errorf("failed to store relay data to the db: %w", err)
	}
	//4. Send to the event bus with destination chain is scalar for confirmation
	if len(confirmTxs.TxHashs) > 0 {
		if c.eventBus != nil {
			log.Debug().Msgf("[ElectrumClient] [VaultTxMessageHandler] Broadcasting confirm tx request: %v", confirmTxs)
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data:             confirmTxs,
			})
		} else {
			log.Warn().Msg("[ElectrumClient] [vaultTxMessageHandler] event bus is undefined")
		}
	} else {
		log.Debug().Msgf("[ElectrumClient] [handleTokenSents] No tokensent have enough %d confirmations to broadcast", c.electrumConfig.Confirmations)
	}
	return nil
}

// Todo: update ContractCallWithToken status with execution confirmation from bitcoin network
func (c *Client) handleRedeemTxs(redeemTxs []*chains.RedeemTx) error {
	log.Debug().Int("CurrentHeight", c.currentHeight).Msgf("[ElectrumClient] [handleRedeemTxs] Received %d redeem transactions", len(redeemTxs))
	//1. Store redeem transactions to the db
	err := c.dbAdapter.SaveRedeemTxs(redeemTxs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store redeem transactions to the db")
		return fmt.Errorf("failed to store redeem transactions to the db: %w", err)
	}
	//2. Update ContractCallWithToken status with execution confirmation from bitcoin network
	txHashes := make([]string, len(redeemTxs))
	for i, tx := range redeemTxs {
		txHashes[i] = tx.TxHash
	}
	c.dbAdapter.UpdateRedeemExecutedCommands(c.electrumConfig.SourceChain, txHashes)
	//3. Filter redeem with enough confirmations and group by custodian group id
	mapTxHashs := make(map[string][]string)

	//If confirmations is 1, send to the event bus with destination chain is scalar for confirmation
	//If confirmations is greater than 1, wait for the next blocks to get more confirmations before broadcasting to the scalar network
	for _, redeemTx := range redeemTxs {
		if c.electrumConfig.Confirmations <= 1 || c.currentHeight-int(redeemTx.BlockNumber) >= c.electrumConfig.Confirmations {
			mapTxHashs[redeemTx.CustodianGroupUid] = append(mapTxHashs[redeemTx.CustodianGroupUid], redeemTx.TxHash)
		}
	}
	if c.eventBus != nil {
		for _, txHashs := range mapTxHashs {
			if len(txHashs) > 0 {
				confirmRedeemTx := events.ConfirmRedeemTxRequest{
					Chain:   c.electrumConfig.SourceChain,
					TxHashs: txHashs,
				}

				log.Debug().Msgf("[ElectrumClient] [handleRedeemTxs] Broadcasting confirm redeem tx request: %v", confirmRedeemTx)
				c.eventBus.BroadcastEvent(&events.EventEnvelope{
					EventType:        events.EVENT_ELECTRS_REDEEM_TRANSACTION,
					DestinationChain: events.SCALAR_NETWORK_NAME,
					Data:             confirmRedeemTx,
				})
			}
		}
	}
	return nil
}
