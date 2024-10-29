package handler

import "github.com/rs/zerolog/log"

// HandleError logs errors with a given tag
func HandleError(tag string, err error) {
	if err != nil {
		log.Error().
			Str("tag", tag).
			Err(err).
			Msg("Error occurred")
	}
}
