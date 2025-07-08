package evm

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/pkg/events"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

func (ec *EvmClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Debug().Str("eventType", event.EventType).
		Str("messageID", event.MessageID).
		Str("destinationChain", event.DestinationChain).
		Msg("[EvmClient] [handleEventBusMessage]")
	switch event.EventType {
	case events.EVENT_SCALAR_BATCHCOMMAND_SIGNED:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarBatchCommandSigned(event.DestinationChain, event.Data.(*chainstypes.BatchedCommandsResponse))
		if err != nil {
			log.Error().
				Err(err).
				Any("eventData", event.Data).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_TOKEN_SENT:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarTokenSent(event.Data.(string))
		if err != nil {
			log.Error().
				Err(err).
				Any("eventData", event.Data).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_DEST_CALL_APPROVED:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarContractCallApproved(event.MessageID, event.Data.(string))
		if err != nil {
			log.Error().
				Err(err).
				Str("messageId", event.MessageID).
				Str("eventData", event.Data.(string)).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_SWITCH_PHASE_STARTED:
		//Emitted from scalar.handleStartedSwitchPhaseEvents with event.Data as startedSwitchPhase
		err := ec.handleStartedSwitchPhase(event.Data.(*covTypes.SwitchPhaseStarted))
		if err != nil {
			log.Error().
				Err(err).
				Any("eventData", event.Data).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	}
	return nil
}

func (ec *EvmClient) handleScalarTokenSent(executeData string) error {
	log.Debug().
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarTokenSent]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.EvmConfig.GetId()).
			Msg("[EvmClient] [handleScalarTokenSent] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarTokenSent] auth is nil")
	}
	signedTx, err := ec.gatewayExecute(decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.EvmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarTokenSent]")
		return err
	} else {
		if ec.auth.NoSend {
			log.Debug().
				Str("chainId", ec.EvmConfig.GetId()).
				Str("contractAddress", ec.EvmConfig.Gateway).
				Str("signer", ec.auth.From.String()).
				Uint64("nonce", signedTx.Nonce()).
				Str("txHash", signedTx.Hash().String()).
				Msg("[EvmClient] [handleScalarTokenSent] successfully sent tx to the network")
		} else {
			log.Debug().
				Str("chainId", ec.EvmConfig.GetId()).
				Str("contractAddress", ec.EvmConfig.Gateway).
				Str("signer", signedTx.To().Hex()).
				Uint64("nonce", signedTx.Nonce()).
				Str("txHash", signedTx.Hash().String()).
				Msg("[EvmClient] [handleScalarTokenSent] successfully signed but not sent tx to the network due to NoSend flag")
		}
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarTokenSent]")
	//2. Add the transaction waiting to be mined
	// ec.pendingTxs.AddTx(signedTx.Hash().String(), time.Now())
	//3. Update status of the event
	// err = ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	// if err != nil {
	// 	log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarContractCallApproved]")
	// 	return err
	// }
	return nil
}

func (ec *EvmClient) handleScalarBatchCommandSigned(chainId string, batchedCmdRes *chainstypes.BatchedCommandsResponse) error {
	log.Debug().
		Str("ChainId", chainId).
		Str("BatchedCommandID", batchedCmdRes.ID).
		Any("CommandIDs", batchedCmdRes.CommandIDs).
		Msg("[EvmClient] [handleScalarBatchCommandSigned]")
	executeDataBytes, err := hex.DecodeString(batchedCmdRes.ExecuteData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	input, err := AbiUnpack(executeDataBytes[4:], "bytes")
	if err != nil {
		log.Debug().Msgf("[EvmClient] [DecodeExecuteData] unpack executeData error: %v", err)
	}
	//decodedExecuteData, err := DecodeExecuteData(batchedCmdRes.ExecuteData)
	decodedExecuteData, err := DecodeInput(input[0].([]byte))
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	broadcastedCommands := ec.countBroadcastedCommands(decodedExecuteData)
	if broadcastedCommands >= int64(len(decodedExecuteData.CommandIds)) {
		log.Debug().Msg("[EvmClient] [handleScalarBatchCommandSigned] tx is already broadcasted")
		return nil
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//TODO: add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.EvmConfig.GetId()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
	}
	//Estimate gas
	gas, err := ec.Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: ec.auth.From,
		To:   &ec.GatewayAddress,
		Data: executeDataBytes,
	})
	gasPrice := uint64(0)
	if ec.auth.GasPrice != nil {
		gasPrice = ec.auth.GasPrice.Uint64()
	}
	log.Debug().Uint64("Estimated gas", gas).
		Uint64("Config gas limit", ec.auth.GasLimit).
		Uint64("Gas price", gasPrice).Msg("[EvmClient] successfully estimate gas")
	ec.auth.GasLimit = gas
	if err != nil {
		log.Error().Err(err).
			Str("BatchedCommandID", batchedCmdRes.ID).
			Str("GatewayAddress", ec.GatewayAddress.Hex()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned] failed to estimate gas of the batched command ")
		return err
	}

	//Todo: check if input is not yet broadcasted to the chain
	//Get signed tx only then try check if the tx is already mined, then get the receipt and process the event
	//If signed tx is not mined, then send the tx to the network
	ec.auth.NoSend = true
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	ec.auth.NoSend = false
	log.Debug().Uint64("Tx gas", signedTx.Gas()).
		Uint64("Tx gas price", signedTx.GasPrice().Uint64()).
		Msg("[EvmClient] successfully estimate gas")
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.EvmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned]")
		return err
	} else {
		//Try find tx on the chain
		_, isPending, err := ec.Client.TransactionByHash(context.Background(), signedTx.Hash())
		if err != nil {
			err = ec.Client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				log.Error().Err(err).
					Str("txHash", signedTx.Hash().String()).
					Msg("[EvmClient] [handleScalarBatchCommandSigned] failed to send tx to the network")
				return err
			} else {
				log.Info().Str("txHash", signedTx.Hash().String()).
					Msg("[EvmClient] [handleScalarBatchCommandSigned] successfully sent tx to the network")
			}
		} else if isPending {
			log.Info().Str("txHash", signedTx.Hash().String()).
				Msg("[EvmClient] [handleScalarBatchCommandSigned] tx is pending")
		} else {
			log.Info().Str("txHash", signedTx.Hash().String()).
				Msg("[EvmClient] [handleScalarBatchCommandSigned] tx is mined")
		}
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarBatchCommandSigned] successfully sent tx to the network")
	//2. Add the transaction waiting to be mined
	// ec.pendingTxs.AddTx(signedTx.Hash().String(), time.Now())
	//3. Todo: Clearify how to update status of the batchcommand
	return nil
}

