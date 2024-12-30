package models

import (
	"time"

	"gorm.io/gorm"
)

// Store last received events from external network
type EventCheckPoint struct {
	gorm.Model
	ChainName   string `gorm:"uniqueIndex:idx_chain_event;type:varchar(255)"`
	EventName   string `gorm:"uniqueIndex:idx_chain_event;type:varchar(255)"`
	BlockNumber uint64 `gorm:"type:bigint"`
	TxHash      string `gorm:"type:varchar(255)"`
	LogIndex    uint
	EventKey    string `gorm:"type:varchar(255)"`
}

type RelayData struct {
	gorm.Model
	ID                    string  `gorm:"primaryKey;type:varchar(255)"`
	PacketSequence        *int    `gorm:"unique"`
	ExecuteHash           *string `gorm:"type:varchar(255)"`
	Status                int     `gorm:"default:0"`
	From                  string  `gorm:"type:varchar(255)"`
	To                    string  `gorm:"type:varchar(255)"`
	CallContract          *CallContract
	CallContractWithToken *CallContractWithToken
	TokenSent             *TokenSent
}

type CommandExecuted struct {
	gorm.Model
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      uint64
	LogIndex         uint
	CommandId        string
	Status           int     `gorm:"default:0"`
	ReferenceTxHash  *string `gorm:"type:varchar(255)"`
	Amount           *string `gorm:"type:varchar(255)"`
}

type CallContract struct {
	gorm.Model
	ID                   string `gorm:"primaryKey"`
	TxHash               string `gorm:"type:varchar(255)"`
	TxHex                []byte
	BlockNumber          uint64 `gorm:"default:0"`
	LogIndex             uint
	ContractAddress      string `gorm:"type:varchar(255)"`
	Amount               uint64 `gorm:"type:bigint"`
	Symbol               string `gorm:"type:varchar(255)"`
	Payload              []byte
	PayloadHash          string  `gorm:"type:varchar(255);uniqueIndex"`
	SourceAddress        string  `gorm:"type:varchar(255)"`
	StakerPublicKey      *string `gorm:"type:varchar(255)"`
	CallContractApproved *CallContractApproved
	RelayDataID          string     `gorm:"type:varchar(255)"`
	RelayData            *RelayData `gorm:"foreignKey:RelayDataID"`
}

type CallContractWithToken struct {
	CallContract
	TokenContractAddress string `gorm:"type:varchar(255)"`
	Amount               uint64 `gorm:"type:bigint"`
}

// type Approved struct {
// 	ID               string `gorm:"primaryKey;type:varchar(255)"`
// 	SourceChain      string `gorm:"type:varchar(255)"`
// 	DestinationChain string `gorm:"type:varchar(255)"`
// 	TxHash           string `gorm:"type:varchar(255)"`
// 	BlockNumber      int
// 	LogIndex         int
// 	SourceAddress    string `gorm:"type:varchar(255)"`
// 	ContractAddress  string `gorm:"type:varchar(255)"`
// 	SourceTxHash     string `gorm:"type:varchar(255)"`
// 	SourceEventIndex *big.Int
// 	PayloadHash      string `gorm:"type:varchar(255)"`
// 	Symbol           string `gorm:"type:varchar(255)"`
// 	Amount           *big.Int
// 	CommandId        string
// 	CreatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
// 	UpdatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
// }

type CallContractApproved struct {
	gorm.Model
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      uint64
	LogIndex         uint
	SourceAddress    string `gorm:"type:varchar(255)"`
	ContractAddress  string `gorm:"type:varchar(255)"`
	SourceTxHash     string `gorm:"type:varchar(255)"`
	SourceEventIndex uint64
	PayloadHash      string `gorm:"type:varchar(255)"`
	CommandId        string
	CallContractID   *string       `gorm:"type:varchar(255);unique"`
	CallContract     *CallContract `gorm:"foreignKey:CallContractID"`
	CreatedAt        time.Time     `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time     `gorm:"type:timestamp(6);default:current_timestamp(6)"`
}

type TokenSent struct {
	gorm.Model
	ID                 string `gorm:"primaryKey"`
	TxHash             string `gorm:"type:varchar(255)"`
	TxHex              []byte
	BlockNumber        uint64 `gorm:"default:0"`
	LogIndex           uint
	SourceAddress      string `gorm:"type:varchar(255)"`
	DestinationAddress string `gorm:"type:varchar(255)"`
	Amount             uint64 `gorm:"type:bigint"`
	Symbol             string `gorm:"type:varchar(255)"`
	TokenSentApproved  *TokenSentApproved
	RelayDataID        string     `gorm:"type:varchar(255)"`
	RelayData          *RelayData `gorm:"foreignKey:RelayDataID"`
}

type TokenSentApproved struct {
	gorm.Model
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      uint64
	LogIndex         uint
	SourceAddress    string `gorm:"type:varchar(255)"`
	ContractAddress  string `gorm:"type:varchar(255)"`
	SourceTxHash     string `gorm:"type:varchar(255)"`
	SourceEventIndex uint64
	PayloadHash      string `gorm:"type:varchar(255)"`
	CommandId        string
	TokenSentID      *string    `gorm:"type:varchar(255);unique"`
	TokenSent        *TokenSent `gorm:"foreignKey:CallContractID"`
	CreatedAt        time.Time  `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time  `gorm:"type:timestamp(6);default:current_timestamp(6)"`
}

type ProtocolInfo struct {
	gorm.Model
	//ID                   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ChainID              string `gorm:"column:chain_id"`               //Evm chain id
	ChainName            string `gorm:"column:chain_name;not null"`    //Evm chain name
	BTCAddressHex        string `gorm:"column:btc_address_hex"`        //Btc address
	PublicKeyHex         string `gorm:"column:public_key_hex"`         //Btc public key
	SmartContractAddress string `gorm:"column:smart_contract_address"` //Evm contract address which is extended from the IScalarExecutable
	TokenContractAddress string `gorm:"column:token_contract_address"` //Evm ERC20 token contract address
	State                bool   `gorm:"column:state"`
	ChainEndpoint        string `gorm:"column:chain_endpoint"` //Service endpoint for handling evm tx
	RPCUrl               string `gorm:"column:rpc_url"`        //Service endpoint for handling psbt signing
	AccessToken          string `gorm:"column:access_token"`   //Access token for the signing service
}
