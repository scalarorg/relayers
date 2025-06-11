package electrs

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/events"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
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

	err = c.findAndHandleRedeemTxs(header.Height)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [BlockchainHeaderHandler] Failed to handle redeem transaction")
		return fmt.Errorf("failed to handle redeem transaction: %w", err)
	}
	// blockHeight := events.ChainBlockHeight{
	// 	Chain:  c.electrumConfig.SourceChain,
	// 	Height: uint64(header.Height),
	// 	Hash:   header.Hash,
	// 	Time:   header.Time,
	// }
	blockHeader := chains.BlockHeader{
		Chain:       c.electrumConfig.SourceChain,
		BlockNumber: uint64(header.Height),
		BlockHash:   header.Hash,
		BlockTime:   uint64(header.Time),
	}
	err = c.dbAdapter.CreateBlockHeader(&blockHeader)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] create block header failed")
	}
	if c.eventBus != nil {
		log.Debug().Msgf("[ElectrumClient] [BlockchainHeaderHandler] Broadcasting new block event: %v", header.Height)
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_NEW_BLOCK,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             &blockHeader,
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
func (c *Client) findAndHandleRedeemTxs(blockHeight int) error {
	// Check executing redeem transactions in the relayer db, if the confirmation is enough, send to the event bus
	lastConfirmedBlockNumber := blockHeight - c.electrumConfig.Confirmations + 1
	redeemTxs, err := c.dbAdapter.FindExecutingRedeemTxs(c.electrumConfig.SourceChain, lastConfirmedBlockNumber)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [findAndHandleRedeemTxs] Failed to get executing redeem transactions from db")
		return fmt.Errorf("failed to get executing redeem transactions from db: %w", err)
	}
	if len(redeemTxs) == 0 {
		log.Debug().Int("blockHeight", blockHeight).Msgf("[ElectrumClient] [findAndHandleRedeemTxs] No pending redeem transactions with %d confirmations found", c.electrumConfig.Confirmations)
		return nil
	}
	return c.handleRedeemTxs(redeemTxs)
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
	//blockNumbers := make([]int64, 0)
	lastCheckpoint := c.getLastCheckpoint()
	for _, tx := range vaultTxs {
		// if !slices.Contains(blockNumbers, int64(tx.Height)) {
		// 	blockNumbers = append(blockNumbers, int64(tx.Height))
		// }
		if uint64(tx.Height) > lastCheckpoint.BlockNumber ||
			(uint64(tx.Height) == lastCheckpoint.BlockNumber && uint(tx.TxPosition) > lastCheckpoint.LogIndex) {
			lastCheckpoint.BlockNumber = uint64(tx.Height)
			lastCheckpoint.TxHash = tx.TxHash
			lastCheckpoint.LogIndex = uint(tx.TxPosition)
			lastCheckpoint.EventKey = tx.Key
		}
	}
	// if c.eventBus != nil {
	// 	c.eventBus.BroadcastEvent(&events.EventEnvelope{
	// 		EventType:        events.EVENT_ELECTRS_NEW_BLOCK,
	// 		DestinationChain: c.electrumConfig.SourceChain,
	// 		Data:             blockNumbers,
	// 	})
	// }

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
	blockNumbers := make([]uint64, 0)
	for _, tokenSent := range tokenSents {
		if !slices.Contains(blockNumbers, tokenSent.BlockNumber) {
			blockNumbers = append(blockNumbers, tokenSent.BlockNumber)
		}
		if c.electrumConfig.Confirmations <= 1 || c.currentHeight-int(tokenSent.BlockNumber) >= c.electrumConfig.Confirmations {
			tokenSent.Status = chains.TokenSentStatusVerifying
			confirmTxs.TxHashs[tokenSent.TxHash] = tokenSent.DestinationChain
		} else {
			log.Debug().Msgf("[ElectrumClient] [handleTokenSents] BridgeTx %s does not have enough %d confirmed yet",
				tokenSent.TxHash, c.electrumConfig.Confirmations)
		}
	}

	//3. store relay data to the db, update last checkpoint
	err := c.dbAdapter.SaveTokenSentsAndRemoveDuplicates(tokenSents)
	if err != nil {
		log.Error().Err(err).Msg("[ElectrumClient] [handleTokenSents] Failed to store token sents to the db")
		return fmt.Errorf("[ElectrumClient] [handleTokenSents] failed to store token sents to the db: %w", err)
	}
	if c.eventBus != nil {
		//4. Send to the event bus with destination chain is scalar for confirmation
		if len(confirmTxs.TxHashs) > 0 {
			log.Debug().Msgf("[ElectrumClient] [VaultTxMessageHandler] Broadcasting confirm tx request: %v", confirmTxs)
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data:             confirmTxs,
			})

		} else {
			log.Debug().Msgf("[ElectrumClient] [handleTokenSents] No tokensent have enough %d confirmations to broadcast", c.electrumConfig.Confirmations)
		}
	} else {
		log.Warn().Msg("[ElectrumClient] [handleTokenSents] event bus is undefined")
	}

	return nil
}

