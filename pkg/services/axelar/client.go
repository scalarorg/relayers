package axelar

import (
	"context"
	"fmt"
	"hash/sha256"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
)

type AxelarClient struct {
	signer *SignerClient
}

func NewClient(signer *SignerClient) (*AxelarClient, error) {
	return &AxelarClient{
		signer: signer,
	}, nil
}

// ConfirmTxResponse represents the response from confirming an EVM transaction
type ConfirmTxResponse struct {
	TransactionHash string
	Status          string
}

// ConfirmEvmTx simulates confirming an EVM transaction on the Axelar network
func (c *AxelarClient) ConfirmEvmTx(sourceChain string, txHash string) (*ConfirmTxResponse, error) {
	// Get payload for confirming gateway tx
	address, err := c.signer.GetAddress()
	if err != nil {
		return nil, err
	}
	msgs, err := GetConfirmGatewayTxPayload(address, sourceChain, txHash)
	if err != nil {
		return nil, err
	}

	// Broadcast the transaction
	resp, err := c.signer.Broadcast(msgs, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to broadcast ConfirmEvmTx")
		return nil, err
	}

	return &ConfirmTxResponse{
		TransactionHash: resp.TxHash,
		Status:          "confirmed",
	}, nil
}

// CallContract sends a contract call request
func (c *AxelarClient) CallContract(ctx context.Context, chain string, contractAddress string, payload string) error {
	address, err := c.signer.GetAddress()
	if err != nil {
		return err
	}
	msgs, err := GetCallContractRequest(address, chain, contractAddress, payload)
	if err != nil {
		return err
	}

	_, err = c.signer.Broadcast(msgs, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to broadcast CallContract")
		return err
	}

	return nil
}

// GetPendingCommands retrieves pending commands for a specific chain
func (c *AxelarClient) GetPendingCommands(ctx context.Context, chain string) ([]string, error) {
	// Implementation would involve querying the Axelar network for pending commands
	return []string{}, nil
}

// SignCommands signs pending commands for a specific chain
func (c *AxelarClient) SignCommands(ctx context.Context, chain string) error {
	address, err := c.signer.GetAddress()
	if err != nil {
		return err
	}
	msgs, err := GetSignCommandPayload(address, chain)
	if err != nil {
		return err
	}

	_, err = c.signer.Broadcast(msgs, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to broadcast SignCommands")
		return err
	}

	return nil
}

// GetExecuteDataFromBatchCommands retrieves execution data for batched commands
func (c *AxelarClient) GetExecuteDataFromBatchCommands(ctx context.Context, chain string, id string) (string, error) {
	// Implementation would involve polling the Axelar network until status is 3
	// and then returning the execute data
	return "", nil
}

// RouteMessageRequest sends a route message request
func (c *AxelarClient) RouteMessageRequest(ctx context.Context, logIndex int, txHash string, payload string) error {
	address, err := c.signer.GetAddress()
	if err != nil {
		return err
	}
	msgs, err := GetRouteMessageRequest(address, txHash, logIndex, payload)
	if err != nil {
		return err
	}

	_, err = c.signer.Broadcast(msgs, "")
	if err != nil {
		if strings.Contains(err.Error(), "already executed") {
			log.Error().
				Str("tx_hash", txHash).
				Int("log_index", logIndex).
				Msg("Already executed")
			return nil
		}
		log.Error().Err(err).Msg("Failed to broadcast RouteMessageRequest")
		return err
	}

	return nil
}

// SetFee sets the fee for the client
func (c *AxelarClient) SetFee(fee string) {
	c.signer.fee = fee
}

// CalculateTokenIBCPath calculates the IBC path hash for tokens
func (c *AxelarClient) CalculateTokenIBCPath(destChannelID, denom string, port string) (string, error) {
	if port == "" {
		port = "transfer"
	}
	path := fmt.Sprintf("%s/%s/%s", port, destChannelID, denom)
	hash := sha256.Sum256([]byte(path))
	return fmt.Sprintf("0x%x", hash), nil
}

// GetBalance gets the balance for an address
func (c *AxelarClient) GetBalance(address string, denom string) (sdk.Coin, error) {
	return c.signer.GetBalance(address, denom)
}

// IsTxSuccess checks if a transaction was successful
func (c *AxelarClient) IsTxSuccess(response *ConfirmTxResponse) bool {
	return response != nil && response.Status == "confirmed"
}

// IsTxFailed checks if a transaction failed
func (c *AxelarClient) IsTxFailed(response *ConfirmTxResponse) bool {
	return response == nil || response.Status != "confirmed"
}
