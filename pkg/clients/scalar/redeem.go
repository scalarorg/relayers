package scalar

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	models "github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/utils"
	chains "github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covenant "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Struct for cache redeem transaction from btc network
type CustodianGroupRedeemTx struct {
	lock            sync.Mutex
	RedeemTxSession map[string]uint64
	GroupRedeemTxs  map[string][]*models.RedeemTx
}

func (s *CustodianGroupRedeemTx) AddRedeemTx(redeemTx *models.RedeemTx) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.GroupRedeemTxs == nil {
		s.GroupRedeemTxs = make(map[string][]*models.RedeemTx)
	}
	if s.RedeemTxSession == nil {
		s.RedeemTxSession = make(map[string]uint64)
	}
	groupUid := redeemTx.CustodianGroupUid
	redeemTxs := s.GroupRedeemTxs[groupUid]
	sessionSequence := s.RedeemTxSession[groupUid]
	if redeemTxs == nil {
		s.GroupRedeemTxs[groupUid] = []*models.RedeemTx{redeemTx}
		s.RedeemTxSession[groupUid] = redeemTx.SessionSequence
	} else {
		//Store only the latest redeem tx (with highest session sequence)
		if redeemTx.SessionSequence > sessionSequence {
			s.GroupRedeemTxs[groupUid] = []*models.RedeemTx{redeemTx}
			s.RedeemTxSession[groupUid] = redeemTx.SessionSequence
		} else if redeemTx.SessionSequence == sessionSequence {
			s.GroupRedeemTxs[groupUid] = append(redeemTxs, redeemTx)
		} else {
			log.Warn().Msgf("[ScalarClient] [AddRedeemTx] session sequence %d is lower than stored one: %d", redeemTx.SessionSequence, sessionSequence)
		}
	}
}

func (s *CustodianGroupRedeemTx) PickRedeemTxsByGroupUid(groupUid string) ([]*models.RedeemTx, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	redeemTxs, ok := s.GroupRedeemTxs[groupUid]
	if !ok {
		return nil, fmt.Errorf("[ScalarClient] [PickRedeemTxsByGroupUid] no redeem txs found for group uid: %s", groupUid)
	}
	delete(s.GroupRedeemTxs, groupUid)
	return redeemTxs, nil
}

