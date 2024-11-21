package evm

import (
	"encoding/hex"
	"fmt"
	"math/big"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func AbiUnpack(data []byte, types ...string) ([]interface{}, error) {
	var arguments ethabi.Arguments
	for _, t := range types {
		typ, err := ethabi.NewType(t, t, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create type: %w", err)
		}
		arguments = append(arguments, ethabi.Argument{Type: typ})
	}
	args, err := arguments.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to get arguments: %w", err)
	}
	return args, nil
}

func AbiUnpackIntoMap(v map[string]interface{}, data []byte, types ...byte) error {
	var arguments ethabi.Arguments
	for _, t := range types {
		arguments = append(arguments, ethabi.Argument{Type: ethabi.Type{T: t}})
	}
	err := arguments.UnpackIntoMap(v, data)
	if err != nil {
		return fmt.Errorf("failed to get arguments: %w", err)
	}
	return nil
}

func DecodeExecuteData(executeData string) (*DecodedExecuteData, error) {
	executeDataBytes, err := hex.DecodeString(executeData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode execute data: %w", err)
	}

	//First 4 bytes are the function selector
	input, err := AbiUnpack(executeDataBytes[4:], "bytes")
	if err != nil {
		log.Debug().Msgf("[EvmClient] [DecodeExecuteData] unpack executeData error: %v", err)
	}
	log.Debug().Msgf("[EvmClient] [DecodeExecuteData] successfull decoded input")
	//Decode the input
	args, err := AbiUnpack(input[0].([]byte), "bytes", "bytes")
	if err != nil {
		return nil, fmt.Errorf("failed to unpack execute input: %w", err)
	}
	//Decode the data
	dataDecoded, err := AbiUnpack(args[0].([]byte), "uint256", "bytes32[]", "string[]", "bytes[]")
	if err != nil {
		return nil, fmt.Errorf("failed to unpack data: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [DecodeExecuteData] decodedData: %v", dataDecoded)
	//Decode the proof
	proofDecoded, err := AbiUnpack(args[1].([]byte), "address[]", "uint256[]", "uint256", "bytes[]")
	if err != nil {
		return nil, fmt.Errorf("failed to unpack proof: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [DecodeExecuteData] proofDecoded: %v", proofDecoded)
	chainId := dataDecoded[0].(*big.Int)
	commandIds := dataDecoded[1].([][32]byte)
	// commandIds := make([]string, len(commandIdsBytes))
	// for i, commandId := range commandIdsBytes {
	// 	commandIds[i] = hex.EncodeToString(commandId[:])
	// }
	commands := dataDecoded[2].([]string)
	params := dataDecoded[3].([][]byte)
	// params := make([]string, len(paramsBytes))
	// for i, param := range paramsBytes {
	// 	params[i] = hex.EncodeToString(param)
	// }
	weights := proofDecoded[1].([]*big.Int)
	weightsUint64 := make([]uint64, len(weights))
	for i, weight := range weights {
		weightsUint64[i] = weight.Uint64()
	}
	threshold := proofDecoded[2].(*big.Int)
	signaturesBytes := proofDecoded[3].([][]byte)
	signatures := make([]string, len(signaturesBytes))
	for i, signature := range signaturesBytes {
		signatures[i] = hex.EncodeToString(signature)
	}
	return &DecodedExecuteData{
		Input:      input[0].([]byte),
		ChainId:    chainId.Uint64(),
		CommandIds: commandIds,
		Commands:   commands,
		Params:     params,
		Operators:  proofDecoded[0].([]common.Address),
		Weights:    weightsUint64,
		Threshold:  threshold.Uint64(),
		Signatures: signatures,
	}, nil
}
