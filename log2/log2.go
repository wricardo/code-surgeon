package log2

import "github.com/rs/zerolog/log"

func Debugf(format string, args ...interface{}) {
	log.Debug().Msgf(format, args...)
}
