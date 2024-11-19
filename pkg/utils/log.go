package utils

import "github.com/rs/zerolog/log"

func LogMap(data map[string][]string) {
	for key, value := range data {
		log.Debug().Msgf("%s=>%s", key, value)
	}
}
