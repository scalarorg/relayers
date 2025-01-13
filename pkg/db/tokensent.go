package db

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
)

// find relay datas by token sent attributes
func (db *DatabaseAdapter) FindPendingBtcTokenSent(sourceChain string, height int) ([]*chains.TokenSent, error) {
	var tokenSents []*chains.TokenSent
	result := db.PostgresClient.
		Where("source_chain = ? AND block_number <= ?",
			sourceChain,
			height).
		Where("status = ?", string(chains.TokenSentStatusPending)).
		Find(&tokenSents)

	if result.Error != nil {
		return tokenSents, fmt.Errorf("FindPendingBtcTokenSent with error: %w", result.Error)
	}
	if len(tokenSents) == 0 {
		log.Warn().
			Str("sourceChain", sourceChain).
			Msgf("[DatabaseAdapter] [FindPendingBtcTokenSent] no token sent with block height before %d found", height)
	}
	return tokenSents, nil
}

func (db *DatabaseAdapter) SaveTokenSents(tokenSents []chains.TokenSent) error {
	result := db.PostgresClient.Save(tokenSents)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
