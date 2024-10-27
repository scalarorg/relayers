package axelar

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

var AxelarClient *Client

// Dummy Message type
type Message struct {
	// Add fields as needed for your use case
	Content string
}

type Client struct {
	dummyClient string
}

func InitAxelarClient() error {
	var err error
	AxelarClient, err = NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Axelar client: %w", err)
	}
	return nil
}

func NewClient() (*Client, error) {
	return &Client{dummyClient: "dummyClient"}, nil
}

// Implement AxelarClient interface methods
func (c *Client) RelayMessage(ctx context.Context, msg *Message) error {
	// Implementation using c.config.RPCUrl, c.config.ChainID, etc.
	// For now, just log or print the message content
	log.Debug().Str("content", msg.Content).Msg("Relaying message")
	return nil
}

// You might want to add more methods here as needed

// ConfirmTxResponse represents the response from confirming an EVM transaction
type ConfirmTxResponse struct {
	TransactionHash string
	Status          string
}

// ConfirmEvmTx simulates confirming an EVM transaction on the Axelar network
func (c *Client) ConfirmEvmTx(sourceChain string, txHash string) (*ConfirmTxResponse, error) {
	// Log the confirmation attempt
	log.Info().
		Str("source_chain", sourceChain).
		Str("tx_hash", txHash).
		Msg("Attempting to confirm EVM transaction")

	// Simulate some processing time
	// In a real implementation, this would involve interacting with the Axelar network
	// time.Sleep(2 * time.Second)

	// Generate a dummy response
	response := &ConfirmTxResponse{
		TransactionHash: fmt.Sprintf("axelar_%s", txHash[2:]), // Remove "0x" prefix and add "axelar_" prefix
		Status:          "confirmed",
	}

	log.Info().
		Str("source_chain", sourceChain).
		Str("original_tx_hash", txHash).
		Str("confirm_tx_hash", response.TransactionHash).
		Msg("EVM transaction confirmed")

	return response, nil
}
