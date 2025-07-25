package relayer

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	pkgTypes "github.com/scalarorg/relayers/pkg/types"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

type SwitchedPhaseEvent struct {
	Events     []*chains.SwitchedPhase
	MinSession *pkgTypes.Session
	MaxSession *pkgTypes.Session
}

func getMaxSession(lastSwitchedPhases []*chains.SwitchedPhase) *pkgTypes.Session {
	var maxSession *pkgTypes.Session
	for _, switchedPhase := range lastSwitchedPhases {
		if maxSession == nil {
			maxSession = &pkgTypes.Session{
				Sequence: switchedPhase.SessionSequence,
				Phase:    switchedPhase.To,
			}
			continue
		}
		if maxSession.Sequence < switchedPhase.SessionSequence {
			maxSession.Sequence = switchedPhase.SessionSequence
			maxSession.Phase = switchedPhase.To
		} else if maxSession.Sequence == switchedPhase.SessionSequence && maxSession.Phase < switchedPhase.To {
			maxSession.Phase = switchedPhase.To
		}
	}
	return maxSession
}

func getMinSession(lastSwitchedPhases []*chains.SwitchedPhase) *pkgTypes.Session {
	var minSession *pkgTypes.Session
	for _, switchedPhase := range lastSwitchedPhases {
		if minSession == nil {
			minSession = &pkgTypes.Session{
				Sequence: switchedPhase.SessionSequence,
				Phase:    switchedPhase.To,
			}
			continue
		}
		if minSession.Sequence > switchedPhase.SessionSequence {
			minSession.Sequence = switchedPhase.SessionSequence
			minSession.Phase = switchedPhase.To
		} else if minSession.Sequence == switchedPhase.SessionSequence && minSession.Phase > switchedPhase.To {
			minSession.Phase = switchedPhase.To
		}
	}
	return minSession
}

func (s *Service) RecoverEvmSessions(groupUid chainExported.Hash) error {
	err := s.ScalarClient.WaitForUtxoSnapshot(groupUid)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot wait for utxo snapshot")
		return err
	}
	lastSwitchedPhases, err := s.DbAdapter.GetLastSwitchedPhases(hex.EncodeToString(groupUid[:]))
	if err != nil {
		return fmt.Errorf("failed to get last switched phases: %w", err)
	}

	maxSession := getMaxSession(lastSwitchedPhases)
	minSession := getMinSession(lastSwitchedPhases)
	switchedPhaseEvents := &SwitchedPhaseEvent{
		Events:     lastSwitchedPhases,
		MinSession: minSession,
		MaxSession: maxSession,
	}
	if maxSession == nil || minSession == nil {
		return fmt.Errorf("no last switched phases found")
	}
	log.Info().Hex("groupUid", groupUid[:]).
		Any("maxSession", maxSession).
		Any("minSession", minSession).
		Msg("[Relayer] [RecoverEvmSessions] recovered redeem session for each group")

	if maxSession.Phase == uint8(covExported.Executing) {
		err := s.processRecoverExecutingPhase(groupUid, switchedPhaseEvents)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot process recover executing phase for group %s", groupUid)
		}
	} else if maxSession.Phase == uint8(covExported.Preparing) {
		err := s.processRecoverPreparingPhase(groupUid, switchedPhaseEvents)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot process recover preparing phase for group %s", groupUid)
		}
	}

	log.Info().Msgf("[Relayer] [RecoverEvmSessions] finished RecoverEvmSessions")
	return nil
}

