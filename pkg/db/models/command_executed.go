package models

import (
	"time"
)

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
