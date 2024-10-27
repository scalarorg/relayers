package models

import (
	"time"

	"gorm.io/gorm"
)

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
