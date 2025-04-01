package events

import "github.com/scalarorg/relayers/pkg/types"

const (
	CUSTODIAL_NETWORK_NAME        = "Custodial.Network"
	SCALAR_NETWORK_NAME           = "Scalar.Network"
	EVENT_BTC_SIGNATURE_REQUESTED = "Btc.SignatureRequested"
	//EVENT_BTC_PSBT_SIGN_REQUEST          = "Btc.PsbtSignRequest"
	EVENT_CUSTODIAL_SIGNATURES_CONFIRMED = "Custodial.SignaturesConfirmed"
	EVENT_ELECTRS_VAULT_TRANSACTION      = "Electrs.VaultTransaction"
	EVENT_ELECTRS_REDEEM_TRANSACTION     = "Electrs.RedeemTransaction"
	EVENT_ELECTRS_NEW_BLOCK              = "Electrs.NewBlock"
	EVENT_SCALAR_TOKEN_SENT              = "Scalar.TokenSent"
	EVENT_SCALAR_DEST_CALL_APPROVED      = "Scalar.ContractCallApproved"
	EVENT_SCALAR_BATCHCOMMAND_SIGNED     = "Scalar.BatchCommandSigned"
	EVENT_SCALAR_COMMAND_EXECUTED        = "Scalar.CommandExecuted"
	EVENT_SCALAR_CREATE_PSBT_REQUEST     = "Scalar.CreatePsbtRequest"
	EVENT_SCALAR_SWITCH_PHASE_STARTED    = "Scalar.StartedSwitchPhase"
	EVENT_EVM_CONTRACT_CALL_APPROVED     = "ContractCallApproved"
	EVENT_EVM_CONTRACT_CALL              = "ContractCall"
	EVENT_EVM_CONTRACT_CALL_WITH_TOKEN   = "ContractCallWithToken"
	EVENT_EVM_TOKEN_SENT                 = "TokenSent"
	EVENT_EVM_COMMAND_EXECUTED           = "Executed"
	EVENT_EVM_TOKEN_DEPLOYED             = "TokenDeployed"
	EVENT_EVM_SWITCHED_PHASE             = "SwitchPhase"
)

type EventEnvelope struct {
	DestinationChain string      // The source chain of the event
	EventType        string      // The name of the event in format "ComponentName.EventName"
	MessageID        string      // The message id of the event used add RelayData'id
	CommandIDs       []string    // The command ids
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

type ConfirmRedeemTxRequest struct {
	Chain   string
	TxHashs []string
}
type ChainBlockHeight struct {
	Chain  string
	Height uint64
}
