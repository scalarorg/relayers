package types

type BtcTransaction struct {
	EventType                   int    `json:"event_type"`
	VaultTxHashHex              string `json:"vault_tx_hash_hex"`
	VaultTxHex                  string `json:"vault_tx_hex"`
	StakerPkHex                 string `json:"staker_pk_hex"`
	FinalityProviderPkHex       string `json:"finality_provider_pk_hex"`
	StakingValue                int64  `json:"staking_value"`
	StakingStartHeight          int    `json:"staking_start_height"`
	StakingStartTimestamp       int64  `json:"staking_start_timestamp"`
	StakingOutputIndex          int    `json:"staking_output_index"`
	ChainID                     string `json:"chain_id"`
	ChainIDUserAddress          string `json:"chain_id_user_address"`
	ChainIDSmartContractAddress string `json:"chain_id_smart_contract_address"`
	AmountMinting               string `json:"amount_minting"`
	IsOverflow                  bool   `json:"is_overflow"`
}

type BtcEventTransaction struct {
	TxHash                     string         `json:"txHash"`
	LogIndex                   uint           `json:"logIndex"`
	BlockNumber                uint64         `json:"blockNumber"`
	Sender                     string         `json:"sender"`
	SourceChain                string         `json:"sourceChain"`
	DestinationChain           string         `json:"destinationChain"`
	DestinationContractAddress string         `json:"destinationContractAddress"`
	MintingAmount              string         `json:"mintingAmount"`
	Payload                    string         `json:"payload"`
	PayloadHash                string         `json:"payloadHash"`
	Args                       BtcTransaction `json:"args"`
	StakerPublicKey            string         `json:"stakerPublicKey"`
	VaultTxHex                 string         `json:"vaultTxHex"`
}
