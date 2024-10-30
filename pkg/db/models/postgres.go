package models

import (
	"math/big"
	"time"

	"gorm.io/gorm"
)

type Operatorship struct {
	ID   int `gorm:"primaryKey;autoIncrement"`
	Hash string
}

type RelayData struct {
	gorm.Model
	ID                    string    `gorm:"primaryKey;type:varchar(255)"`
	PacketSequence        *int      `gorm:"unique"`
	ExecuteHash           *string   `gorm:"type:varchar(255)"`
	Status                int       `gorm:"default:0"`
	From                  string    `gorm:"type:varchar(255)"`
	To                    string    `gorm:"type:varchar(255)"`
	CreatedAt             time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt             time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	CallContract          *CallContract
	CallContractWithToken *CallContractWithToken
}

type CommandExecuted struct {
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      int
	LogIndex         int
	CommandId        string
	Status           int       `gorm:"default:0"`
	ReferenceTxHash  *string   `gorm:"type:varchar(255)"`
	Amount           *string   `gorm:"type:varchar(255)"`
	CreatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
}

type CallContract struct {
	ID                   string `gorm:"primaryKey;type:varchar(255)"`
	TxHash               string `gorm:"type:varchar(255)"`
	TxHex                []byte
	BlockNumber          *int `gorm:"default:0"`
	LogIndex             *int
	ContractAddress      string  `gorm:"type:varchar(255)"`
	Amount               *string `gorm:"type:varchar(255)"`
	Symbol               *string `gorm:"type:varchar(255)"`
	Payload              []byte
	PayloadHash          string    `gorm:"type:varchar(255);uniqueIndex"`
	SourceAddress        string    `gorm:"type:varchar(255)"`
	StakerPublicKey      *string   `gorm:"type:varchar(255)"`
	SenderAddress        *string   `gorm:"type:varchar(255)"`
	CreatedAt            time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt            time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	CallContractApproved *CallContractApproved
	RelayDataID          string    `gorm:"type:varchar(255)"`
	RelayData            RelayData `gorm:"foreignKey:RelayDataID"`
}

type CallContractWithToken struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(255)"`
	BlockNumber     *int      `json:"blockNumber" gorm:"default:0"`
	ContractAddress string    `json:"contractAddress" gorm:"type:varchar(255)"`
	Amount          string    `json:"amount" gorm:"type:varchar(255)"`
	Symbol          string    `json:"symbol" gorm:"type:varchar(255)"`
	Payload         string    `json:"payload"`
	PayloadHash     string    `json:"payloadHash" gorm:"type:varchar(255)"`
	SourceAddress   string    `json:"sourceAddress" gorm:"type:varchar(255)"`
	CreatedAt       time.Time `json:"createdAt" gorm:"autoCreateTime;type:timestamp(6)"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"autoUpdateTime;type:timestamp(6)"`
	RelayData       RelayData `json:"relayData" gorm:"foreignKey:ID"`
}

type Approved struct {
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      int
	LogIndex         int
	SourceAddress    string `gorm:"type:varchar(255)"`
	ContractAddress  string `gorm:"type:varchar(255)"`
	SourceTxHash     string `gorm:"type:varchar(255)"`
	SourceEventIndex *big.Int
	PayloadHash      string `gorm:"type:varchar(255)"`
	Symbol           string `gorm:"type:varchar(255)"`
	Amount           *big.Int
	CommandId        string
	CreatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
}

type CallContractApproved struct {
	ID               string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain      string `gorm:"type:varchar(255)"`
	DestinationChain string `gorm:"type:varchar(255)"`
	TxHash           string `gorm:"type:varchar(255)"`
	BlockNumber      int
	LogIndex         int
	SourceAddress    string `gorm:"type:varchar(255)"`
	ContractAddress  string `gorm:"type:varchar(255)"`
	SourceTxHash     string `gorm:"type:varchar(255)"`
	SourceEventIndex int
	PayloadHash      string `gorm:"type:varchar(255)"`
	CommandId        string
	CallContractID   *string       `gorm:"type:varchar(255);unique"`
	CallContract     *CallContract `gorm:"foreignKey:CallContractID"`
	CreatedAt        time.Time     `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time     `gorm:"type:timestamp(6);default:current_timestamp(6)"`
}
