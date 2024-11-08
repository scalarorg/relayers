package models

import (
	"time"

	"gorm.io/gorm"
)

type LastBlock struct {
	ChainName   string `gorm:"primaryKey;type:varchar(255)"`
	BlockNumber int64  `gorm:"type:bigint"`
}

type Operatorship struct {
	ID   int `gorm:"primaryKey;autoIncrement"`
	Hash string
}

type RelayData struct {
	gorm.Model
	ID             string    `gorm:"primaryKey;type:varchar(255)"`
	PacketSequence *int      `gorm:"unique"`
	ExecuteHash    *string   `gorm:"type:varchar(255)"`
	Status         int       `gorm:"default:0"`
	From           string    `gorm:"type:varchar(255)"`
	To             string    `gorm:"type:varchar(255)"`
	CreatedAt      time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt      time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	CallContract   *CallContract
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
