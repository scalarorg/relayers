package models

type Operatorship struct {
	ID   int `gorm:"primaryKey;autoIncrement"`
	Hash string
}
