package types

import (
	"github.com/btcsuite/btcd/btcutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/scalarorg/bitcoin-vault/ffi/go-vault"
	address "github.com/scalarorg/bitcoin-vault/go-utils/address"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
)

type ExecuteParams struct {
	SourceChain      string
	SourceAddress    string
	ContractAddress  common.Address
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex uint64
}

// AxelarListenerEvent represents an event with generic type T for parsed events
type ScalarListenerEvent[T any] struct {
	TopicID    string
	Type       string
	ParseEvent func(events map[string][]string) (T, error)
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
	ChainsProtobufPackage = "scalar.chains.v1beta1"
	AxelarProtobufPackage = "scalar.scalarnet.v1beta1"
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

type PsbtParams struct {
	ScalarTag      []byte
	Version        uint8
	ProtocolTag    []byte
	NetworkKind    vault.NetworkKind //mainnet, testnet
	NetworkType    string            //mainnet, testnet, testnet4, regtest
	CovenantPubKey []vault.PublicKey
	CovenantQuorum uint8
	CovenantScript []byte
}

type PsbtSigningRequest struct {
	Commands []chainstypes.QueryCommandResponse
	Params   PsbtParams
}

// Calculate taproot address from covenant pubkey and network
func (p *PsbtParams) GetTaprootAddress() (btcutil.Address, error) {
	return address.ScriptPubKeyToAddress(p.CovenantScript, p.NetworkType)
}
