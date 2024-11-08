package evm

import (
	"encoding/hex"

	"github.com/rs/zerolog/log"
	evm_clients "github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

func (ea *EvmAdapter) handleCosmosToEvmCallContractCompleteEvent(
	evmClient *evm_clients.EvmClient,
	data types.HandleCosmosToEvmCallContractCompleteEventData,
) ([]types.HandleCosmosToEvmCallContractCompleteEventExecuteResult, error) {
	relayDatas := data.RelayDatas
	event := data.Event

	if len(relayDatas) == 0 {
		log.Info().
			Str("payloadHash", hex.EncodeToString(event.Args.PayloadHash[:])).
			Str("commandId", hex.EncodeToString(event.Args.CommandId[:])).
			Msg("[Scalar][Evm Execute]: Cannot find payload from given payloadHash")
		return nil, nil
	}

	var results []types.HandleCosmosToEvmCallContractCompleteEventExecuteResult

	for idx, data := range relayDatas {
		log.Info().
			Str("payload", hex.EncodeToString(data.Payload)).
			Msgf("[Scalar][Evm Execute]: check data %d", idx)
		if len(data.Payload) == 0 {
			continue
		}

		// Check if already executed
		isExecuted, err := evmClient.IsCallContractExecuted(
			event.Args.CommandId,
			event.Args.SourceChain,
			event.Args.SourceAddress,
			event.Args.ContractAddress,
			event.Args.PayloadHash,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to check if contract is executed")
			continue
		}

		if isExecuted {
			results = append(results, types.HandleCosmosToEvmCallContractCompleteEventExecuteResult{
				ID:     data.ID,
				Status: types.SUCCESS,
			})
			log.Info().
				Str("txId", data.ID).
				Str("commandId", hex.EncodeToString(event.Args.CommandId[:])).
				Msg("[Scalar][Evm Execute]: Already executed txId. Will mark the status in the DB as Success")
			continue
		}

		log.Debug().
			Str("contractAddress", event.Args.ContractAddress.String()).
			Str("commandId", hex.EncodeToString(event.Args.CommandId[:])).
			Str("sourceChain", event.Args.SourceChain).
			Str("sourceAddress", event.Args.SourceAddress).
			Str("payload", hex.EncodeToString(data.Payload)).
			Msg("[Scalar][Prepare to Execute]: Execute")

		receipt, err := evmClient.Execute(
			event.Args.ContractAddress,
			event.Args.CommandId,
			event.Args.SourceChain,
			event.Args.SourceAddress,
			[]byte(data.Payload),
		)
		if err != nil || receipt == nil {
			results = append(results, types.HandleCosmosToEvmCallContractCompleteEventExecuteResult{
				ID:     data.ID,
				Status: types.FAILED,
			})
			log.Error().
				Str("id", data.ID).
				Err(err).
				Msg("[Scalar][Evm Execute]: Execute failed. Will mark the status in the DB as Failed")
			continue
		}

		log.Info().
			Interface("receipt", receipt).
			Msg("[Scalar][Evm Execute]: Executed")

		results = append(results, types.HandleCosmosToEvmCallContractCompleteEventExecuteResult{
			ID:     data.ID,
			Status: types.SUCCESS,
		})
	}

	return results, nil
}

func (ea *EvmAdapter) handleWaitForTransaction(evmClient *evm_clients.EvmClient, hash string) {
	receipt, err := evmClient.WaitForTransaction(hash)
	if err != nil {
		log.Error().Err(err).Str("hash", hash).Msg("[EvmAdapter] Failed to wait for transaction")
	}
	log.Info().Interface("receipt", receipt).Msg("[EvmAdapter] Transaction confirmed")
}