// Call ScalarGateway's execute method
// executeData is raw transaction data in hex string
// It is ready to be sent directly to the network
// If call to the contract method, then we to unpack the input arguments
// After the execute method is called, the gateway contract will emit 2 events
// 1. ContractCallApproved event -> relayer will handle this event for execute protocol's contract method
// 2. Executed event -> relayer will handle this event for create a record in the db for scanner

func (ec *EvmClient) handleScalarContractCallApproved(messageID string, executeData string) error {
	log.Debug().
		Str("messageID", messageID).
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarContractCallApproved]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.EvmConfig.GetId()).
			Msg("[EvmClient] [handleScalarContractCallApproved] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarContractCallApproved] auth is nil")
	}
	signedTx, err := ec.gatewayExecute(decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.EvmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarContractCallApproved]")
		return err
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarContractCallApproved]")
	txHash := signedTx.Hash().String()
	//2. Add the transaction waiting to be mined
	// ec.pendingTxs.AddTx(txHash, time.Now())
	//3. Update status of the event
	err = ec.dbAdapter.UpdateCallContractWithTokenExecuteHash(messageID, chains.ContractCallStatusSuccess, txHash)
	if err != nil {
		log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarContractCallApproved]")
		return err
	}
	return nil
}

func (ec *EvmClient) observeScalarExecuteData(decodedExecuteData *DecodedExecuteData) error {
	commandIds := make([]string, len(decodedExecuteData.CommandIds))
	for i, commandId := range decodedExecuteData.CommandIds {
		commandIds[i] = hex.EncodeToString(commandId[:])
	}
	log.Debug().
		Int("inputLength", len(decodedExecuteData.Input)).
		Strs("commandIds", commandIds).
		Msg("[EvmClient] [observeScalarExecuteData]")
	return nil
}

// Check if execute data is already broadcasted
// TODO: improve this function for handling other types of commands
func (ec *EvmClient) countBroadcastedCommands(decodedExecuteData *DecodedExecuteData) int64 {
	txHashes := []string{}
	for _, commandId := range decodedExecuteData.CommandIds {
		txHashes = append(txHashes, hex.EncodeToString(commandId[:]))
	}
	count, err := ec.dbAdapter.CountExecutedCommandByCommandIds(txHashes)
	if err != nil {
		log.Error().Err(err).Msg("[EvmClient] [checkBroadcasted] failed to count token sents by tx hashes")
		return 0
	}
	return count
}

func (ec *EvmClient) handleStartedSwitchPhase(event *covTypes.SwitchPhaseStarted) error {
	log.Debug().
		Str("chain", string(event.Chain)).
		Uint64("sessionSequence", event.Sequence).
		Uint8("phase", uint8(event.Phase)).
		Msg("[EvmClient] [handleStartedSwitchPhase]")

	decodedExecuteData, err := DecodeExecuteData(event.ExecuteData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	log.Info().Any("decodedExecuteData params", decodedExecuteData.Params)
	redeemPhase, err := DecodeStartedSwitchPhase(decodedExecuteData.Params[0])

	if err != nil {
		return fmt.Errorf("failed to decode redeem phase: %w", err)
	}
	//Get current phase before call evm tx
	// callOpt, err := ec.CreateCallOpts()
	// if err != nil {
	// 	return fmt.Errorf("failed to create call opts: %w", err)
	// }
	//Find out custodian group uid from the symbol execute data

	// currentPhase, err := ec.Gateway.GetSession(callOpt, redeemPhase.Symbol)
	// if err != nil {
	// 	log.Error().Err(err).
	// 		Str("input", hex.EncodeToString(decodedExecuteData.Input)).
	// 		Str("contractAddress", ec.EvmConfig.Gateway).
	// 		Str("signer", ec.auth.From.String()).
	// 		Msg("[EvmClient] [handleStartedSwitchPhase]")
	// 	return err
	// }
	// if currentPhase.Sequence != redeemPhase.Sequence || currentPhase.Phase != redeemPhase.Phase {
	// log.Debug().
	// 	Str("chainId", string(event.Chain)).
	// 	Uint64("Current session sequence", currentPhase.Sequence).
	// 	Uint8("Current phase", currentPhase.Phase).
	// 	Msg("[EvmClient] [handleStartedSwitchPhase] current phase on the gatewayis not the same as the payload. Call transaction to update it")
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.EvmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleStartedSwitchPhase]")
		return err
	} else {
		log.Info().
			Str("chainId", string(event.Chain)).
			Uint64("Session sequence", redeemPhase.Sequence).
			Uint8("phase", redeemPhase.Phase).
			Str("txHash", signedTx.Hash().String()).
			Msg("[EvmClient] [handleStartedSwitchPhase] successfully sent start switch phase tx to the network")
	}
	//}
	return nil
}
