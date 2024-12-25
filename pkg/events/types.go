package events

import "github.com/scalarorg/relayers/pkg/types"

const (
	CUSTODIAL_NETWORK_NAME               = "Custodial.Network"
	SCALAR_NETWORK_NAME                  = "Scalar.Network"
	EVENT_BTC_SIGNATURE_REQUESTED        = "Btc.SignatureRequested"
	EVENT_CUSTODIAL_SIGNATURES_CONFIRMED = "Custodial.SignaturesConfirmed"
	EVENT_ELECTRS_VAULT_TRANSACTION      = "Electrs.VaultTransaction"
	EVENT_SCALAR_DEST_CALL_APPROVED      = "Scalar.DestCallApproved"
	EVENT_SCALAR_COMMAND_EXECUTED        = "Scalar.CommandExecuted"
	EVENT_EVM_CONTRACT_CALL_APPROVED     = "ContractCallApproved"
	EVENT_EVM_CONTRACT_CALL              = "ContractCall"
	EVENT_EVM_COMMAND_EXECUTED           = "Executed"
)

type EventEnvelope struct {
	DestinationChain string      // The source chain of the event
	EventType        string      // The name of the event in format "ComponentName.EventName"
	MessageID        string      // The message id of the event used add RelayData'id
	Data             interface{} // The actual event data
}
type SignatureRequest struct {
	ExecuteParams *types.ExecuteParams
	Base64Psbt    string
}
type ConfirmTxsRequest struct {
	ChainName string
	TxHashs   map[string]string //Map txHash to DestinationChain, user for validate destination chain
}
