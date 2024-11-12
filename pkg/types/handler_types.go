package types

type BTCExecuteResult struct {
	TxID string
	Psbt string
}

type HandleEvmToCosmosEventExecuteResult struct {
	Status         Status
	PacketSequence *int64 `json:"packetSequence,omitempty"`
}
