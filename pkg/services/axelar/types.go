package axelar

import sdk "github.com/cosmos/cosmos-sdk/types"

// AxelarListenerEvent represents an event with generic type T for parsed events
type AxelarListenerEvent[T any] struct {
	TopicID    string
	Type       string
	ParseEvent func(events map[string][]string) (T, error)
}

// ContractCallSubmitted represents a contract call submission event
type ContractCallSubmitted struct {
	MessageID        string `json:"messageId"`
	Sender           string `json:"sender"`
	SourceChain      string `json:"sourceChain"`
	DestinationChain string `json:"destinationChain"`
	ContractAddress  string `json:"contractAddress"`
	Payload          string `json:"payload"`
	PayloadHash      string `json:"payloadHash"`
}

// ContractCallWithTokenSubmitted represents a contract call with token submission event
type ContractCallWithTokenSubmitted struct {
	MessageID        string `json:"messageId"`
	Sender           string `json:"sender"`
	SourceChain      string `json:"sourceChain"`
	DestinationChain string `json:"destinationChain"`
	ContractAddress  string `json:"contractAddress"`
	Payload          string `json:"payload"`
	PayloadHash      string `json:"payloadHash"`
	Symbol           string `json:"symbol"`
	Amount           string `json:"amount"`
}

// ExecuteRequest represents an execute request
type ExecuteRequest struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

// IBCEvent represents a generic IBC event with generic type T for Args
type IBCEvent[T any] struct {
	Hash        string `json:"hash"`
	SrcChannel  string `json:"srcChannel,omitempty"`
	DestChannel string `json:"destChannel,omitempty"`
	Args        T      `json:"args"`
}

// IBCPacketEvent represents an IBC packet event
type IBCPacketEvent struct {
	Hash        string      `json:"hash"`
	SrcChannel  string      `json:"srcChannel"`
	DestChannel string      `json:"destChannel"`
	Denom       string      `json:"denom"`
	Amount      string      `json:"amount"`
	Sequence    int         `json:"sequence"`
	Memo        interface{} `json:"memo"`
}

// ------ Payloads ------
// TODO: USING COSMOS SDK TO DEFINE THESE TYPES LATER
const (
	EvmProtobufPackage    = "axelar.evm.v1beta1"
	AxelarProtobufPackage = "axelar.axelarnet.v1beta1"
)

// Fee represents a fee structure with amount and recipient
type Fee struct {
	Amount    *sdk.Coin `json:"amount,omitempty"` // Optional amount field using Cosmos SDK Coin type
	Recipient []byte    `json:"recipient"`        // Recipient as byte array
}

// ConfirmGatewayTxRequest represents a request to confirm a gateway transaction
type ConfirmGatewayTxRequest struct {
	Sender []byte `json:"sender"`
	Chain  string `json:"chain"`
	TxID   []byte `json:"txId"`
}

// CallContractRequest represents a request to call a contract
type CallContractRequest struct {
	Sender          []byte `json:"sender"`
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Payload         []byte `json:"payload"`
	Fee             *Fee   `json:"fee,omitempty"`
}

// RouteMessageRequest represents a request to route a message
type RouteMessageRequest struct {
	Sender  []byte `json:"sender"`
	ID      string `json:"id"`
	Payload []byte `json:"payload"`
}

// SignCommandsRequest represents a request to sign commands
type SignCommandsRequest struct {
	Sender []byte `json:"sender"`
	Chain  string `json:"chain"`
}