func (s *Service) processRecoverExecutingPhase(groupUid chainExported.Hash, switchedPhaseEvents *SwitchedPhaseEvent) error {
	log.Info().Hex("groupUid", groupUid[:]).
		Msg("[Relayer] [RecoverEvmSessions] processRecoverExecutingPhase")

	//0. Check if the redeem session is broadcasted to bitcoin network
	lastRedeemTxs, err := s.DbAdapter.GetLastRedeemTxs(hex.EncodeToString(groupUid[:]), switchedPhaseEvents.MinSession.Sequence)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutingPhase] cannot get last redeem transactions")
		return err
	}
	if len(lastRedeemTxs) == 0 {
		log.Info().Msgf("[Relayer] [processRecoverExecutingPhase] no last redeem transactions found")
		return nil
	}

	isBroadcasted, err := s.isRedeemSessionBroadcasted(lastRedeemTxs[0])
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutingPhase] cannot check if the redeem session is broadcasted to bitcoin network")
		return err
	}

	if !isBroadcasted {
		log.Info().Msgf("[Relayer] [processRecoverExecutingPhase] redeem session is not broadcasted to bitcoin network")

		//1. Replay all switch to preparing phase then replay all redeem token events for signing
		evmCounter := s.replaySwitchPhase(switchedPhaseEvents.Events, uint8(covExported.Preparing))
		log.Info().Int32("evmCounter", evmCounter).
			Msg("[Relayer] [processRecoverExecutingPhase] replay preparing switch phase events")
		if evmCounter != int32(len(s.EvmClients)) {
			panic(fmt.Sprintf("[Relayer] [processRecoverExecutingPhase] cannot recover all evm switch phase events, evm counter is %d", evmCounter))
		}

		//2. wait for group's session switch to preparing then replay all redeem token events
		err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Preparing)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to preparing phase", groupUid)
			return err
		}

		// Get redeem token events from database
		redeemTokenEvents, err := s.DbAdapter.GetRedeemTokenEventsByGroupAndSequence(hex.EncodeToString(groupUid[:]), switchedPhaseEvents.MinSession.Sequence)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot get redeem token events from database")
			return err
		}

		if len(redeemTokenEvents) > 0 {
			mapTxHashes, err := s.replayRedeemTransactions(groupUid, redeemTokenEvents)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot replay redeem transactions")
				return err
			}
			log.Info().Any("mapTxHashes", mapTxHashes).Msg("[Relayer] [processRecoverExecutionPhase] finished replay redeem transactions")
			s.waitingForPendingCommands(mapTxHashes)
			log.Info().Msgf("[Relayer] [processRecoverExecutionPhase] all pending commands are ready")
		}
	}

	//5. Replay all switch to executing phase events from database
	evmCounter := s.replaySwitchPhase(switchedPhaseEvents.Events, uint8(covExported.Executing))
	log.Info().Int32("evmCounter", evmCounter).
		Msg("[Relayer] [processRecoverExecutingPhase] replay executing switch phase events")

	if evmCounter == int32(len(s.EvmClients)) {
		log.Info().Int32("evmCounter", evmCounter).Msg("[Relayer] [processRecoverExecutionPhase] all evm chains a in executing phase")
		//All evm chains are in executing phase
		err = s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Executing)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to executing phase", groupUid)
			return err
		}
		err = s.replayBtcRedeemTxs(groupUid)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot replay btc redeem transactions")
			return err
		}
	} else {
		log.Warn().Int32("evmCounter", evmCounter).Msg("[Relayer] [processRecoverExecutionPhase] not all evm chains are in executing phase")
	}
	return nil
}

