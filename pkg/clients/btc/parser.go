package btc

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

func ParseExecuteParams(params []byte) (*types.ExecuteParams, error) {
	args, err := evm.AbiUnpack(params, "string", "string", "address", "bytes32", "bytes32", "uint256")
	if err != nil {
		return nil, fmt.Errorf("failed to parse execute params: %w", err)
	}
	return &types.ExecuteParams{
		SourceChain:      args[0].(string),
		SourceAddress:    args[1].(string),
		ContractAddress:  args[2].(common.Address),
		PayloadHash:      args[3].([32]byte),
		SourceTxHash:     args[4].([32]byte),
		SourceEventIndex: args[5].(*big.Int).Uint64(),
	}, nil
}
