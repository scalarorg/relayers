package types

import "github.com/ethereum/go-ethereum/core/types"

type EvmEvent[T any] struct {
	Hash             string
	BlockNumber      uint64
	LogIndex         uint
	SourceChain      string
	DestinationChain string
	WaitForFinality  func() (*types.Receipt, error)
	Args             T
}
