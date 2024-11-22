package btc

import (
	"fmt"

	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

func ParseExecuteParams(params []byte) (*types.ExecuteParams, error) {
	args, err := evm.AbiUnpack(params, "string", "string", "address", "byte32", "byte32", "int")
	if err != nil {
		return nil, fmt.Errorf("failed to parse execute params: %w", err)
	}
	return &types.ExecuteParams{
		SourceChain:      args[0].(string),
		SourceAddress:    args[1].(string),
		ContractAddress:  args[2].(string),
		PayloadHash:      args[3].([32]byte),
		SourceTxHash:     args[4].([32]byte),
		SourceEventIndex: args[5].(uint64),
	}, nil
}
