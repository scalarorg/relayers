package models

import (
	"time"
)

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
