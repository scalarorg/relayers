package rabbitmq

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/types"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/services/axelar"
)

var RabbitMQClient *Client

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func InitRabbitMQClient() error {
	var err error
	RabbitMQClient, err = NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ client: %w", err)
	}
	return nil
}

func NewClient() (*Client, error) {
	connectionString := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		config.GlobalConfig.RabbitMQ.User,
		config.GlobalConfig.RabbitMQ.Password,
		config.GlobalConfig.RabbitMQ.Host,
		strconv.Itoa(config.GlobalConfig.RabbitMQ.Port))

	conn, err := amqp.Dial(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	args := amqp.Table{
		"x-queue-type":              "quorum",
		"x-dead-letter-exchange":    "common_dlx",
		"x-dead-letter-routing-key": config.GlobalConfig.RabbitMQ.RoutingKey,
	}

	q, err := ch.QueueDeclare(
		config.GlobalConfig.RabbitMQ.Queue, // name
		true,                               // durable
		false,                              // delete when unused
		false,                              // exclusive
		false,                              // no-wait
		args,                               // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &Client{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (c *Client) Consume() error {
	msgs, err := c.channel.Consume(
		c.queue.Name,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		for msg := range msgs {
			log.Debug().Bytes("content", msg.Body).Msg("[RabbitMQ] Received message")

			var btcTx types.BtcTransaction
			err := json.Unmarshal(msg.Body, &btcTx)
			if err != nil {
				log.Error().Err(err).Msg("[RabbitMQ] Failed to unmarshal message")
				msg.Nack(false, false) // Negative acknowledgement, don't requeue
				continue
			}

			log.Debug().Interface("transaction", btcTx).Msg("[RabbitMQ] Received BTC Event")

			// Check if we should stop consuming based on height
			if config.GlobalConfig.RabbitMQ.StopHeight != nil && *config.GlobalConfig.RabbitMQ.StopHeight > 0 && btcTx.StakingStartHeight > int(*config.GlobalConfig.RabbitMQ.StopHeight) {
				log.Info().
					Int64("stop_height", *config.GlobalConfig.RabbitMQ.StopHeight).
					Int("current_height", btcTx.StakingStartHeight).
					Msg("[RabbitMQ] Stop consume at height")
				msg.Ack(false)
				continue
			}

			// Create BTC event object
			btcEvent, err := CreateBtcEventTransaction(&btcTx)
			if err != nil {
				log.Error().Err(err).Msg("[RabbitMQ] Failed to create BTC event")
				msg.Nack(false, true) // Requeue the message
				continue
			}

			log.Debug().Interface("event", btcEvent).Msg("[RabbitMQ] Created BTC Event")

			// Create the event in the database
			err = db.CreateBtcCallContractEvent(btcEvent)
			if err != nil {
				log.Error().Err(err).Msg("[RabbitMQ] Failed to create BTC event in database")
				msg.Nack(false, true) // Requeue the message
				continue
			}

			// Handle the event by sending the confirm tx to the axelar network
			err = handleBtcEvent(btcEvent, &msg)
			if err != nil {
				log.Error().Err(err).Msg("[RabbitMQ] Failed to handle BTC event")
				msg.Nack(false, true) // Requeue the message
				continue
			}

			msg.Ack(false) // Acknowledge the message
		}
	}()

	return nil
}

func (c *Client) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

// Helper functions (you'll need to implement these)

func CreateBtcEventTransaction(btcTransaction *types.BtcTransaction) (*types.BtcEventTransaction, error) {
	log.Debug().Interface("btcTransaction", btcTransaction).Msg("[RabbitMQ][BTC Event Parser]")

	toAddressHex, err := base64ToHex(btcTransaction.ChainIDUserAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to decode user address: %w", err)
	}

	amountDecode, err := base64ToDecimal(btcTransaction.AmountMinting)
	if err != nil {
		return nil, fmt.Errorf("failed to decode minting amount: %w", err)
	}

	chainId, err := base64ToDecimal(btcTransaction.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode chain ID: %w", err)
	}

	toAddress := common.HexToAddress(toAddressHex)
	amount := new(big.Int)
	amount.SetString(amountDecode, 10)
	blockTime := uint64(btcTransaction.StakingStartTimestamp)

	// Create ABI arguments
	arguments := abi.Arguments{
		{Type: abi.Type{T: abi.AddressTy}},
		{Type: abi.Type{T: abi.UintTy, Size: 256}},
		{Type: abi.Type{T: abi.UintTy, Size: 64}},
	}

	// Pack the arguments
	payload, err := arguments.Pack(toAddress, amount, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to pack payload: %w", err)
	}

	log.Debug().Str("payload", hex.EncodeToString(payload)).Msg("[RabbitMQ][BTC Event Parser]")

	payloadHash := crypto.Keccak256Hash(payload)

	log.Debug().Str("payloadHash", payloadHash.Hex()).Msg("[RabbitMQ][BTC Event Parser]")

	destinationChain, err := GetChainNameById(chainId)
	if err != nil {
		return nil, fmt.Errorf("destination chain not found: %s", chainId)
	}

	chainIDSmartContractAddressHex, err := base64ToHex(btcTransaction.ChainIDSmartContractAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to decode smart contract address: %w", err)
	}

	btcEvent := &types.BtcEventTransaction{
		TxHash:                     "0x" + strings.ToLower(btcTransaction.VaultTxHashHex),
		LogIndex:                   0,
		BlockNumber:                uint64(btcTransaction.StakingStartHeight),
		MintingAmount:              amount.String(),
		Sender:                     toAddress.Hex(),
		SourceChain:                config.GlobalConfig.RabbitMQ.SourceChain,
		DestinationChain:           destinationChain,
		DestinationContractAddress: chainIDSmartContractAddressHex,
		Payload:                    hex.EncodeToString(payload),
		PayloadHash:                payloadHash.Hex(),
		Args:                       *btcTransaction,
		StakerPublicKey:            btcTransaction.StakerPkHex,
		VaultTxHex:                 btcTransaction.VaultTxHex,
	}

	return btcEvent, nil
}

func base64ToHex(b64 string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func base64ToDecimal(b64 string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return new(big.Int).SetBytes(bytes).String(), nil
}

// Implement GetChainNameById function
func GetChainNameById(chainId string) (string, error) {
	for _, network := range config.GlobalConfig.EvmNetworks {
		if network.ChainID == chainId {
			return network.Name, nil
		}
	}
	return "", fmt.Errorf("chain name not found for chainId: %s", chainId)
}

func handleBtcEvent(btcEvent *types.BtcEventTransaction, msg *amqp.Delivery) error {
	confirmTx, err := axelar.AxelarService.ConfirmEvmTx(btcEvent.SourceChain, btcEvent.TxHash)
	if err != nil {
		return fmt.Errorf("failed to confirm BTC Event: %w", err)
	}

	if confirmTx != nil {
		log.Info().
			Str("tx_hash", btcEvent.TxHash).
			Str("source_chain", btcEvent.SourceChain).
			Str("confirm_tx_hash", "0x"+confirmTx.TransactionHash).
			Msg("[RabbitMQ][Scalar]: Confirmed BTC Event")
		msg.Ack(false)
	} else {
		log.Error().
			Str("tx_hash", btcEvent.TxHash).
			Msg("[RabbitMQ][Scalar]: Failed to confirm BTC Event")
		msg.Nack(false, false)
	}

	return nil
}