func (c *Client) WaitForSwitchingToPhase(groupHex string, expectedPhase covExported.Phase) error {
	for {
		groupBytes32, err := utils.DecodeGroupUid(groupHex)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [RecoverAllEvents] failed to decode group uid: %s", groupHex)
			return err
		}
		redeemSession, err := c.GetRedeemSession(groupBytes32)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [RecoverAllEvents] failed to get current redeem session from scalarnet")
			continue
		} else {
			if redeemSession.Session.CurrentPhase == expectedPhase {
				log.Info().Str("GroupUid", groupHex).
					Any("Session", redeemSession.Session).
					Msgf("[EvmClient] [WaitForSwitchingToPhase] Current group has switched to expected phase %+v", expectedPhase)
				return nil
			} else {
				log.Info().Str("GroupUid", groupHex).
					Any("Session", redeemSession.Session).
					Msgf("[EvmClient] [WaitForSwitchingToPhase] waiting for group to switch to expected phase %v", expectedPhase)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *Client) WaitForUtxoSnapshot(groupHex string) error {
	covenantClient := c.GetCovenantQueryClient()
	groupBytes, err := hex.DecodeString(strings.TrimPrefix(groupHex, "0x"))
	if err != nil {
		log.Error().Err(err).Msgf("[EvmClient] [WaitForUtxoSnapshot] failed to decode group uid: %s", groupHex)
		return err
	}
	request := &covenant.UTXOSnapshotRequest{
		UID: groupBytes,
	}
	for {
		utxoSnapshot, err := covenantClient.UTXOSnapshot(context.Background(), request)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [WaitForUtxoSnapshot] failed to get utxo snapshot")
			return err
		}
		//TODO: check with block height
		if utxoSnapshot != nil && utxoSnapshot.UtxoSnapshot != nil && len(utxoSnapshot.UtxoSnapshot.Utxos) > 0 {
			log.Info().Msgf("[EvmClient] [WaitForUtxoSnapshot] found utxo snapshot")
			return nil
		}
		log.Info().Msgf("[EvmClient] [WaitForUtxoSnapshot] waiting for utxo snapshot")
		time.Sleep(2 * time.Second)
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
			txHash := command.Params["sourceTxHash"]
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

func (c *Client) handleElectrsEventRedeemTx(redeemTxEvents events.RedeemTxEvents) error {
	//TODO: implement recovering
	mapTxHashes := make(map[string][]*models.RedeemTx)
	for _, redeemTx := range redeemTxEvents.RedeemTxs {
		txHashes := mapTxHashes[redeemTx.CustodianGroupUid]
		mapTxHashes[redeemTx.CustodianGroupUid] = append(txHashes, redeemTx)
	}
	for groupUid, redeemTxs := range mapTxHashes {
		broadcastingRedeemTxs := []*models.RedeemTx{}
		groupBytes32, err := utils.DecodeGroupUid(groupUid)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to decode group uid: %s", groupUid)
			continue
		}
		redeemSession, err := c.GetRedeemSession(groupBytes32)
		if err != nil || redeemSession == nil || redeemSession.Session == nil {
			if err != nil {
				log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to get redeem session: %s", err)
			}
			continue
		} else if redeemSession.Session.CurrentPhase != covExported.Executing {
			log.Info().Str("groupUid", groupUid).
				Any("Session", redeemSession.Session).
				Msgf("[EvmClient] [handleElectrsEventRedeemTx] redeem session is not in executing phase")
			c.AddRedeemTxsToCache(redeemTxEvents.Chain, redeemSession.Session, redeemTxs)
		} else {
			broadcastingRedeemTxs = append(broadcastingRedeemTxs, redeemTxs...)
		}
		err = c.BroadcastRedeemTxsConfirmRequest(redeemTxEvents.Chain, redeemSession.Session, broadcastingRedeemTxs)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to broadcast redeem txs confirm request")
			continue
		}
	}
	return nil
}

func (c *Client) AddRedeemTxsToCache(chainId string, redeemSession *covenant.RedeemSession, redeemTxs []*models.RedeemTx) {
	groupRedeemTx, ok := c.redeemTxCache[chainId]
	if !ok {
		groupRedeemTx = &CustodianGroupRedeemTx{
			GroupRedeemTxs:  make(map[string][]*models.RedeemTx),
			RedeemTxSession: make(map[string]uint64),
		}
	}
	for _, tx := range redeemTxs {
		if tx.SessionSequence >= redeemSession.Sequence {
			//Add to cache txes with sequence number greater than or equal to current redeem session sequence
			groupRedeemTx.AddRedeemTx(tx)
		}
	}
	c.redeemTxCache[chainId] = groupRedeemTx
}
func (c *Client) BroadcastRedeemTxsConfirmRequest(chainId string, redeemSession *covenant.RedeemSession, redeemTxs []*models.RedeemTx) error {
	if len(redeemTxs) == 0 {
		return nil
	}
	txHashes := []string{}
	for _, tx := range redeemTxs {
		if tx.SessionSequence == redeemSession.Sequence {
			txHashes = append(txHashes, tx.TxHash)
		}
	}
	confirmRedeemTxRequest := events.ConfirmRedeemTxRequest{
		Chain:   chainId,
		TxHashs: txHashes,
	}
	err := c.broadcaster.ConfirmRedeemTxRequest(confirmRedeemTxRequest)
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [BroadcastRedeemTxsConfirmRequest] failed to confirm redeem tx: %s", err)
		return err
	}
	return nil
}
func (c *Client) PickCacheRedeemTx(groupUid string) map[string][]*models.RedeemTx {
	result := make(map[string][]*models.RedeemTx)
	for chainId, redeemTxCache := range c.redeemTxCache {
		redeemTxs, err := redeemTxCache.PickRedeemTxsByGroupUid(groupUid)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [PickCacheRedeemTx] failed to get redeem txs")
			continue
		}
		result[chainId] = redeemTxs
	}
	return result
}
