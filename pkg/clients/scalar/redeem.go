package scalar

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/utils"
	chains "github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covenant "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

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
func (c *Client) WaitForPendingCommands(chainId string, sourceTxs map[string]bool) error {
	chainClient := c.GetChainQueryServiceClient()
	request := &chains.PendingCommandsRequest{
		Chain: chainId,
	}
	waitingTxs := map[string]bool{}
	for tx := range sourceTxs {
		waitingTxs[tx] = true
	}
	for len(waitingTxs) > 0 {
		pendingCommands, err := chainClient.PendingCommands(context.Background(), request)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [waitForPendingCommands] failed to get pending commands")
			return err
		}
		for _, command := range pendingCommands.Commands {
			txHash := command.Params["sourceTxHash"]
			if _, ok := waitingTxs[txHash]; ok {
				log.Info().Str("Chain", chainId).Str("TxHash", txHash).Msg("[EvmClient] [waitForPendingCommands] found pending command")
				delete(waitingTxs, txHash)
			}
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}

func (c *Client) handleElectrsEventRedeemTx(confirmRequest events.ConfirmTxsRequest) error {
	//TODO: implement recovering
	return c.requestConfirmEvmTxs(confirmRequest)
}
