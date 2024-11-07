package handler

// TODO: MIGRATE TO ADAPTERS

// import (
// 	"bytes"
// 	"encoding/json"
// 	"net/http"
// 	"strconv"

// 	"encoding/hex"
// 	"fmt"
// 	"math/big"
// 	"strings"

// 	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/rs/zerolog/log"

// 	"github.com/ethereum/go-ethereum/accounts/abi"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/crypto"
// 	"github.com/scalarorg/relayers/pkg/db"
// 	"github.com/scalarorg/relayers/pkg/types"
// )

// type GatewayExecuteData struct {
// 	Data struct {
// 		ChainID    *big.Int
// 		CommandIDs [][32]byte
// 		Commands   []string
// 		Params     [][]byte
// 	}
// 	Proof struct {
// 		Operators  []common.Address
// 		Weights    []*big.Int
// 		Threshold  *big.Int
// 		Signatures [][]byte
// 	}
// }

// func decodeGatewayExecuteData(executeData string) (*GatewayExecuteData, error) {
// 	// Remove "0x" prefix if present
// 	if strings.HasPrefix(executeData, "0x") {
// 		executeData = executeData[2:]
// 	}

// 	// Convert hex string to bytes
// 	inputBytes, err := hex.DecodeString(executeData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode execute data hex: %w", err)
// 	}

// 	// Define ABI types needed for decoding
// 	var (
// 		abiBytes   = abi.Type{T: abi.BytesTy}
// 		abiString  = abi.Type{T: abi.StringTy}
// 		abiAddress = abi.Type{T: abi.AddressTy}
// 		abiBytes32 = abi.Type{T: abi.FixedBytesTy, Size: 32}
// 		abiUint256 = abi.Type{T: abi.UintTy, Size: 256}
// 	)

// 	// First decode the input into data and proof bytes
// 	var data, proof []byte
// 	unpacked, err := abi.Arguments{
// 		{Type: abiBytes}, // data
// 		{Type: abiBytes}, // proof
// 	}.Unpack(inputBytes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode input into data and proof: %w", err)
// 	}
// 	data = unpacked[0].([]byte)
// 	proof = unpacked[1].([]byte)

// 	// Decode data portion
// 	dataUnpacked, err := abi.Arguments{
// 		{Type: abiUint256}, // chainId
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiBytes32}}, // commandIds
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiString}},  // commands
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiBytes}},   // params
// 	}.Unpack(data)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode data portion: %w", err)
// 	}

// 	// Decode proof portion
// 	proofUnpacked, err := abi.Arguments{
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiAddress}}, // operators
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiUint256}}, // weights
// 		{Type: abiUint256}, // threshold
// 		{Type: abi.Type{T: abi.ArrayTy, Elem: &abiBytes}}, // signatures
// 	}.Unpack(proof)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode proof portion: %w", err)
// 	}

// 	// Construct result
// 	result := &GatewayExecuteData{}

// 	// Set Data fields
// 	result.Data.ChainID = dataUnpacked[0].(*big.Int)
// 	result.Data.CommandIDs = dataUnpacked[1].([][32]byte)
// 	result.Data.Commands = dataUnpacked[2].([]string)
// 	result.Data.Params = dataUnpacked[3].([][]byte)

// 	// Set Proof fields
// 	result.Proof.Operators = proofUnpacked[0].([]common.Address)
// 	result.Proof.Weights = proofUnpacked[1].([]*big.Int)
// 	result.Proof.Threshold = proofUnpacked[2].(*big.Int)
// 	result.Proof.Signatures = proofUnpacked[3].([][]byte)

// 	return result, nil
// }

// func getPacketSequenceFromExecuteTx(executeTx *cosmosTypes.TxResponse) int64 {
// 	var rawLog []map[string]interface{}
// 	if err := json.Unmarshal([]byte(executeTx.RawLog), &rawLog); err != nil {
// 		log.Error().Err(err).Msg("failed to unmarshal raw log")
// 		return 0
// 	}

// 	if len(rawLog) == 0 {
// 		return 0
// 	}

// 	events, ok := rawLog[0]["events"].([]interface{})
// 	if !ok {
// 		return 0
// 	}

// 	for _, event := range events {
// 		eventMap, ok := event.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		if eventMap["type"] == "send_packet" {
// 			attributes, ok := eventMap["attributes"].([]interface{})
// 			if !ok {
// 				continue
// 			}