// Todo: update ContractCallWithToken status with execution confirmation from bitcoin network
func (c *Client) handleRedeemTxs(redeemTxs []*chains.BtcRedeemTx) error {
	log.Debug().Int("CurrentHeight", c.currentHeight).Msgf("[ElectrumClient] [handleRedeemTxs] Received %d redeem transactions", len(redeemTxs))

	//Group redeem txs by custodian group id, in each group we keep only txs with highest sequence number
	mapRedeemTxs := c.groupRedeemTxs(redeemTxs)

	if c.eventBus != nil && len(mapRedeemTxs) > 0 {
		for groupUid, redeemTxEvents := range mapRedeemTxs {
			log.Debug().Str("groupUid", groupUid).
				Uint64("Current SessionSequence", redeemTxEvents.Sequence).
				Uint64("RedeemTx BlockNumber", redeemTxEvents.BlockNumber).
				Msgf("[ElectrumClient] [handleRedeemTxs] Broadcasting confirm redeem tx request: %v", redeemTxEvents)
			if redeemTxEvents.Phase == covExported.Executing {
				for _, tx := range redeemTxEvents.RedeemTxs {
					tx.Status = string(chains.RedeemStatusVerifying)
				}
				c.eventBus.BroadcastEvent(&events.EventEnvelope{
					EventType:        events.EVENT_ELECTRS_REDEEM_TRANSACTION,
					DestinationChain: events.SCALAR_NETWORK_NAME,
					Data:             redeemTxEvents,
				})
			} else {
				log.Debug().Msg("[ElectrumClient] [handleRedeemTxs] current session is not in executing phase")
			}
		}
	}
	//1. Store redeem transactions to the db

	err := c.dbAdapter.SaveBtcRedeemTxs(c.electrumConfig.SourceChain, redeemTxs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store redeem transactions to the db")
		return fmt.Errorf("failed to store redeem transactions to the db: %w", err)
	}
	return nil
}

// Group redeem txs by custodian group uid with condition:
// Redeem txs have enough confirmations and highest sequence number
func (c *Client) groupRedeemTxs(redeemTxs []*chains.BtcRedeemTx) map[string]*events.BtcRedeemTxEvents {
	mapRedeemTxs := make(map[string]*events.BtcRedeemTxEvents)
	redeemSessions := make(map[string]*covTypes.RedeemSession)
	//Store group uids that have been queried to ensure each group uid is only queried once
	queriedGroupUids := make(map[string]bool)
	for _, redeemTx := range redeemTxs {
		txConfirmations := c.currentHeight - int(redeemTx.BlockNumber) + 1
		if txConfirmations < c.electrumConfig.Confirmations {
			continue
		}
		//Find current redeem session
		redeemSession, ok := redeemSessions[redeemTx.CustodianGroupUid]

		queried := queriedGroupUids[redeemTx.CustodianGroupUid]
		if !ok && !queried {
			queriedGroupUids[redeemTx.CustodianGroupUid] = true
			redeemSessionRes, err := c.scalarClient.GetRedeemSession(redeemTx.CustodianGroupUid)
			if err != nil || redeemSessionRes == nil || redeemSessionRes.Session == nil {
				continue
			} else {
				redeemSession = redeemSessionRes.Session
				redeemSessions[redeemTx.CustodianGroupUid] = redeemSession
			}
		}
		if redeemSession == nil {
			log.Debug().Str("groupUid", redeemTx.CustodianGroupUid).
				Msgf("[ElectrumClient] [groupRedeemTxs] Cannot find redeem session for group uid")
			continue
		}
		if redeemSession.Sequence != redeemTx.SessionSequence {
			log.Debug().Str("groupUid", redeemTx.CustodianGroupUid).
				Str("Sequence", fmt.Sprintf("CurrentSession %d, RedeemTx %d", redeemSession.Sequence, redeemTx.SessionSequence)).
				Any("Current RedeemSession", redeemSession).
				Any("RedeemTx", redeemTx).
				Msgf("[ElectrumClient] [groupRedeemTxs] RedeemTx don't belong to current redeem session")
			continue
		}
		redeemTxEvents := mapRedeemTxs[redeemTx.CustodianGroupUid]
		if redeemTxEvents == nil {
			redeemTxEvents = &events.BtcRedeemTxEvents{
				Chain:       c.electrumConfig.SourceChain,
				GroupUid:    redeemTx.CustodianGroupUid,
				Sequence:    redeemSession.Sequence,
				Phase:       redeemSession.CurrentPhase,
				BlockNumber: redeemTx.BlockNumber,
				RedeemTxs:   []*chains.BtcRedeemTx{redeemTx},
			}
		} else {
			//TODO: keep only latest redeem tx
			if redeemTx.BlockNumber > redeemTxEvents.BlockNumber {
				redeemTxEvents.BlockNumber = redeemTx.BlockNumber
				redeemTxEvents.RedeemTxs = []*chains.BtcRedeemTx{redeemTx}
			} else if redeemTx.BlockNumber == redeemTxEvents.BlockNumber {
				redeemTxEvents.RedeemTxs = append(redeemTxEvents.RedeemTxs, redeemTx)
			}
		}
		mapRedeemTxs[redeemTx.CustodianGroupUid] = redeemTxEvents
	}
	return mapRedeemTxs
}
