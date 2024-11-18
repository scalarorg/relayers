package types

type ExecuteParams struct {
	SourceChain      string
	SourceAddress    string
	ContractAddress  string
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex uint64
}
