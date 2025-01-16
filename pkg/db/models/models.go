package models

import (
	"time"

	"gorm.io/gorm"
)

// // TokenSentStatus represents the status of a token transfer
// type TokenSentStatus string

// const (
// 	// TokenSentStatusPending indicates the token is just received from the source chain
// 	TokenSentStatusPending TokenSentStatus = "`pending`"
// 	// TokenSentStatusVerifying indicates the transfer is broadcasting to the scalar for verification
// 	TokenSentStatusVerifying TokenSentStatus = "verifying"
// 	// TokenSentStatusApproved indicates the transfer is approved by the scalar network
// 	TokenSentStatusApproved TokenSentStatus = "approved"
// 	// TokenSentStatusSigning indicates the transfer is signing in the scalar network
// 	TokenSentStatusSigning TokenSentStatus = "signing"
// 	// TokenSentStatusExecuting indicates the transfer is executing on the destination chain
// 	TokenSentStatusExecuting TokenSentStatus = "executing"
// 	// TokenSentStatusSuccess indicates the transfer was successful executed on the destination chain
// 	TokenSentStatusSuccess TokenSentStatus = "success"
// 	// TokenSentStatusFailed indicates the transfer failed
// 	TokenSentStatusFailed TokenSentStatus = "failed"
// 	// TokenSentStatusCancelled indicates the transfer was cancelled
// 	TokenSentStatusCancelled TokenSentStatus = "cancelled"
// )

// // String converts the TokenSentStatus to a string
// func (s TokenSentStatus) String() string {
// 	return string(s)
// }

// // IsValid checks if the status is a valid TokenSentStatus
// func (s TokenSentStatus) IsValid() bool {
// 	switch s {
// 	case TokenSentStatusPending, TokenSentStatusSuccess, TokenSentStatusFailed, TokenSentStatusCancelled:
// 		return true
// 	default:
// 		return false
// 	}
// }

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

// type RelayData struct {
// 	gorm.Model
// 	ID                    string  `gorm:"primaryKey;type:varchar(255)"`
// 	PacketSequence        *int    `gorm:"unique"`
// 	ExecuteHash           *string `gorm:"type:varchar(255)"`
// 	Status                int     `gorm:"default:0"`
// 	From                  string  `gorm:"type:varchar(255)"`
// 	To                    string  `gorm:"type:varchar(255)"`
// 	CallContract          *CallContract
// 	CallContractWithToken *CallContractWithToken
// 	//TokenSent             *TokenSent
// }

type MintCommand struct {
	gorm.Model
	TxHash           string `gorm:"type:varchar(64)"`
	SourceChain      string `gorm:"type:varchar(20);not null"`
	DestinationChain string `gorm:"type:varchar(20);not null"`
	TransferID       uint64 `gorm:"type:varchar(50);not null"`
	CommandID        string `gorm:"type:varchar(64);not null"`
	Amount           int64
	Symbol           string `gorm:"type:varchar(10);not null"`
	Recipient        string `gorm:"type:varchar(64);not null"`
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
	Status           int `gorm:"default:0"`
}

type ContractCallApprovedWithMint struct {
	ContractCallApproved
	Symbol string `gorm:"type:varchar(255)"`
	Amount uint64 `gorm:"type:bigint"`
}
type ContractCallApproved struct {
	EventID          string    `gorm:"primaryKey"`
	TxHash           string    `gorm:"type:varchar(255)"`
	SourceChain      string    `gorm:"type:varchar(255)"`
	DestinationChain string    `gorm:"type:varchar(255)"`
	CommandID        string    `gorm:"type:varchar(255)"`
	Sender           string    `gorm:"type:varchar(255)"`
	ContractAddress  string    `gorm:"type:varchar(255)"`
	PayloadHash      string    `gorm:"type:varchar(255)"`
	Status           int       `gorm:"default:1"`
	SourceTxHash     string    `gorm:"type:varchar(255)"`
	SourceEventIndex uint64    `gorm:"type:bigint"`
	CreatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt        time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	DeletedAt        gorm.DeletedAt
}

// type TokenSent struct {
// 	EventID              string `gorm:"primaryKey"`
// 	TxHash               string `gorm:"type:varchar(255)"`
// 	TxHex                []byte
// 	BlockNumber          uint64 `gorm:"default:0"`
// 	LogIndex             uint
// 	SourceChain          string          `gorm:"type:varchar(64)"`
// 	SourceAddress        string          `gorm:"type:varchar(255)"`
// 	DestinationChain     string          `gorm:"type:varchar(64)"`
// 	DestinationAddress   string          `gorm:"type:varchar(255)"`
// 	TokenContractAddress string          `gorm:"type:varchar(255)"`
// 	Amount               uint64          `gorm:"type:bigint"`
// 	Symbol               string          `gorm:"type:varchar(255)"`
// 	Status               TokenSentStatus `gorm:"default:pending"`
// 	RelayDataID          string          `gorm:"type:varchar(255)"`
// 	RelayData            *RelayData      `gorm:"foreignKey:RelayDataID"`
// 	CreatedAt            time.Time       `gorm:"type:timestamp(6);default:current_timestamp(6)"`
// 	UpdatedAt            time.Time       `gorm:"type:timestamp(6);default:current_timestamp(6)"`
// 	DeletedAt            gorm.DeletedAt
// }

type TokenSentApproved struct {
	EventID            string `gorm:"primaryKey;type:varchar(255)"`
	SourceChain        string `gorm:"type:varchar(255)"`
	SourceAddress      string `gorm:"type:varchar(255)"`
	DestinationChain   string `gorm:"type:varchar(255)"`
	DestinationAddress string `gorm:"type:varchar(255)"`
	TxHash             string `gorm:"type:varchar(255)"`
	BlockNumber        uint64
	LogIndex           uint
	Amount             uint64 `gorm:"type:bigint"`
	Symbol             string `gorm:"type:varchar(255)"`
	Status             int    `gorm:"default:0"`
	ContractAddress    string `gorm:"type:varchar(255)"`
	SourceTxHash       string `gorm:"type:varchar(255)"`
	SourceEventIndex   uint64
	CommandId          string
	TransferID         uint64    `gorm:"type:bigint"`
	CreatedAt          time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	UpdatedAt          time.Time `gorm:"type:timestamp(6);default:current_timestamp(6)"`
	DeletedAt          gorm.DeletedAt
}
