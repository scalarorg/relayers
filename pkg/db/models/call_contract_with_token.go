package models

import "time"

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
