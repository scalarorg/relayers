package types

import "github.com/ethereum/go-ethereum/common"

type ExecuteParams struct {
	SourceChain      string
	SourceAddress    string
	ContractAddress  common.Address
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex uint64
}
