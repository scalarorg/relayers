package contracts_abi

var eventAbis = map[string]string{
	"ContractCallApproved": `[
		{
			"type": "event",
			"name": "ContractCallApproved",
			"inputs": [
				{"indexed": true, "name": "commandId", "type": "bytes32"},
				{"indexed": false, "name": "sourceChain", "type": "string"},
				{"indexed": false, "name": "sourceAddress", "type": "string"},
				{"indexed": true, "name": "contractAddress", "type": "address"},
				{"indexed": true, "name": "payloadHash", "type": "bytes32"},
				{"indexed": false, "name": "sourceTxHash", "type": "bytes32"},
				{"indexed": false, "name": "sourceEventIndex", "type": "uint256"}
			]
		}
	]`,
}

// GetContractCallApprovedABI returns the ABI string for the ContractCallApproved event
func GetEventABI(name string) string {
	return eventAbis[name]
}
