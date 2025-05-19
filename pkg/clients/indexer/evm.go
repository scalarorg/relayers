package indexer

import (
	"gorm.io/gorm"
)

type EVMIndexer struct {
	DbClient *gorm.DB
}

func NewEVMIndexer(dbClient *gorm.DB) *EVMIndexer {
	return &EVMIndexer{
		DbClient: dbClient,
	}
}
