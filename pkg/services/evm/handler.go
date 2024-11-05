package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/types"
)

func (ea *EvmAdapter) handleCosmosToEvmCallContractCompleteEvent(
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

	for _, data := range relayDatas {
		if len(data.Payload) == 0 {
			continue
		}

		// Check if already executed
		isExecuted, err := ea.IsCallContractExecuted(
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

		receipt, err := ea.Execute(
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

func (ea *EvmAdapter) IsCallContractExecuted(
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	contractAddress common.Address,
	payloadHash [32]byte,
) (bool, error) {
	return ea.gateway.IsContractCallApproved(nil, commandId, sourceChain, sourceAddress, contractAddress, payloadHash)
}

func (ea *EvmAdapter) Execute(
	contractAddress common.Address,
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	payload []byte,
) (*eth_types.Receipt, error) {
	executable, err := contracts.NewIAxelarExecutable(contractAddress, ea.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create executable contract: %w", err)
	}

	tx, err := executable.Execute(ea.auth, commandId, sourceChain, sourceAddress, payload)
	if err != nil {
		log.Error().Msgf("[EvmClient.Execute] Failed: %v", err)
		return nil, err
	}

	receipt, err := ea.submitTx(tx, 0)
	if err != nil {
		return nil, err
	}

	return receipt, nil
}

func (ea *EvmAdapter) submitTx(tx *eth_types.Transaction, retryAttempt int) (*eth_types.Receipt, error) {
	if retryAttempt >= ea.config.MaxRetry {
		return nil, fmt.Errorf("max retry exceeded")
	}

	// Create a new context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ea.config.TxTimeout)
	defer cancel()

	// Log transaction details
	log.Debug().
		Interface("tx", tx).
		Msg("Submitting transaction")

	// Send the transaction using the new context
	err := ea.client.SendTransaction(ctx, tx)
	if err != nil {
		log.Error().
			Err(err).
			Str("rpcUrl", ea.config.RPCUrl).
			Str("walletAddress", ea.auth.From.String()).
			Str("to", tx.To().String()).
			Str("data", hex.EncodeToString(tx.Data())).
			Msg("[EvmAdapter.submitTx] Failed to submit transaction")

		// Sleep before retry
		time.Sleep(ea.config.RetryDelay)

		log.Debug().
			Int("attempt", retryAttempt+1).
			Msg("Retrying transaction")

		return ea.submitTx(tx, retryAttempt+1)
	}

	// Wait for transaction receipt using the new context
	receipt, err := bind.WaitMined(ctx, ea.client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction receipt: %w", err)
	}

	log.Debug().
		Interface("receipt", receipt).
		Msg("Transaction receipt received")

	return receipt, nil
}
