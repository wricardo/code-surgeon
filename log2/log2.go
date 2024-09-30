package log2

import (
	"io"
	"os"

	stdlog "log"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Debugf(format string, args ...interface{}) {
	log.Debug().Msgf(format, args...)
}

func Configure() {
	// use UNIX Time, which is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	writers := []io.Writer{}

	if false {
		writers = append(writers, os.Stderr)
	}

	// if we are in local development, we want to write to the console
	if true {
		f, err := os.OpenFile("main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open log file")
		}
		consoleWriter := zerolog.ConsoleWriter{Out: f}
		writers = append(writers, &consoleWriter)
	}

	level := zerolog.DebugLevel
	if false {
		level = zerolog.TraceLevel
	}
	if false {
		level = zerolog.InfoLevel
	}

	// configure the global logger
	logger := zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp().Caller().Logger().Level(level)
	zerolog.DefaultContextLogger = &logger
	log.Logger = logger

	zerolog.SetGlobalLevel(level)

	// this only shows the file name and line, not the full path
	// zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
	// 	return filepath.Base(file) + ":" + strconv.Itoa(line)
	// }

	// if in production, set the log level to info

	// configure stdlog to write to the zerolog logger just in case some libraries use stdlog
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}
