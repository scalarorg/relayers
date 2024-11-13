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
	"ContractCall": `[
		{
			"type": "event",
			"name": "ContractCall",
			"inputs": [
				{"indexed": true, "name": "sender", "type": "address"},
				{"indexed": false, "name": "destinationChain", "type": "string"},
				{"indexed": false, "name": "destinationContractAddress", "type": "string"},
				{"indexed": true, "name": "payloadHash", "type": "bytes32"},
				{"indexed": false, "name": "payload", "type": "bytes"}
			]
		}
	]`,
	"Executed": `[
		{
			"type": "event",
			"name": "Executed",
			"inputs": [
				{"indexed": true, "name": "commandId", "type": "bytes32"}
			]
		}
	]`,
}

// GetContractCallApprovedABI returns the ABI string for the ContractCallApproved event
// func GetEventABI(name string) string {
// 	return eventAbis[name]
// }
