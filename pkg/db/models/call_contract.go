package models

import (
	"time"
)

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
