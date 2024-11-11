package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
)

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
) (*types.Receipt, error) {
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

func (ec *EvmClient) SubmitTx(tx *types.Transaction, retryAttempt int) (*types.Receipt, error) {
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

func (ec *EvmClient) WaitForTransaction(hash string) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ec.config.TxTimeout)
	defer cancel()

	txHash := common.HexToHash(hash)
	tx, _, err := ec.Client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	return bind.WaitMined(ctx, ec.Client, tx)
}
