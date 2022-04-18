// Logging package for tunnel server

package lumber

import "github.com/LambdaTest/test-at-scale/pkg/errs"

// LoggingConfig stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type LoggingConfig struct {
	EnableConsole     bool
	ConsoleJSONFormat bool
	ConsoleLevel      string
	EnableFile        bool
	FileJSONFormat    bool
	FileLevel         string
	FileLocation      string
}

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

const (
	// Debug has verbose message
	Debug = "debug"
	// Info is default log level
	Info = "info"
	// Warn is for logging messages about possible issues
	Warn = "warn"
	// Error is for logging errors
	Error = "error"
	// Fatal is for logging fatal messages. The system shutsdown after logging the message.
	Fatal = "fatal"
)

// List of supported loggers.
const (
	InstanceZapLogger int = iota
	InstanceLogrusLogger
)

// Logger is our contract for the logger
type Logger interface {
	// Debugf logs a message at level Debug on the standard logger.
	Debugf(format string, args ...interface{})
	// Infof logs a message at level Info on the standard logger.
	Infof(format string, args ...interface{})
	// Warnf logs a message at level Warn on the standard logger.
	Warnf(format string, args ...interface{})
	// Errorf logs a message at level Error on the standard logger.
	Errorf(format string, args ...interface{})
	// Fatalf logs a message at level Fatal on the standard logger then the process will exit with status set to 1.
	Fatalf(format string, args ...interface{})
	// Panicf logs a message at level Panic on the standard logger.
	Panicf(format string, args ...interface{})
	// WithField creates an entry from the standard logger and adds a field to
	// it. If you want multiple fields, use `WithFields`
	// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
	// or Panic on the Entry it returns.
	WithFields(keyValues Fields) Logger
}

// NewLogger returns an instance of logger
func NewLogger(config LoggingConfig, verbose bool, loggerInstance int) (Logger, error) {
	switch loggerInstance {
	case InstanceZapLogger:
		logger := newZapLogger(config, verbose)
		return logger, nil

	case InstanceLogrusLogger:
		logger, err := newLogrusLogger(config, verbose)
		if err != nil {
			return nil, err
		}
		return logger, nil

	default:
		return nil, errs.ErrInvalidLoggerInstance
	}
}
