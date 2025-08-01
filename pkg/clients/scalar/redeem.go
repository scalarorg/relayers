package scalar

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	models "github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/events"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
	chains "github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covenant "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Struct for cache redeem transaction from btc network
type CustodianGroupRedeemTx struct {
	lock         sync.Mutex
	mapSequences map[string]uint64
	mapRedeemTxs map[string][]*models.BtcRedeemTx
}

func (s *CustodianGroupRedeemTx) AddRedeemTxs(redeemTxEvents *events.BtcRedeemTxEvents) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.mapRedeemTxs == nil {
		s.mapRedeemTxs = make(map[string][]*models.BtcRedeemTx)
	}
	if s.mapSequences == nil {
		s.mapSequences = make(map[string]uint64)
	}
	groupUid := redeemTxEvents.GroupUid
	redeemTxs := s.mapRedeemTxs[groupUid]
	sessionSequence := s.mapSequences[groupUid]
	if redeemTxEvents.Sequence > sessionSequence {
		s.mapRedeemTxs[groupUid] = redeemTxEvents.RedeemTxs
		s.mapSequences[groupUid] = redeemTxEvents.Sequence
	} else if redeemTxEvents.Sequence == sessionSequence {
		s.mapRedeemTxs[groupUid] = append(redeemTxs, redeemTxEvents.RedeemTxs...)
	} else {
		log.Warn().Msgf("[ScalarClient] [AddRedeemTx] session sequence %d is lower than stored one: %d", redeemTxEvents.Sequence, sessionSequence)
	}
}

func (s *CustodianGroupRedeemTx) PickRedeemTxsByGroupUid(groupUid string, sequence uint64) ([]*models.BtcRedeemTx, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	storedSequence, ok := s.mapSequences[groupUid]
	if !ok {
		log.Debug().Str("GroupUid", groupUid).Msg("[ScalarClient] [PickRedeemTxsByGroupUid] sequence not found")
		return nil, nil
	}
	if sequence != storedSequence {
		return nil, fmt.Errorf("[ScalarClient] [PickRedeemTxsByGroupUid] request and stored sequence do not match %d != %d", sequence, storedSequence)
	}
	redeemTxs, ok := s.mapRedeemTxs[groupUid]
	if !ok {
		return nil, fmt.Errorf("[ScalarClient] [PickRedeemTxsByGroupUid] no redeem txs found for group uid: %s", groupUid)
	}
	delete(s.mapRedeemTxs, groupUid)
	delete(s.mapSequences, groupUid)
	return redeemTxs, nil
}