// MaxSession is in preparing phase
// Some slower evm chains may in executing phase of the previous session
func (s *Service) processRecoverPreparingPhase(groupUid chainExported.Hash, switchedPhaseEvents *SwitchedPhaseEvent) error {
	log.Info().Hex("groupUid", groupUid[:]).
		Msg("[Relayer] [RecoverEvmSessions] processRecoverPreparingPhase")
	var expectedPhase covExported.Phase
	//1. For each evm chain, replay last switch event from database. It can be Preparing or executing from previous session
	if switchedPhaseEvents.MaxSession.Phase == switchedPhaseEvents.MinSession.Phase {
		// All evm chains are in the same phase preparing
		evmCounter := s.replaySwitchPhase(switchedPhaseEvents.Events, uint8(covExported.Preparing))
		if evmCounter != int32(len(s.EvmClients)) {
			panic(fmt.Sprintf("[Relayer] [processRecoverPreparingPhase] cannot recover all evm switch phase events, evm counter is %d", evmCounter))
		}
		expectedPhase = covExported.Preparing
	} else {
		evmCounter := s.replaySwitchPhase(switchedPhaseEvents.Events, uint8(covExported.Executing))
		if evmCounter != int32(len(s.EvmClients)) {
			panic(fmt.Sprintf("[Relayer] [processRecoverPreparingPhase] cannot recover all evm switch phase events, evm counter is %d", evmCounter))
		}
		expectedPhase = covExported.Executing
	}
	//2. Waiting for group session switch to expected phase
	err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, expectedPhase)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot wait for group %s to switch to executing phase", groupUid)
		return err
	}

	if expectedPhase == covExported.Preparing {
		//3. Replay all redeem transactions from database
		redeemTokenEvents, err := s.DbAdapter.GetRedeemTokenEventsByGroupAndSequence(hex.EncodeToString(groupUid[:]), switchedPhaseEvents.MinSession.Sequence)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot get redeem token events from database")
			return err
		}

		mapTxHashes, err := s.replayRedeemTransactions(groupUid, redeemTokenEvents)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot replay redeem transactions")
			return err
		}
		log.Info().Any("mapTxHashes", mapTxHashes).Msg("[Relayer] [processRecoverPreparingPhase] finished replay redeem transactions")
	} else if expectedPhase == covExported.Executing {
		err := s.replayBtcRedeemTxs(groupUid)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot replay btc redeem transactions")
			return err
		}
	}
	return nil
}

// replaySwitchPhaseEventsFromDB replays switch phase events from database instead of EVM network
func (s *Service) replaySwitchPhase(switchedPhases []*chains.SwitchedPhase, expectedPhase uint8) int32 {
	var evmCounter int32

	// Group switched phases by chain
	chainSwitchPhase := make(map[string]*chains.SwitchedPhase)
	for _, phase := range switchedPhases {
		if phase.To != expectedPhase {
			continue
		}
		chainSwitchPhase[phase.SourceChain] = phase
	}

	for _, evmClient := range s.EvmClients {
		chainId := evmClient.EvmConfig.GetId()
		switchPhaseEvent, ok := chainSwitchPhase[chainId]
		if !ok {
			log.Warn().Msgf("[Relayer] [replaySwitchPhaseEventsFromDB] cannot find switch phase event for evm client %s", chainId)
			continue
		}

		// Convert database model to contract event and handle it
		//contractEvent := s.convertSwitchedPhaseToContractEvent(switchPhaseEvent)
		err := evmClient.HandleSwitchPhase(switchPhaseEvent)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [replaySwitchPhaseEventsFromDB] cannot handle switch phase event for evm client %s", chainId)
		} else {
			evmCounter++
		}
	}

	return evmCounter
}

// replayRedeemTransactionsFromDB replays redeem transactions from database instead of EVM network
func (s *Service) replayRedeemTransactions(groupUid chainExported.Hash, redeemTokenEvents []*chains.EvmRedeemTx) (map[string][]string, error) {
	mapTxHashes := make(map[string][]string)
	log.Info().Any("redeemTokenEvents", redeemTokenEvents).Msg("[Relayer] [replayRedeemTransactions] redeem token events")
	for _, redeemTx := range redeemTokenEvents {
		// Convert database model to contract event and handle it
		//contractEvent := s.convertRedeemTxToContractEvent(redeemTx)
		err := s.EvmClients[0].HandleRedeemToken(redeemTx) // Assuming we have at least one EVM client
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [replayRedeemTransactionsFromDB] cannot handle redeem token event")
			continue
		}
		if mapTxHashes[redeemTx.DestinationChain] == nil {
			mapTxHashes[redeemTx.DestinationChain] = []string{}
		}
		mapTxHashes[redeemTx.DestinationChain] = append(mapTxHashes[redeemTx.DestinationChain], redeemTx.TxHash)
	}

	return mapTxHashes, nil
}

