package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
)

func (ec *EvmClient) handleContractCall(event *contracts.IAxelarGatewayContractCall) error {
	//0. Preprocess the event
	ec.preprocessContractCall(event)
	//1. Convert into a RelayData instance then store to the db
	relayData, err := ec.ContractCallEvent2RelayData(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(relayData)
	if err != nil {
		return fmt.Errorf("failed to create evm contract call: %w", err)
	}
	//2. Send to the bus
	ec.eventBus.BroadcastEvent(&types.EventEnvelope{
		EventType: events.EVENT_EVM_CONTRACT_CALL,
		Data:      event,
	})
	return nil
}
func (ec *EvmClient) preprocessContractCall(event *contracts.IAxelarGatewayContractCall) error {
	log.Info().Msgf("Start handle Contract call: %v", event)
	//Todo: validate the event
	return nil
}
func (ec *EvmClient) handleContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) error {
	//0. Preprocess the event
	ec.preprocessContractCallApproved(event)
	//1. Convert into a RelayData instance then store to the db
	contractCallApproved, err := ec.ContractCallApprovedEvent2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallApprovedEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(contractCallApproved)
	if err != nil {
		return fmt.Errorf("failed to create contract call approved: %w", err)
	}
	//2. Send to the bus
	destinationChain := extractDestChainFromEvmGwContractCallApproved(event)
	ec.eventBus.BroadcastEvent(&types.EventEnvelope{
		EventType:        events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		DestinationChain: destinationChain,
		Data:             event,
	})
	return nil
}

func (ec *EvmClient) preprocessContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) error {
	log.Info().Msgf("Start handle Contract call approved: %v", event)
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) handleCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	//0. Preprocess the event
	ec.preprocessCommandExecuted(event)
	//1. Convert into a RelayData instance then store to the db
	cmdExecuted, err := ec.CommandExecutedEvent2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert EVMExecutedEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(cmdExecuted)
	if err != nil {
		return fmt.Errorf("failed to create evm executed: %w", err)
	}
	//2. Send to the bus
	ec.eventBus.BroadcastEvent(&types.EventEnvelope{
		EventType: events.EVENT_EVM_COMMAND_EXECUTED,
		Data:      event,
	})
	return nil
}

func (ec *EvmClient) preprocessCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	log.Info().Msgf("Start handle EVM Command executed: %v", event)
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) IsCallContractExecuted(
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	contractAddress common.Address,
	payloadHash [32]byte,
) (bool, error) {
	return ec.Gateway.IsContractCallApproved(nil, commandId, sourceChain, sourceAddress, contractAddress, payloadHash)
}

func (ec *EvmClient) Execute(
	contractAddress common.Address,
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	payload []byte,
) (*ethtypes.Receipt, error) {
	executable, err := contracts.NewIAxelarExecutable(contractAddress, ec.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create executable contract: %w", err)
	}

	tx, err := executable.Execute(ec.auth, commandId, sourceChain, sourceAddress, payload)
	if err != nil {
		log.Error().Msgf("[EvmClient.Execute] Failed: %v", err)
		return nil, err
	}

	receipt, err := ec.SubmitTx(tx, 0)
	if err != nil {
		return nil, err
	}

	return receipt, nil
}

func (ec *EvmClient) SubmitTx(tx *ethtypes.Transaction, retryAttempt int) (*ethtypes.Receipt, error) {
	if retryAttempt >= ec.config.MaxRetry {
		return nil, fmt.Errorf("max retry exceeded")
	}

	// Create a new context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ec.config.TxTimeout)
	defer cancel()

	// Log transaction details
	log.Debug().
		Interface("tx", tx).
		Msg("Submitting transaction")

	// Send the transaction using the new context
	err := ec.Client.SendTransaction(ctx, tx)
	if err != nil {
		log.Error().
			Err(err).
			Str("rpcUrl", ec.config.RPCUrl).
			Str("walletAddress", ec.auth.From.String()).
			Str("to", tx.To().String()).
			Str("data", hex.EncodeToString(tx.Data())).
			Msg("[EvmClient.SubmitTx] Failed to submit transaction")

		// Sleep before retry
		time.Sleep(ec.config.RetryDelay)

		log.Debug().
			Int("attempt", retryAttempt+1).
			Msg("Retrying transaction")

		return ec.SubmitTx(tx, retryAttempt+1)
	}

	// Wait for transaction receipt using the new context
	receipt, err := bind.WaitMined(ctx, ec.Client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction receipt: %w", err)
	}

	log.Debug().
		Interface("receipt", receipt).
		Msg("Transaction receipt received")

	return receipt, nil
}

func (ec *EvmClient) WaitForTransaction(hash string) (*ethtypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ec.config.TxTimeout)
	defer cancel()

	txHash := common.HexToHash(hash)
	tx, _, err := ec.Client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	return bind.WaitMined(ctx, ec.Client, tx)
}
