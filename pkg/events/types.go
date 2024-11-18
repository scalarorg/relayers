package events

import "github.com/scalarorg/relayers/pkg/types"

const (
	CUSTODIAL_NETWORK_NAME               = "Custodial.Network"
	SCALAR_NETWORK_NAME                  = "Scalar.Network"
	EVENT_BTC_SIGNATURE_REQUESTED        = "Btc.SignatureRequested"
	EVENT_CUSTODIAL_SIGNATURES_CONFIRMED = "Custodial.SinaturesConfirmed"
	EVENT_ELECTRS_VAULT_TRANSACTION      = "Electrs.VaultTransaction"
	EVENT_SCALAR_CONTRACT_CALL_APPROVED  = "Scalar.ContractCallApproved"
	EVENT_SCALAR_COMMAND_EXECUTED        = "Scalar.CommandExecuted"
	EVENT_EVM_CONTRACT_CALL_APPROVED     = "Evm.ContractCallApproved"
	EVENT_EVM_CONTRACT_CALL              = "Evm.ContractCall"
	EVENT_EVM_COMMAND_EXECUTED           = "Evm.CommandExecuted"
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
