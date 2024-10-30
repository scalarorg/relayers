package handler

import (
	"fmt"
	"strconv"
	"strings"

	"bytes"
	"encoding/base64"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/services/axelar"
	"github.com/scalarorg/relayers/pkg/services/btc"
	"github.com/scalarorg/relayers/pkg/services/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

// HandleError logs errors with a given tag
func HandleError(tag string, err error) {
	if err != nil {
		log.Error().
			Str("tag", tag).
			Err(err).
			Msg("Error occurred")
	}
}

func PrepareHandler(event interface{}, label string) {
	log.Info().
		Interface("event", event).
		Str("label", label).
		Msg("Event received")
}

func HandleEvmToCosmosConfirmEvent(vxClient *axelar.AxelarClient, executeParams types.ExecuteRequest) (*types.HandleEvmToCosmosEventExecuteResult, error) {
	idParts := strings.Split(executeParams.ID, "-")
	if len(idParts) != 2 {
		return nil, fmt.Errorf("invalid id format: %s", executeParams.ID)
	}

	hash := idParts[0]
	logIndex, err := strconv.ParseInt(idParts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid logIndex: %v", err)
	}

	log.Debug().
		Str("id", executeParams.ID).
		Str("hash", hash).
		Int64("logIndex", logIndex).
		Msg("[handleEvmToCosmosConfirmEvent] Scalar")

	routeMessageTx, err := vxClient.RouteMessageRequest(int(logIndex), hash, executeParams.Payload)
	if err != nil {
		return &types.HandleEvmToCosmosEventExecuteResult{Status: types.FAILED}, nil
	}

	if routeMessageTx == nil {
		return &types.HandleEvmToCosmosEventExecuteResult{Status: types.FAILED}, nil
	}

	isAlreadyExecuted := strings.Contains(routeMessageTx.RawLog, "already executed")
	if isAlreadyExecuted {
		log.Info().
			Str("id", executeParams.ID).
			Msg("[handleEvmToCosmosConfirmEvent] Already sent an executed tx. Marked it as success")
		return &types.HandleEvmToCosmosEventExecuteResult{Status: types.SUCCESS}, nil
	}

	log.Info().
		Str("transactionHash", routeMessageTx.TxHash).
		Msg("[handleEvmToCosmosConfirmEvent] Executed")

	packetSequence := getPacketSequenceFromExecuteTx(routeMessageTx)
	return &types.HandleEvmToCosmosEventExecuteResult{
		Status:         types.SUCCESS,
		PacketSequence: &packetSequence,
	}, nil
}

func HandleEvmToCosmosEvent(vxClient *axelar.AxelarClient, event interface{}) error {
	var sourceChain string
	var hash string

	switch e := event.(type) {
	case *evm.EvmEvent[contracts.IAxelarGatewayContractCallWithToken]:
		sourceChain = e.SourceChain
		hash = e.Hash
	case *evm.EvmEvent[contracts.IAxelarGatewayContractCall]:
		sourceChain = e.SourceChain
		hash = e.Hash
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}

	confirmTx, err := vxClient.ConfirmEvmTx(sourceChain, hash)
	if err != nil {
		return fmt.Errorf("failed to confirm evm tx: %v", err)
	}

	if confirmTx != nil {
		log.Info().
			Str("transactionHash", confirmTx.TransactionHash).
			Msg("[handleEvmToCosmosEvent] Confirmed")
	}

	return nil
}

func HandleCosmosToEvmCallContractCompleteEvent(
	evmClient *evm.EvmClient,
	event *evm.EvmEvent[contracts.IAxelarGatewayContractCallApproved],
	relayDatas []struct {
		ID      string
		Payload string
	},
) ([]types.HandleCosmosToEvmCallContractCompleteEventExecuteResult, error) {
	if len(relayDatas) == 0 {
		log.Info().
			Str("payloadHash", hex.EncodeToString(event.Args.PayloadHash[:])).
			Str("commandId", hex.EncodeToString(event.Args.CommandId[:])).
			Msg("[Scalar][Evm Execute]: Cannot find payload from given payloadHash")
		return nil, nil
	}

	var results []types.HandleCosmosToEvmCallContractCompleteEventExecuteResult

	for _, data := range relayDatas {
		if data.Payload == "" {
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
			Str("payload", data.Payload).
			Msg("[Scalar][Prepare to Execute]: Execute")

		tx, err := evmClient.Execute(
			event.Args.ContractAddress,
			event.Args.CommandId,
			event.Args.SourceChain,
			event.Args.SourceAddress,
			[]byte(data.Payload),
		)
		if err != nil || tx == nil {
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
			Interface("tx", tx).
			Msg("[Scalar][Evm Execute]: Executed")

		results = append(results, types.HandleCosmosToEvmCallContractCompleteEventExecuteResult{
			ID:     data.ID,
			Status: types.SUCCESS,
		})
	}

	return results, nil
}

func HandleCosmosToEvmApprovedEvent(
	vxClient *axelar.AxelarClient,
	evmClient *evm.EvmClient,
	event interface{},
) (*ethTypes.Transaction, error) {
	var destinationChain string

	switch e := event.(type) {
	case *types.IBCEvent[types.ContractCallSubmitted]:
		destinationChain = e.Args.DestinationChain
	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
		destinationChain = e.Args.DestinationChain
	default:
		return nil, fmt.Errorf("unsupported event type: %T", event)
	}

	pendingCommands, err := vxClient.GetPendingCommands(destinationChain)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending commands: %v", err)
	}

	log.Info().
		Interface("pendingCommands", pendingCommands).
		Msg("[Scalar][CallEvm] PendingCommands")

	if len(pendingCommands) == 0 {
		return nil, nil
	}

	signCommand, err := vxClient.SignCommands(destinationChain)
	if err != nil {
		return nil, fmt.Errorf("failed to sign commands: %v", err)
	}

	log.Debug().
		Interface("signCommand", signCommand).
		Msg("[Scalar][CallEvm] SignCommand")

	if signCommand != nil && strings.Contains(signCommand.RawLog, "failed") {
		return nil, fmt.Errorf("sign command failed: %s", signCommand.RawLog)
	}

	if signCommand == nil {
		return nil, fmt.Errorf("cannot sign command")
	}

	batchedCommandId := getBatchCommandIdFromSignTx(signCommand)

	executeData, err := vxClient.GetExecuteDataFromBatchCommands(destinationChain, batchedCommandId)
	if err != nil {
		return nil, fmt.Errorf("failed to get execute data: %v", err)
	}

	decodedExecuteData, err := decodeGatewayExecuteData(executeData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execute data: %v", err)
	}

	log.Info().
		Str("batchCommandId", batchedCommandId).
		Str("executeData", executeData).
		Interface("decodedExecuteData", decodedExecuteData).
		Msg("[Scalar][CallEvm]")

	tx, err := evmClient.GatewayExecute([]byte(executeData))
	if err != nil {
		log.Error().
			Err(err).
			Msg("[Scalar][CallEvm] Execute failed")
		return nil, err
	}

	if tx == nil {
		log.Error().
			Msg("[Scalar][CallEvm] Execute failed: tx is nil")
		return nil, nil
	}

	log.Debug().
		Str("txHash", tx.Hash().Hex()).
		Msg("[Scalar][CallEvm] Evm TxHash")

	return tx, nil
}

func HandleCosmosToBTCApprovedEvent(
	vxClient *axelar.AxelarClient,
	db *db.DatabaseClient,
	event interface{},
) (*types.BTCExecuteResult, string, error) {
	var destinationChain string

	switch e := event.(type) {
	case *types.IBCEvent[types.ContractCallSubmitted]:
		destinationChain = e.Args.DestinationChain
	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
		destinationChain = e.Args.DestinationChain
	default:
		return nil, "", fmt.Errorf("unsupported event type: %T", event)
	}

	log.Info().
		Interface("event", event).
		Msg("[handleCosmosToBTCApprovedEvent] Event")

	pendingCommands, err := vxClient.GetPendingCommands(destinationChain)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get pending commands: %v", err)
	}

	log.Info().
		Interface("pendingCommands", pendingCommands).
		Msg("[handleCosmosToBTCApprovedEvent] PendingCommands")

	if len(pendingCommands) == 0 {
		return nil, "", nil
	}

	signCommand, err := vxClient.SignCommands(destinationChain)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign commands: %v", err)
	}

	log.Debug().
		Interface("signCommand", signCommand).
		Msg("[handleCosmosToBTCApprovedEvent] SignCommand")

	if signCommand != nil && strings.Contains(signCommand.RawLog, "failed") {
		return nil, "", fmt.Errorf("sign command failed: %s", signCommand.RawLog)
	}

	if signCommand == nil {
		return nil, "", fmt.Errorf("cannot sign command")
	}

	batchedCommandId := getBatchCommandIdFromSignTx(signCommand)
	log.Info().
		Str("batchedCommandId", batchedCommandId).
		Msg("[handleCosmosToBTCApprovedEvent] BatchCommandId")

	executeData, err := vxClient.GetExecuteDataFromBatchCommands(destinationChain, batchedCommandId)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get execute data: %v", err)
	}

	log.Info().
		Interface("executeData", executeData).
		Msg("[handleCosmosToBTCApprovedEvent] BatchCommands")

	executedResult, err := HandleBTCExecute(db, executeData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to handle BTC execute: %v", err)
	}

	return executedResult, batchedCommandId, nil
}

