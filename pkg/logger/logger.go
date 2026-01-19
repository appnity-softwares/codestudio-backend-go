package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

// Init initializes the global logger
func Init(env string) {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	if env == "development" {
		// Pretty console output for development
		Log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		// JSON output for production
		Log = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}
}

// Helper functions for common log levels
func Info() *zerolog.Event {
	return Log.Info()
}

func Error() *zerolog.Event {
	return Log.Error()
}

func Warn() *zerolog.Event {
	return Log.Warn()
}

func Debug() *zerolog.Event {
	return Log.Debug()
}

func Fatal() *zerolog.Event {
	return Log.Fatal()
}
