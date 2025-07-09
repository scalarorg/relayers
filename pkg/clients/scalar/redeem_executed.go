package scalar

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
)

type BtcRedeemTxGroup struct {
	Chain           string
	GroupUid        string //Group uid is empty for upc redeem
	SessionSequence uint64
	TxHashs         []*chains.BtcRedeemTx
}

func (c *Client) StartRedeemExecutedProcessing(ctx context.Context) {
	log.Info().Msg("[ScalarClient] Starting bridge processing")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ScalarClient] Context cancelled, stopping bridge processing")
			return
		case <-ticker.C:
			if err := c.processNextRedeemExecuted(); err != nil {
				log.Error().Err(err).Msg("[ScalarClient] Failed to process vault block")
			}
		}
	}
}

// Find lastest redeem executed and process them
func (c *Client) processNextRedeemExecuted() error {
	var btcRedeemTxs []*chains.BtcRedeemTx
	var err error
	if c.processCheckPoint.LastRedeemBlock == nil {
		c.processCheckPoint.LastRedeemBlock, err = c.getLastRedeemBlock()
		if err != nil {
			return fmt.Errorf("failed to get next uncompleted vault block: %w", err)
		}
	}

	if c.processCheckPoint.LastRedeemBlock == nil {
		//No executed vault tx command, get the new vault txs
		btcRedeemTxs, err = c.dbAdapter.GetNextBtcRedeemExecuteds(0)
		if err != nil {
			return fmt.Errorf("failed to get new vault txs: %w", err)
		}
	} else {
		// Get uncompleted vault transactions for this block (not in command_executed)
		btcRedeemTxs, err = c.dbAdapter.GetBtcRedeemExecutedsByBlock(c.processCheckPoint.LastRedeemBlock.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted vault transactions for block %d: %w", c.processCheckPoint.LastRedeemBlock.BlockNumber, err)
		}
		if len(btcRedeemTxs) == 0 {
			log.Info().Uint64("blockNumber", c.processCheckPoint.LastRedeemBlock.BlockNumber).
				Str("status", c.processCheckPoint.LastRedeemBlock.Status).
				Msg("[ScalarClient] new unfinished redeem block to process. Get redeem txs from next block")
			btcRedeemTxs, err = c.dbAdapter.GetNextBtcRedeemExecuteds(c.processCheckPoint.LastRedeemBlock.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get redeem transactions for block %d: %w", c.processCheckPoint.LastRedeemBlock.BlockNumber, err)
			}
		}
	}

	if len(btcRedeemTxs) > 0 {
		redeemTxGroups := c.groupRedeemTxs(btcRedeemTxs)
		for _, redeemTxGroup := range redeemTxGroups {
			if redeemTxGroup.GroupUid == "" {
				//For upc redeem, we need to broadcast the redeem txs confirm request to the scalar gateway
				err = c.BroadcastRedeemTxsConfirmRequest(redeemTxGroup.Chain, redeemTxGroup.GroupUid, redeemTxGroup.TxHashs)
				if err != nil {
					log.Error().Err(err).Msg("[ScalarClient] Failed to confirm redeem executed transactions")
					return err
				}
			} else {
				err = c.BroadcastRedeemTxsConfirmRequest(redeemTxGroup.Chain, redeemTxGroup.GroupUid, redeemTxGroup.TxHashs)
				if err != nil {
					log.Error().Err(err).Msgf("[ScalarClient] [broadcastRedeemTxsConfirm] failed to broadcast redeem txs confirm request")
					return err
				}
			}
		}
	} else {
		log.Info().Msg("[ScalarClient] No redeem executed transactions to process.")
	}
	c.processCheckPoint.LastRedeemBlock = &relayer.RedeemBlock{
		BlockNumber:      btcRedeemTxs[0].BlockNumber,
		BlockHash:        btcRedeemTxs[0].BlockHash,
		Chain:            btcRedeemTxs[0].Chain,
		Status:           string(relayer.BlockStatusProcessing),
		TransactionCount: len(btcRedeemTxs),
		ProcessedTxCount: 0,
	}
	//Store vault block to the relayerdb
	c.dbAdapter.CreateRedeemBlock(c.processCheckPoint.LastRedeemBlock)
	return nil
}

func (c *Client) getLastRedeemBlock() (*relayer.RedeemBlock, error) {
	redeemBlock, err := c.dbAdapter.GetLastRedeemBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get last vault block: %w", err)
	}
	if redeemBlock == nil {
		log.Debug().Msg("[ScalarClient] No vault blocks processed")
		commandExecuteds, err := c.dbAdapter.FindLastExecutedCommands(c.networkConfig.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get uncompleted vault transactions for block %d: %w", 0, err)
		}
		if len(commandExecuteds) > 0 {
			redeemBlock = &relayer.RedeemBlock{
				BlockNumber:      commandExecuteds[0].BlockNumber,
				Chain:            c.networkConfig.GetId(),
				Status:           string(relayer.BlockStatusProcessing),
				TransactionCount: len(commandExecuteds),
				ProcessedTxCount: len(commandExecuteds),
			}
			err = c.dbAdapter.CreateRedeemBlock(redeemBlock)
			if err != nil {
				return nil, fmt.Errorf("failed to create vault block: %w", err)
			}
			log.Info().Int("vaultTxsCount", len(commandExecuteds)).
				Msg("[ScalarClient] create redeem new block")
			return redeemBlock, nil
		}
	}
	return redeemBlock, nil
}

func (c *Client) groupRedeemTxs(btcRedeemTxs []*chains.BtcRedeemTx) []*BtcRedeemTxGroup {
	redeemTxGroups := make([]*BtcRedeemTxGroup, 0)
	for _, tx := range btcRedeemTxs {
		var group *BtcRedeemTxGroup
		for _, group := range redeemTxGroups {
			if group.Chain == tx.Chain && group.GroupUid == tx.CustodianGroupUid && group.SessionSequence == tx.SessionSequence {
				group = group
				break
			}
		}
		if group == nil {
			group = &BtcRedeemTxGroup{
				Chain:           tx.Chain,
				GroupUid:        tx.CustodianGroupUid,
				SessionSequence: tx.SessionSequence,
			}
			redeemTxGroups = append(redeemTxGroups, group)
		}
		group.TxHashs = append(group.TxHashs, tx)
	}
	return redeemTxGroups
}