func HandleBTCExecute(db *db.DatabaseClient, executeData string) (*types.BTCExecuteResult, error) {
	// Define the ABI
	const executeABI = `[{"inputs":[{"internalType":"bytes","name":"input","type":"bytes"}],"name":"execute","type":"function"}]`

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(executeABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}

	// Remove '0x' prefix if present
	if strings.HasPrefix(executeData, "0x") {
		executeData = executeData[2:]
	}

	// Convert hex string to bytes
	executeDataBytes, err := hex.DecodeString(executeData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execute data: %v", err)
	}

	// Decode the function input
	method, err := parsedABI.MethodById(executeDataBytes[:4])
	if err != nil {
		return nil, fmt.Errorf("failed to get method: %v", err)
	}

	// Decode the parameters
	decodedData, err := method.Inputs.Unpack(executeDataBytes[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack input: %v", err)
	}

	// Get the input bytes from the decoded data
	input := decodedData[0].([]byte)

	log.Info().
		Bytes("input", input).
		Msg("[execute] Input")

	return execute(db, string(input))
}

func HandleCosmosApprovedEvent(
	event interface{},
	vxClient *axelar.AxelarClient,
	db *db.DatabaseClient,
	evmClients []*evm.EvmClient,
	btcClients []*btc.BtcClient,
) error {
	var destinationChain string

	// Extract destination chain based on event type
	switch e := event.(type) {
	case *types.IBCEvent[types.ContractCallSubmitted]:
		destinationChain = strings.ToLower(e.Args.DestinationChain)
	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
		destinationChain = strings.ToLower(e.Args.DestinationChain)
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}

	log.Debug().
		Interface("event", event).
		Msg("[Scalar] Scalar Event")

	// Find matching EVM client
	var evmClient *evm.EvmClient
	for _, client := range evmClients {
		if strings.ToLower(client.ChainName()) == destinationChain {
			evmClient = client
			break
		}
	}

	if evmClient != nil {
		tx, err := HandleCosmosToEvmApprovedEvent(vxClient, evmClient, event)
		if err != nil {
			return fmt.Errorf("failed to handle cosmos to evm approved event: %v", err)
		}

		txHash := tx.Hash().Hex()
		if err := db.UpdateCosmosToEvmEvent(event, &txHash); err != nil {
			return fmt.Errorf("failed to update cosmos to evm event: %v", err)
		}
		return nil
	}

	// Find matching BTC clients
	var btcBroadcastClient, btcSignerClient *btc.BtcClient
	for _, client := range btcClients {
		if client.Config().Name == destinationChain {
			if client.IsBroadcast() {
				btcBroadcastClient = client
			}
			if client.IsSigner() {
				btcSignerClient = client
			}
		}
	}

	if btcBroadcastClient != nil && btcSignerClient != nil {
		result, batchedCommandId, err := HandleCosmosToBTCApprovedEvent(vxClient, db, event)
		if err != nil {
			return fmt.Errorf("failed to handle cosmos to BTC approved event: %v", err)
		}

		if result == nil {
			log.Error().Msg("[handleCosmosApprovedEvent] Failed to execute BTC Tx: result is nil")
			return fmt.Errorf("failed to execute BTC tx: result is nil")
		}

		log.Info().
			Interface("executedResult", result).
			Msg("[handleCosmosApprovedEvent] ExecutedResult")

		log.Info().
			Str("tx", result.TxID).
			Msg("[BTC Tx Executed] BTC Tx")

		// Use btcutil to decode PSBT
		psbtBytes, err := base64.StdEncoding.DecodeString(result.Psbt)
		if err != nil {
			return fmt.Errorf("failed to decode PSBT base64: %v", err)
		}

		var psbt wire.MsgTx
		if err := psbt.Deserialize(bytes.NewReader(psbtBytes)); err != nil {
			return fmt.Errorf("failed to deserialize PSBT: %v", err)
		}

		// Get the first input's hash
		txInputHash := hex.EncodeToString(psbt.TxIn[0].PreviousOutPoint.Hash[:])
		refTxHash := addHexPrefix(txInputHash)

		log.Info().
			Str("refTxHash", refTxHash).
			Msg("[handleCosmosApprovedEvent] RefTxHash")

		var receipt *btcjson.GetTransactionResult
		receipt, err = btcBroadcastClient.GetTransaction(result.TxID)
		if err != nil {
			log.Error().Msg("Error when fetching btc tx from testnet node")

			receipt, err = btc.GetMempoolTx(result.TxID, btcBroadcastClient.Config().Network)
			if err != nil {
				log.Error().Msg("Failed to retrieve mempool transaction data")
				return fmt.Errorf("failed to get transaction receipt: %v", err)
			}
			log.Info().
				Interface("receipt", receipt).
				Msg("Mempool transaction")
		}

		if receipt == nil {
			return fmt.Errorf("not found btc receipt tx")
		}

		if err := db.HandleMultipleEvmToBtcEventsTx(event, receipt, refTxHash, batchedCommandId); err != nil {
			return fmt.Errorf("failed to handle multiple evm to btc events tx: %v", err)
		}

		log.Info().
			Interface("receipt", receipt).
			Msg("[BTC Tx Executed] BTC Receipt")

		return nil
	}

	return fmt.Errorf("no client found for chainId: %s", destinationChain)
}
