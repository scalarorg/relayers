package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
)

func AbiUnpack(data []byte, types ...byte) ([]interface{}, error) {
	var arguments abi.Arguments
	for _, t := range types {
		arguments = append(arguments, abi.Argument{Type: abi.Type{T: t}})
	}
	args, err := arguments.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to get arguments: %w", err)
	}
	return args, nil
}

func AbiUnpackIntoMap(v map[string]interface{}, data []byte, types ...byte) error {
	var arguments abi.Arguments
	for _, t := range types {
		arguments = append(arguments, abi.Argument{Type: abi.Type{T: t}})
	}
	err := arguments.UnpackIntoMap(v, data)
	if err != nil {
		return fmt.Errorf("failed to get arguments: %w", err)
	}
	return nil
}

func DecodeExecuteData(executeData string) (*DecodedExecuteData, error) {
	executeDataBytes := []byte(executeData)
	abi, err := getScalarGwExecuteAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse abi: %w", err)
	}
	execute := struct {
		input []byte
	}{}
	err = abi.UnpackIntoInterface(&executeData, "execute", executeDataBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack execute data: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [observeScalarContractCallApproved] execute.input: %v", execute.input)
	//Decode the input
	args, err := AbiUnpack(execute.input, ethabi.BytesTy, ethabi.BytesTy)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack execute input: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [observeScalarContractCallApproved] decodedInput: %v", args)
	dataDecoded, err := AbiUnpack(args[0].([]byte), ethabi.UintTy, ethabi.FixedBytesTy, ethabi.StringTy, ethabi.BytesTy)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack data: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [observeScalarContractCallApproved] decodedData: %v", dataDecoded)
	proofDecoded, err := AbiUnpack(args[1].([]byte), ethabi.ArrayTy, ethabi.ArrayTy, ethabi.UintTy, ethabi.BytesTy)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack proof: %w", err)
	}
	log.Debug().Msgf("[EvmClient] [observeScalarContractCallApproved] proofDecoded: %v", proofDecoded)
	return &DecodedExecuteData{
		ChainId:    dataDecoded[0].(uint64),
		CommandIds: dataDecoded[1].([]Byte32),
		Commands:   dataDecoded[2].([]string),
		Params:     dataDecoded[3].([]Bytes),
		Operators:  proofDecoded[0].([]string),
		Weights:    proofDecoded[1].([]uint64),
		Threshold:  proofDecoded[2].(uint64),
		Signatures: proofDecoded[3].([]Bytes),
	}, nil
}
