package scalar

import (
	"context"
	"encoding/hex"
	"strings"
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
	RedeemTxSession map[string]uint64
	GroupRedeemTxs  map[string][]*models.RedeemTx
}

func (s *CustodianGroupRedeemTx) AddRedeemTx(redeemTx *models.RedeemTx) {
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
		groupBytes32, err := utils.DecodeGroupUid(groupUid)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to decode group uid: %s", groupUid)
			continue
		}
		redeemSession, err := c.GetRedeemSession(groupBytes32)
		if err != nil || redeemSession == nil || redeemSession.Session == nil {
			log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to get redeem session: %s", err)
			groupRedeemTx, ok := c.redeemTxCache[redeemTxEvents.Chain]
			if !ok {
				groupRedeemTx = &CustodianGroupRedeemTx{
					GroupRedeemTxs:  make(map[string][]*models.RedeemTx),
					RedeemTxSession: make(map[string]uint64),
				}
			}
			for _, tx := range redeemTxs {
				groupRedeemTx.AddRedeemTx(tx)
			}
			c.redeemTxCache[redeemTxEvents.Chain] = groupRedeemTx
			continue
		}
		txHashes := []string{}
		for _, tx := range redeemTxs {
			if tx.SessionSequence == redeemSession.Session.Sequence {
				txHashes = append(txHashes, tx.TxHash)
			}
		}
		confirmRedeemTxRequest := events.ConfirmRedeemTxRequest{
			Chain:   redeemTxEvents.Chain,
			TxHashs: txHashes,
		}
		err = c.broadcaster.ConfirmRedeemTxRequest(confirmRedeemTxRequest)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [handleElectrsEventRedeemTx] failed to confirm redeem tx: %s", err)
			continue
		}

	}
	return nil
}