// 			for _, attr := range attributes {
// 				attrMap, ok := attr.(map[string]interface{})
// 				if !ok {
// 					continue
// 				}

// 				if attrMap["key"] == "packet_sequence" {
// 					seq, err := strconv.ParseInt(attrMap["value"].(string), 10, 64)
// 					if err != nil {
// 						log.Error().Err(err).Msg("failed to parse packet sequence")
// 						return 0
// 					}
// 					return seq
// 				}
// 			}
// 		}
// 	}

// 	return 0
// }

// func getBatchCommandIdFromSignTx(signTx *cosmosTypes.TxResponse) string {
// 	var rawLog []map[string]interface{}
// 	if err := json.Unmarshal([]byte(signTx.RawLog), &rawLog); err != nil {
// 		log.Error().Err(err).Msg("failed to unmarshal raw log")
// 		return ""
// 	}

// 	if len(rawLog) == 0 {
// 		return ""
// 	}

// 	events, ok := rawLog[0]["events"].([]interface{})
// 	if !ok {
// 		return ""
// 	}

// 	for _, event := range events {
// 		eventMap, ok := event.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		if eventMap["type"] == "sign" {
// 			attributes, ok := eventMap["attributes"].([]interface{})
// 			if !ok {
// 				continue
// 			}

// 			for _, attr := range attributes {
// 				attrMap, ok := attr.(map[string]interface{})
// 				if !ok {
// 					continue
// 				}

// 				if attrMap["key"] == "batchedCommandID" {
// 					if value, ok := attrMap["value"].(string); ok {
// 						return value
// 					}
// 				}
// 			}
// 		}
// 	}

// 	return ""
// }

// func execute(dbClient *db.DatabaseClient, executeData string) (*types.BTCExecuteResult, error) {
// 	// Additional ABI type constants needed
// 	var (
// 		abiBytes   = abi.Type{T: abi.BytesTy}
// 		abiString  = abi.Type{T: abi.StringTy}
// 		abiAddress = abi.Type{T: abi.AddressTy}
// 		abiBytes32 = abi.Type{T: abi.FixedBytesTy, Size: 32}
// 		abiUint256 = abi.Type{T: abi.UintTy, Size: 256}
// 	)

// 	// Remove "0x" prefix if present
// 	if strings.HasPrefix(executeData, "0x") {
// 		executeData = executeData[2:]
// 	}

// 	// Convert hex string to bytes
// 	inputBytes, err := hex.DecodeString(executeData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode execute data hex: %w", err)
// 	}

// 	// First decode the input into data and proof bytes
// 	var data, proof []byte
// 	unpacked, err := abi.Arguments{
// 		{Type: abiBytes}, // data
// 		{Type: abiBytes}, // proof
// 	}.Unpack(inputBytes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode input into data and proof: %w", err)
// 	}
// 	data = unpacked[0].([]byte)
// 	proof = unpacked[1].([]byte)

// 	log.Info().
// 		Hex("data", data).
// 		Hex("proof", proof).
// 		Msg("decoded input data and proof")

// 	// Now decode the data portion
// 	gatewayData, err := decodeGatewayExecuteData(hex.EncodeToString(data))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode gateway data: %w", err)
// 	}

// 	// Validate chain ID
// 	if gatewayData.Data.ChainID.Cmp(big.NewInt(2)) != 0 {
// 		return nil, fmt.Errorf("invalid chain ID")
// 	}

// 	// Validate command lengths
// 	commandsLength := len(gatewayData.Data.CommandIDs)
// 	if commandsLength != len(gatewayData.Data.Commands) || commandsLength != len(gatewayData.Data.Params) {
// 		return nil, fmt.Errorf("invalid commands length")
// 	}

// 	for i := 0; i < commandsLength; i++ {
// 		// Hash the command
// 		commandHash := crypto.Keccak256Hash([]byte(gatewayData.Data.Commands[i]))
// 		if commandHash != crypto.Keccak256Hash([]byte("approveContractCall")) {
// 			continue
// 		}

// 		// Decode params
// 		var (
// 			sourceChain      string
// 			sourceAddress    string
// 			contractAddress  common.Address
// 			payloadHash      [32]byte
// 			sourceTxHash     [32]byte
// 			sourceEventIndex *big.Int
// 		)

// 		// Create ABI arguments for decoding params
// 		arguments := abi.Arguments{
// 			{Type: abiString},  // sourceChain
// 			{Type: abiString},  // sourceAddress
// 			{Type: abiAddress}, // contractAddress
// 			{Type: abiBytes32}, // payloadHash
// 			{Type: abiBytes32}, // sourceTxHash
// 			{Type: abiUint256}, // sourceEventIndex
// 		}