func (c *Client) WaitForSwitchingToPhase(groupHash chainExported.Hash, expectedPhase covExported.Phase) error {
	for {
		redeemSession, err := c.GetRedeemSession(groupHash)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [RecoverAllEvents] failed to get current redeem session from scalarnet")
			continue
		} else {
			if redeemSession.Session.CurrentPhase == expectedPhase {
				log.Info().Str("GroupUid", groupHash.String()).
					Any("Session", redeemSession.Session).
					Msgf("[EvmClient] [WaitForSwitchingToPhase] Current group has switched to expected phase %+v", expectedPhase)
				return nil
			} else {
				log.Info().Str("GroupUid", groupHash.String()).
					Any("Session", redeemSession.Session).
					Msgf("[EvmClient] [WaitForSwitchingToPhase] waiting for group to switch to expected phase %v", expectedPhase)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *Client) WaitForUtxoSnapshot(groupHex chainExported.Hash) error {
	covenantClient := c.GetCovenantQueryClient()
	log.Info().Msgf("[EvmClient] [WaitForUtxoSnapshot] waiting for utxo snapshot for group %s", groupHex.String())
	request := &covenant.UTXOSnapshotRequest{
		UID: groupHex.Bytes(),
	}
	for {
		utxoSnapshot, err := covenantClient.UTXOSnapshot(context.Background(), request)
		//TODO: check with block height
		if utxoSnapshot != nil && utxoSnapshot.UtxoSnapshot != nil && len(utxoSnapshot.UtxoSnapshot.Utxos) > 0 {
			log.Info().Msgf("[EvmClient] [WaitForUtxoSnapshot] found utxo snapshot")
			return nil
		} else if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [WaitForUtxoSnapshot] failed to get utxo snapshot. Waiting for utxo snapshot again")
			time.Sleep(2 * time.Second)
		}
	}
}
func (c *Client) ReserveUtxo(sourceChain string, redeemTokenEvent *contracts.IScalarGatewayRedeemToken) error {
	params := covenant.RedeemTokenPayloadWithType{}
	err := params.AbiUnpack(redeemTokenEvent.Raw.Data)
	if err != nil {
		log.Error().Err(err).Msgf("[EvmClient] [ReserveUtxo] failed to unpack redeem token event")
		return err
	}

	request := &covenant.ReserveRedeemUtxoRequest{
		Sender:        c.network.GetAddress(),
		Address:       redeemTokenEvent.Sender.String(),
		SourceChain:   exported.ChainName(sourceChain),
		DestChain:     exported.ChainName(redeemTokenEvent.DestinationChain),
		Symbol:        redeemTokenEvent.Symbol,
		Amount:        redeemTokenEvent.Amount.Uint64(),
		LockingScript: params.LockingScript, //Bit coin address
	}
	return c.broadcaster.QueueTxMsg(request)
}
func (c *Client) WaitForPendingCommands(chainId string, sourceTxs []string) error {
	log.Info().Str("Chain", chainId).Any("SourceTxs", sourceTxs).Msg("[EvmClient] [waitForPendingCommands] waiting for pending commands")
	chainClient := c.GetChainQueryServiceClient()
	request := &chains.PendingCommandsRequest{
		Chain: chainId,
	}
	waitingTxs := make(map[string]bool)
	for _, txHash := range sourceTxs {
		waitingTxs[txHash] = true
	}
	for len(waitingTxs) > 0 {
		pendingCommands, err := chainClient.PendingCommands(context.Background(), request)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [waitForPendingCommands] failed to get pending commands")
			return err
		}
		for _, command := range pendingCommands.Commands {
			txHash := strings.TrimPrefix(command.Params["sourceTxHash"], "0x")
			log.Info().Str("Chain", chainId).Str("TxHash", txHash).Msg("[EvmClient] [waitForPendingCommands] found pending command")
			if _, ok := waitingTxs[txHash]; ok {
				log.Info().Str("Chain", chainId).Str("TxHash", txHash).Msg("[EvmClient] [waitForPendingCommands] found pending command")
				delete(waitingTxs, txHash)
			}
		}
		if len(waitingTxs) > 0 {
			log.Info().Str("Chain", chainId).
				Msgf("[EvmClient] [waitForPendingCommands] waiting for pending commands %+v", waitingTxs)
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}

func (c *Client) broadcastRedeemTxsConfirm(redeemTxEvents *events.BtcRedeemTxEvents) error {
	err := c.BroadcastRedeemTxsConfirmRequest(redeemTxEvents.Chain, redeemTxEvents.GroupUid, redeemTxEvents.RedeemTxs)
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [broadcastRedeemTxsConfirm] failed to broadcast redeem txs confirm request")
		return err
	} else {
		log.Info().Msg("[ScalarClient] broadcastRedeemTxsConfirm broadcasted RedeemTxsConfirmRequest")
		return nil
	}
}

func (c *Client) AddRedeemTxsToCache(chainId string, redeemTxEvents *events.BtcRedeemTxEvents) {
	if c.redeemTxCache == nil {
		c.redeemTxCache = make(map[string]*CustodianGroupRedeemTx)
	}
	groupRedeemTx, ok := c.redeemTxCache[chainId]
	if !ok {
		groupRedeemTx = &CustodianGroupRedeemTx{
			mapRedeemTxs: make(map[string][]*models.BtcRedeemTx),
			mapSequences: make(map[string]uint64),
		}
	}
	groupRedeemTx.AddRedeemTxs(redeemTxEvents)
	c.redeemTxCache[chainId] = groupRedeemTx
}
func (c *Client) BroadcastRedeemTxsConfirmRequest(chainId string, groupUid string, redeemTxs []*models.BtcRedeemTx) error {
	if len(redeemTxs) == 0 {
		log.Info().Msgf("[ScalarClient] BroadcastRedeemTxsConfirmRequest, redeemTxs is empty")
		return nil
	}
	txHashes := []string{}
	for _, tx := range redeemTxs {
		txHashes = append(txHashes, tx.TxHash)
	}
	confirmRedeemTxRequest := events.ConfirmRedeemTxRequest{
		Chain:    chainId,
		GroupUid: groupUid,
		TxHashs:  txHashes,
	}
	err := c.broadcaster.ConfirmRedeemTxRequest(confirmRedeemTxRequest)
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [BroadcastRedeemTxsConfirmRequest] failed to confirm redeem tx: %s", err)
		return err
	}
	return nil
}
func (c *Client) PickCacheRedeemTx(groupUid chainExported.Hash, sequence uint64) map[string][]*models.BtcRedeemTx {
	result := make(map[string][]*models.BtcRedeemTx)
	for chainId, redeemTxCache := range c.redeemTxCache {
		redeemTxs, err := redeemTxCache.PickRedeemTxsByGroupUid(groupUid.String(), sequence)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [PickCacheRedeemTx] failed to get redeem txs")
			continue
		}
		if len(redeemTxs) > 0 {
			result[chainId] = redeemTxs
		}
	}
	return result
}