// convertSwitchedPhaseToContractEvent converts database model to contract event
// func (s *Service) convertSwitchedPhaseToContractEvent(switchedPhase *chains.SwitchedPhase) *contracts.IScalarGatewaySwitchPhase {
// 	// Convert hex string to bytes32
// 	var custodianGroupId [32]byte
// 	copy(custodianGroupId[:], common.FromHex(switchedPhase.CustodianGroupUid))

// 	// Create a mock log for the event
// 	mockLog := ethTypes.Log{
// 		TxHash: common.HexToHash(switchedPhase.TxHash),
// 		Index:  0, // Use default value since LogIndex doesn't exist in SwitchedPhase
// 	}

// 	return &contracts.IScalarGatewaySwitchPhase{
// 		CustodianGroupId: custodianGroupId,
// 		Sequence:         switchedPhase.SessionSequence,
// 		From:             switchedPhase.From,
// 		To:               switchedPhase.To,
// 		Raw:              mockLog,
// 	}
// }

// convertRedeemTxToContractEvent converts database model to contract event
func (s *Service) convertRedeemTxToContractEvent(redeemTx *chains.EvmRedeemTx) *contracts.IScalarGatewayRedeemToken {
	// Convert hex string to bytes32
	var custodianGroupUid [32]byte
	copy(custodianGroupUid[:], common.FromHex(redeemTx.CustodianGroupUid))

	var payloadHash [32]byte
	copy(payloadHash[:], common.FromHex(redeemTx.PayloadHash))

	// Create a mock log for the event
	mockLog := types.Log{
		TxHash: common.HexToHash(redeemTx.TxHash),
		Index:  uint(redeemTx.LogIndex),
	}

	return &contracts.IScalarGatewayRedeemToken{
		Sender:                     common.HexToAddress(redeemTx.SourceAddress),
		Sequence:                   redeemTx.SessionSequence,
		CustodianGroupUID:          custodianGroupUid,
		DestinationChain:           redeemTx.DestinationChain,
		DestinationContractAddress: redeemTx.TokenContractAddress,
		PayloadHash:                payloadHash,
		Payload:                    redeemTx.Payload,
		Symbol:                     redeemTx.Symbol,
		Amount:                     new(big.Int).SetUint64(redeemTx.Amount),
		Raw:                        mockLog,
	}
}

/*
 * check if the current redeem session is broadcasted to bitcoin network by checking if the first input utxo is present in the bitcoin network
 */
func (s *Service) isRedeemSessionBroadcasted(redeemTx *chains.EvmRedeemTx) (bool, error) {
	log.Info().Msgf("[Relayer] [isRedeemSessionBroadcasted] checking if the current redeem session is broadcasted to bitcoin network")
	if s.BtcClient == nil {
		return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] btc client is undefined")
	}
	params := covTypes.RedeemTokenPayloadWithType{}
	err := params.AbiUnpack(redeemTx.Payload)
	if err != nil {
		return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] cannot unpack redeem token payload: %s", err)
	}
	log.Info().Any("First EvmRedeemTx", params).Msg("[Service] check if redeem session is broadcasted")
	if len(params.Utxos) == 0 {
		//Todo: for safety, we do not rebroadcast redeem session if no utxos found in redeem token payload
		return true, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] no utxos found in redeem token payload")
	}
	for _, btcClient := range s.BtcClient {
		if btcClient.Config().ID == redeemTx.DestinationChain {
			txId := strings.TrimPrefix(params.Utxos[0].TxID.Hex(), "0x")
			outspend, err := btcClient.GetOutSpend(txId, params.Utxos[0].Vout)
			// outResult, err := btcClient.GetTxOut(params.Utxos[0].TxID.Hex(), params.Utxos[0].Vout)
			if err != nil {
				return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] cannot get outspend for redeem token event: %s", err)
			}
			log.Info().Any("OutSpend", outspend).Msg("Successfully get outspend from mempool")
			// if outResult == nil {
			// 	return true, nil
			// }
			return outspend.Spent, nil
		}
	}
	return false, nil
}