// 		values, err := arguments.Unpack(gatewayData.Data.Params[i])
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to decode params: %w", err)
// 		}

// 		sourceChain = values[0].(string)
// 		sourceAddress = values[1].(string)
// 		contractAddress = values[2].(common.Address)
// 		payloadHash = values[3].([32]byte)
// 		sourceTxHash = values[4].([32]byte)
// 		sourceEventIndex = values[5].(*big.Int)

// 		log.Info().
// 			Str("sourceChain", sourceChain).
// 			Str("sourceAddress", sourceAddress).
// 			Str("contractAddress", contractAddress.Hex()).
// 			Str("payloadHash", hex.EncodeToString(payloadHash[:])).
// 			Str("sourceTxHash", hex.EncodeToString(sourceTxHash[:])).
// 			Str("sourceEventIndex", sourceEventIndex.String()).
// 			Msg("execute btc tx params decoded")

// 		// Get burning tx from database
// 		burningPsbtEncode, err := dbClient.GetBurningTx(hex.EncodeToString(payloadHash[:]))
// 		if err != nil {
// 			return nil, fmt.Errorf("burning PSBT not found: %w", err)
// 		}

// 		// Decode burning PSBT
// 		if !strings.HasPrefix(burningPsbtEncode, "0x") {
// 			burningPsbtEncode = "0x" + burningPsbtEncode
// 		}

// 		decodedBytes, err := hex.DecodeString(strings.TrimPrefix(burningPsbtEncode, "0x"))
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to decode burning PSBT: %w", err)
// 		}

// 		// Decode the string from the bytes
// 		burningPsbtDecoded := string(decodedBytes)

// 		log.Info().
// 			Str("burningPsbtEncoded", burningPsbtEncode).
// 			Str("burningPsbtDecoded", burningPsbtDecoded).
// 			Msg("burning PSBT decoded")

// 		// Process burning transactions
// 		txID, err := submitPsbt(dbClient, burningPsbtDecoded, sourceChain, hex.EncodeToString(sourceTxHash[:]))
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to process burning txs: %w", err)
// 		}

// 		log.Info().
// 			Str("tx", txID).
// 			Str("psbtBase64", burningPsbtDecoded).
// 			Msg("successfully processed burning PSBT")

// 		return &types.BTCExecuteResult{
// 			TxID: txID,
// 			Psbt: burningPsbtDecoded,
// 		}, nil
// 	}

// 	return nil, nil
// }

// func submitPsbt(dbClient *db.DatabaseClient, psbtb64, sourceChain, sourceTxHash string) (string, error) {
// 	// Find chain configuration from MongoDB
// 	chainConfig, err := dbClient.GetChainConfig(sourceChain)
// 	if err != nil {
// 		return "", fmt.Errorf("chain config not found: %w", err)
// 	}

// 	// Prepare request body
// 	reqBody := map[string]string{
// 		"evm_chain_name": sourceChain,
// 		"evm_tx_id":      sourceTxHash,
// 		"unbonding_psbt": psbtb64,
// 	}

// 	jsonBody, err := json.Marshal(reqBody)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to marshal request body: %w", err)
// 	}

// 	// Create HTTP request
// 	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/sign-unbonding-tx", chainConfig.RpcUrl), bytes.NewBuffer(jsonBody))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create request: %w", err)
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", chainConfig.AccessToken))

// 	// Send request
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to send request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Parse response
// 	var response struct {
// 		Data struct {
// 			TxID string `json:"tx_id"`
// 		} `json:"data"`
// 	}

// 	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
// 		return "", fmt.Errorf("failed to decode response: %w", err)
// 	}

// 	log.Info().
// 		Interface("response", response).
// 		Msg("Submit PSBT response received")

// 	return response.Data.TxID, nil
// }

// // Helper function to reverse a byte slice
// func reverseBytes(b []byte) []byte {
// 	reversed := make([]byte, len(b))
// 	for i := 0; i < len(b); i++ {
// 		reversed[i] = b[len(b)-1-i]
// 	}
// 	return reversed
// }

// // Helper function to add 0x prefix to hex string if not present
// func addHexPrefix(s string) string {
// 	if strings.HasPrefix(s, "0x") {
// 		return s
// 	}
// 	return "0x" + s
// }
